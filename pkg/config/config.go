package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultAPIPort is the default port for the Web GUI and API server
// Can be overridden at compile time via ldflags
var DefaultAPIPort = 9090

// SetDefaultAPIPort sets the default API port (called from main with compile-time value)
func SetDefaultAPIPort(port int) {
	DefaultAPIPort = port
}

// Config represents the main configuration
type Config struct {
	Server  ServerConfig   `yaml:"server" json:"server"`
	NICs    NICsConfig     `yaml:"nics" json:"nics"`
	Egress  []EgressPolicy `yaml:"egress_policies" json:"egress_policies"`
	Routing RoutingConfig  `yaml:"routing" json:"routing"`
	API     APIConfig      `yaml:"api" json:"api"`
}

// ServerConfig represents proxy server settings
type ServerConfig struct {
	Enabled bool        `yaml:"enabled" json:"enabled"`
	Bind    string      `yaml:"bind" json:"bind"`
	HTTP    PortConfig  `yaml:"http" json:"http"`
	SOCKS5  SOCKSConfig `yaml:"socks5" json:"socks5"`
}

// PortConfig represents a port configuration
type PortConfig struct {
	Port    int  `yaml:"port" json:"port"`
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// SOCKSConfig represents SOCKS5 configuration
type SOCKSConfig struct {
	Port int              `yaml:"port" json:"port"`
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Auth SOCKSAuthConfig  `yaml:"auth" json:"auth"`
}

// SOCKSAuthConfig represents SOCKS5 authentication
type SOCKSAuthConfig struct {
	Enabled bool         `yaml:"enabled" json:"enabled"`
	Users   []SOCKSUser `yaml:"users" json:"users"`
}

// SOCKSUser represents a SOCKS5 user
type SOCKSUser struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// NICsConfig represents NIC configuration
type NICsConfig struct {
	Default      string            `yaml:"default" json:"default"`
	DisplayNames map[string]string `yaml:"display_names" json:"display_names"`
}

// EgressPolicy represents an egress policy (NIC + optional proxy)
type EgressPolicy struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	NIC         string       `yaml:"nic" json:"nic"`
	Proxy       *ProxyConfig `yaml:"proxy" json:"proxy"`
	Description string       `yaml:"description" json:"description"`
}

// ProxyConfig represents upstream proxy configuration
type ProxyConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Protocol string `yaml:"protocol" json:"protocol"` // "http" or "socks5"
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// RoutingConfig represents routing rules configuration
type RoutingConfig struct {
	DefaultEgress string `yaml:"default_egress" json:"default_egress"`
	Rules        []Rule `yaml:"rules" json:"rules"`
}

// Rule represents a routing rule
type Rule struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Priority    int      `yaml:"priority" json:"priority"`
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	ListType    string   `yaml:"list_type" json:"list_type"` // "none", "whitelist", "blacklist"
	Domains     []string `yaml:"domains" json:"domains"`
	CIDRs       []string `yaml:"cidrs" json:"cidrs"`
	Ports       []int    `yaml:"ports" json:"ports"`
	Action      string   `yaml:"action" json:"action"` // "forward" or "reject"
	EgressID    string   `yaml:"egress_id" json:"egress_id"`
	Source      string   `yaml:"source,omitempty" json:"source,omitempty"`           // Import source (URL or file path)
	DomainCount int      `yaml:"domain_count,omitempty" json:"domain_count,omitempty"` // Number of domains for imported rules
}

// APIConfig represents API server configuration
type APIConfig struct {
	Bind string    `yaml:"bind" json:"bind"`
	Port int       `yaml:"port" json:"port"`
	Auth AuthConfig `yaml:"auth" json:"auth"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Enabled: true,
			Bind:    "", // Will be auto-selected (Ethernet first, then WLAN)
			HTTP: PortConfig{
				Port:    8009,
				Enabled: true,
			},
			SOCKS5: SOCKSConfig{
				Port:    8010,
				Enabled: true,
				Auth: SOCKSAuthConfig{
					Enabled: false,
				},
			},
		},
		NICs: NICsConfig{
			Default:      "",
			DisplayNames: make(map[string]string),
		},
		Egress: []EgressPolicy{},
		Routing: RoutingConfig{
			DefaultEgress: "",
			Rules: []Rule{
				{
					ID:       "default-catch-all",
					Name:     "默认规则",
					Priority: 100,
					Enabled:  true,
					ListType: "none",
					Domains:  []string{"*"},
					Action:   "forward",
					EgressID: "", // 运行时自动选择默认出口
				},
			},
		},
		API: APIConfig{
			Bind: "127.0.0.1",
			Port: DefaultAPIPort,
			Auth: AuthConfig{
				Enabled:  false,
				Username: "",
				Password: "",
			},
		},
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
