---
name: project-context
description: Essential project context, development guidelines, and common solutions for NetDispatch. Read this first to understand the project.
---

# NetDispatch 项目上下文

> **AI 助手指南**：阅读此文件以快速理解项目并开始工作。此文件包含项目上下文、开发指南、常用命令和问题解决方案。

---

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

## 开发完成后的标准操作

> **重要**：每次开发完成后，必须按照以下顺序执行操作。

### 1. 代码验证

```bash
# 后端编译检查
go build ./...

# 后端测试
go test ./...

# 前端构建检查
cd web && npm run build && cd ..
```

### 2. 版本更新

编辑 `docs/architecture.md`，在"变更日志"章节添加新版本记录：

```markdown
### v1.x.x (YYYY-MM-DD)

**新功能**：
- 功能描述

**改进**：
- 改进描述

**修复**：
- Bug 修复描述
```

### 3. 提交代码

```bash
# 查看更改
git status

# 添加所有更改
git add -A

# 提交（使用规范的提交信息格式）
git commit -m "feat: 简短描述

- 详细说明 1
- 详细说明 2

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

提交类型规范：
- `feat:` 新功能
- `fix:` Bug 修复
- `docs:` 文档更新
- `refactor:` 重构
- `perf:` 性能优化
- `test:` 测试相关
- `chore:` 构建/工具相关

### 4. 推送到 GitHub

```bash
git push origin main
```

### 5. 创建 Release（如有需要）

参考 `release` skill 或手动执行：

```bash
./build.sh v1.x.x
git tag v1.x.x
git push origin v1.x.x
```

---

## 开发时需要具备的能力

### 能力 1：理解代理协议

必须理解以下协议的工作原理：
- **HTTP 代理**：处理普通 HTTP 请求和 CONNECT 隧道
- **SOCKS5 代理**：理解握手流程、认证、CONNECT 命令
- **上游代理**：如何连接上游 HTTP/SOCKS5 代理

关键文件：
- `internal/handler/http.go`
- `internal/handler/socks5.go`

### 能力 2：理解路由引擎

必须理解：
- 规则优先级（数值越小优先级越高）
- 匹配条件（域名/IP CIDR/端口）
- 通配符匹配规则（`*.example.com` 不匹配 `example.com`）
- 黑白名单机制

关键文件：
- `internal/router/manager.go`
- `internal/router/rule.go`
- `internal/router/tree.go`

### 能力 3：理解出口策略

必须理解：
- 出口策略 = 网卡 + 上游代理（可选）
- 循环检测机制
- 如何选择出口策略

关键文件：
- `internal/egress/manager.go`
- `internal/egress/validation.go`

### 能力 4：前端开发能力

必须熟悉：
- React + TypeScript
- Ant Design 组件库
- TanStack Query 数据获取
- WebSocket 实时通信

关键文件：
- `web/src/pages/Dashboard.tsx`（WebSocket 实时更新示例）
- `web/src/services/api.ts`（API 客户端）
- `web/src/components/Sidebar.tsx`（版本显示示例）

### 能力 5：配置管理

必须理解：
- YAML 配置结构
- 配置验证逻辑
- 配置热更新

关键文件：
- `pkg/config/config.go`
- `configs/config.yaml`

---

## 技术栈详解

**后端 (Go)**:
| 组件 | 库 | 用途 |
|------|-----|------|
| HTTP 框架 | `net/http` + `gorilla/mux` | HTTP 服务器和路由 |
| 日志 | `rs/zerolog` | 结构化日志 |
| CLI | `spf13/cobra` | 命令行界面 |
| 托盘 | `fyne.io/systray` | 系统托盘 |
| WebSocket | `gorilla/websocket` | WebSocket 支持 |
| 配置 | `gopkg.in/yaml.v3` | YAML 解析 |

**前端 (React)**:
| 组件 | 库 | 用途 |
|------|-----|------|
| 框架 | React 18 | UI 框架 |
| 构建 | Vite | 构建工具 |
| UI | Ant Design | UI 组件库 |
| 图表 | ECharts | 流量图表 |
| 数据 | TanStack Query | 数据获取和缓存 |
| 实时 | Native WebSocket | 实时更新 |
| 样式 | Tailwind CSS | CSS 框架 |

---

## 项目结构详解

```
netdispatch/
├── cmd/netdispatch/              # 程序入口
│   ├── main.go                   # Cobra CLI 定义、启动逻辑
│   ├── console_windows.go        # Windows 控制台隐藏
│   ├── console_other.go          # 其他平台空实现
│   ├── messagebox_windows.go     # Windows 弹窗
│   └── messagebox_other.go       # 其他平台 stderr 输出
│
├── internal/                     # 内部模块（不对外暴露）
│   ├── connmgr/                  # 连接管理
│   ├── egress/                   # 出口策略
│   ├── handler/                  # 协议处理器
│   ├── nic/                      # 网卡管理
│   ├── proxy/                    # 代理服务器核心
│   ├── router/                   # 路由引擎
│   └── tray/                     # 系统托盘
│
├── pkg/                          # 公共模块（可对外暴露）
│   ├── api/                      # REST API
│   ├── config/                   # 配置管理
│   ├── version/                  # 版本信息
│   ├── singleinstance/           # 单实例检测
│   ├── ws/                       # WebSocket
│   └── crashlog/                 # 崩溃日志
│
├── web/                          # 前端项目
│   └── dist/                     # 构建产物（嵌入二进制）
│
├── configs/config.yaml           # 示例配置
├── build.sh                      # 构建脚本
├── release.sh                    # 发布脚本
├── AGENTS.md                     # 本文件（AI 助手指南）
├── README.md                     # 项目介绍
└── docs/
    ├── architecture.md           # 详细架构文档
    └── build-guide.md            # 编译指南
```

---

## 开发命令速查

### 后端开发

```bash
# 编译检查
go build ./...

# 运行测试
go test ./...

# 运行服务
go run ./cmd/netdispatch start

# 指定配置文件
go run ./cmd/netdispatch start -c configs/config.yaml

# 查看版本
go run ./cmd/netdispatch version
```

### 前端开发

```bash
cd web

# 安装依赖
npm install

# 开发服务器
npm run dev

# 生产构建
npm run build

# 类型检查
npx tsc --noEmit
```

### 构建与发布

```bash
# 构建指定版本
./build.sh v1.0.0

# 发布新版本（创建 tag 触发 GitHub Actions）
git tag v1.0.0 && git push origin v1.0.0
```

---

## API 端点完整列表

### 网卡管理
```
GET  /api/v1/nics           # 获取网卡列表
GET  /api/v1/nics/{name}    # 获取单个网卡信息
```

### 出口策略
```
GET    /api/v1/egress           # 列表
POST   /api/v1/egress           # 创建
PUT    /api/v1/egress/{id}      # 更新
DELETE /api/v1/egress/{id}      # 删除
POST   /api/v1/egress/{id}/test # 测试连接
```

### 路由规则
```
GET    /api/v1/rules              # 列表
POST   /api/v1/rules              # 创建
PUT    /api/v1/rules/{id}         # 更新
DELETE /api/v1/rules/{id}         # 删除
POST   /api/v1/rules/import       # 从 URL 导入域名
POST   /api/v1/rules/import/file  # 从文件导入域名
```

### 连接管理
```
GET    /api/v1/connections          # 活跃连接列表
GET    /api/v1/connections/recent   # 最近关闭的连接
GET    /api/v1/connections/{id}     # 获取单个连接
DELETE /api/v1/connections/{id}     # 关闭连接
```

### 统计信息
```
GET  /api/v1/stats/overview  # 总览统计
GET  /api/v1/stats/traffic   # 流量统计
GET  /api/v1/stats/history   # 流量历史（60个数据点）
```

### 系统管理
```
GET  /api/v1/config       # 获取配置
PUT  /api/v1/config       # 更新配置
GET  /api/v1/system/info  # 系统信息（版本等）
GET  /api/v1/status       # 服务状态
GET  /api/v1/health       # 健康检查
```

### WebSocket
```
GET  /ws  # WebSocket 连接
```

---

## 常见问题与解决方案

### 1. 单实例检测误报

**现象**：进程已退出但仍提示"已在运行"

**解决**：
```bash
# 检查是否有进程
tasklist | grep netdispatch

# 手动清理 lock 文件
rm -f ~/.local/run/netdispatch/netdispatch.lock
# 或 Windows
del "%USERPROFILE%\.local\run\netdispatch\netdispatch.lock"
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
2. 检查浏览器控制台是否有连接错误
3. 前端会自动重连

### 5. 网卡绑定失败

**现象**：启动时提示网卡不存在

**解决**：
- 检查网卡名称是否正确
- Windows 网卡名：`以太网`, `WLAN`
- 使用 `ipconfig`（Windows）或 `ifconfig`（Linux）查看实际名称

### 6. 前端构建后页面空白

**解决**：
1. 确保运行 `npm run build`
2. 检查 `web/dist` 目录是否有文件
3. 重新编译后端

---

## 代理测试命令

```bash
# HTTP 代理
curl -x http://<YOUR_SERVER_IP>:8009 https://www.google.com -I

# SOCKS5 代理（注意使用 socks5h 让代理解析 DNS）
curl -x socks5h://<YOUR_SERVER_IP>:8010 https://www.google.com -I

# 测试 GitHub
curl -x http://<YOUR_SERVER_IP>:8009 https://github.com -I

# 测试国内网站（默认路由）
curl -x http://<YOUR_SERVER_IP>:8009 https://www.baidu.com -I
```

---

## 注意事项

1. **版本号注入**：始终使用 `./build.sh` 或手动指定 ldflags，否则版本显示为 `dev`

2. **前端构建**：后端构建前必须先构建前端（`web/dist` 会嵌入二进制）

3. **Windows 托盘**：需要 `-H windowsgui` ldflag 来隐藏控制台窗口

4. **单实例**：程序启动时会检测是否已有实例运行

5. **循环检测**：创建出口策略时会验证上游代理不指向本机

6. **代理协议**：SOCKS5 使用 `socks5h://` 让代理解析 DNS，避免本地 DNS 污染

7. **域名通配符**：`*.example.com` 不匹配 `example.com`，需分别添加

---

## 相关文档

- **详细架构设计**：`docs/architecture.md`
- **编译指南**：`docs/build-guide.md`
- **Release 工作流**：`.claude/skills/release.md`
