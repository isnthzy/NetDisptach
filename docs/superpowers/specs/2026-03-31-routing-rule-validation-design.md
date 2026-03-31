# 路由规则管理鲁棒性增强设计

## 概述

增强路由规则管理的健壮性，确保：
1. 优先级范围限制在 0-100
2. 优先级不允许重复
3. 首次启动自动创建默认规则
4. 出口策略 ID/名称唯一性
5. 端口范围验证
6. API 端口冲突检查

## 问题分析

### 路由规则问题

| 位置 | 问题 | 严重程度 |
|------|------|----------|
| `web/src/pages/Rules.tsx:345` | 优先级范围 `min={1} max={10000}` 错误 | 高 |
| `web/src/pages/Rules.tsx:472` | 导入弹窗优先级范围同样错误 | 高 |
| 前端 Rules.tsx | 无优先级重复检查 | 高 |
| `pkg/api/handlers.go:118-137` | createRule 无验证逻辑 | 高 |
| `pkg/api/handlers.go:139-162` | updateRule 无验证逻辑 | 高 |
| `internal/router/importer.go` | 导入功能无验证 | 高 |
| `pkg/config/config.go` | 无默认路由规则 | 中 |
| 前端 Rules.tsx | 端口范围未验证（应 1-65535） | 中 |
| `pkg/api/handlers.go:501-538` | importRuleFromURL 返回 500 而非 400 | 低 |

### 出口策略问题

| 位置 | 问题 | 严重程度 |
|------|------|----------|
| `pkg/api/handlers.go:44-58` | createEgress 无验证（ID 重复、名称重复） | 高 |
| `pkg/api/handlers.go:60-78` | updateEgress 无验证 | 高 |
| `internal/egress/manager.go:57-62` | Add 方法直接覆盖同名 ID 策略 | 中 |
| 前端 Egress.tsx | 代理端口范围未验证（使用 Input 而非 InputNumber） | 中 |
| 前端 Egress.tsx | 无 NIC 存在性验证 | 低 |

### 端口冲突问题

| 位置 | 问题 | 严重程度 |
|------|------|----------|
| `pkg/api/handlers.go:287-308` | validateConfig 未检查 API 端口与 HTTP/SOCKS5 冲突 | 中 |

### SOCKS5 认证问题

| 位置 | 问题 | 严重程度 |
|------|------|----------|
| 前端 Settings.tsx | SOCKS5 用户名可重复添加 | 低 |

### 影响分析

- 用户可能创建优先级超出范围的规则，导致排序混乱
- 重复优先级导致规则匹配不稳定
- 新用户首次启动无任何规则，所有流量行为不明确
- 出口策略 ID 重复导致意外覆盖
- 端口冲突导致服务启动失败

## 设计方案

### 1. 路由规则验证层

新增文件 `internal/router/validation.go`：

```go
package router

import "fmt"

const (
    MinPriority = 0
    MaxPriority = 100
    MinPort     = 1
    MaxPort     = 65535
)

// ValidateRule 验证规则的有效性
func ValidateRule(rule Rule, existingRules []Rule, excludeID string) error {
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

    // 检查规则名称
    if rule.Name == "" {
        return fmt.Errorf("规则名称不能为空")
    }

    return nil
}
```

### 2. API 层验证

修改 `pkg/api/handlers.go`：

**createRule 函数：**
```go
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
    var rule router.Rule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // 验证规则
    existingRules := s.routerMgr.ListRules()
    if err := router.ValidateRule(rule, existingRules, ""); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    // ... 原有逻辑
}
```

**updateRule 函数：**
```go
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    var rule router.Rule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    rule.ID = id

    // 验证规则（排除自身 ID）
    existingRules := s.routerMgr.ListRules()
    if err := router.ValidateRule(rule, existingRules, id); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    // ... 原有逻辑
}
```

### 3. 导入验证

修改 `internal/router/importer.go` 的 `createImportedRule` 函数：

```go
func (i *Importer) createImportedRule(name string, domains []string, source, egressID string, priority int, enabled bool) (*ImportResult, error) {
    if len(domains) == 0 {
        return nil, fmt.Errorf("no valid domains found")
    }

    // 验证优先级
    existingRules := i.manager.ListRules()
    tempRule := Rule{Priority: priority}
    if err := ValidateRule(tempRule, existingRules, ""); err != nil {
        return nil, err
    }

    // ... 原有逻辑
}
```

### 4. 前端验证

修改 `web/src/pages/Rules.tsx`：

**优先级输入组件：**
```tsx
// 规则表单
<Form.Item
  name="priority"
  label="优先级"
  rules={[
    { required: true, message: '请输入优先级' },
    {
      validator: (_, value) => {
        const rules = form.getFieldValue('rules') || [];
        const isDuplicate = rules.some((r: Rule) =>
          r.priority === value && r.id !== editingRule?.id
        );
        if (isDuplicate) {
          return Promise.reject('优先级已被其他规则使用');
        }
        return Promise.resolve();
      }
    }
  ]}
>
  <InputNumber min={0} max={100} style={{ width: '100%' }} />
</Form.Item>
```

**导入表单：**
```tsx
<Form.Item name="priority" label="优先级">
  <InputNumber min={0} max={100} style={{ width: '100%' }} />
</Form.Item>
```

### 5. 出口策略验证层

新增文件 `internal/egress/validation.go`：

```go
package egress

import "fmt"

// ValidatePolicy 验证出口策略的有效性
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

    // 检查名称唯一性（可选，但建议）
    for _, existing := range existingPolicies {
        if existing.ID == excludeID {
            continue
        }
        if existing.Name == policy.Name {
            return fmt.Errorf("策略名称 '%s' 已存在", policy.Name)
        }
    }

    // 检查 NIC 是否有效
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
```

### 6. 端口冲突检查

修改 `pkg/api/handlers.go` 的 `validateConfig` 函数：

```go
func (s *Server) validateConfig(cfg *config.Config) error {
    // 端口范围验证
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

    // 端口冲突检查
    ports := make(map[int]string)
    if cfg.Server.HTTP.Enabled {
        ports[cfg.Server.HTTP.Port] = "HTTP"
    }
    if cfg.Server.SOCKS5.Enabled {
        if existing, ok := ports[cfg.Server.SOCKS5.Port]; ok {
            return fmt.Errorf("SOCKS5 端口与 %s 端口冲突: %d", existing, cfg.Server.SOCKS5.Port)
        }
        ports[cfg.Server.SOCKS5.Port] = "SOCKS5"
    }
    // API 端口冲突检查
    if _, ok := ports[cfg.API.Port]; ok {
        return fmt.Errorf("API 端口与 %s 端口冲突: %d", ports[cfg.API.Port], cfg.API.Port)
    }

    return nil
}
```

### 7. 前端验证增强

**修改 `web/src/pages/Rules.tsx`：**
- 优先级范围修正为 `min={0} max={100}`
- 添加端口范围验证
- 添加前端优先级重复检查

**修改 `web/src/pages/Egress.tsx`：**
- 代理端口改为 InputNumber，范围 `min={1} max={65535}`
- 添加代理主机必填验证

**修改 `web/src/pages/Settings.tsx`：**
- SOCKS5 用户名重复检查

### 8. 默认规则

修改 `pkg/config/config.go` 的 `DefaultConfig` 函数：

```go
func DefaultConfig() *Config {
    return &Config{
        Server: ServerConfig{
            Enabled: true,
            Bind:    "",
            HTTP: PortConfig{
                Port:    809,
                Enabled: true,
            },
            SOCKS5: SOCKSConfig{
                Port:    810,
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
```

## 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/router/validation.go` | 新增 | 路由规则验证逻辑 |
| `internal/egress/validation.go` | 新增 | 出口策略验证逻辑 |
| `pkg/api/handlers.go` | 修改 | createRule/updateRule/createEgress/updateEgress 添加验证 |
| `pkg/api/handlers.go` | 修改 | validateConfig 添加 API 端口冲突检查 |
| `internal/router/importer.go` | 修改 | 导入前验证优先级 |
| `pkg/config/config.go` | 修改 | DefaultConfig 添加默认规则 |
| `web/src/pages/Rules.tsx` | 修改 | 优先级范围修正 + 端口验证 + 前端验证 |
| `web/src/pages/Egress.tsx` | 修改 | 代理端口改为 InputNumber + 验证 |
| `web/src/pages/Settings.tsx` | 修改 | SOCKS5 用户名重复检查 |

## 错误消息设计

### 路由规则错误

| 场景 | 错误消息 |
|------|----------|
| 优先级超出范围 | `优先级必须在 0-100 之间，当前值: {value}` |
| 优先级重复 | `优先级 {value} 已被规则 '{rule_name}' 使用` |
| 端口超出范围 | `端口必须在 1-65535 之间，当前值: {value}` |
| 规则名称为空 | `规则名称不能为空` |

### 出口策略错误

| 场景 | 错误消息 |
|------|----------|
| 策略 ID 为空 | `策略 ID 不能为空` |
| 策略 ID 重复 | `策略 ID '{id}' 已存在` |
| 策略名称为空 | `策略名称不能为空` |
| 策略名称重复 | `策略名称 '{name}' 已存在` |
| 网卡不存在 | `网卡 '{nic}' 不存在` |
| 代理端口超出范围 | `代理端口必须在 1-65535 之间` |
| 代理协议无效 | `代理协议必须是 http 或 socks5` |

### 端口冲突错误

| 场景 | 错误消息 |
|------|----------|
| HTTP/SOCKS5 端口冲突 | `SOCKS5 端口与 HTTP 端口冲突: {port}` |
| API 端口冲突 | `API 端口与 {service} 端口冲突: {port}` |

### SOCKS5 认证错误

| 场景 | 错误消息 |
|------|----------|
| 用户名重复 | `用户名 '{username}' 已存在` |

## 测试用例

### 路由规则验证

1. 创建规则时优先级为 -1 → 返回 400 错误
2. 创建规则时优先级为 101 → 返回 400 错误
3. 创建规则时优先级重复 → 返回 400 错误
4. 更新规则时保持原优先级 → 成功
5. 更新规则时改为其他规则的优先级 → 返回 400 错误
6. 导入规则时优先级无效 → 返回 400 错误
7. 创建规则时端口为 0 → 返回 400 错误
8. 创建规则时端口为 65536 → 返回 400 错误
9. 创建规则时名称为空 → 返回 400 错误
10. 首次启动无配置文件 → 自动创建默认规则
11. 前端输入超出范围 → 输入框限制 + 验证提示

### 出口策略验证

1. 创建策略时 ID 为空 → 返回 400 错误
2. 创建策略时 ID 重复 → 返回 400 错误
3. 创建策略时名称为空 → 返回 400 错误
4. 创建策略时名称重复 → 返回 400 错误
5. 创建策略时网卡不存在 → 返回 400 错误（可选，允许配置不存在的网卡）
6. 创建策略时代理端口超出范围 → 返回 400 错误
7. 创建策略时代理协议无效 → 返回 400 错误

### 端口冲突检查

1. HTTP 和 SOCKS5 使用相同端口 → 返回 400 错误
2. API 端口与 HTTP 端口相同 → 返回 400 错误
3. API 端口与 SOCKS5 端口相同 → 返回 400 错误
4. 所有端口不同 → 成功

### SOCKS5 认证

1. 添加重复用户名 → 前端验证阻止
2. 删除用户后重新添加相同用户名 → 成功
