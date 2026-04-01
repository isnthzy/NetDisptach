package tray

import (
	"runtime"
	"sync/atomic"

	"github.com/getlantern/systray"
)

var (
	onOpenBrowser   func()
	onQuit          func()
	onStatusChange  func(running bool)
	currentStatus   int32 = 1 // 1 = running, 0 = stopped
	mOpen           *systray.MenuItem
	mToggle         *systray.MenuItem
	mQuit           *systray.MenuItem
)

// SetOnOpenBrowser sets the callback for opening browser
func SetOnOpenBrowser(fn func()) {
	onOpenBrowser = fn
}

// SetOnQuit sets the callback for quit
func SetOnQuit(fn func()) {
	onQuit = fn
}

// SetStatusChangeCallback sets the callback for status change
func SetStatusChangeCallback(fn func(running bool)) {
	onStatusChange = fn
}

// SetStatus sets the tray icon status ("running" or "stopped")
func SetStatus(status string) {
	if status == "running" {
		atomic.StoreInt32(&currentStatus, 1)
	} else {
		atomic.StoreInt32(&currentStatus, 0)
	}
	updateIcon()
	if mToggle != nil {
		if isRunning() {
			mToggle.SetTitle("停止代理")
			mToggle.SetTooltip("停止代理服务")
		} else {
			mToggle.SetTitle("启动代理")
			mToggle.SetTooltip("启动代理服务")
		}
	}
}

// isRunning returns true if the current status is "running"
func isRunning() bool {
	return atomic.LoadInt32(&currentStatus) == 1
}

// Quit quits the system tray
func Quit() {
	systray.Quit()
}

// Run starts the system tray (blocking call, should run on main thread)
func Run() {
	// Lock to OS thread for GUI operations
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
}

// RunExternalLoop starts the system tray with external loop control
// IMPORTANT: This runs systray on a locked OS thread to prevent freeze issues
func RunExternalLoop() (start, stop func()) {
	start = func() {
		go func() {
			runtime.LockOSThread()
			systray.Run(onReady, onExit)
		}()
	}
	stop = func() {
		systray.Quit()
	}
	return start, stop
}

func onReady() {
	// Set initial icon and title
	systray.SetIcon(IconICO)
	systray.SetTitle("NetDispatch")
	systray.SetTooltip("NetDispatch 网络调度器 - 运行中")

	// Add menu items
	mOpen = systray.AddMenuItem("打开网页", "打开 Web 控制台")

	systray.AddSeparator()

	mToggle = systray.AddMenuItem("停止代理", "停止代理服务")
	mQuit = systray.AddMenuItem("退出", "退出程序")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				if onOpenBrowser != nil {
					go onOpenBrowser()
				}
			case <-mToggle.ClickedCh:
				if isRunning() {
					SetStatus("stopped")
					if onStatusChange != nil {
						go onStatusChange(false)
					}
				} else {
					SetStatus("running")
					if onStatusChange != nil {
						go onStatusChange(true)
					}
				}
			case <-mQuit.ClickedCh:
				if onQuit != nil {
					go onQuit()
				}
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Cleanup if needed
}

// updateIcon updates the tray icon based on current status
func updateIcon() {
	if isRunning() {
		systray.SetIcon(IconICO)
		systray.SetTooltip("NetDispatch 网络调度器 - 运行中")
	} else {
		systray.SetIcon(IconDisabledICO)
		systray.SetTooltip("NetDispatch 网络调度器 - 已停止")
	}
}
