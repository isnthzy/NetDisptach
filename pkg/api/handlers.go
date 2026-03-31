package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"netdispatch/internal/egress"
	"netdispatch/internal/router"
	"netdispatch/pkg/config"
)

// listNICs returns all available NICs
func (s *Server) listNICs(w http.ResponseWriter, r *http.Request) {
	nics := s.nicManager.List()
	respondJSON(w, http.StatusOK, nics)
}

// getNIC returns a specific NIC
func (s *Server) getNIC(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	nic := s.nicManager.Get(name)
	if nic == nil {
		respondError(w, http.StatusNotFound, "NIC not found")
		return
	}

	respondJSON(w, http.StatusOK, nic)
}

// listEgress returns all egress policies
func (s *Server) listEgress(w http.ResponseWriter, r *http.Request) {
	policies := s.egressMgr.List()
	respondJSON(w, http.StatusOK, policies)
}

// createEgress creates a new egress policy
func (s *Server) createEgress(w http.ResponseWriter, r *http.Request) {
	var policy egress.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.egressMgr.Add(&policy)
	s.syncEgressToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after creating egress")
	}
	respondJSON(w, http.StatusCreated, policy)
}

// updateEgress updates an egress policy
func (s *Server) updateEgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var policy egress.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	policy.ID = id
	s.egressMgr.Add(&policy)
	s.syncEgressToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after updating egress")
	}
	respondJSON(w, http.StatusOK, policy)
}

// deleteEgress deletes an egress policy
func (s *Server) deleteEgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.egressMgr.Remove(id)
	s.syncEgressToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after deleting egress")
	}
	respondJSON(w, http.StatusNoContent, nil)
}

// testEgress tests an egress policy connection
func (s *Server) testEgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	policy := s.egressMgr.Get(id)
	if policy == nil {
		respondError(w, http.StatusNotFound, "Egress policy not found")
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": "Connection test not implemented",
	}

	respondJSON(w, http.StatusOK, result)
}

// listRules returns all routing rules
func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rules := s.routerMgr.ListRules()
	respondJSON(w, http.StatusOK, rules)
}

// createRule creates a new routing rule
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var rule router.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := rule.CompileCIDRs(); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid CIDR: "+err.Error())
		return
	}

	s.routerMgr.AddRule(rule)
	s.syncRulesToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after creating rule")
	}
	respondJSON(w, http.StatusCreated, rule)
}

// updateRule updates a routing rule
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var rule router.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	rule.ID = id
	if err := rule.CompileCIDRs(); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid CIDR: "+err.Error())
		return
	}

	s.routerMgr.AddRule(rule)
	s.syncRulesToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after updating rule")
	}
	respondJSON(w, http.StatusOK, rule)
}

// deleteRule deletes a routing rule
func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.routerMgr.RemoveRule(id)
	s.syncRulesToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after deleting rule")
	}
	respondJSON(w, http.StatusNoContent, nil)
}

// listConnections returns all active connections
func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	connections := s.connMgr.List()
	respondJSON(w, http.StatusOK, connections)
}

// getConnection returns a specific connection
func (s *Server) getConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	conn := s.connMgr.Get(id)
	if conn == nil {
		respondError(w, http.StatusNotFound, "Connection not found")
		return
	}

	respondJSON(w, http.StatusOK, conn)
}

// closeConnection closes a connection
func (s *Server) closeConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	s.connMgr.Close(id)
	respondJSON(w, http.StatusNoContent, nil)
}

// getStatsOverview returns overview statistics
func (s *Server) getStatsOverview(w http.ResponseWriter, r *http.Request) {
	stats := s.connMgr.GetStats()
	respondJSON(w, http.StatusOK, stats)
}

// getStatsTraffic returns traffic statistics
func (s *Server) getStatsTraffic(w http.ResponseWriter, r *http.Request) {
	stats := s.connMgr.GetStats()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"bytes_in":  stats.BytesIn,
		"bytes_out": stats.BytesOut,
		"by_nic":    stats.ByNIC,
	})
}

// getTrafficHistory returns traffic history
func (s *Server) getTrafficHistory(w http.ResponseWriter, r *http.Request) {
	history := s.connMgr.GetTrafficHistory()
	respondJSON(w, http.StatusOK, history)
}

// getRecentConnections returns recent connections
func (s *Server) getRecentConnections(w http.ResponseWriter, r *http.Request) {
	connections := s.connMgr.GetRecentConnections()
	respondJSON(w, http.StatusOK, connections)
}

// getConfig returns the current configuration
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	respondJSON(w, http.StatusOK, s.config)
}

// updateConfig updates the configuration
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate configuration
	if err := s.validateConfig(&newCfg); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.configMu.RLock()
	oldServerCfg := s.config.Server
	s.configMu.RUnlock()

	// Sync server config (proxy service)
	s.applyServerConfig(&oldServerCfg, &newCfg.Server)

	// Sync egress policies
	s.applyEgressConfig(newCfg.Egress)

	// Sync routing config
	s.applyRoutingConfig(&newCfg.Routing)

	// Sync NICs config
	s.applyNICsConfig(&newCfg.NICs)

	// Update memory config
	s.configMu.Lock()
	s.config = &newCfg
	s.configMu.Unlock()

	// Save to file
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config")
		respondError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	respondJSON(w, http.StatusOK, newCfg)
}

// validateConfig validates the configuration
func (s *Server) validateConfig(cfg *config.Config) error {
	// Port range validation
	if cfg.Server.HTTP.Enabled {
		if cfg.Server.HTTP.Port < 1 || cfg.Server.HTTP.Port > 65535 {
			return fmt.Errorf("invalid HTTP port: %d", cfg.Server.HTTP.Port)
		}
	}
	if cfg.Server.SOCKS5.Enabled {
		if cfg.Server.SOCKS5.Port < 1 || cfg.Server.SOCKS5.Port > 65535 {
			return fmt.Errorf("invalid SOCKS5 port: %d", cfg.Server.SOCKS5.Port)
		}
	}

	// Port conflict check
	if cfg.Server.HTTP.Enabled && cfg.Server.SOCKS5.Enabled {
		if cfg.Server.HTTP.Port == cfg.Server.SOCKS5.Port {
			return fmt.Errorf("HTTP and SOCKS5 cannot use the same port: %d", cfg.Server.HTTP.Port)
		}
	}

	return nil
}

// applyServerConfig syncs proxy server configuration
func (s *Server) applyServerConfig(oldCfg, newCfg *config.ServerConfig) {
	if s.proxyServer == nil {
		return
	}

	bind := newCfg.Bind
	if bind == "" {
		bind = "0.0.0.0"
	}

	// Update SOCKS5 users if authentication settings changed
	if newCfg.SOCKS5.Auth.Enabled && len(newCfg.SOCKS5.Auth.Users) > 0 {
		users := make(map[string]string)
		for _, u := range newCfg.SOCKS5.Auth.Users {
			users[u.Username] = u.Password
		}
		s.proxyServer.SetSOCKS5Users(users)
	} else {
		s.proxyServer.SetSOCKS5Users(nil)
	}

	// Handle master switch - when disabled, stop all services
	// Note: we don't modify the individual service states in newCfg
	if !newCfg.Enabled {
		s.proxyServer.StopHTTP()
		s.proxyServer.StopSOCKS()
		return
	}

	// HTTP service - sync enabled state
	if newCfg.HTTP.Enabled {
		if err := s.proxyServer.RestartHTTP(bind, newCfg.HTTP.Port); err != nil {
			log.Error().Err(err).Msg("Failed to restart HTTP proxy")
		}
	} else {
		s.proxyServer.StopHTTP()
	}

	// SOCKS5 service - sync enabled state
	if newCfg.SOCKS5.Enabled {
		if err := s.proxyServer.RestartSOCKS(bind, newCfg.SOCKS5.Port); err != nil {
			log.Error().Err(err).Msg("Failed to restart SOCKS5 proxy")
		}
	} else {
		s.proxyServer.StopSOCKS()
	}
}

// applyEgressConfig syncs egress policies
func (s *Server) applyEgressConfig(policies []config.EgressPolicy) {
	egressPolicies := make([]*egress.Policy, len(policies))
	for i, p := range policies {
		egressPolicies[i] = &egress.Policy{
			ID:          p.ID,
			Name:        p.Name,
			NIC:         p.NIC,
			Description: p.Description,
		}
		if p.Proxy != nil {
			egressPolicies[i].Proxy = &egress.ProxyConfig{
				Host:     p.Proxy.Host,
				Port:     p.Proxy.Port,
				Protocol: p.Proxy.Protocol,
				Username: p.Proxy.Username,
				Password: p.Proxy.Password,
			}
		}
	}
	s.egressMgr.SetPolicies(egressPolicies)
}

// applyRoutingConfig syncs routing configuration
func (s *Server) applyRoutingConfig(cfg *config.RoutingConfig) {
	// Sync default egress
	s.egressMgr.SetDefault(cfg.DefaultEgress)
	s.routerMgr.SetDefaultEgress(cfg.DefaultEgress)

	// Sync all routing rules
	rules := make([]router.Rule, len(cfg.Rules))
	for i, r := range cfg.Rules {
		rules[i] = router.Rule{
			ID:          r.ID,
			Priority:    r.Priority,
			Enabled:     r.Enabled,
			ListType:    router.ListType(r.ListType),
			Domains:     r.Domains,
			CIDRs:       r.CIDRs,
			Ports:       r.Ports,
			Action:      r.Action,
			EgressID:    r.EgressID,
			Source:      r.Source,
			DomainCount: r.DomainCount,
		}
		// Compile CIDRs for each rule
		if err := rules[i].CompileCIDRs(); err != nil {
			log.Warn().Err(err).Str("rule", r.ID).Msg("Failed to compile CIDRs for rule")
		}
		// Build domain tree for rules with many domains
		if len(r.Domains) > 0 {
			rules[i].BuildDomainTree()
		}
	}
	s.routerMgr.SetRules(rules)
}

// applyNICsConfig syncs NIC configuration
func (s *Server) applyNICsConfig(cfg *config.NICsConfig) {
	// NIC display names are for UI only, no runtime sync needed
	_ = cfg
}

// getSystemInfo returns system information
func (s *Server) getSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":    "0.1.0",
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}

	respondJSON(w, http.StatusOK, info)
}

// healthCheck returns health status
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// getServerStatus returns actual proxy service status
func (s *Server) getServerStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"http_running":   s.proxyServer.IsHTTPRunning(),
		"socks5_running": s.proxyServer.IsSOCKSRunning(),
		"running":        s.proxyServer.IsRunning(),
	}
	respondJSON(w, http.StatusOK, status)
}

// syncEgressToConfig syncs egress policies from manager to config
func (s *Server) syncEgressToConfig() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	policies := s.egressMgr.List()
	s.config.Egress = make([]config.EgressPolicy, len(policies))
	for i, p := range policies {
		s.config.Egress[i] = config.EgressPolicy{
			ID:          p.ID,
			Name:        p.Name,
			NIC:         p.NIC,
			Description: p.Description,
		}
		if p.Proxy != nil {
			s.config.Egress[i].Proxy = &config.ProxyConfig{
				Host:     p.Proxy.Host,
				Port:     p.Proxy.Port,
				Protocol: p.Proxy.Protocol,
				Username: p.Proxy.Username,
				Password: p.Proxy.Password,
			}
		}
	}
}

// syncRulesToConfig syncs routing rules from manager to config
func (s *Server) syncRulesToConfig() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	rules := s.routerMgr.ListRules()
	s.config.Routing.Rules = make([]config.Rule, len(rules))
	for i, r := range rules {
		s.config.Routing.Rules[i] = config.Rule{
			ID:          r.ID,
			Priority:    r.Priority,
			Enabled:     r.Enabled,
			ListType:    string(r.ListType),
			Domains:     r.Domains,
			CIDRs:       r.CIDRs,
			Ports:       r.Ports,
			Action:      r.Action,
			EgressID:    r.EgressID,
			Source:      r.Source,
			DomainCount: r.DomainCount,
		}
	}
}

// importRuleFromURL imports a domain list from a URL
func (s *Server) importRuleFromURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		EgressID string `json:"egress_id"`
		Priority int    `json:"priority"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	if req.Name == "" {
		req.Name = "Imported from URL"
	}

	result, err := s.importer.ImportFromURL(req.URL, req.Name, req.EgressID, req.Priority, req.Enabled)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Import failed: "+err.Error())
		return
	}

	// Sync to config
	s.syncRulesToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after importing rule")
	}

	respondJSON(w, http.StatusCreated, result)
}

// importRuleFromFile imports a domain list from an uploaded file
func (s *Server) importRuleFromFile(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		respondError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = "Imported from file"
	}

	egressID := r.FormValue("egress_id")
	priority := 0
	fmt.Sscanf(r.FormValue("priority"), "%d", &priority)
	enabled := r.FormValue("enabled") != "false"

	// Import from reader
	result, err := s.importer.ImportFromReader(file, name, "file-upload", egressID, priority, enabled)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Import failed: "+err.Error())
		return
	}

	// Sync to config
	s.syncRulesToConfig()
	if err := s.saveConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to save config after importing rule")
	}

	respondJSON(w, http.StatusCreated, result)
}

// importRuleFromReader is a helper for testing
func (s *Server) importRuleFromReader(reader io.Reader, name, source, egressID string, priority int, enabled bool) (*router.ImportResult, error) {
	return s.importer.ImportFromReader(reader, name, source, egressID, priority, enabled)
}
