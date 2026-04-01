package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

var (
	onOpenBrowser func()
	onQuit         func()
	startFunc      func()
	stopFunc       func()
)

// SetOnOpenBrowser sets the callback for opening browser
func SetOnOpenBrowser(fn func()) {
	onOpenBrowser = fn
}

// SetOnQuit sets the callback for quit
func SetOnQuit(fn func()) {
	onQuit = fn
}

// Quit quits the system tray
func Quit() {
	if stopFunc != nil {
		stopFunc()
	}
	systray.Quit()
}

// Run starts the system tray (blocking call, should run on main thread)
// Deprecated: Use RunExternalLoop for better compatibility
func Run() {
	systray.Run(onReady, onExit)
}

// RunExternalLoop starts the system tray with external loop control
// This is the recommended way to run systray when you have other
// event loops or need to run it alongside other services.
// Returns start and stop functions.
func RunExternalLoop() (start, stop func()) {
	start, stop = systray.RunWithExternalLoop(onReady, onExit)
	startFunc = start
	stopFunc = stop
	return start, stop
}

func onReady() {
	// Set icon - use embedded ICO
	systray.SetIcon(IconICO)
	systray.SetTitle("NetDispatch")
	systray.SetTooltip("NetDispatch 网络调度器")

	// Set up click handlers for better Windows compatibility
	// Left click opens browser
	systray.SetOnClick(func(menu systray.IMenu) {
		if onOpenBrowser != nil {
			onOpenBrowser()
		}
	})

	// Double click also opens browser
	systray.SetOnDClick(func(menu systray.IMenu) {
		if onOpenBrowser != nil {
			onOpenBrowser()
		}
	})

	// Right click shows the menu - this is the key fix for the menu not appearing
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	mOpen := systray.AddMenuItem("打开网页", "打开 Web 控制台")
	mOpen.Click(func() {
		if onOpenBrowser != nil {
			onOpenBrowser()
		}
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("退出", "退出程序")
	mQuit.Click(func() {
		if onQuit != nil {
			onQuit()
		}
		systray.Quit()
	})
}

func onExit() {
	// Cleanup if needed
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use cmd.exe with start command for better compatibility
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err != nil {
			// Log error silently
			fmt.Println("Failed to open browser:", err)
		} else {
			// Reap zombie process
			go cmd.Wait()
		}
	}
}
