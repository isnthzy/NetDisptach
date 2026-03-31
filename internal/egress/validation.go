package egress

import "fmt"

// ValidatePolicy 验证出口策略的有效性
// existingPolicies: 用于检查 ID 和名称重复
// excludeID: 更新时排除自身的 ID
// validNICs: 有效的网卡名称列表（可选，为空则跳过网卡验证）
func ValidatePolicy(policy *Policy, existingPolicies []*Policy, excludeID string, validNICs []string) error {
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
	}

	return nil
}
