package handler

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/rs/zerolog/log"

	"netdispatch/internal/connmgr"
	"netdispatch/internal/egress"
)

// SOCKS5Handler handles SOCKS5 proxy requests
type SOCKS5Handler struct {
	ctx     *Context
	noAuth  bool
	users   map[string]string
}

// NewSOCKS5Handler creates a new SOCKS5 handler
func NewSOCKS5Handler(ctx *Context, users map[string]string) *SOCKS5Handler {
	return &SOCKS5Handler{
		ctx:    ctx,
		noAuth: len(users) == 0,
		users:  users,
	}
}

// Handle handles an incoming SOCKS5 connection
func (h *SOCKS5Handler) Handle(conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()

	if err := h.handleGreeting(conn); err != nil {
		log.Debug().Err(err).Str("client", clientAddr).Msg("SOCKS5 greeting failed")
		return
	}

	targetAddr, err := h.handleRequest(conn)
	if err != nil {
		log.Debug().Err(err).Str("client", clientAddr).Msg("SOCKS5 request failed")
		return
	}

	connRecord := h.ctx.ConnMgr.Create(clientAddr, targetAddr, "socks5")
	if connRecord == nil {
		log.Error().Str("client", clientAddr).Str("target", targetAddr).Msg("Failed to create connection record")
		h.sendReply(conn, 0x05, nil)
		return
	}
	defer h.ctx.ConnMgr.Close(connRecord.ID)

	egressPolicy, err := h.ctx.EgressMgr.Select(targetAddr)
	if err != nil {
		log.Error().Err(err).Str("target", targetAddr).Msg("Failed to select egress policy")
		h.sendReply(conn, 0x05, nil)
		return
	}
	if egressPolicy == nil {
		log.Error().Str("target", targetAddr).Msg("No egress policy available")
		h.sendReply(conn, 0x05, nil)
		return
	}

	h.ctx.ConnMgr.Update(connRecord.ID, egressPolicy.ID, egressPolicy.NIC, egressPolicy.Proxy != nil, "")

	var targetConn net.Conn

	if egressPolicy.Proxy != nil {
		targetConn, err = h.dialViaProxy(egressPolicy, targetAddr)
	} else {
		targetConn, err = h.ctx.Dialer.Dial("tcp", targetAddr, egressPolicy.NIC)
	}

	if err != nil {
		log.Error().Err(err).Str("target", targetAddr).Msg("Failed to connect to target")
		h.sendReply(conn, 0x05, nil)
		return
	}
	defer targetConn.Close()

	h.sendReply(conn, 0x00, targetConn.LocalAddr())

	bytesIn, bytesOut := connmgr.CopyTraffic(conn, targetConn)
	h.ctx.ConnMgr.AddBytes(connRecord.ID, bytesIn, bytesOut)

	log.Info().
		Str("client", clientAddr).
		Str("target", targetAddr).
		Str("nic", connRecord.NIC).
		Bool("proxy", connRecord.ProxyUsed).
		Int64("bytes_in", bytesIn).
		Int64("bytes_out", bytesOut).
		Msg("SOCKS5 connection closed")
}

func (h *SOCKS5Handler) handleGreeting(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != 0x05 {
		return fmt.Errorf("invalid SOCKS version: %d", buf[0])
	}

	nMethods := int(buf[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	if h.noAuth {
		if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
			return err
		}
		return nil
	}

	for _, m := range methods {
		if m == 0x02 {
			if _, err := conn.Write([]byte{0x05, 0x02}); err != nil {
				return err
			}
			return h.handleAuth(conn)
		}
	}

	conn.Write([]byte{0x05, 0xff})
	return fmt.Errorf("no acceptable auth method")
}

func (h *SOCKS5Handler) handleAuth(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	ulen := int(buf[1])
	user := make([]byte, ulen)
	if _, err := io.ReadFull(conn, user); err != nil {
		return err
	}

	// Read password length
	plenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, plenBuf); err != nil {
		return err
	}
	plen := int(plenBuf[0])
	pass := make([]byte, plen)
	if _, err := io.ReadFull(conn, pass); err != nil {
		return err
	}

	expected, ok := h.users[string(user)]
	if !ok || expected != string(pass) {
		conn.Write([]byte{0x01, 0x01})
		return fmt.Errorf("auth failed")
	}

	_, err := conn.Write([]byte{0x01, 0x00})
	return err
}

func (h *SOCKS5Handler) handleRequest(conn net.Conn) (string, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != 0x05 {
		return "", fmt.Errorf("invalid SOCKS version: %d", buf[0])
	}

	if buf[1] != 0x01 {
		return "", fmt.Errorf("unsupported command: %d", buf[1])
	}

	var host string
	var port uint16

	switch buf[3] {
	case 0x01:
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(conn, ipv4); err != nil {
			return "", err
		}
		host = net.IP(ipv4).String()

	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)

	case 0x04:
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(conn, ipv6); err != nil {
			return "", err
		}
		host = net.IP(ipv6).String()

	default:
		return "", fmt.Errorf("unsupported address type: %d", buf[3])
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port = binary.BigEndian.Uint16(portBuf)

	return fmt.Sprintf("%s:%d", host, port), nil
}

func (h *SOCKS5Handler) sendReply(conn net.Conn, status byte, localAddr net.Addr) {
	var reply []byte

	if localAddr != nil {
		if addr, ok := localAddr.(*net.TCPAddr); ok && addr != nil {
			if addr.IP.To4() != nil {
				// IPv4
				reply = make([]byte, 10)
				reply[3] = 0x01
				copy(reply[4:8], addr.IP.To4())
				binary.BigEndian.PutUint16(reply[8:10], uint16(addr.Port))
			} else {
				// IPv6
				reply = make([]byte, 22)
				reply[3] = 0x04
				copy(reply[4:20], addr.IP)
				binary.BigEndian.PutUint16(reply[20:22], uint16(addr.Port))
			}
		}
	}

	if reply == nil {
		reply = make([]byte, 10)
		reply[3] = 0x01
	}

	reply[0] = 0x05
	reply[1] = status
	reply[2] = 0x00

	if _, err := conn.Write(reply); err != nil {
		log.Debug().Err(err).Msg("Failed to send SOCKS5 reply")
	}
}

func (h *SOCKS5Handler) dialViaProxy(policy *egress.Policy, targetAddr string) (net.Conn, error) {
	if policy == nil || policy.Proxy == nil {
		return nil, fmt.Errorf("invalid egress policy: nil policy or proxy")
	}

	proxyAddr := fmt.Sprintf("%s:%d", policy.Proxy.Host, policy.Proxy.Port)

	var proxyConn net.Conn
	var err error

	if policy.NIC != "" {
		proxyConn, err = h.ctx.Dialer.Dial("tcp", proxyAddr, policy.NIC)
	} else {
		proxyConn, err = net.Dial("tcp", proxyAddr)
	}

	if err != nil {
		return nil, err
	}

	if policy.Proxy.Protocol == "socks5" {
		return h.socks5Handshake(proxyConn, policy.Proxy, targetAddr)
	}

	// For HTTP proxy, send CONNECT request
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)

	// Add basic auth if credentials are provided
	if policy.Proxy.Username != "" && policy.Proxy.Password != "" {
		auth := policy.Proxy.Username + ":" + policy.Proxy.Password
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	connectReq += "\r\n"

	if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to send CONNECT: %w", err)
	}

	// Use PeekReader to preserve buffered data
	peekReader := NewPeekReader(proxyConn)

	// Read response line
	line, err := peekReader.ReadLine()
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	if !strings.Contains(line, "200") {
		proxyConn.Close()
		return nil, fmt.Errorf("proxy connect failed: %s", strings.TrimSpace(line))
	}

	// Skip remaining headers
	if err := peekReader.SkipHeaders(); err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read proxy headers: %w", err)
	}

	// Return connection with any buffered data preserved
	return peekReader.BufferedConn(), nil
}

func (h *SOCKS5Handler) socks5Handshake(conn net.Conn, proxy *egress.ProxyConfig, targetAddr string) (net.Conn, error) {
	host, portStr, _ := net.SplitHostPort(targetAddr)
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	if err := h.sendSocks5Greeting(conn, proxy.Username, proxy.Password); err != nil {
		conn.Close()
		return nil, err
	}

	if err := h.sendSocks5Connect(conn, host, port); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (h *SOCKS5Handler) sendSocks5Greeting(conn net.Conn, username, password string) error {
	var req []byte
	if username != "" && password != "" {
		// Only support authentication method
		req = []byte{0x05, 0x01, 0x02}
	} else {
		req = []byte{0x05, 0x01, 0x00}
	}

	if _, err := conn.Write(req); err != nil {
		return err
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}

	if resp[0] != 0x05 {
		return fmt.Errorf("invalid socks version: %d", resp[0])
	}

	if username != "" && password != "" {
		// Must select authentication method
		if resp[1] != 0x02 {
			return fmt.Errorf("server did not select authentication method: %d", resp[1])
		}

		authReq := make([]byte, 3+len(username)+len(password))
		authReq[0] = 0x01
		authReq[1] = byte(len(username))
		copy(authReq[2:], username)
		authReq[2+len(username)] = byte(len(password))
		copy(authReq[3+len(username):], password)

		if _, err := conn.Write(authReq); err != nil {
			return err
		}

		authResp := make([]byte, 2)
		if _, err := io.ReadFull(conn, authResp); err != nil {
			return err
		}

		if authResp[1] != 0x00 {
			return fmt.Errorf("socks5 auth failed")
		}
	} else {
		if resp[1] != 0x00 {
			return fmt.Errorf("server rejected no-auth method: %d", resp[1])
		}
	}

	return nil
}

func (h *SOCKS5Handler) sendSocks5Connect(conn net.Conn, host string, port int) error {
	req := make([]byte, 7+len(host))
	req[0] = 0x05
	req[1] = 0x01
	req[2] = 0x00
	req[3] = 0x03
	req[4] = byte(len(host))
	copy(req[5:], host)
	req[5+len(host)] = byte(port >> 8)
	req[6+len(host)] = byte(port & 0xff)

	if _, err := conn.Write(req); err != nil {
		return err
	}

	// Read response header (first 4 bytes)
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != 0x05 {
		return fmt.Errorf("invalid socks version: %d", header[0])
	}

	if header[1] != 0x00 {
		return fmt.Errorf("socks5 connect failed: %d", header[1])
	}

	// Read remaining address based on address type
	switch header[3] {
	case 0x01: // IPv4
		_, err := io.ReadFull(conn, make([]byte, 4+2)) // 4 bytes IP + 2 bytes port
		return err
	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return err
		}
		_, err := io.ReadFull(conn, make([]byte, int(lenBuf[0])+2)) // domain + port
		return err
	case 0x04: // IPv6
		_, err := io.ReadFull(conn, make([]byte, 16+2)) // 16 bytes IP + 2 bytes port
		return err
	default:
		return fmt.Errorf("unknown address type: %d", header[3])
	}
}
