---
name: windows-native-systray
description: Implement Windows system tray using native Win32 API to fix freeze issues
type: project
created: 2026-04-01
status: approved
---

# Windows 原生托盘实现设计文档

## 背景

### 问题描述

NetDispatch 在 Windows 平台上，系统托盘图标的右键菜单有概率出现假死现象（右键无响应）。问题在尝试了多个第三方库后仍然存在：
- `fyne.io/systray` - 原始库
- `github.com/energye/systray` - 第一次替换
- `github.com/getlantern/systray` - 第二次替换，仍然卡死

### 解决方案

在 Windows 平台使用原生 Win32 API 实现，其他平台继续使用第三方库。

**Why:** 第三方库在 Windows 消息循环处理上存在问题，原生 API 可以完全控制消息处理逻辑。

**How to apply:** 使用 Go 的 `syscall` 调用 Windows API，通过 build tags 实现平台特定代码。

---

## 设计详情

### 文件结构

```
internal/tray/
├── tray.go              # 公共 API 定义（平台无关）
├── tray_windows.go      # Windows 原生实现 (//go:build windows)
├── tray_other.go        # 其他平台库实现 (//go:build !windows)
├── icon.go              # 图标数据
└── icon_disabled.go     # 禁用状态图标
```

### 公共 API (tray.go)

```go
package tray

// SetOnOpenBrowser sets the callback for opening browser
func SetOnOpenBrowser(fn func())

// SetOnQuit sets the callback for quit
func SetOnQuit(fn func())

// SetStatusChangeCallback sets the callback for status change
func SetStatusChangeCallback(fn func(running bool))

// SetStatus sets the tray icon status ("running" or "stopped")
func SetStatus(status string)

// Quit quits the system tray
func Quit()

// Run starts the system tray (blocking call)
func Run()

// RunExternalLoop starts the system tray with external loop control
func RunExternalLoop() (start, stop func())
```

### Windows 原生实现 (tray_windows.go)

#### 使用的 Windows API

| API | 用途 |
|-----|------|
| `Shell_NotifyIcon` | 托盘图标管理 |
| `CreateWindowEx` | 创建隐藏窗口 |
| `RegisterClassEx` | 注册窗口类 |
| `CreatePopupMenu` | 创建右键菜单 |
| `TrackPopupMenu` | 显示菜单 |
| `GetMessage` | 消息循环 |
| `DefWindowProc` | 默认消息处理 |

#### 核心实现逻辑

```
初始化流程:
1. 注册窗口类 (WNDCLASSEX)
2. 创建隐藏窗口 (CreateWindowEx)
3. 加载图标资源
4. 创建托盘图标 (Shell_NotifyIcon NIM_ADD)
5. 创建菜单 (CreatePopupMenu)
6. 启动消息循环

消息处理:
WM_USER+1 (托盘事件)
  ├── WM_LBUTTONUP → 打开浏览器
  ├── WM_RBUTTONUP → TrackPopupMenu 显示菜单
  └── WM_MOUSEMOVE → 更新 tooltip

WM_COMMAND (菜单项点击)
  ├── ID_OPEN → 打开浏览器
  ├── ID_TOGGLE → 切换状态
  └── ID_QUIT → 退出程序
```

#### 关键代码结构

```go
//go:build windows

package tray

import (
    "syscall"
    "unsafe"
)

const (
    NIM_ADD    = 0x00000000
    NIM_MODIFY = 0x00000001
    NIM_DELETE = 0x00000002

    WM_USER       = 0x0400
    WM_TRAYEVENT  = WM_USER + 1
    WM_COMMAND    = 0x0111
    WM_LBUTTONUP  = 0x0202
    WM_RBUTTONUP  = 0x0205
)

var (
    user32  = syscall.NewLazyDLL("user32.dll")
    shell32 = syscall.NewLazyDLL("shell32.dll")

    // Windows API 函数指针
)

type trayInstance struct {
    hwnd       syscall.Handle
    menu       syscall.Handle
    iconData   NOTIFYICONDATA
}

func initTray() *trayInstance {
    // 初始化托盘
}

func (t *trayInstance) run() {
    // 消息循环
}
```

### 其他平台实现 (tray_other.go)

```go
//go:build !windows

package tray

import (
    "github.com/getlantern/systray"
)

// 使用 getlantern/systray 实现
// 保持现有代码逻辑不变
```

### 依赖管理

**go.mod 变更：**

```go
// 使用 replace 或 build tags 确保只在非 Windows 平台引入 systray
```

实际做法是将 getlantern/systray 的 import 放在 `tray_other.go` 中，该文件有 `//go:build !windows` 标签，因此 Windows 编译时不会包含此依赖。

---

## 菜单结构

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

---

## 测试计划

### 功能测试

| 测试项 | 预期结果 |
|-------|---------|
| 左键单击 | 打开浏览器 |
| 右键菜单显示 | 立即显示菜单 |
| 菜单项点击 | 执行对应操作 |
| 退出功能 | 程序正常退出 |
| 状态切换 | 图标和菜单正确更新 |

### 稳定性测试

| 测试项 | 要求 |
|-------|------|
| 长时间运行 | 24小时+ 无假死 |
| 频繁操作 | 连续右键100次无问题 |
| 内存监控 | 无内存泄漏 |

---

## 风险评估

### 低风险

- API 稳定：Windows Shell API 长期稳定
- 隔离清晰：通过 build tags 完全隔离平台代码

### 需要注意

- 主线程要求：Windows 消息循环必须运行在主线程
- 图标格式：需要正确的 ICO 格式

---

## 实现步骤

1. 创建 `tray.go` 定义公共 API
2. 创建 `tray_windows.go` 实现 Windows 原生托盘
3. 创建 `tray_other.go` 移动现有代码
4. 测试 Windows 版本
5. 测试其他平台编译

---

## 参考资料

- [Shell_NotifyIcon API](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shell_notifyiconw)
- [NOTIFYICONDATA structure](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-notifyicondataw)
- [TrackPopupMenu API](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-trackpopupmenu)
