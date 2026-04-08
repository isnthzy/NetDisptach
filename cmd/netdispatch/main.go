package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"netdispatch/internal/egress"
	"netdispatch/internal/nic"
	"netdispatch/internal/proxy"
	"netdispatch/internal/router"
	"netdispatch/internal/tray"
	"netdispatch/pkg/api"
	"netdispatch/pkg/config"
	"netdispatch/pkg/crashlog"
	"netdispatch/pkg/singleinstance"
	"netdispatch/pkg/version"
	"netdispatch/pkg/ws"
)

// Build-time variables (set via ldflags)
var (
	// defaultAPIPortStr is the default port for the Web GUI and API server (string for ldflags)
	defaultAPIPortStr = "9090"
)

// getParsedAPIPort returns the API port as an integer
func getParsedAPIPort() int {
	port := 9090
	fmt.Sscanf(defaultAPIPortStr, "%d", &port)
	return port
}

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:   "netdispatch",
	Short: "A multi-protocol proxy server with smart NIC routing",
	Long: `NetDispatch is a high-performance proxy server that supports HTTP/HTTPS/SOCKS5 protocols
with intelligent routing based on egress policies (NIC + optional upstream proxy).`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		v := version.Get()
		fmt.Printf("NetDispatch v%s\n", v.Version)
		fmt.Printf("Git Commit: %s\n", v.GitCommit)
		fmt.Printf("Build Date: %s\n", v.BuildDate)
		fmt.Printf("Default API Port: %d\n", getParsedAPIPort())
	},
}

// broadcastWriter is a global log writer that can be connected to the WebSocket hub
var broadcastWriter *ws.BroadcastLogWriter

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// Create broadcast writer that wraps stderr
	broadcastWriter = ws.NewBroadcastLogWriter(os.Stderr, nil)
	// Use ConsoleWriter with broadcast writer for both console output and WebSocket broadcast
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: broadcastWriter, TimeFormat: "15:04:05"})

	// Set the default API port from compile-time variable
	config.SetDefaultAPIPort(getParsedAPIPort())

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	// If no arguments provided (double-click on Windows), default to 'start' command
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "start")
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getConfigPath returns the absolute path to the config file
// Priority: -c flag > user config dir > executable dir
func getConfigPath() string {
	// If -c flag is provided, use it
	if cfgFile != "" {
		absPath, err := filepath.Abs(cfgFile)
		if err == nil {
			return absPath
		}
		return cfgFile
	}

	// Use user config directory based on OS
	var configDir string
	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%/NetDispatch
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("LOCALAPPDATA")
		}
		configDir = filepath.Join(appData, "NetDispatch")
	case "darwin":
		// macOS: ~/.config/NetDispatch
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config", "NetDispatch")
	default:
		// Linux: ~/.config/NetDispatch
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config", "NetDispatch")
	}

	return filepath.Join(configDir, "config.yaml")
}

// ensureConfigDir ensures the config directory exists
func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0755)
}

func runServer() {
	// Set up panic recovery first
	defer func() {
		if r := recover(); r != nil {
			// Capture stack trace
			buf := make([]byte, 64*1024)
			n := runtime.Stack(buf, false)
			stack := buf[:n]

			// Log the panic
			log.Error().Interface("panic", r).Msg("Kernel panic - application crashed")

			// Write crash log
			crashPath := crashlog.WriteCrashLog(r, stack)
			if crashPath != "" {
				fmt.Fprintf(os.Stderr, "Crash log written to: %s\n", crashPath)
				ShowMessageBox("NetDispatch 崩溃", fmt.Sprintf("程序发生崩溃。\n\n崩溃日志已保存到:\n%s", crashPath))
			} else {
				ShowMessageBox("NetDispatch 崩溃", "程序发生崩溃，无法保存崩溃日志。")
			}

			// Exit with error code
			os.Exit(1)
		}
	}()

	// Check for single instance using default API port
	// The port check helps detect existing instances quickly
	release, err := singleinstance.Acquire("netdispatch", getParsedAPIPort())
	if err != nil {
		// Another instance is already running
		// Show a message box on Windows, print to stderr on other platforms
		ShowMessageBox("NetDispatch", "应用程序已在运行中。\n\nApplication is already running.")
		focusExistingInstance()
		os.Exit(1)
	}
	defer release()

	configPath := getConfigPath()
	log.Info().Str("path", configPath).Msg("Using config file")

	if err := ensureConfigDir(configPath); err != nil {
		log.Warn().Err(err).Msg("Failed to create config directory")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	log.Info().Msg("Starting NetDispatch...")

	nicManager := nic.NewManager()
	if err := nicManager.Refresh(); err != nil {
		log.Warn().Err(err).Msg("Failed to detect NICs")
	}

	nics := nicManager.List()
	log.Info().Int("count", len(nics)).Msg("Detected network interfaces")
	for _, n := range nics {
		log.Info().
			Str("name", n.Name).
			Str("ip", n.IP).
			Bool("default", n.IsDefault).
			Msg("NIC detected")
	}

	// Auto-select bind address if not set (WLAN first, then Ethernet)
	if cfg.Server.Bind == "" {
		cfg.Server.Bind = autoSelectBindAddress(nicManager)
		log.Info().Str("bind", cfg.Server.Bind).Msg("Auto-selected bind address")
	}

	egressMgr := egress.NewManager()

	for _, policy := range cfg.Egress {
		egressMgr.Add(&egress.Policy{
			ID:          policy.ID,
			Name:        policy.Name,
			NIC:         policy.NIC,
			Proxy:       convertProxyConfig(policy.Proxy),
			Description: policy.Description,
		})
	}

	routerMgr := router.NewManager()

	// Load routing rules from config
	for _, rule := range cfg.Routing.Rules {
		r := router.Rule{
			ID:       rule.ID,
			Priority: rule.Priority,
			Enabled:  rule.Enabled,
			ListType: router.ListType(rule.ListType),
			Domains:  rule.Domains,
			CIDRs:    rule.CIDRs,
			Ports:    rule.Ports,
			Action:   rule.Action,
			EgressID: rule.EgressID,
		}
		if err := r.CompileCIDRs(); err != nil {
			log.Warn().Err(err).Str("rule", rule.ID).Msg("Failed to compile CIDRs for rule")
		}
		routerMgr.AddRule(r)
	}

	// Connect router manager to egress manager
	egressMgr.SetRouterManager(routerMgr)

	if cfg.Routing.DefaultEgress != "" {
		egressMgr.SetDefault(cfg.Routing.DefaultEgress)
		routerMgr.SetDefaultEgress(cfg.Routing.DefaultEgress)
	} else if nicManager.Default() != nil {
		egressMgr.SetDefault(nicManager.Default().Name)
	}

	proxyServer := proxy.NewServer(nicManager, egressMgr)

	// Set SOCKS5 authentication users if enabled
	if cfg.Server.SOCKS5.Auth.Enabled && len(cfg.Server.SOCKS5.Auth.Users) > 0 {
		users := make(map[string]string)
		for _, u := range cfg.Server.SOCKS5.Auth.Users {
			users[u.Username] = u.Password
		}
		proxyServer.SetSOCKS5Users(users)
		log.Info().Int("users", len(users)).Msg("SOCKS5 authentication enabled")
	}

	// Log the config state for debugging
	log.Info().Bool("server_enabled", cfg.Server.Enabled).Bool("http_enabled", cfg.Server.HTTP.Enabled).Bool("socks5_enabled", cfg.Server.SOCKS5.Enabled).Msg("Config loaded")

	// Only start proxy if enabled
	if cfg.Server.Enabled {
		if cfg.Server.HTTP.Enabled {
			if err := proxyServer.StartHTTP(cfg.Server.Bind, cfg.Server.HTTP.Port); err != nil {
				log.Error().Err(err).Msg("Failed to start HTTP proxy")
			}
		}

		if cfg.Server.SOCKS5.Enabled {
			if err := proxyServer.StartSOCKS(cfg.Server.Bind, cfg.Server.SOCKS5.Port); err != nil {
				log.Error().Err(err).Msg("Failed to start SOCKS5 proxy")
			}
		}
	} else {
		log.Info().Msg("Proxy forwarding is disabled")
	}

	wsHub := ws.NewHub()
	go wsHub.Run()

	apiServer := api.NewServer(cfg, nicManager, egressMgr, routerMgr, proxyServer.ConnectionManager(), proxyServer)
	apiServer.SetConfigPath(configPath)
	apiServer.SetWSHub(wsHub)

	// Connect the broadcast writer to the WebSocket hub
	broadcastWriter.SetHub(wsHub)

	// Create shutdown context for goroutine coordination
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Start traffic history recorder with proper shutdown
	connMgr := proxyServer.ConnectionManager()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-shutdownCtx.Done():
				return
			case <-ticker.C:
				connMgr.RecordTraffic()
					// Broadcast real-time stats via WebSocket
					stats := connMgr.GetStats()
					wsHub.BroadcastTraffic(stats.BytesIn, stats.BytesOut, stats.ActiveConnections)
			}
		}
	}()

	go func() {
		if err := apiServer.Start(); err != nil {
			log.Error().Err(err).Msg("API server error")
		}
	}()

	log.Info().Msg("NetDispatch started successfully")

	// Web UI is served by API server on API port
	webURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.API.Port)

	tray.SetOnOpenBrowser(func() {
		openBrowser(webURL)
	})

	tray.SetOnQuit(func() {
		log.Info().Msg("Shutting down from tray...")
		doShutdown(shutdownCancel, proxyServer, apiServer, wsHub)
		// Exit after cleanup - tray.Quit() is called automatically
		os.Exit(0)
	})

	tray.SetStatusChangeCallback(func(running bool) {
		if running {
			log.Info().Msg("Starting proxy from tray...")
			started := false
			if cfg.Server.HTTP.Enabled {
				if err := proxyServer.StartHTTP(cfg.Server.Bind, cfg.Server.HTTP.Port); err != nil {
					log.Error().Err(err).Msg("Failed to start HTTP proxy")
				} else {
					started = true
				}
			}
			if cfg.Server.SOCKS5.Enabled {
				if err := proxyServer.StartSOCKS(cfg.Server.Bind, cfg.Server.SOCKS5.Port); err != nil {
					log.Error().Err(err).Msg("Failed to start SOCKS5 proxy")
				} else {
					started = true
				}
			}
			if !started {
				// Revert to stopped state if start failed
				tray.SetStatus("stopped")
			}
		} else {
			log.Info().Msg("Stopping proxy from tray...")
			proxyServer.Stop()
		}
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutting down...")
		doShutdown(shutdownCancel, proxyServer, apiServer, wsHub)
		tray.Quit()
		os.Exit(0)
	}()

	// Run systray with external loop on main thread
	// This prevents menu freeze issues caused by running in a goroutine
	start, _ := tray.RunExternalLoop()
	start()

	// Keep main thread alive
	select {}
}

// autoSelectBindAddress selects the best bind address (Ethernet first, then WLAN)
func autoSelectBindAddress(nicMgr *nic.Manager) string {
	nics := nicMgr.List()

	// Priority: Ethernet > WLAN > Default
	// Skip APIPA addresses (169.254.x.x) which indicate no network
	var ethernetIP string
	var wlanIP string
	var defaultIP string

	for _, n := range nics {
		// Skip APIPA addresses
		if strings.HasPrefix(n.IP, "169.254.") {
			continue
		}

		nameLower := strings.ToLower(n.Name)

		// Ethernet patterns (cross-platform)
		// Windows: 以太, 网线, ethernet
		// Linux: eth, enp, ens, eno, enx (enp0s3, ens33, etc.)
		isEthernet := strings.Contains(nameLower, "eth") ||
			strings.Contains(nameLower, "enp") ||
			strings.Contains(nameLower, "ens") ||
			strings.Contains(nameLower, "eno") ||
			strings.Contains(nameLower, "enx") ||
			strings.Contains(nameLower, "以太") ||
			strings.Contains(nameLower, "网线") ||
			strings.Contains(nameLower, "ethernet")

		// Wireless patterns (cross-platform)
		// Windows: wlan, wifi
		// Linux: wlp, wlo, wlx, wlan (wlp2s0, wlo1, etc.)
		isWireless := strings.Contains(nameLower, "wlan") ||
			strings.Contains(nameLower, "wlp") ||
			strings.Contains(nameLower, "wlo") ||
			strings.Contains(nameLower, "wlx") ||
			strings.Contains(nameLower, "wifi") ||
			strings.Contains(nameLower, "无线")

		if isEthernet && ethernetIP == "" {
			ethernetIP = n.IP
		} else if isWireless && wlanIP == "" {
			wlanIP = n.IP
		}
		if n.IsDefault && defaultIP == "" {
			defaultIP = n.IP
		}
	}

	// Return in priority order: Ethernet first
	if ethernetIP != "" {
		return ethernetIP
	}
	if wlanIP != "" {
		return wlanIP
	}
	if defaultIP != "" {
		return defaultIP
	}
	return "0.0.0.0"
}

func convertProxyConfig(cfg *config.ProxyConfig) *egress.ProxyConfig {
	if cfg == nil {
		return nil
	}
	return &egress.ProxyConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Protocol: cfg.Protocol,
		Username: cfg.Username,
		Password: cfg.Password,
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use cmd.exe with start command for better compatibility
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Debug().Err(err).Msg("Failed to open browser")
		} else {
			// Reap zombie process
			go cmd.Wait()
		}
	}
}

// doShutdown performs graceful shutdown of all components
func doShutdown(cancel context.CancelFunc, proxyServer *proxy.Server, apiServer *api.Server, wsHub *ws.Hub) {
	cancel() // Signal all goroutines to stop
	proxyServer.Stop()
	ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()
	apiServer.Stop(ctx)
	wsHub.Stop()
}
