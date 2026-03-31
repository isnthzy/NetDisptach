package egress

import (
	"fmt"
	"net"
)

// ServerConfig contains server configuration for loop detection
type ServerConfig struct {
	BindIP    string
	HTTPPort  int
	SOCKS5Port int
}

// ValidatePolicy 验证出口策略的有效性
// existingPolicies: 用于检查 ID 和名称重复
// excludeID: 更新时排除自身的 ID
// validNICs: 有效的网卡名称列表（可选，为空则跳过网卡验证）
// serverCfg: 服务器配置（可选，用于检测循环引用）
func ValidatePolicy(policy *Policy, existingPolicies []*Policy, excludeID string, validNICs []string, serverCfg *ServerConfig) error {
	// 检查 ID 不能为空
	if policy.ID == "" {
		return fmt.Errorf("策略 ID 不能为空")
	}

	// 检查名称不能为空
	if policy.Name == "" {
		return fmt.Errorf("策略名称不能为空")
	}

	// 检查 ID 唯一性
	for _, existing := range existingPolicies {
		if existing.ID == excludeID {
			continue
		}
		if existing.ID == policy.ID {
			return fmt.Errorf("策略 ID '%s' 已存在", policy.ID)
		}
	}

	// 检查名称唯一性
	for _, existing := range existingPolicies {
		if existing.ID == excludeID {
			continue
		}
		if existing.Name == policy.Name {
			return fmt.Errorf("策略名称 '%s' 已存在", policy.Name)
		}
	}

	// 检查 NIC 是否有效（仅当提供了有效网卡列表时）
	if policy.NIC != "" && len(validNICs) > 0 {
		found := false
		for _, nic := range validNICs {
			if nic == policy.NIC {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("网卡 '%s' 不存在", policy.NIC)
		}
	}

	// 检查代理配置
	if policy.Proxy != nil {
		if policy.Proxy.Host == "" {
			return fmt.Errorf("代理地址不能为空")
		}
		if policy.Proxy.Port < 1 || policy.Proxy.Port > 65535 {
			return fmt.Errorf("代理端口必须在 1-65535 之间")
		}
		if policy.Proxy.Protocol != "http" && policy.Proxy.Protocol != "socks5" {
			return fmt.Errorf("代理协议必须是 http 或 socks5")
		}

		// 检查是否指向自己（避免循环）
		if serverCfg != nil {
			if isLoopAddress(policy.Proxy.Host, policy.Proxy.Port, serverCfg) {
				return fmt.Errorf("上游代理地址 %s:%d 指向本机代理端口，会导致循环，请使用其他代理地址",
					policy.Proxy.Host, policy.Proxy.Port)
			}
		}
	}

	return nil
}

// isLoopAddress checks if the proxy address points to local proxy ports
func isLoopAddress(host string, port int, serverCfg *ServerConfig) bool {
	// Check if port matches HTTP or SOCKS5 proxy port
	if port != serverCfg.HTTPPort && port != serverCfg.SOCKS5Port {
		return false
	}

	// Check if host points to local machine
	// 1. Direct IP match with bind address
	if host == serverCfg.BindIP {
		return true
	}

	// 2. Localhost variants
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		// These point to local machine, check if bind is also local
		if serverCfg.BindIP == "" || serverCfg.BindIP == "0.0.0.0" ||
		   serverCfg.BindIP == "127.0.0.1" || serverCfg.BindIP == "::1" {
			return true
		}
	}

	// 3. Check if host is a local IP address
	localIPs, err := getLocalIPs()
	if err == nil {
		for _, ip := range localIPs {
			if host == ip {
				return true
			}
		}
	}

	return false
}

// getLocalIPs returns all local IP addresses
func getLocalIPs() ([]string, error) {
	var ips []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	return ips, nil
}
