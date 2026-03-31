package handler

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"

	"netdispatch/internal/connmgr"
	"netdispatch/internal/egress"
)

// HTTPHandler handles HTTP/HTTPS proxy requests
type HTTPHandler struct {
	ctx *Context
}

// countWriter wraps an io.Writer and counts bytes written
type countWriter struct {
	writer io.Writer
	count  *int64
}

func (cw *countWriter) Write(p []byte) (int, error) {
	n, err := cw.writer.Write(p)
	*cw.count += int64(n)
	return n, err
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(ctx *Context) *HTTPHandler {
	return &HTTPHandler{ctx: ctx}
}

// Handle handles an incoming connection
func (h *HTTPHandler) Handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read HTTP request")
		return
	}

	clientAddr := conn.RemoteAddr().String()
	targetAddr := req.URL.Host

	if !strings.Contains(targetAddr, ":") {
		if req.Method == http.MethodConnect {
			targetAddr += ":443"
		} else {
			targetAddr += ":80"
		}
	}

	connRecord := h.ctx.ConnMgr.Create(clientAddr, targetAddr, "http")
	if connRecord == nil {
		log.Error().Str("client", clientAddr).Str("target", targetAddr).Msg("Failed to create connection record")
		conn.Close()
		return
	}
	defer h.ctx.ConnMgr.Close(connRecord.ID)

	if req.Method == http.MethodConnect {
		h.handleConnect(conn, connRecord, targetAddr)
	} else {
		h.handleHTTP(conn, connRecord, req, targetAddr)
	}
}

func (h *HTTPHandler) handleConnect(conn net.Conn, connRecord *connmgr.Connection, targetAddr string) {
	egressPolicy, err := h.ctx.EgressMgr.Select(targetAddr)
	if err != nil {
		log.Error().Err(err).Str("target", targetAddr).Msg("Failed to select egress policy")
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	if egressPolicy == nil {
		log.Error().Str("target", targetAddr).Msg("No egress policy available")
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
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
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to write connect response")
		return
	}

	bytesIn, bytesOut := connmgr.CopyTraffic(conn, targetConn)
	h.ctx.ConnMgr.AddBytes(connRecord.ID, bytesIn, bytesOut)

	log.Info().
		Str("client", connRecord.ClientAddr).
		Str("target", targetAddr).
		Str("nic", connRecord.NIC).
		Bool("proxy", connRecord.ProxyUsed).
		Int64("bytes_in", bytesIn).
		Int64("bytes_out", bytesOut).
		Msg("HTTPS tunnel closed")
}

func (h *HTTPHandler) handleHTTP(conn net.Conn, connRecord *connmgr.Connection, req *http.Request, targetAddr string) {
	egressPolicy, err := h.ctx.EgressMgr.Select(targetAddr)
	if err != nil {
		log.Error().Err(err).Str("target", targetAddr).Msg("Failed to select egress policy")
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	if egressPolicy == nil {
		log.Error().Str("target", targetAddr).Msg("No egress policy available")
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
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
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer targetConn.Close()

	// Write request and count bytes using counting writer
	var bytesOut int64
	countWriter := &countWriter{writer: targetConn, count: &bytesOut}
	if err := req.Write(countWriter); err != nil {
		log.Debug().Err(err).Msg("Failed to write request to target")
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(targetConn), req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read response from target")
		return
	}
	defer resp.Body.Close()

	// Write response headers directly and count bytes
	var headerBuf bytes.Buffer
	resp.Header.Write(&headerBuf)
	headerLine := fmt.Sprintf("HTTP/%d.%d %s\r\n", resp.ProtoMajor, resp.ProtoMinor, resp.Status)
	headerBytes := len(headerLine) + headerBuf.Len() + 2 // +2 for final \r\n

	if _, err := fmt.Fprintf(conn, "%s%s\r\n", headerLine, headerBuf.String()); err != nil {
		log.Debug().Err(err).Msg("Failed to write response headers")
		return
	}

	// Stream body directly and count bytes
	bodyBytes, err := io.Copy(conn, resp.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to write response body")
	}

	bytesIn := int64(headerBytes) + bodyBytes
	h.ctx.ConnMgr.AddBytes(connRecord.ID, bytesIn, int64(bytesOut))

	log.Info().
		Str("client", connRecord.ClientAddr).
		Str("target", targetAddr).
		Str("nic", connRecord.NIC).
		Str("method", req.Method).
		Int("status", resp.StatusCode).
		Int64("bytes_in", bytesIn).
		Int64("bytes_out", int64(bytesOut)).
		Msg("HTTP request completed")
}

func (h *HTTPHandler) dialViaProxy(policy *egress.Policy, targetAddr string) (net.Conn, error) {
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

	// Read response
	reader := bufio.NewReader(proxyConn)
	line, err := reader.ReadString('\n')
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	if !strings.Contains(line, "200") {
		proxyConn.Close()
		return nil, fmt.Errorf("proxy connect failed: %s", strings.TrimSpace(line))
	}

	// Skip remaining headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			proxyConn.Close()
			return nil, fmt.Errorf("failed to read proxy headers: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	return proxyConn, nil
}

func (h *HTTPHandler) socks5Handshake(conn net.Conn, proxy *egress.ProxyConfig, targetAddr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	portNum := 0
	fmt.Sscanf(port, "%d", &portNum)

	if err := h.sendSocks5Greeting(conn, proxy.Username, proxy.Password); err != nil {
		conn.Close()
		return nil, err
	}

	if err := h.sendSocks5Connect(conn, host, portNum); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (h *HTTPHandler) sendSocks5Greeting(conn net.Conn, username, password string) error {
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

func (h *HTTPHandler) sendSocks5Connect(conn net.Conn, host string, port int) error {
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
