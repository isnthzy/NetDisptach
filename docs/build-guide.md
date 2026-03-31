# NetDispatch 编译手册

## 环境要求

| 工具 | 版本要求 | 说明 |
|------|---------|------|
| Go | >= 1.21 | 后端编译 |
| Node.js | >= 18 | 前端编译 |
| Make | 任意版本 | 构建工具 |

## 快速开始

### 1. 安装依赖

```bash
# Go 依赖
make deps

# 前端依赖
make web-deps
```

### 2. 编译

```bash
# 编译后端
make build

# 编译前端
make web-build
```

### 3. 运行

```bash
# Windows 双击启动 (从资源管理器双击，控制台会自动隐藏)
双击 bin\netdispatch.exe

# 或使用启动脚本
双击 bin\启动NetDispatch.bat

# 命令行运行 (控制台保持可见，可查看日志)
make dev

# 或直接运行
./bin/netdispatch start -c configs/config.yaml
```

> **智能控制台隐藏**：程序会自动检测启动方式。从资源管理器双击启动时，控制台自动隐藏；从命令行启动时，控制台保持可见。

> **单实例限制**：程序仅允许运行一个实例，重复启动会弹出提示框告知用户。

## Makefile 命令说明

| 命令 | 说明 |
|------|------|
| `make build` | 编译后端，输出到 `bin/netdispatch.exe` |
| `make build-custom` | 编译并指定自定义端口（如 `make build-custom API_PORT=8080`） |
| `make run` | 直接运行（不编译，用于开发调试） |
| `make dev` | 编译后运行 |
| `make clean` | 清理编译产物 |
| `make test` | 运行测试 |
| `make deps` | 下载并整理 Go 依赖 |
| `make web-deps` | 安装前端依赖 |
| `make web-dev` | 启动前端开发服务器 |
| `make web-build` | 编译前端 |
| `make docker-build` | 构建 Docker 镜像 |
| `make lint` | 运行代码检查 |

> **注意**：由于程序内置智能控制台隐藏功能，不再需要单独的 `build-console` 命令。

## 编译时配置

### 指定 Web 控制台端口

默认情况下，Web 控制台监听端口为 **9090**。可以通过编译时变量自定义：

```bash
# 使用 Makefile
make build API_PORT=8080

# 或自定义输出文件名
make build-custom API_PORT=8080
```

### 指定版本号

```bash
make build VERSION=1.0.0
```

### 使用 Go 命令直接编译

```bash
# 指定 API 端口
go build -ldflags "-H windowsgui -X 'main.DefaultAPIPort=8080'" -o bin/netdispatch.exe ./cmd/netdispatch

# 指定版本和端口
go build -ldflags "-H windowsgui -X 'main.Version=1.0.0' -X 'main.DefaultAPIPort=8080'" -o bin/netdispatch.exe ./cmd/netdispatch
```

### 查看编译信息

```bash
./bin/netdispatch version
# 输出：
# NetDispatch v1.0.0
# Default API Port: 8080
```

## 手动编译

### 后端编译

```bash
# 编译到 bin 目录
go build -o bin/netdispatch ./cmd/netdispatch

# 带版本信息的编译
go build -ldflags "-X 'main.version=1.0.0' -X 'main.defaultAPIPortStr=9090'" -o bin/netdispatch.exe ./cmd/netdispatch

# 直接编译到当前目录
go build ./cmd/netdispatch
```

> **注意**：程序内置智能控制台隐藏功能，会根据启动方式自动决定是否隐藏控制台窗口，无需使用 `-H windowsgui` 编译选项。

### 前端编译

```bash
cd web
npm install
npm run build
```

## 跨平台编译

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

## 配置文件

默认配置文件路径：`configs/config.yaml`

启动时可通过 `-c` 参数指定：
```bash
./bin/netdispatch start -c /path/to/config.yaml
```

## 目录结构

```
netdispatch/
├── bin/                    # 编译输出
│   └── netdispatch.exe
├── cmd/netdispatch/        # 程序入口
│   └── main.go
├── configs/                # 配置文件
│   └── config.yaml
├── internal/               # 内部模块
│   ├── connmgr/           # 连接管理
│   ├── egress/            # 出口策略
│   ├── handler/           # 协议处理器
│   ├── nic/               # 网卡管理
│   ├── proxy/             # 代理服务
│   ├── router/            # 路由引擎
│   └── tray/              # 系统托盘
├── pkg/                    # 公共模块
│   ├── api/               # REST API
│   ├── config/            # 配置管理
│   └── ws/                # WebSocket
├── web/                    # 前端项目
│   ├── src/               # 源代码
│   ├── dist/              # 编译输出
│   ├── embed.go           # 资源嵌入
│   └── package.json
├── docs/                   # 文档
│   ├── architecture.md    # 架构设计
│   └── build-guide.md     # 编译手册
├── Makefile               # 构建脚本
├── go.mod                 # Go 模块定义
└── README.md              # 项目说明
```

## 常见问题

### 编译失败：找不到模块

```bash
go mod tidy
go mod download
```

### Windows 下 make 命令不可用

直接使用 Go 命令：
```bash
go build -o bin/netdispatch.exe ./cmd/netdispatch
```

### 前端编译失败

确保 Node.js 版本 >= 18：
```bash
node --version
```
