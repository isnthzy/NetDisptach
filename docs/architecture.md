# NetDispatch 架构设计文档

> **AI 助手指南**：本项目提供了 `AGENTS.md` 文件，包含项目上下文、开发指南和常见问题解决方案。
>
> **重要**：在开始开发之前，请先阅读 `AGENTS.md` 文件以快速理解项目。
>
> 如果你使用的是 Claude Code，可以运行 `/read AGENTS.md` 来加载项目上下文。

---

## 1. 项目概述

### 1.1 项目名称
NetDispatch - 多协议代理网络调度器

### 1.2 项目目标
构建一个高性能、可扩展的代理服务内核，支持多协议代理、智能出口路由、实时流量监控。

### 1.3 核心功能
| 功能模块 | 描述 |
|---------|------|
| 多协议代理 | 支持 HTTP/HTTPS/SOCKS5 协议 |
| 出口策略 | 网卡 + 代理服务器的组合策略 |
| 智能路由 | 基于黑白名单匹配出口策略 |
| Web 控制台 | 实时流量监控、策略配置、规则管理 |
| 系统托盘 | 最小化到托盘，右键菜单操作 |
| 中文界面 | 全中文 Web 控制台界面 |

### 1.4 核心概念

**出口策略 (Egress Policy)** = 网卡 + 代理服务器（可选）

```
┌─────────────────────────────────────────────┐
│ 策略 A: "网线直连"                            │
│   - NIC: eth0 (网线)                        │
│   - Proxy: 无 (直连目标)                     │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 策略 B: "WiFi走代理"                          │
│   - NIC: wlan0 (WiFi)                       │
│   - Proxy: 192.168.1.100:1080 (SOCKS5)      │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 策略 C: "网线走代理"                          │
│   - NIC: eth0 (网线)                        │
│   - Proxy: 10.0.0.50:8080 (HTTP)            │
└─────────────────────────────────────────────┘
```

**路由规则** = 匹配条件 → 出口策略

```
*.google.com     → 策略B (WiFi + 代理)
*.internal.com   → 策略A (网线直连)
10.0.0.0/8       → 策略A (网线直连)
黑名单 IP        → 拒绝
默认             → 策略A
```

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web GUI (前端)                            │
│    ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│    │ 流量监控  │  │ 规则配置  │  │ 状态面板  │  │ 日志查看  │      │
│    └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└───────────────────────────┬─────────────────────────────────────┘
                            │ WebSocket / REST API
┌───────────────────────────▼─────────────────────────────────────┐
│                        API Gateway Layer                         │
│    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│    │  REST API    │  │  WebSocket   │  │  Auth Module │        │
│    └──────────────┘  └──────────────┘  └──────────────┘        │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                        Core Engine Layer                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │HTTP Handler │ │HTTPS Handler│ │SOCKS Handler│              │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘              │
│         └────────────────┼────────────────┘                     │
│                          ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  Connection Manager                       │  │
│  └──────────────────────────┬───────────────────────────────┘  │
│                             ▼                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Route Engine │  │ Upstream Mgr │  │  NIC Selector│         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                     Infrastructure Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Traffic Stats│  │ Config Store │  │  Logger      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│                        Network Layer                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐           │
│  │  NIC 0  │  │  NIC 1  │  │  NIC 2  │  │  NIC N  │           │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 分层职责

| 层级 | 职责 | 核心组件 |
|-----|------|---------|
| Web GUI | 用户交互、数据可视化 | Dashboard, Monitor, Config |
| API Gateway | 接口暴露、认证、协议转换 | HTTP Server, WS Hub |
| Core Engine | 代理逻辑、路由决策 | Protocol Handlers, Router |
| Infrastructure | 配置、监控、日志 | Config, Metrics, Logger |
| Network | 网络I/O、网卡绑定 | NIC Manager |

---

## 3. 核心模块设计

### 3.1 协议处理器 (Protocol Handlers)

#### 3.1.1 HTTP/HTTPS 代理

```
┌─────────────────────────────────────────┐
│            HTTP/HTTPS Handler           │
├─────────────────────────────────────────┤
│  - CONNECT 方法隧道处理                  │
│  - HTTP 请求转发                         │
│  - TLS 透传 (HTTPS)                      │
│  - 证书动态生成 (MITM 可选)              │
└─────────────────────────────────────────┘
```

**处理流程**：
1. 解析客户端请求
2. 判断是否为 CONNECT 方法
3. CONNECT → 建立 TCP 隧道，开始双向转发
4. 普通 HTTP → 转发请求并返回响应

#### 3.1.2 SOCKS 代理

```
┌─────────────────────────────────────────┐
│            SOCKS Handler                │
├─────────────────────────────────────────┤
│  - SOCKS4 协议支持                       │
│  - SOCKS4a 协议支持                      │
│  - SOCKS5 协议支持                       │
│  - 用户名密码认证 (SOCKS5)               │
│  - UDP Associate (可选)                  │
└─────────────────────────────────────────┘
```

**SOCKS5 握手流程**：
```
Client                    Server                    Target
   │                        │                        │
   │─── greeting ──────────►│                        │
   │◄── method selection ───│                        │
   │─── auth (optional) ───►│                        │
   │◄── auth result ────────│                        │
   │─── connect request ───►│                        │
   │                        │──── establish ────────►│
   │◄── connect reply ──────│                        │
   │◄═══════════════════════╪══════════════════════►│
   │        Bidirectional Data Transfer               │
```

### 3.2 路由引擎 (Route Engine)

```
┌─────────────────────────────────────────────────────────────┐
│                      Route Engine                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ Rule Parser │───►│ Rule Matcher│───►│ Route Table │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│                                              │               │
│                                              ▼               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Match Rules                       │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │    │
│  │  │ IP/CIDR │ │ Domain  │ │  Port   │ │ Protocol│   │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                              │               │
│                                              ▼               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  Action Types                        │    │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────────────┐ │    │
│  │  │ DIRECT    │ │ REJECT    │ │ FORWARD -> NIC    │ │    │
│  │  └───────────┘ └───────────┘ └───────────────────┘ │    │
│  │  ┌───────────┐ ┌───────────────────────────────┐   │    │
│  │  │ UPSTREAM  │ │ UPSTREAM -> NIC               │   │    │
│  │  └───────────┘ └───────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 网卡选择器 (NIC Selector)

```
┌─────────────────────────────────────────────────────────────┐
│                      NIC Selector                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  NIC Registry                         │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐        │   │
│  │  │ eth0   │ │ eth1   │ │ wlan0  │ │ tun0   │        │   │
│  │  │Default │ │VPN Line│ │Backup  │ │Tunnel  │        │   │
│  │  └────────┘ └────────┘ └────────┘ └────────┘        │   │
│  └──────────────────────────────────────────────────────┘   │
│                           │                                  │
│                           ▼                                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  Bind Strategy                        │   │
│  │                                                       │   │
│  │  type BindStrategy interface {                        │   │
│  │      SelectNIC(target net.Addr) (*NIC, error)         │   │
│  │  }                                                    │   │
│  │                                                       │   │
│  │  implementations:                                     │   │
│  │  - RuleBasedStrategy  (by whitelist/blacklist)        │   │
│  │  - RoundRobinStrategy (load balance)                  │   │
│  │  - WeightedStrategy   (by NIC weight)                │   │
│  │  - StickyStrategy     (same target -> same NIC)       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.4 出口策略管理器 (Egress Manager)

```
┌─────────────────────────────────────────────────────────────┐
│                     Egress Manager                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Egress Policy                         │  │
│  │                                                        │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │ Egress Config                                    │  │  │
│  │  │ - ID: string                                     │  │  │
│  │  │ - Name: string (策略名称)                         │  │  │
│  │  │ - NIC: string (网卡名称)                          │  │  │
│  │  │ - Proxy: ProxyConfig (可选)                       │  │  │
│  │  │   - Host: string                                 │  │  │
│  │  │   - Port: int                                    │  │  │
│  │  │   - Protocol: http | socks5                      │  │  │
│  │  │   - Username/Password: (可选)                     │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  │                                                        │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐     │  │
│  │  │ 网线直连    │ │ WiFi走代理  │ │ 网线走代理  │     │  │
│  │  │ NIC: eth0   │ │ NIC: wlan0  │ │ NIC: eth0   │     │  │
│  │  │ Proxy: 无   │ │ Proxy: 有   │ │ Proxy: 有   │     │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘     │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Connection Factory                        │  │
│  │  - CreateDirectConn(nic) → 直连目标                    │  │
│  │  - CreateProxyConn(nic, proxy) → 经代理连接            │  │
│  │  - Bind to specific NIC                                │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**连接流程**：

```
客户端请求
    │
    ▼
解析目标地址
    │
    ▼
匹配路由规则 → 获取 EgressPolicy
    │
    ├─ Proxy == nil ────────────────────┐
    │                                    │
    └─ Proxy != nil                      │
         │                               │
         ▼                               ▼
    连接代理服务器                    直连目标服务器
         │                               │
         └───────────┬───────────────────┘
                     ▼
              绑定指定网卡 (bind local IP)
                     │
                     ▼
                建立连接
                     │
                     ▼
               双向转发数据
```

---

## 4. 数据模型

### 4.1 核心数据结构

```go
// NIC represents a network interface card
type NIC struct {
    ID          string
    Name        string    // "eth0", "wlan0"
    DisplayName string    // "网线", "WiFi"
    IP          net.IP
    Netmask     net.IPMask
    IsUp        bool
    IsDefault   bool
}

// ProxyConfig represents an upstream proxy configuration
type ProxyConfig struct {
    Host     string    // 代理服务器 IP
    Port     int       // 代理端口
    Protocol string    // "http", "socks5"
    Username string    // 可选认证
    Password string    // 可选认证
}

// EgressPolicy represents an egress policy (NIC + optional proxy)
type EgressPolicy struct {
    ID          string
    Name        string       // "网线直连", "WiFi走代理"
    NIC         string       // "eth0", "wlan0"
    Proxy       *ProxyConfig // nil = 直连，非 nil = 走代理
    Description string
}

// Rule represents a routing rule
type Rule struct {
    ID          string
    Priority    int
    Enabled     bool

    // Match conditions
    Domains     []string   // "*.google.com", "*.youtube.com"
    CIDRs       []string   // "192.168.0.0/16", "10.0.0.0/8"
    Ports       []int      // 80, 443
    Protocols   []string   // "http", "https", "socks"

    // Action
    Action      ActionType
    EgressID    string      // EgressPolicy.ID for FORWARD action
}

type ActionType int

const (
    ActionForward  ActionType = iota  // Forward via egress policy
    ActionReject                        // Reject connection
)

// Connection represents an active connection
type Connection struct {
    ID           string
    ClientAddr   net.Addr
    TargetAddr   net.Addr
    Protocol     string
    EgressID     string    // 使用的出口策略
    NIC          string    // 实际使用的网卡
    ProxyUsed    bool      // 是否使用了代理
    StartTime    time.Time
    BytesIn      int64
    BytesOut     int64
    RuleMatched  string    // 匹配的规则 ID
}

// TrafficStats represents traffic statistics
type TrafficStats struct {
    Timestamp          time.Time
    TotalConnections   int64
    ActiveConnections  int64
    BytesIn            int64
    BytesOut           int64

    // By NIC
    ConnectionsByNIC   map[string]int64
    TrafficByNIC       map[string]int64

    // By Protocol
    ConnectionsByProtocol map[string]int64

    // By Egress Policy
    ConnectionsByEgress   map[string]int64
}
```

### 4.2 配置模型

```yaml
# config.yaml
server:
  # 是否启用代理服务
  enabled: true
  # 代理服务绑定地址 (自动选择或指定IP)
  bind: "0.0.0.0"

  # HTTP 代理端口 (支持 CONNECT 隧道)
  http:
    port: 809
    enabled: true

  # SOCKS5 代理端口
  socks5:
    port: 810
    enabled: true
    auth:
      enabled: false
      users: []

# 网卡配置 (自动检测)
nics:
  default: ""
  display_names: {}

# 出口策略
egress:
  - id: "eth0-direct"
    name: "网线直连"
    nic: "eth0"

  - id: "wlan0-proxy"
    name: "WiFi走代理"
    nic: "wlan0"
    proxy:
      host: "192.168.1.100"
      port: 1080
      protocol: "socks5"
      username: ""
      password: ""

# 路由规则
routing:
  default_egress: "eth0-direct"

  rules:
    - id: "rule-001"
      priority: 100
      enabled: true
      domains: ["*.google.com", "*.youtube.com"]
      action: "forward"
      egress_id: "wlan0-proxy"

    - id: "rule-002"
      priority: 200
      enabled: true
      cidrs: ["10.0.0.0/8", "192.168.0.0/16"]
      action: "forward"
      egress_id: "eth0-direct"

    - id: "rule-003"
      priority: 300
      enabled: true
      domains: ["*.badsite.com"]
      action: "reject"

# API 服务 (Web 控制台由 API 服务提供)
api:
  bind: "127.0.0.1"
  port: 9090
  auth:
    enabled: false
    username: ""
    password: ""
```

---

## 5. 项目结构

```
netdispatch/
├── cmd/
│   └── netdispatch/
│       ├── main.go              # 程序入口
│       ├── console_windows.go   # Windows 控制台隐藏
│       ├── console_other.go     # 其他平台控制台空实现
│       ├── messagebox_windows.go # Windows 弹窗
│       └── messagebox_other.go  # 其他平台弹窗空实现
│
├── internal/
│   ├── connmgr/                 # 连接管理
│   │   └── manager.go           # 连接追踪与统计
│   │
│   ├── handler/                 # 协议处理器
│   │   ├── handler.go           # Handler 接口
│   │   ├── http.go              # HTTP/HTTPS 处理器
│   │   └── socks.go             # SOCKS5 处理器
│   │
│   ├── router/                  # 路由引擎
│   │   ├── router.go            # 路由器
│   │   ├── manager.go           # 规则管理
│   │   └── rule.go              # 规则匹配
│   │
│   ├── egress/                  # 出口策略管理
│   │   ├── manager.go           # 策略管理器
│   │   └── errors.go            # 错误定义
│   │
│   ├── nic/                     # 网卡管理
│   │   ├── manager.go           # 网卡发现与管理
│   │   ├── binding.go           # NIC 绑定连接
│   │   └── errors.go            # 错误定义
│   │
│   ├── proxy/                   # 核心代理逻辑
│   │   └── server.go            # 代理服务器
│   │
│   └── tray/                    # 系统托盘
│       ├── tray.go              # 托盘实现
│       └── icon.go              # 图标嵌入
│
├── pkg/
│   ├── api/                     # REST API
│   │   ├── server.go            # API 服务器
│   │   └── handlers.go          # API 处理器
│   │
│   ├── ws/                      # WebSocket
│   │   ├── hub.go               # 连接中心
│   │   └── loghook.go           # 日志广播
│   │
│   ├── config/                  # 配置
│   │   └── config.go            # 配置加载
│   │
│   ├── crashlog/                # 崩溃日志
│   │   └── crashlog.go          # 崩溃日志记录
│   │
│   └── singleinstance/          # 单实例检测
│       ├── singleinstance.go    # 单实例接口
│       ├── lock_windows.go      # Windows 文件锁
│       └── lock_unix.go         # Unix 文件锁
│
├── web/                         # Web GUI (前端)
│   ├── src/
│   │   ├── components/          # 通用组件
│   │   ├── pages/               # 页面组件
│   │   ├── services/            # API 服务
│   │   └── App.tsx              # 主应用
│   ├── embed.go                 # 嵌入前端资源
│   └── package.json
│
├── configs/
│   └── config.yaml              # 默认配置
│
├── docs/
│   ├── architecture.md          # 本文档
│   └── build-guide.md           # 编译指南
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. API 设计

### 6.1 REST API Endpoints

```
# 连接管理
GET    /api/v1/connections              # 获取连接列表
GET    /api/v1/connections/:id          # 获取连接详情
DELETE /api/v1/connections/:id          # 关闭连接

# 出口策略管理
GET    /api/v1/egress                   # 获取所有出口策略
POST   /api/v1/egress                   # 创建出口策略
PUT    /api/v1/egress/:id               # 更新出口策略
DELETE /api/v1/egress/:id               # 删除出口策略
POST   /api/v1/egress/:id/test          # 测试出口策略连通性

# 路由规则管理
GET    /api/v1/rules                    # 获取所有规则
POST   /api/v1/rules                    # 创建规则
PUT    /api/v1/rules/:id                # 更新规则
DELETE /api/v1/rules/:id                # 删除规则
POST   /api/v1/rules/reorder            # 重新排序规则

# 网卡管理
GET    /api/v1/nics                     # 获取可用网卡列表
GET    /api/v1/nics/:name               # 获取网卡详情

# 统计信息
GET    /api/v1/stats/overview           # 概览统计
GET    /api/v1/stats/traffic            # 流量历史
GET    /api/v1/stats/connections        # 连接历史

# 配置管理
GET    /api/v1/config                   # 获取当前配置
PUT    /api/v1/config                   # 更新配置
POST   /api/v1/config/reload            # 重新加载配置文件

# 系统
GET    /api/v1/system/info              # 系统信息
GET    /api/v1/health                   # 健康检查
```

### 6.2 WebSocket Events

```typescript
// Client -> Server
{
  "type": "subscribe",
  "channels": ["traffic", "connections", "logs"]
}

// Server -> Client: 流量更新 (每2秒推送)
{
  "type": "traffic",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "bytes_in": 1024000,
    "bytes_out": 2048000,
    "active_connections": 42
  }
}

// Server -> Client: 连接事件
{
  "type": "connection",
  "action": "created" | "closed",
  "data": {
    "id": "conn-123",
    "client": "192.168.1.100:54321",
    "target": "example.com:443",
    "protocol": "https",
    "egress": "wlan0-proxy",
    "nic": "wlan0",
    "proxy_used": true
  }
}

// Server -> Client: 日志事件
{
  "type": "log",
  "level": "info" | "warn" | "error",
  "message": "Connection established",
  "fields": {
    "connection_id": "conn-123",
    "egress": "wlan0-proxy"
  }
}
```

---

## 7. Web GUI 设计

### 7.1 页面结构

```
┌─────────────────────────────────────────────────────────────┐
│  NetDispatch    Dashboard   出口策略   路由规则   日志   设置  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │                    Main Content                     │     │
│  │                                                     │     │
│  │  - Dashboard: 流量监控、连接列表、状态卡片          │     │
│  │  - 出口策略: 策略列表、添加/编辑策略               │     │
│  │  - 路由规则: 规则列表、添加/编辑规则、拖拽排序     │     │
│  │  - 日志: 实时日志查看、过滤                        │     │
│  │  - 设置: 系统配置、认证设置                        │     │
│  │                                                     │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│  状态栏: 连接: 42 | 入站: 1.2 MB/s | 出站: 2.4 MB/s         │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Dashboard 页面

```
┌─────────────────────────────────────────────────────────────┐
│                        Dashboard                             │
├──────────────────┬──────────────────┬───────────────────────┤
│ 活跃连接         │ 入站流量         │ 出站流量              │
│      42          │    1.2 MB/s      │    2.4 MB/s           │
│  [▲ 12%]         │  [▲ 8%]          │  [▼ 3%]               │
├──────────────────┴──────────────────┴───────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              实时流量图表                            │    │
│  │   ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁                     │    │
│  │   ── 入站 (绿色)  ── 出站 (蓝色)                     │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
├────────────────────────┬────────────────────────────────────┤
│   网卡流量分布          │    出口策略使用                    │
│   ┌────────────────┐   │   ┌────────────────────────────┐   │
│   │ 网线: ████████ │   │   │ 网线直连  ████████  60%   │   │
│   │ WiFi: ████     │   │   │ WiFi走代理 ████      30%   │   │
│   └────────────────┘   │   │ 拒绝      ██        10%   │   │
│                        │   └────────────────────────────┘   │
├────────────────────────┴────────────────────────────────────┤
│                     最近连接                                 │
│ ┌────────┬──────────────┬────────┬──────────┬───────────┐  │
│ │ 客户端 │ 目标         │ 协议   │ 出口策略  │ 持续时间  │  │
│ ├────────┼──────────────┼────────┼──────────┼───────────┤  │
│ │ 10.0...│ google.com   │ HTTPS  │ WiFi代理 │ 2m 34s    │  │
│ │ 10.0...│ internal.com │ HTTPS  │ 网线直连 │ 1m 12s    │  │
│ └────────┴──────────────┴────────┴──────────┴───────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 7.3 出口策略管理页面

```
┌─────────────────────────────────────────────────────────────┐
│  出口策略管理                              [+ 添加策略]      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 网线直连                                    [编辑] [删除] │
│  │ ─────────────────────────────────────────────────── │   │
│  │ 网卡: 网线 (eth0)                                   │   │
│  │ 代理: 不使用                                        │   │
│  │ 状态: ● 正常                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ WiFi走代理                                  [编辑] [删除] │
│  │ ─────────────────────────────────────────────────── │   │
│  │ 网卡: WiFi (wlan0)                                  │   │
│  │ 代理: SOCKS5://192.168.1.100:1080                   │   │
│  │ 状态: ● 正常                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 网线走代理                                  [编辑] [删除] │
│  │ ─────────────────────────────────────────────────── │   │
│  │ 网卡: 网线 (eth0)                                   │   │
│  │ 代理: HTTP://10.0.0.50:8080                         │   │
│  │ 状态: ● 正常                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**添加/编辑策略弹窗**：

```
┌─────────────────────────────────────────────────────────────┐
│  添加出口策略                                          [×]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  策略名称                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ WiFi走代理                                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  选择网卡                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ WiFi (wlan0)                                    ▼  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ [√] 使用代理服务器                                    │ │
│  └───────────────────────────────────────────────────────┘ │
│                                                             │
│  代理协议                                                   │
│  ┌──────────────┐                                          │
│  │ SOCKS5    ▼ │                                          │
│  └──────────────┘                                          │
│                                                             │
│  代理地址               端口                                │
│  ┌────────────────────┐  ┌────────────┐                    │
│  │ 192.168.1.100      │  │ 1080       │                    │
│  └────────────────────┘  └────────────┘                    │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ [ ] 需要认证                                          │ │
│  └───────────────────────────────────────────────────────┘ │
│                                                             │
│  用户名                密码                                 │
│  ┌────────────────────┐  ┌────────────┐                    │
│  │                    │  │            │                    │
│  └────────────────────┘  └────────────┘                    │
│                                                             │
│                          [测试连接]  [取消]  [保存]         │
└─────────────────────────────────────────────────────────────┘
```

### 7.4 路由规则管理页面

```
┌─────────────────────────────────────────────────────────────┐
│  路由规则管理                              [+ 添加规则]      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  优先级 │ 匹配条件          │ 动作          │ 状态 │ 操作   │
│  ──────┼──────────────────┼───────────────┼──────┼───────   │
│   100  │ *.google.com     │ WiFi走代理    │ ● 开 │ [编辑]   │
│        │ *.youtube.com    │               │      │ [删除]   │
│  ──────┼──────────────────┼───────────────┼──────┼───────   │
│   200  │ 10.0.0.0/8       │ 网线直连      │ ● 开 │ [编辑]   │
│        │ 192.168.0.0/16   │               │      │ [删除]   │
│  ──────┼──────────────────┼───────────────┼──────┼───────   │
│   300  │ *.badsite.com    │ 拒绝连接      │ ● 开 │ [编辑]   │
│        │                  │               │      │ [删除]   │
│  ──────┼──────────────────┼───────────────┼──────┼───────   │
│  默认  │ * (所有其他)      │ 网线直连      │ ● 开 │ [编辑]   │
│                                                             │
│  提示: 规则按优先级从高到低匹配，拖拽行可调整顺序            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**添加/编辑规则弹窗**：

```
┌─────────────────────────────────────────────────────────────┐
│  添加路由规则                                          [×]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  规则名称                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Google服务走WiFi代理                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  匹配条件                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 域名 (每行一个，支持通配符 *)                        │   │
│  │ *.google.com                                        │   │
│  │ *.youtube.com                                       │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ IP/CIDR (每行一个)                                  │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 端口 (逗号分隔)                                     │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  动作                                                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ ○ 转发到出口策略                                      │  │
│  │   ┌────────────────────────────────────────────────┐ │  │
│  │   │ WiFi走代理                                  ▼  │ │  │
│  │   └────────────────────────────────────────────────┘ │  │
│  │                                                       │  │
│  │ ○ 拒绝连接                                           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│                          [取消]  [保存]                     │
└─────────────────────────────────────────────────────────────┘
```

### 7.5 设置页面

```
┌─────────────────────────────────────────────────────────────┐
│  系统设置                                                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  代理服务设置                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 绑定地址 (本机IP，客户端通过此IP连接代理)            │   │
│  │ ┌─────────────────────────────────────────────────┐ │   │
│  │ │ 0.0.0.0 (所有网卡)                        ▼    │ │   │
│  │ │                                                  │ │   │
│  │ │ 自动检测到的本机IP:                              │ │   │
│  │ │ ● 192.168.1.50  (网线 eth0)                     │ │   │
│  │ │ ● 192.168.2.100 (WiFi wlan0)                    │ │   │
│  │ │ ● 127.0.0.1     (本地回环)                       │ │   │
│  │ └─────────────────────────────────────────────────┘ │   │
│  │                                                      │   │
│  │ 代理端口设置                                         │   │
│  │ ┌────────────────────────────────────────────────┐  │   │
│  │ │ HTTP/HTTPS 端口: [8009     ]   [√] 启用        │  │   │
│  │ │ SOCKS5 端口:     [8010     ]   [√] 启用        │  │   │
│  │ └────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Web 控制台设置                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 监听地址: [0.0.0.0       ]                          │   │
│  │ 监听端口: [3000          ]                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  认证设置                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [√] 启用 Web 控制台认证                              │   │
│  │ 用户名: [admin        ]                              │   │
│  │ 密码:   [••••••       ]                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [ ] 启用代理服务认证 (客户端需要认证才能使用代理)    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│                          [取消]  [保存设置]                 │
└─────────────────────────────────────────────────────────────┘
```

**绑定地址下拉框详情**：

```
┌─────────────────────────────────────────────┐
│ 绑定地址                                ▼   │
├─────────────────────────────────────────────┤
│ ● 0.0.0.0 (所有网卡)          ← 推荐        │
│ ○ 192.168.1.50 (网线 eth0)                  │
│ ○ 192.168.2.100 (WiFi wlan0)                │
│ ○ 127.0.0.1 (仅本机)                        │
│ ○ 自定义...                                 │
└─────────────────────────────────────────────┘

说明：
- 0.0.0.0: 监听所有网卡，局域网内其他设备可访问
- 具体IP: 只监听该网卡，仅该网段可访问
- 127.0.0.1: 仅本机可访问代理服务
```

### 7.6 技术栈

| 层级 | 技术选型 |
|-----|---------|
| Framework | React 18 + TypeScript |
| Build | Vite |
| UI Components | Ant Design |
| Charts | ECharts |
| State | Zustand |
| Data Fetching | TanStack Query |
| Real-time | native WebSocket |
| Styling | Tailwind CSS |

### 7.7 响应式设计

界面支持多种屏幕尺寸自适应：

**桌面端 (>= 768px)**：
- 固定 220px 侧边栏
- 多列卡片布局
- 完整表格显示

**移动端 (< 768px)**：
- 可折叠抽屉式菜单（点击汉堡图标展开）
- 单列卡片堆叠
- 表格水平滚动
- 顶部固定导航栏

```typescript
// 响应式断点检测
const [isMobile, setIsMobile] = useState(false)

useEffect(() => {
  const checkMobile = () => setIsMobile(window.innerWidth < 768)
  checkMobile()
  window.addEventListener('resize', checkMobile)
  return () => window.removeEventListener('resize', checkMobile)
}, [])
```

### 7.8 实时更新

仪表盘使用 WebSocket 实现实时数据更新：

- 后端每 2 秒广播流量统计
- 自动重连机制
- 连接状态指示器

```typescript
// WebSocket 连接示例
const ws = new WebSocket(`ws://${window.location.host}/ws`)
ws.onmessage = (event) => {
  const data = JSON.parse(event.data)
  if (data.type === 'traffic') {
    setStats(data.data)
  }
}
```

---

## 8. 技术选型

### 8.1 Go 后端

| 功能 | 库/工具 | 说明 |
|-----|--------|------|
| HTTP Server | `net/http` + `github.com/gorilla/mux` | 标准库 + 路由增强 |
| WebSocket | `github.com/gorilla/websocket` | 成熟的 WebSocket 库 |
| Config | `gopkg.in/yaml.v3` | YAML 配置解析 |
| Logging | `github.com/rs/zerolog` | 高性能结构化日志 |
| CLI | `github.com/spf13/cobra` | 命令行框架 |
| System Tray | `github.com/getlantern/systray` | 跨平台系统托盘（使用 LockOSThread 防止冻结） |

### 8.2 关键实现考虑

```go
// Socket binding to specific NIC
func dialWithNIC(network, address, nicName string) (net.Conn, error) {
    localAddr, err := getNICAddress(nicName)
    if err != nil {
        return nil, err
    }

    dialer := &net.Dialer{
        LocalAddr: &net.TCPAddr{IP: localAddr},
    }

    return dialer.Dial(network, address)
}

// SOCKS5 client for upstream
func dialViaUpstream(upstream *Upstream, target string) (net.Conn, error) {
    var conn net.Conn
    var err error

    if upstream.BindNIC != "" {
        conn, err = dialWithNIC("tcp",
            fmt.Sprintf("%s:%d", upstream.Host, upstream.Port),
            upstream.BindNIC)
    } else {
        conn, err = net.Dial("tcp",
            fmt.Sprintf("%s:%d", upstream.Host, upstream.Port))
    }

    if err != nil {
        return nil, err
    }

    // SOCKS5 handshake...
    return doSocks5Handshake(conn, upstream, target)
}
```

---

## 9. 开发计划

### Phase 1: 核心代理 ✅
- [x] 项目脚手架搭建
- [x] HTTP/HTTPS 代理处理器
- [x] SOCKS5 代理处理器
- [x] 基础连接管理
- [x] 配置文件加载

### Phase 2: 网卡与出口策略 ✅
- [x] 网卡自动检测与管理
- [x] Socket 绑定实现
- [x] 出口策略定义与存储
- [x] 直连/代理连接器

### Phase 3: 路由引擎 ✅
- [x] 规则匹配引擎 (域名/IP CIDR/端口)
- [x] 路由决策逻辑
- [x] 默认策略处理

### Phase 4: API 服务 ✅
- [x] REST API 实现
- [x] WebSocket 实时事件
- [x] 统计数据收集

### Phase 5: Web GUI ✅
- [x] React 项目搭建
- [x] Dashboard 流量监控
- [x] 出口策略管理界面
- [x] 路由规则管理界面
- [x] 系统设置界面 (端口/IP绑定)

### Phase 6: 完善 ✅
- [x] 系统托盘
- [x] 中文界面
- [x] 使用手册
- [x] 实时日志

### 后续优化
- [ ] 性能优化
- [ ] 单元测试
- [ ] API 认证

---

## 10. 扩展考虑

### 10.1 可选功能
- MITM 支持（HTTPS 解密）
- 缓存支持
- 访问控制列表 (ACL)
- 流量限速
- 负载均衡策略
- 集群模式

### 10.2 安全考虑
- API 认证 (JWT)
- TLS 加密
- 输入验证
- 速率限制
- 审计日志

---

## 附录

### A. 参考项目
- CCProxy (功能参考)
- Squid (架构参考)
- V2Ray (协议实现参考)
- Clash (路由规则参考)

### B. 相关协议
- RFC 7230-7235: HTTP/1.1
- RFC 1928: SOCKS5
- RFC 1918: SOCKS4

---

## 11. 系统托盘

### 11.1 功能说明

程序启动后会最小化到 Windows 系统托盘，提供以下功能：

| 操作 | 功能 |
|-----|------|
| 左键双击 | 打开 Web 控制台 |
| 右键菜单 | 显示菜单选项 |
| 右键 → 打开网页 | 打开 Web 控制台 |
| 右键 → 退出 | 关闭程序 |

### 11.2 技术实现

使用 `fyne.io/systray` 库实现跨平台系统托盘。

```go
// 托盘初始化
func onReady() {
    systray.SetIcon(IconICO)
    systray.SetTitle("NetDispatch")
    systray.SetTooltip("NetDispatch 网络调度器")

    mOpen := systray.AddMenuItem("打开网页", "打开 Web 控制台")
    systray.AddSeparator()
    mQuit := systray.AddMenuItem("退出", "退出程序")

    go func() {
        for {
            select {
            case <-mOpen.ClickedCh:
                openBrowser("http://127.0.0.1:9090")
            case <-mQuit.ClickedCh:
                systray.Quit()
            }
        }
    }()
}
```

---

## 12. 黑白名单功能

### 12.1 名单类型

路由规则支持三种名单类型：

| 类型 | 说明 |
|-----|------|
| 普通规则 | 按优先级匹配，匹配后执行指定动作 |
| 白名单 | 仅允许匹配的地址通过，其他拒绝 |
| 黑名单 | 阻止匹配的地址访问，其他允许 |

### 12.2 数据模型

```go
type ListType string

const (
    ListTypeNone      ListType = "none"       // 普通规则
    ListTypeWhitelist ListType = "whitelist"  // 白名单
    ListTypeBlacklist ListType = "blacklist"  // 黑名单
)

type Rule struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Priority    int      `json:"priority"`
    Enabled     bool     `json:"enabled"`
    ListType    ListType `json:"list_type"`
    Domains     []string `json:"domains"`
    CIDRs       []string `json:"cidrs"`
    Ports       []int    `json:"ports"`
    Action      string   `json:"action"`
    EgressID    string   `json:"egress_id"`
    Description string   `json:"description"`
}
```

### 12.3 使用场景

**白名单示例**：
```
类型: 白名单
域名: *.internal.company.com, *.trusted.com
动作: 仅允许这些域名访问，其他全部拒绝
```

**黑名单示例**：
```
类型: 黑名单
域名: *.badsite.com, *.malware.net
CIDR: 192.168.100.0/24
动作: 阻止这些地址，其他全部允许
```

---

## 13. 中文界面

### 13.1 界面语言

Web 控制台全中文界面，包括：

- 仪表盘
- 出口策略
- 路由规则
- 日志
- 设置
- 使用手册

### 13.2 使用手册

内置使用手册页面，包含：

1. 什么是出口策略？
2. 如何配置出口策略？
3. 什么是路由规则？
4. 什么是黑白名单？
5. 如何配置路由规则？
6. 如何设置代理端口？
7. 如何设置绑定地址？
8. 客户端如何配置代理？
9. 如何查看实时流量？
10. 系统托盘功能
11. Web 监听控制台是什么？
12. 如何筛选不同网卡的流量日志？
13. 常见问题

---

## 13.3 代理服务设置

### 13.3.1 代理总开关

在设置页面可以开启/关闭代理转发功能。关闭后，所有代理端口将停止监听。

### 13.3.2 绑定地址

绑定地址决定代理服务监听哪个网络接口。

**自动选择规则**（优先级从高到低）：
1. WLAN（WiFi）
2. 以太网（网线）
3. 默认网关所在网卡
4. 0.0.0.0（所有网卡）

### 13.3.3 默认端口

| 协议 | 默认端口 | 说明 |
|-----|---------|------|
| HTTP/HTTPS | 8009 | 支持 CONNECT 隧道 (HTTPS) |
| SOCKS5 | 8010 | 支持 SOCKS5 协议 |
| API | 9090 | REST API + Web 控制台 |

---

## 13.4 Web 监听控制台

Web 监听控制台是 NetDispatch 提供的 Web 管理界面，通过浏览器访问。

**功能包括**：
- 仪表盘：查看实时流量、活跃连接、流量图表
- 出口策略：管理网卡和代理服务器组合
- 路由规则：配置流量路由策略
- 日志：实时查看连接日志
- 设置：配置代理端口、绑定地址等

**访问方式**：
- 系统托盘右键 → 打开网页
- 系统托盘左键双击
- 浏览器访问 http://localhost:端口（端口自动选择）

**监听设置**：
- 启用/关闭：可通过开关控制是否启用 Web 控制台
- 监听地址：默认 `0.0.0.0`（监听所有网卡），可选 `127.0.0.1`（仅本机）
- 监听端口：自动选择可用端口

**性能说明**：
监听所有网卡（0.0.0.0）的性能损耗极小，因为只是监听端口等待连接。如需更高安全性，可关闭 Web 控制台或改为仅本机访问。

---

## 13.4 日志筛选

在日志页面，提供下拉式筛选功能查看不同出口策略的流量。

**筛选选项**：
- 全部：显示所有日志
- 网线：仅显示通过网线（以太网）转发的日志
- WiFi：仅显示通过 WiFi 转发的日志
- 出口策略名：动态显示已配置的出口策略，选择筛选对应策略的流量

**使用场景**：
- 排查某个网卡的网络问题
- 分析不同网络接口的流量分布
- 监控特定出口策略的使用情况

---

## 14. 实时流量监控

### 14.1 刷新频率

- 流量图表：每 30 秒刷新一次
- 连接列表：每 30 秒刷新一次
- 统计数据：每 30 秒刷新一次

### 14.2 图表展示

- X 轴：最近 30 个时间点（每点 30 秒）
- Y 轴：流量速度 (MB/s)
- 系列：入站流量（绿色）、出站流量（蓝色）

---

## 15. 单实例检测

### 15.1 功能说明

程序只允许运行一个实例，重复启动时会：
- Windows: 弹出提示框 "应用程序已在运行中"
- Linux/macOS: 输出错误信息到 stderr

### 15.2 技术实现

单实例检测采用双重机制：

1. **端口检测**：尝试连接 API 端口，快速检测已运行实例
2. **文件锁**：使用平台特定的文件锁机制

```
┌─────────────────────────────────────────────────────────────┐
│                   单实例检测流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  启动程序                                                    │
│      │                                                      │
│      ▼                                                      │
│  检查 API 端口是否被占用                                      │
│      │                                                      │
│      ├─ 是 → 已有实例运行 → 显示提示 → 退出                    │
│      │                                                      │
│      ▼                                                      │
│  尝试获取文件锁                                               │
│      │                                                      │
│      ├─ 成功 → 继续启动                                       │
│      │                                                      │
│      └─ 失败 → 已有实例运行 → 显示提示 → 退出                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 15.3 平台差异

| 平台 | 文件锁实现 | 提示方式 |
|-----|----------|---------|
| Windows | `LockFileEx` | MessageBox 弹窗 |
| Linux | `flock` | stderr 输出 |
| macOS | `flock` | stderr 输出 |

---

## 16. 智能控制台隐藏 (Windows)

### 16.1 功能说明

在 Windows 上，程序根据启动方式智能决定是否隐藏控制台窗口：

| 启动方式 | 控制台行为 |
|---------|----------|
| 双击启动 (Explorer) | 隐藏控制台，仅显示托盘图标 |
| 命令行启动 (cmd/powershell) | 保持控制台可见 |

### 16.2 检测原理

通过检测父进程名称判断启动方式：

```
父进程是终端程序 (cmd.exe, powershell.exe, etc.)
    → 保持控制台可见

父进程是 Explorer.exe
    → 隐藏控制台

无法确定
    → 默认隐藏控制台（适配双击启动场景）
```

### 16.3 支持的终端程序

- cmd.exe
- powershell.exe / pwsh.exe
- WindowsTerminal.exe / wt.exe
- ConEmu.exe / ConEmu64.exe
- Alacritty.exe
- Hyper.exe
- MobaXterm.exe
- Tabby.exe

---

## 17. 崩溃日志机制

### 17.1 功能说明

当程序发生 panic（内核崩溃）时，自动记录崩溃信息到日志文件。

### 17.2 崩溃日志内容

```
========================================
NetDispatch Crash Log
========================================

Time: 2024-01-15T10:30:00Z
OS: windows
Arch: amd64
Go Version: go1.21.0

----------------------------------------
Panic Value:
----------------------------------------
runtime error: invalid memory address

----------------------------------------
Stack Trace:
----------------------------------------
goroutine 1 [running]:
main.runServer()
    /path/to/main.go:123
main.main()
    /path/to/main.go:456
```

### 17.3 日志存储位置

| 平台 | 路径 |
|-----|------|
| Windows | `%APPDATA%/NetDispatch/crash_logs/` |
| Linux | `~/.config/NetDispatch/crash_logs/` |
| macOS | `~/.config/NetDispatch/crash_logs/` |

### 17.4 技术实现

```go
defer func() {
    if r := recover(); r != nil {
        buf := make([]byte, 64*1024)
        n := runtime.Stack(buf, false)
        stack := buf[:n]

        crashPath := crashlog.WriteCrashLog(r, stack)
        ShowMessageBox("NetDispatch 崩溃",
            fmt.Sprintf("程序发生崩溃。\n\n崩溃日志已保存到:\n%s", crashPath))
        os.Exit(1)
    }
}()
```

---

## 18. 跨平台兼容性

### 18.1 平台支持

| 功能 | Windows | Linux | macOS |
|-----|---------|-------|-------|
| HTTP/HTTPS 代理 | ✅ | ✅ | ✅ |
| SOCKS5 代理 | ✅ | ✅ | ✅ |
| 智能路由 | ✅ | ✅ | ✅ |
| Web 控制台 | ✅ | ✅ | ✅ |
| 系统托盘 | ✅ | ✅ | ✅ |
| 智能控制台隐藏 | ✅ | - | - |
| 单实例弹窗提示 | ✅ | - | - |
| 崩溃日志 | ✅ | ✅ | ✅ |

### 18.2 平台特定代码

使用 Go build tags 实现平台特定代码：

```
cmd/netdispatch/
├── main.go              # 跨平台主入口
├── console_windows.go   # Windows 控制台隐藏 (//go:build windows)
├── console_other.go     # 其他平台空实现 (//go:build !windows)
├── messagebox_windows.go # Windows 弹窗 (//go:build windows)
└── messagebox_other.go  # 其他平台 stderr 输出 (//go:build !windows)

pkg/singleinstance/
├── singleinstance.go    # 跨平台接口
├── lock_windows.go      # Windows 文件锁 (//go:build windows)
└── lock_unix.go         # Unix 文件锁 (//go:build !windows)
```

### 18.3 编译说明

```bash
# Windows (原生编译)
go build -o bin/netdispatch.exe ./cmd/netdispatch

# Linux (交叉编译)
GOOS=linux GOARCH=amd64 go build -o bin/netdispatch ./cmd/netdispatch

# macOS (需要 CGO，建议在 macOS 上编译)
GOOS=darwin GOARCH=amd64 go build -o bin/netdispatch ./cmd/netdispatch
```

> **注意**：macOS 版本因 systray 库依赖 CGO，建议在 macOS 系统上原生编译。

---

## 19. 版本管理

### 19.1 版本注入

版本信息在编译时通过 ldflags 注入：

```bash
# 使用构建脚本
./build.sh v1.0.0

# 或手动指定
go build -ldflags "-s -w \
    -X netdispatch/pkg/version.Version=1.0.0 \
    -X netdispatch/pkg/version.GitCommit=$(git rev-parse --short HEAD) \
    -X netdispatch/pkg/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o netdispatch.exe ./cmd/netdispatch
```

### 19.2 版本包结构

```
pkg/version/
└── version.go    # Version, GitCommit, BuildDate 变量
```

### 19.3 版本显示

- **命令行**：`./netdispatch.exe version`
- **Web GUI**：侧边栏标题下方
- **API**：`GET /api/v1/system/info`

---

## 20. 安全特性

### 20.1 出口策略循环检测

防止用户配置上游代理指向本机代理端口，导致无限循环。

**检测逻辑**（`internal/egress/validation.go`）：

```go
func isLoopAddress(host string, port int, serverCfg *ServerConfig) bool {
    // 检查端口是否匹配本机代理端口
    if port != serverCfg.HTTPPort && port != serverCfg.SOCKS5Port {
        return false
    }

    // 检查主机是否指向本机
    // 1. 直接 IP 匹配
    // 2. localhost / 127.0.0.1 / ::1
    // 3. 本机所有网卡 IP
}
```

**效果**：创建出口策略时，如果上游代理地址为本机代理端口，会返回错误。

### 20.2 单实例检测

确保同一时间只有一个实例运行：

```
~/.local/run/netdispatch/netdispatch.lock
```

---

## 21. 开发脚本

### 21.1 构建脚本 (build.sh)

```bash
./build.sh v1.0.0
```

功能：
1. 编译前端 (`npm run build`)
2. 编译后端并注入版本号
3. 输出可执行文件

### 21.2 发布脚本 (release.sh)

```bash
./release.sh v1.1.0
```

功能：
1. 执行构建
2. 创建 git tag
3. 推送到 GitHub
4. 创建 GitHub Release 并上传可执行文件

---

## 22. GitHub Actions 自动发布

### 22.1 功能说明

项目使用 GitHub Actions 实现自动构建和发布。当推送格式为 `v*.*.*` 的 Git Tag 时，自动触发构建流程。

### 22.2 触发条件

```bash
# 创建并推送 tag 触发发布
git tag v1.0.0
git push origin v1.0.0
```

### 22.3 构建流程

```
┌─────────────────────────────────────────────────────────────┐
│                   GitHub Actions Release                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. 检出代码 (fetch-depth: 0 获取完整历史)                    │
│      │                                                       │
│      ▼                                                       │
│  2. 构建前端 (npm install && npm run build)                  │
│      │                                                       │
│      ▼                                                       │
│  3. 生成 Changelog (从 git commits)                          │
│      │                                                       │
│      ▼                                                       │
│  4. 构建二进制                                                │
│      ├─ Windows amd64 (netdispatch-windows-amd64.exe)       │
│      └─ Linux amd64 (netdispatch-linux-amd64)               │
│      │                                                       │
│      ▼                                                       │
│  5. 创建 GitHub Release                                       │
│      ├─ 上传二进制文件                                        │
│      └─ 包含 Changelog                                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 22.4 构建产物

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Windows | amd64 | `netdispatch-windows-amd64.exe` |
| Linux | amd64 | `netdispatch-linux-amd64` |

### 22.5 版本信息注入

编译时通过 ldflags 注入版本信息：

```bash
go build -ldflags "-s -w \
    -X main.defaultAPIPortStr=9090 \
    -X netdispatch/pkg/version.Version=v1.0.0 \
    -X netdispatch/pkg/version.GitCommit=$(git rev-parse HEAD) \
    -X netdispatch/pkg/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

### 22.6 默认配置嵌入

二进制文件包含嵌入的默认配置 (`pkg/config/default.yaml`)。

首次启动时，如果配置文件不存在，会自动创建：

| 平台 | 配置路径 |
|------|----------|
| Windows | `%APPDATA%/NetDispatch/config.yaml` |
| Linux | `~/.config/NetDispatch/config.yaml` |
| macOS | `~/.config/NetDispatch/config.yaml` |

### 22.7 工作流文件

```
.github/
└── workflows/
    └── release.yml    # 发布工作流定义
```

### 22.8 发布注意事项

1. **本地测试**：发布前先本地构建测试
   ```bash
   ./build.sh v1.0.0
   ./netdispatch.exe start
   ```

2. **版本命名**：使用语义化版本 `vMAJOR.MINOR.PATCH`
   - MAJOR: 不兼容的 API 变更
   - MINOR: 向后兼容的新功能
   - PATCH: Bug 修复

3. **检查 Actions**：推送 tag 后检查 GitHub Actions 日志

---

## 23. 项目文件索引

### 23.1 核心文件

| 文件 | 描述 |
|------|------|
| `AGENTS.md` | AI 助手指南，包含项目上下文和开发指南 |
| `README.md` | 项目介绍和快速开始 |
| `docs/architecture.md` | 本文档，详细架构设计 |
| `docs/build-guide.md` | 编译指南 |
| `configs/config.yaml` | 示例配置文件 |

### 23.2 后端关键模块

| 路径 | 描述 |
|------|------|
| `cmd/netdispatch/main.go` | 程序入口，CLI 定义 |
| `internal/handler/http.go` | HTTP/HTTPS 代理处理 |
| `internal/handler/socks5.go` | SOCKS5 代理处理 |
| `internal/egress/manager.go` | 出口策略管理 |
| `internal/egress/validation.go` | 出口策略验证（含循环检测） |
| `internal/router/manager.go` | 路由规则管理 |
| `internal/router/rule.go` | 规则匹配逻辑 |
| `internal/router/tree.go` | 域名树优化 |
| `internal/connmgr/manager.go` | 连接管理、流量统计 |
| `pkg/api/server.go` | REST API 路由定义 |
| `pkg/api/handlers.go` | API 处理函数 |
| `pkg/version/version.go` | 版本信息 |
| `pkg/ws/hub.go` | WebSocket Hub |

### 23.3 前端关键文件

| 路径 | 描述 |
|------|------|
| `web/src/App.tsx` | 主应用组件 |
| `web/src/components/Sidebar.tsx` | 侧边栏（含版本显示） |
| `web/src/pages/Dashboard.tsx` | 仪表盘（WebSocket 实时更新） |
| `web/src/pages/Egress.tsx` | 出口策略管理 |
| `web/src/pages/Rules.tsx` | 路由规则管理（含优先级说明） |
| `web/src/pages/Settings.tsx` | 系统设置 |
| `web/src/pages/Help.tsx` | 使用手册 |
| `web/src/services/api.ts` | API 客户端定义 |

---

## 24. 变更日志

### v1.2.0 (2026-04-08)

**修复**：
- Linux 网卡自动识别：添加 `enp`、`ens`、`eno`、`enx`、`wlp`、`wlo`、`wlx` 等模式，支持 Ubuntu 等发行版的可预测网卡命名
- Windows 进程检测：使用 Windows API `OpenProcess` + `GetExitCodeProcess` 正确检测进程是否存在

**改进**：
- 单实例检测添加调试日志，便于诊断启动问题

### v1.1.0 (2026-04-07)

**新功能**：
- GitHub Actions 自动发布工作流
- 默认配置文件嵌入二进制，首次启动自动创建
- Claude Code Skills 系统（`.claude/skills/`）

**改进**：
- AI 配置整合到 `.claude/` 目录，支持多 AI 工具
- README 添加 GitHub Releases 下载说明
- README 添加 Windows 命令行启动建议

**文档**：
- 新增 `release.md` skill（发布流程指南）
- 新增 `project-context.md` skill（项目上下文）
- 更新架构文档添加 GitHub Actions 章节

### v1.0.0 (2026-03-31)

**新功能**：
- 版本号注入系统（`pkg/version`）
- 出口策略循环检测
- WebSocket 实时流量更新（每 2 秒）
- 路由规则优先级说明（Web GUI）
- 响应式布局（移动端适配）
- 现代化 UI 主题

**改进**：
- 默认端口改为 8009/8010
- 域名树优化（`internal/router/tree.go`）
- 端口映射优化（O(1) 查找）
- 流量复制使用 sync.Pool 减少内存分配

**文档**：
- 添加 `AGENTS.md` AI 助手指南
- 更新架构文档
- 更新 README
