package router

import "fmt"

const (
	MinPriority = 0
	MaxPriority = 100
	MinPort     = 1
	MaxPort     = 65535
)

// ValidateRule 验证规则的有效性
// existingRules: 用于检查优先级重复
// excludeID: 更新时排除自身的 ID
func ValidateRule(rule Rule, existingRules []Rule, excludeID string) error {
	// 检查规则名称
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	// 检查优先级范围
	if rule.Priority < MinPriority || rule.Priority > MaxPriority {
		return fmt.Errorf("优先级必须在 %d-%d 之间，当前值: %d", MinPriority, MaxPriority, rule.Priority)
	}

	// 检查优先级唯一性
	for _, existing := range existingRules {
		if existing.ID == excludeID {
			continue
		}
		if existing.Priority == rule.Priority {
			return fmt.Errorf("优先级 %d 已被规则 '%s' 使用", rule.Priority, existing.Name)
		}
	}

	// 检查端口范围
	for _, port := range rule.Ports {
		if port < MinPort || port > MaxPort {
			return fmt.Errorf("端口必须在 %d-%d 之间，当前值: %d", MinPort, MaxPort, port)
		}
	}

	return nil
}
