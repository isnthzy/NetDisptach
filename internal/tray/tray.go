package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

var (
	onOpenBrowser         func()
	onQuit                func()
	onStatusChange        func(running bool)
	currentStatus         = "running"
	mOpen                 *systray.MenuItem
	mToggle               *systray.MenuItem
	mQuit                 *systray.MenuItem
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
	currentStatus = status
	updateIcon()
	if mToggle != nil {
		if currentStatus == "running" {
			mToggle.SetTitle("停止代理")
			mToggle.SetTooltip("停止代理服务")
		} else {
			mToggle.SetTitle("启动代理")
			mToggle.SetTooltip("启动代理服务")
		}
	}
}

// Quit quits the system tray
func Quit() {
	systray.Quit()
}

// Run starts the system tray (blocking call, should run on main thread)
func Run() {
	systray.Run(onReady, onExit)
}

// RunExternalLoop starts the system tray with external loop control
// Note: getlantern/systray doesn't have RunWithExternalLoop
// This starts systray in a goroutine for compatibility
func RunExternalLoop() (start, stop func()) {
	start = func() {
		go systray.Run(onReady, onExit)
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
					onOpenBrowser()
				}
			case <-mToggle.ClickedCh:
				if currentStatus == "running" {
					SetStatus("stopped")
					if onStatusChange != nil {
						onStatusChange(false)
					}
				} else {
					SetStatus("running")
					if onStatusChange != nil {
						onStatusChange(true)
					}
				}
			case <-mQuit.ClickedCh:
				if onQuit != nil {
					onQuit()
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
	if currentStatus == "running" {
		systray.SetIcon(IconICO)
		systray.SetTooltip("NetDispatch 网络调度器 - 运行中")
	} else {
		systray.SetIcon(IconDisabledICO)
		systray.SetTooltip("NetDispatch 网络调度器 - 已停止")
	}
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err != nil {
			fmt.Println("Failed to open browser:", err)
		} else {
			go cmd.Wait()
		}
	}
}
