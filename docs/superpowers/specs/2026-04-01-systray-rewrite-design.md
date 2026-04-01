---
name: systray-rewrite
description: Rewrite system tray module using getlantern/systray to fix right-click freeze issues on Windows
type: project
created: 2026-04-01
status: approved
---

# 系统托盘模块重写设计文档

## 背景

### 问题描述

NetDispatch 在 Windows 平台上，系统托盘图标的右键菜单有概率出现假死现象（右键无响应）。问题通常在程序运行一段时间后出现。

### 问题分析

- 问题仅在 Windows 平台出现
- 使用 `github.com/energye/systray` 库
- 最近的修复尝试（切换到 `RunExternalLoop`）未能完全解决问题
- 问题可能是库的内部实现与 Windows 消息循环的兼容性问题

### 解决方案

将托盘库从 `github.com/energye/systray` 替换为 `github.com/getlantern/systray`。

**Why:** getlantern/systray 是经过大规模生产验证的库（Lantern VPN 使用），在 Windows 平台上有更好的稳定性。

**How to apply:** 在实现时需要调整 API 调用方式，两个库的接口略有不同。

---

## 设计详情

### 模块架构

```
internal/tray/
├── tray.go              # 托盘实现（重写）
├── icon.go              # 图标资源（已存在，继续使用）
└── icon_disabled.go     # 禁用状态图标（新增）
```

### 文件变更清单

| 文件 | 变更类型 | 说明 |
|-----|---------|------|
| `go.mod` | 修改 | 替换依赖 |
| `internal/tray/tray.go` | 重写 | 使用新库 API |
| `internal/tray/icon_disabled.go` | 新增 | 禁用状态图标 |
| `cmd/netdispatch/main.go` | 修改 | 调用新的状态切换 API |

### API 设计

**保持现有 API（向后兼容）：**

```go
// Run starts the system tray (blocking call)
func Run()

// RunExternalLoop starts the system tray with external loop control
func RunExternalLoop() (start, stop func())

// SetOnOpenBrowser sets the callback for opening browser
func SetOnOpenBrowser(fn func())

// SetOnQuit sets the callback for quit
func SetOnQuit(fn func())

// Quit quits the system tray
func Quit()
```

**新增 API：**

```go
// SetStatus sets the tray icon status
// status: "running" | "stopped"
func SetStatus(status string)

// SetStatusChangeCallback sets the callback for status change
func SetStatusChangeCallback(fn func(running bool))
```

### 库差异对比

| 功能 | energye/systray | getlantern/systray |
|-----|----------------|-------------------|
| 基本菜单 | ✅ | ✅ |
| 左键点击 | `SetOnClick` | 需要外部检测 |
| 右键点击 | `SetOnRClick` | 自动显示菜单 |
| 状态更新 | 手动调用 | `SetIcon`/`SetTooltip` |
| 线程安全 | 需要外部同步 | 内置支持 |

### 实现要点

1. **依赖替换**
   - 移除 `github.com/energye/systray`
   - 添加 `github.com/getlantern/systray`

2. **图标管理**
   - 准备两套图标：运行中（彩色）和停止（灰色）
   - 使用 `systray.SetIcon()` 动态切换

3. **右键菜单**
   - getlantern/systray 会自动处理右键菜单显示
   - 无需手动调用 `ShowMenu()`

4. **主线程**
   - 继续使用 `RunExternalLoop()` 模式
   - 确保在主线程运行

### 菜单结构

```
托盘图标
├── 左键单击 → 打开浏览器
├── 左键双击 → 打开浏览器
└── 右键菜单
    ├── 打开网页
    ├── ─────────
    ├── 停止代理 (当运行时) / 启动代理 (当停止时)
    └── 退出
```

### 状态图标

| 状态 | 图标 | 提示文本 |
|-----|------|---------|
| 运行中 | 彩色图标 | "NetDispatch - 运行中" |
| 已停止 | 灰色图标 | "NetDispatch - 已停止" |

---

## 测试计划

### 功能测试

| 测试项 | 预期结果 |
|-------|---------|
| 左键单击 | 打开浏览器 |
| 左键双击 | 打开浏览器 |
| 右键菜单显示 | 显示菜单选项 |
| 菜单项点击 | 执行对应操作 |
| 退出功能 | 程序正常退出 |
| 状态切换 | 图标和提示文本正确更新 |

### 稳定性测试

| 测试项 | 要求 |
|-------|------|
| 长时间运行 | 24小时+ 无假死 |
| 频繁操作 | 连续右键100次无问题 |
| 内存监控 | 无内存泄漏 |

### 平台测试

| 平台 | 测试要求 |
|-----|---------|
| Windows 10 | 全功能测试 |
| Windows 11 | 全功能测试 |
| 高 DPI 显示器 | 图标显示正常 |

---

## 风险评估

### 低风险

- 库替换：getlantern/systray 是成熟稳定的库
- API 兼容：现有代码改动较小

### 需要注意

- CGO 依赖：getlantern/systray 依赖 CGO，需要确保编译环境正确配置
- 跨平台：需要验证 Linux 和 macOS 的兼容性

---

## 实现步骤

1. 更新 `go.mod`，替换依赖
2. 重写 `internal/tray/tray.go`
3. 添加禁用状态图标 `icon_disabled.go`
4. 更新 `main.go` 调用新 API
5. 编译测试
6. 长时间运行稳定性验证

---

## 参考资料

- [getlantern/systray GitHub](https://github.com/getlantern/systray)
- [Windows Shell_NotifyIcon API](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shell_notifyiconw)
