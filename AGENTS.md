# NetDispatch 项目 Skills

> **AI 助手指南**：阅读此文件以快速理解项目并开始工作。此文件包含项目上下文、开发指南、常用命令和问题解决方案。

## 项目概述

**NetDispatch** 是一个高性能多协议代理网络调度器，支持智能出口路由。

### 核心功能
- 多协议代理：HTTP/HTTPS/SOCKS5
- 出口策略：网卡 + 上游代理组合
- 智能路由：域名/IP CIDR/端口匹配
- Web 控制台：实时流量监控
- 系统托盘：后台运行

### 默认端口
| 协议 | 端口 |
|------|------|
| HTTP/HTTPS 代理 | 8009 |
| SOCKS5 代理 | 8010 |
| API/Web 控制台 | 9090 |

---

## 技术栈

**后端 (Go)**:
- 框架：`net/http` + `gorilla/mux`
- 日志：`rs/zerolog`
- CLI：`spf13/cobra`
- 托盘：`fyne.io/systray`
- WebSocket：`gorilla/websocket`

**前端 (React)**:
- React 18 + TypeScript
- Vite 构建
- Ant Design UI
- ECharts 图表
- TanStack Query 数据获取
- WebSocket 实时更新

---

## 项目结构

```
netdispatch/
├── cmd/netdispatch/          # 程序入口
│   └── main.go              # Cobra CLI 定义
├── internal/                 # 内部模块（不对外暴露）
│   ├── connmgr/             # 连接管理、流量统计
│   ├── egress/              # 出口策略管理、验证
│   ├── handler/             # 协议处理器 (HTTP/SOCKS5)
│   ├── nic/                 # 网卡检测与管理
│   ├── proxy/               # 代理服务器核心
│   ├── router/              # 路由引擎、规则匹配
│   └── tray/                # 系统托盘
├── pkg/                      # 公共模块（可对外暴露）
│   ├── api/                 # REST API + WebSocket 处理
│   ├── config/              # YAML 配置管理
│   ├── version/             # 版本信息（编译时注入）
│   ├── singleinstance/      # 单实例检测
│   └── ws/                  # WebSocket Hub
├── web/                      # 前端项目
│   ├── src/
│   │   ├── components/      # Sidebar 等组件
│   │   ├── pages/           # Dashboard, Egress, Rules 等
│   │   └── services/api.ts  # API 客户端
│   └── dist/                # 构建产物（嵌入二进制）
├── configs/config.yaml       # 示例配置
├── build.sh                  # 构建脚本
├── release.sh                # 发布脚本
└── docs/architecture.md      # 详细架构文档
```

---

## 开发命令

### 构建项目

```bash
# 使用构建脚本（推荐，自动注入版本号）
./build.sh v1.0.0

# 或手动构建
cd web && npm run build && cd ..
go build -ldflags "-s -w \
    -X netdispatch/pkg/version.Version=1.0.0 \
    -X netdispatch/pkg/version.GitCommit=$(git rev-parse --short HEAD) \
    -X netdispatch/pkg/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o netdispatch.exe ./cmd/netdispatch
```

### 前端开发

```bash
cd web
npm install          # 安装依赖
npm run dev          # 开发服务器
npm run build        # 生产构建
```

### 后端开发

```bash
go build ./...       # 编译检查
go test ./...        # 运行测试
go run ./cmd/netdispatch start   # 运行服务
```

### 发布流程

```bash
./release.sh v1.1.0  # 自动构建、打标签、创建 GitHub Release
```

---

## 核心概念

### 出口策略 (Egress Policy)

出口策略定义代理请求如何出去：

```
出口策略 = 网卡 + 上游代理（可选）

示例：
- "网线直连"：eth0 网卡，无代理
- "WiFi走代理"：wlan0 网卡，SOCKS5 代理 192.168.1.100:1080
```

**重要验证**：
- 不能配置上游代理指向本机代理端口（会导致循环）
- 验证逻辑在 `internal/egress/validation.go`

### 路由规则

路由规则按优先级匹配（数值越小优先级越高）：

```
规则匹配顺序：
1. 按优先级排序
2. 从低到高匹配
3. 第一个匹配的规则生效

匹配条件：
- 域名：支持 *.example.com 通配符
- IP/CIDR：如 10.0.0.0/8
- 端口：如 80, 443
```

**注意事项**：
- `*.example.com` 匹配 `www.example.com` 但**不匹配** `example.com`
- 如需匹配主域名，需同时添加 `example.com` 和 `*.example.com`

### WebSocket 实时更新

后端每 2 秒广播流量统计到 WebSocket 客户端：

```go
// cmd/netdispatch/main.go
wsHub.BroadcastTraffic(stats.BytesIn, stats.BytesOut, stats.ActiveConnections)
```

前端通过 `useWebSocketStats` hook 接收实时数据。

---

## API 端点

### REST API

```
GET    /api/v1/nics                # 网卡列表
GET    /api/v1/egress              # 出口策略列表
POST   /api/v1/egress              # 创建出口策略
PUT    /api/v1/egress/{id}         # 更新出口策略
DELETE /api/v1/egress/{id}         # 删除出口策略
GET    /api/v1/rules               # 路由规则列表
POST   /api/v1/rules               # 创建规则
PUT    /api/v1/rules/{id}          # 更新规则
DELETE /api/v1/rules/{id}          # 删除规则
POST   /api/v1/rules/import        # 从 URL 导入域名列表
POST   /api/v1/rules/import/file   # 从文件导入域名列表
GET    /api/v1/connections         # 活跃连接
GET    /api/v1/connections/recent  # 最近连接
GET    /api/v1/stats/overview      # 统计概览
GET    /api/v1/stats/history       # 流量历史
GET    /api/v1/config              # 配置
PUT    /api/v1/config              # 更新配置
GET    /api/v1/system/info         # 系统信息（含版本）
GET    /api/v1/status              # 服务状态
GET    /ws                         # WebSocket 端点
```

### WebSocket 事件

```json
// 服务端 -> 客户端：流量更新（每2秒）
{
  "type": "traffic",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "bytes_in": 1024000,
    "bytes_out": 2048000,
    "active_connections": 42
  }
}
```

---

## 常见问题与解决方案

### 1. 单实例检测误报

**现象**：进程已退出但仍提示"已在运行"

**原因**：lock 文件未正确清理

**解决**：
```bash
# 检查是否有进程
tasklist | grep netdispatch

# 手动清理 lock 文件
rm -f ~/.local/run/netdispatch/netdispatch.lock
```

### 2. 代理循环问题

**现象**：请求卡死或超时

**原因**：出口策略的上游代理指向本机

**解决**：已添加验证，创建策略时会检测并拒绝

### 3. 域名匹配不生效

**现象**：`*.example.com` 规则不匹配 `example.com`

**解决**：同时添加主域名和通配符：
```yaml
domains:
  - example.com
  - '*.example.com'
```

### 4. WebSocket 连接失败

**现象**：仪表盘不更新

**排查**：
1. 检查 WebSocket 路径：`/ws`
2. 检查控制台是否有连接错误
3. 前端会自动重连

### 5. 网卡绑定失败

**现象**：启动时提示网卡不存在

**解决**：
- 检查网卡名称是否正确
- Windows 网卡名：`以太网`, `WLAN`
- 使用 `ipconfig` 查看实际名称

---

## 代理测试命令

```bash
# HTTP 代理
curl -x http://<YOUR_SERVER_IP>:8009 https://www.google.com -I

# SOCKS5 代理（注意使用 socks5h 让代理解析 DNS）
curl -x socks5h://<YOUR_SERVER_IP>:8010 https://www.google.com -I

# 测试 GitHub
curl -x http://<YOUR_SERVER_IP>:8009 https://github.com -I
```

---

## 配置文件示例

```yaml
server:
  enabled: true
  bind: "0.0.0.0"
  http:
    port: 8009
    enabled: true
  socks5:
    port: 8010
    enabled: true
    auth:
      enabled: false

nics:
  default: ""
  display_names: {}

egress_policies:
  - id: ethernet-direct
    name: 网线直连
    nic: 以太网
    proxy: null
  - id: wifi-proxy
    name: WiFi代理
    nic: WLAN
    proxy:
      host: 10.93.187.36
      port: 7890
      protocol: http

routing:
  default_egress: ethernet-direct
  rules:
    - id: google-proxy
      name: Google走代理
      priority: 50
      enabled: true
      list_type: none
      domains:
        - '*.google.com'
        - google.com
        - '*.github.com'
        - github.com
      action: forward
      egress_id: wifi-proxy
    - id: default-catch-all
      name: 默认路由
      priority: 100
      enabled: true
      domains: ['*']
      action: forward
      egress_id: ethernet-direct

api:
  bind: 127.0.0.1
  port: 9090
```

---

## 架构文档

详细架构设计请参阅：`docs/architecture.md`

---

## GitHub Release

发布新版本：

```bash
./release.sh v1.1.0
```

这会：
1. 构建前端和后端
2. 注入版本号
3. 创建 git tag
4. 推送到 GitHub
5. 创建 GitHub Release 并上传可执行文件

---

## 注意事项

1. **版本号注入**：始终使用 `./build.sh` 或手动指定 ldflags，否则版本显示为 `dev`

2. **前端构建**：后端构建前必须先构建前端（`web/dist` 会嵌入二进制）

3. **Windows 托盘**：需要 `-H windowsgui` ldflag 来隐藏控制台窗口

4. **单实例**：程序启动时会检测是否已有实例运行

5. **循环检测**：创建出口策略时会验证上游代理不指向本机
