package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/energye/systray"
)

var (
	onOpenBrowser func()
	onQuit        func()
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
	systray.Quit()
}

// Run starts the system tray
func Run() {
	systray.Run(onReady, onExit)
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
