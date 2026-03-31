package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/rs/zerolog/log"

	"netdispatch/internal/connmgr"
	"netdispatch/internal/egress"
	"netdispatch/internal/handler"
	"netdispatch/internal/nic"
)

// Server represents the proxy server
type Server struct {
	httpListener  net.Listener
	httpsListener net.Listener
	socksListener net.Listener

	nicManager *nic.Manager
	egressMgr  *egress.Manager
	connMgr    *connmgr.Manager
	dialer     *nic.Dialer

	httpAddr  string
	socksAddr string

	socks5Users map[string]string

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer creates a new proxy server
func NewServer(nicManager *nic.Manager, egressMgr *egress.Manager) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		nicManager: nicManager,
		egressMgr:  egressMgr,
		connMgr:    connmgr.NewManager(),
		dialer:     nic.NewDialer(nicManager),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// ConnectionManager returns the connection manager
func (s *Server) ConnectionManager() *connmgr.Manager {
	return s.connMgr
}

// SetSOCKS5Users sets the SOCKS5 authentication users
func (s *Server) SetSOCKS5Users(users map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.socks5Users = users
}

// StartHTTP starts the HTTP proxy server
func (s *Server) StartHTTP(bind string, port int) error {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.httpListener = listener
	s.httpAddr = addr
	s.mu.Unlock()

	log.Info().Str("addr", addr).Msg("HTTP proxy started")

	go s.acceptHTTP(listener)
	return nil
}

// StartHTTPS starts the HTTPS proxy server (CONNECT method tunnel)
func (s *Server) StartHTTPS(bind string, port int) error {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.httpsListener = listener
	s.mu.Unlock()

	log.Info().Str("addr", addr).Msg("HTTPS proxy started")

	go s.acceptHTTPS(listener)
	return nil
}

// StartSOCKS starts the SOCKS5 proxy server
func (s *Server) StartSOCKS(bind string, port int) error {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.socksListener = listener
	s.socksAddr = addr
	s.mu.Unlock()

	log.Info().Str("addr", addr).Msg("SOCKS5 proxy started")

	go s.acceptSOCKS(listener)
	return nil
}

// Stop stops all proxy servers
func (s *Server) Stop() {
	s.cancel()

	s.mu.Lock()
	if s.httpListener != nil {
		s.httpListener.Close()
	}
	if s.httpsListener != nil {
		s.httpsListener.Close()
	}
	if s.socksListener != nil {
		s.socksListener.Close()
	}
	s.mu.Unlock()

	log.Info().Msg("Proxy server stopped")
}

// IsHTTPRunning returns whether HTTP proxy is running
func (s *Server) IsHTTPRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.httpListener != nil
}

// IsSOCKSRunning returns whether SOCKS5 proxy is running
func (s *Server) IsSOCKSRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.socksListener != nil
}

// IsRunning returns whether any proxy is running
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.httpListener != nil || s.socksListener != nil
}

// StopHTTP stops the HTTP proxy server
func (s *Server) StopHTTP() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpListener != nil {
		s.httpListener.Close()
		s.httpListener = nil
		s.httpAddr = ""
		log.Info().Msg("HTTP proxy stopped")
	}
}

// StopSOCKS stops the SOCKS5 proxy server
func (s *Server) StopSOCKS() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.socksListener != nil {
		s.socksListener.Close()
		s.socksListener = nil
		s.socksAddr = ""
		log.Info().Msg("SOCKS5 proxy stopped")
	}
}

// RestartHTTP restarts the HTTP proxy server if address changed
func (s *Server) RestartHTTP(bind string, port int) error {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))

	s.mu.Lock()
	if s.httpListener != nil && s.httpAddr == addr {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.StopHTTP()
	return s.StartHTTP(bind, port)
}

// RestartSOCKS restarts the SOCKS5 proxy server if address changed
func (s *Server) RestartSOCKS(bind string, port int) error {
	addr := net.JoinHostPort(bind, fmt.Sprintf("%d", port))

	s.mu.Lock()
	if s.socksListener != nil && s.socksAddr == addr {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.StopSOCKS()
	return s.StartSOCKS(bind, port)
}

func (s *Server) acceptHTTP(listener net.Listener) {
	ctx := &handler.Context{
		ConnMgr:    s.connMgr,
		NICManager: s.nicManager,
		EgressMgr:  s.egressMgr,
		Dialer:     s.dialer,
	}
	h := handler.NewHTTPHandler(ctx)

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener closed, exit gracefully
			select {
			case <-s.ctx.Done():
				return
			default:
				// Check if it's a temporary error
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					log.Warn().Err(err).Msg("Temporary error accepting HTTP connection")
					continue
				}
				// Listener was closed, exit
				log.Debug().Msg("HTTP listener closed")
				return
			}
		}

		go h.Handle(conn)
	}
}

func (s *Server) acceptHTTPS(listener net.Listener) {
	ctx := &handler.Context{
		ConnMgr:    s.connMgr,
		NICManager: s.nicManager,
		EgressMgr:  s.egressMgr,
		Dialer:     s.dialer,
	}
	h := handler.NewHTTPHandler(ctx)

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener closed, exit gracefully
			select {
			case <-s.ctx.Done():
				return
			default:
				// Check if it's a temporary error
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					log.Warn().Err(err).Msg("Temporary error accepting HTTPS connection")
					continue
				}
				// Listener was closed, exit
				log.Debug().Msg("HTTPS listener closed")
				return
			}
		}

		go h.Handle(conn)
	}
}

func (s *Server) acceptSOCKS(listener net.Listener) {
	ctx := &handler.Context{
		ConnMgr:    s.connMgr,
		NICManager: s.nicManager,
		EgressMgr:  s.egressMgr,
		Dialer:     s.dialer,
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener closed, exit gracefully
			select {
			case <-s.ctx.Done():
				return
			default:
				// Check if it's a temporary error
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					log.Warn().Err(err).Msg("Temporary error accepting SOCKS connection")
					continue
				}
				// Listener was closed, exit
				log.Debug().Msg("SOCKS listener closed")
				return
			}
		}

		// Get current users snapshot
		s.mu.Lock()
		users := s.socks5Users
		s.mu.Unlock()

		h := handler.NewSOCKS5Handler(ctx, users)
		go h.Handle(conn)
	}
}
