# 路由规则管理鲁棒性增强设计

## 概述

增强路由规则管理的健壮性，确保：
1. 优先级范围限制在 0-100
2. 优先级不允许重复
3. 首次启动自动创建默认规则

## 问题分析

### 当前问题

| 位置 | 问题 |
|------|------|
| `web/src/pages/Rules.tsx:345` | 优先级范围 `min={1} max={10000}` 错误 |
| `web/src/pages/Rules.tsx:472` | 导入弹窗优先级范围同样错误 |
| 前端 | 无优先级重复检查 |
| `pkg/api/handlers.go` | createRule/updateRule 无验证逻辑 |
| `internal/router/importer.go` | 导入功能无验证 |
| `pkg/config/config.go` | 无默认路由规则 |

### 影响

- 用户可能创建优先级超出范围的规则，导致排序混乱
- 重复优先级导致规则匹配不稳定
- 新用户首次启动无任何规则，所有流量行为不明确

## 设计方案

### 1. 后端验证层

新增文件 `internal/router/validation.go`：

```go
package router

import "fmt"

const (
    MinPriority = 0
    MaxPriority = 100
)

// ValidateRule 验证规则的有效性
// existingRules: 用于检查优先级重复
// excludeID: 更新时排除自身的 ID
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

### 5. 默认规则

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
| `internal/router/validation.go` | 新增 | 验证逻辑 |
| `internal/router/manager.go` | 修改 | 添加 `GetPriorities()` 辅助方法（可选） |
| `pkg/api/handlers.go` | 修改 | createRule/updateRule 添加验证 |
| `internal/router/importer.go` | 修改 | 导入前验证优先级 |
| `pkg/config/config.go` | 修改 | DefaultConfig 添加默认规则 |
| `web/src/pages/Rules.tsx` | 修改 | 优先级范围修正 + 前端验证 |

## 错误消息设计

| 场景 | 错误消息 |
|------|----------|
| 优先级超出范围 | `优先级必须在 0-100 之间，当前值: {value}` |
| 优先级重复 | `优先级 {value} 已被规则 '{rule_name}' 使用` |

## 测试用例

1. 创建规则时优先级为 -1 → 返回 400 错误
2. 创建规则时优先级为 101 → 返回 400 错误
3. 创建规则时优先级重复 → 返回 400 错误
4. 更新规则时保持原优先级 → 成功
5. 更新规则时改为其他规则的优先级 → 返回 400 错误
6. 导入规则时优先级无效 → 返回错误
7. 首次启动无配置文件 → 自动创建默认规则
8. 前端输入超出范围 → 输入框限制 + 验证提示
