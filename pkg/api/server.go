package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"netdispatch/internal/connmgr"
	"netdispatch/internal/egress"
	"netdispatch/internal/nic"
	"netdispatch/internal/proxy"
	"netdispatch/internal/router"
	"netdispatch/pkg/config"
	"netdispatch/pkg/ws"
	"netdispatch/web"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server represents the API server
type Server struct {
	server      *http.Server
	nicManager  *nic.Manager
	egressMgr   *egress.Manager
	routerMgr   *router.Manager
	connMgr     *connmgr.Manager
	proxyServer *proxy.Server
	config      *config.Config
	configPath  string
	configMu    sync.RWMutex
	wsHub       *ws.Hub
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, nicManager *nic.Manager, egressMgr *egress.Manager, routerMgr *router.Manager, connMgr *connmgr.Manager, proxyServer *proxy.Server) *Server {
	return &Server{
		nicManager:  nicManager,
		egressMgr:   egressMgr,
		routerMgr:   routerMgr,
		connMgr:     connMgr,
		proxyServer: proxyServer,
		config:      cfg,
		configPath:  "configs/config.yaml",
		wsHub:       ws.NewHub(),
	}
}

// SetWSHub sets the WebSocket hub
func (s *Server) SetWSHub(hub *ws.Hub) {
	s.wsHub = hub
}

// GetWSHub returns the WebSocket hub
func (s *Server) GetWSHub() *ws.Hub {
	return s.wsHub
}

// SetConfigPath sets the configuration file path
func (s *Server) SetConfigPath(path string) {
	s.configPath = path
}

// saveConfig saves the current configuration to file
func (s *Server) saveConfig() error {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config.Save(s.configPath)
}

// Start starts the API server
func (s *Server) Start() error {
	routerMux := mux.NewRouter()

	routerMux.HandleFunc("/api/v1/nics", s.listNICs).Methods("GET")
	routerMux.HandleFunc("/api/v1/nics/{name}", s.getNIC).Methods("GET")

	routerMux.HandleFunc("/api/v1/egress", s.listEgress).Methods("GET")
	routerMux.HandleFunc("/api/v1/egress", s.createEgress).Methods("POST")
	routerMux.HandleFunc("/api/v1/egress/{id}", s.updateEgress).Methods("PUT")
	routerMux.HandleFunc("/api/v1/egress/{id}", s.deleteEgress).Methods("DELETE")
	routerMux.HandleFunc("/api/v1/egress/{id}/test", s.testEgress).Methods("POST")

	routerMux.HandleFunc("/api/v1/rules", s.listRules).Methods("GET")
	routerMux.HandleFunc("/api/v1/rules", s.createRule).Methods("POST")
	routerMux.HandleFunc("/api/v1/rules/{id}", s.updateRule).Methods("PUT")
	routerMux.HandleFunc("/api/v1/rules/{id}", s.deleteRule).Methods("DELETE")

	routerMux.HandleFunc("/api/v1/connections", s.listConnections).Methods("GET")
	routerMux.HandleFunc("/api/v1/connections/recent", s.getRecentConnections).Methods("GET")
	routerMux.HandleFunc("/api/v1/connections/{id}", s.getConnection).Methods("GET")
	routerMux.HandleFunc("/api/v1/connections/{id}", s.closeConnection).Methods("DELETE")

	routerMux.HandleFunc("/api/v1/stats/overview", s.getStatsOverview).Methods("GET")
	routerMux.HandleFunc("/api/v1/stats/traffic", s.getStatsTraffic).Methods("GET")
	routerMux.HandleFunc("/api/v1/stats/history", s.getTrafficHistory).Methods("GET")

	routerMux.HandleFunc("/api/v1/config", s.getConfig).Methods("GET")
	routerMux.HandleFunc("/api/v1/config", s.updateConfig).Methods("PUT")

	routerMux.HandleFunc("/api/v1/system/info", s.getSystemInfo).Methods("GET")
	routerMux.HandleFunc("/api/v1/health", s.healthCheck).Methods("GET")
	routerMux.HandleFunc("/api/v1/status", s.getServerStatus).Methods("GET")

	// WebSocket endpoint
	routerMux.HandleFunc("/ws", s.handleWebSocket)

	// Serve embedded web UI
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		log.Error().Err(err).Msg("Failed to load embedded web files")
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		routerMux.PathPrefix("/").Handler(s.spaHandler(fileServer, distFS))
	}

	addr := fmt.Sprintf("%s:%d", s.config.API.Bind, s.config.API.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(routerMux),
	}

	log.Info().Str("addr", addr).Msg("API server started")

	return s.server.ListenAndServe()
}

// spaHandler handles SPA routing by serving index.html for non-API routes
func (s *Server) spaHandler(fileServer http.Handler, distFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve static file
		if strings.Contains(path, ".") {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For SPA routes, serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}
}

// Stop stops the API server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	client := ws.NewClient(s.wsHub, conn)
	s.wsHub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
