# NetDispatch

**网络调度器** - 高性能多协议代理服务器，支持智能网卡路由。

## 功能特性

- **多协议支持**：HTTP、HTTPS、SOCKS5 代理协议
- **智能路由**：基于域名、IP、端口的规则匹配
- **出口策略**：网卡 + 上游代理的组合配置
- **Web 控制台**：实时流量监控、配置管理
- **系统托盘**：后台运行，托盘图标操作
- **中文界面**：完整的中文 Web 管理界面

## 快速开始

### 环境要求

| 工具 | 版本 |
|------|------|
| Go | >= 1.21 |
| Node.js | >= 18 |
| Make | 任意版本 |

### 编译

```bash
# 克隆仓库
git clone https://github.com/yourusername/netdispatch.git
cd netdispatch

# 编译（会自动编译前端并嵌入二进制）
make build
```

### 自定义编译

#### 指定 Web 控制台端口

```bash
# 编译时指定默认 API 端口（默认 9090）
make build API_PORT=8080

# 或使用自定义构建命令
go build -ldflags "-H windowsgui -X 'main.DefaultAPIPort=8080'" -o bin/netdispatch.exe ./cmd/netdispatch
```

#### 指定版本号

```bash
make build VERSION=1.0.0
```

### 跨平台编译

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o bin/netdispatch.exe ./cmd/netdispatch

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/netdispatch ./cmd/netdispatch

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/netdispatch ./cmd/netdispatch
```

> **平台兼容性**：
> - Windows：支持智能控制台隐藏和单实例弹窗提示
> - Linux/macOS：控制台保持可见，单实例检测输出到标准错误

### Web 控制台端口

Web 控制台默认监听端口 **9090**，可在配置文件中修改：

```yaml
api:
  bind: "127.0.0.1"  # 监听地址
  port: 9090         # 监听端口
```

启动后访问：`http://127.0.0.1:9090`

### 运行

```bash
# Windows 双击启动 (从资源管理器双击，控制台会自动隐藏)
双击 bin\netdispatch.exe

# 或使用启动脚本
双击 bin\启动NetDispatch.bat

# 命令行运行 (控制台保持可见，可查看日志)
./bin/netdispatch start

# 指定配置文件
./bin/netdispatch start -c configs/config.yaml
```

程序启动后会自动打开 Web 控制台，或通过系统托盘图标访问。

### 启动行为说明

- **双击启动**：控制台窗口自动隐藏，仅显示系统托盘图标
- **命令行启动**：控制台窗口保持可见，可实时查看日志输出
- **单实例限制**：程序仅允许运行一个实例，重复启动会弹出提示框

## 核心概念

### 出口策略

出口策略定义了代理请求如何出去到互联网：

```
出口策略 = 网卡 + 代理服务器（可选）

示例：
- "网线直连"：eth0 网卡，无代理
- "WiFi走代理"：wlan0 网卡，SOCKS5 代理
```

### 路由规则

路由规则决定哪些请求使用哪个出口策略：

| 匹配条件 | 出口策略 |
|---------|---------|
| *.google.com | WiFi + 代理 |
| 10.0.0.0/8 | 网线直连 |
| 黑名单 | 拒绝连接 |

## 目录结构

```
netdispatch/
├── cmd/netdispatch/     # 程序入口
├── internal/            # 内部模块
│   ├── connmgr/        # 连接管理
│   ├── egress/         # 出口策略
│   ├── handler/        # 协议处理
│   ├── nic/            # 网卡管理
│   ├── proxy/          # 代理服务
│   ├── router/         # 路由引擎
│   └── tray/           # 系统托盘
├── pkg/                 # 公共模块
│   ├── api/            # REST API
│   ├── config/         # 配置管理
│   └── ws/             # WebSocket
├── web/                 # 前端项目
├── configs/             # 配置文件
└── docs/                # 文档
```

## 配置示例

```yaml
server:
  enabled: true
  bind: "0.0.0.0"
  http:
    enabled: true
    port: 809
  socks5:
    enabled: true
    port: 810

egress:
  - id: "direct-eth"
    name: "网线直连"
    nic: "eth0"
  - id: "wifi-proxy"
    name: "WiFi走代理"
    nic: "wlan0"
    proxy:
      host: "192.168.1.100"
      port: 1080
      protocol: "socks5"

routing:
  default_egress: "direct-eth"
  rules:
    - id: "google-wifi"
      domains: ["*.google.com"]
      egress_id: "wifi-proxy"
```

## 文档

- [编译手册](docs/build-guide.md)
- [架构设计](docs/architecture.md)

## 技术栈

- **后端**：Go、Zerolog、Cobra、Gorilla
- **前端**：React、TypeScript、Ant Design、Vite
- **系统托盘**：fyne.io/systray

## License

MIT License
