package nic

import (
	"context"
	"net"
	"time"
)

// Dialer creates connections bound to a specific NIC
type Dialer struct {
	manager *Manager
	timeout time.Duration
}

// NewDialer creates a new dialer
func NewDialer(manager *Manager) *Dialer {
	return &Dialer{
		manager: manager,
		timeout: 30 * time.Second,
	}
}

// SetTimeout sets the dial timeout
func (d *Dialer) SetTimeout(timeout time.Duration) {
	d.timeout = timeout
}

// Dial dials a connection bound to a specific NIC
func (d *Dialer) Dial(network, address, nicName string) (net.Conn, error) {
	localIP := d.manager.GetIP(nicName)
	if localIP == nil {
		return nil, &net.OpError{
			Op:  "dial",
			Net: network,
			Err: ErrNICNotFound,
		}
	}

	// Parse the target address to check IP family
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		targetIP := net.ParseIP(host)
		// If target is IP (not domain), check if address families match
		if targetIP != nil {
			localIsIPv4 := localIP.To4() != nil
			targetIsIPv4 := targetIP.To4() != nil
			if localIsIPv4 != targetIsIPv4 {
				// Address family mismatch - dial without binding to allow IPv6
				dialer := &net.Dialer{Timeout: d.timeout}
				return dialer.Dial(network, address)
			}
		}
	}

	dialer := &net.Dialer{
		Timeout:   d.timeout,
		LocalAddr: &net.TCPAddr{IP: localIP},
	}

	return dialer.Dial(network, address)
}

// DialContext dials a connection with context, bound to a specific NIC
func (d *Dialer) DialContext(ctx context.Context, network, address, nicName string) (net.Conn, error) {
	localIP := d.manager.GetIP(nicName)
	if localIP == nil {
		return nil, &net.OpError{
			Op:  "dial",
			Net: network,
			Err: ErrNICNotFound,
		}
	}

	dialer := &net.Dialer{
		Timeout:   d.timeout,
		LocalAddr: &net.TCPAddr{IP: localIP},
	}

	return dialer.DialContext(ctx, network, address)
}

// Listen creates a listener bound to a specific NIC
func (d *Dialer) Listen(network, address, nicName string) (net.Listener, error) {
	localIP := d.manager.GetIP(nicName)
	if localIP == nil {
		return nil, &net.OpError{
			Op:  "listen",
			Net: network,
			Err: ErrNICNotFound,
		}
	}

	listenAddr := &net.TCPAddr{
		IP:   localIP,
		Port: 0,
	}

	return net.ListenTCP(network, listenAddr)
}
