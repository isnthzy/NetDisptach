//go:build windows

package main

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// hideConsole intelligently hides the console window on Windows
// It only hides the console if the program was launched from Explorer (double-click)
// If launched from command line (cmd, powershell, etc.), the console remains visible
func hideConsole() {
	// Get console window handle
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		// No console window, nothing to hide
		return
	}

	// Check if we should hide the console
	// Hide only if launched from Explorer (double-click)
	if shouldHideConsole() {
		user32 := windows.NewLazySystemDLL("user32.dll")
		showWindow := user32.NewProc("ShowWindow")
		showWindow.Call(hwnd, 0) // SW_HIDE = 0
	}
}

// shouldHideConsole determines if the console should be hidden
// Returns true if the program was launched from Explorer (double-click)
// Returns false if launched from command line (cmd, powershell, etc.)
func shouldHideConsole() bool {
	// Get parent process name
	parentName := getParentProcessName()
	if parentName == "" {
		// Can't determine, default to hiding (safer for double-click launch)
		return true
	}

	// Convert to lowercase for comparison
	parentName = strings.ToLower(parentName)

	// List of shell/terminal processes - if parent is one of these, keep console visible
	terminalProcesses := []string{
		"cmd.exe",
		"powershell.exe",
		"pwsh.exe",
		"windowsterminal.exe",
		"wt.exe",
		"conemu64.exe",
		"conemu.exe",
		"alacritty.exe",
		"fluentterminal.exe",
		"hyper.exe",
		"terminus.exe",
		"mobaxterm.exe",
		"tabby.exe",
	}

	for _, term := range terminalProcesses {
		if strings.Contains(parentName, term) {
			return false // Keep console visible for terminal launches
		}
	}

	// If parent is explorer.exe, hide console (double-click launch)
	if strings.Contains(parentName, "explorer.exe") {
		return true
	}

	// For other cases, hide console (likely double-click from file manager)
	return true
}

// getParentProcessName returns the name of the parent process on Windows
func getParentProcessName() string {
	// Use Windows API to get parent process
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")

	// Get current process ID
	getCurrentProcessId := kernel32.NewProc("GetCurrentProcessId")
	pid, _, _ := getCurrentProcessId.Call()

	// Create snapshot of processes
	toolhelp := windows.NewLazySystemDLL("kernel32.dll")
	createToolhelp32Snapshot := toolhelp.NewProc("CreateToolhelp32Snapshot")

	const TH32CS_SNAPPROCESS = 0x00000002
	snapshot, _, _ := createToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 {
		return ""
	}
	defer windows.CloseHandle(windows.Handle(snapshot))

	// Process entry structure
	type PROCESSENTRY32 struct {
		Size              uint32
		CntUsage          uint32
		ProcessID         uint32
		DefaultHeapID     uintptr
		ModuleID          uint32
		CntThreads        uint32
		ParentProcessID   uint32
		PriorityClassBase int32
		Flags             uint32
		ExeFile           [260]uint16
	}

	var entry PROCESSENTRY32
	entry.Size = uint32(unsafe.Sizeof(entry))

	// First process
	process32First := toolhelp.NewProc("Process32FirstW")
	ret, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return ""
	}

	// Find current process and get parent PID
	currentPID := uint32(pid)
	for {
		if entry.ProcessID == currentPID {
			// Found our process, now find parent process name
			parentPID := entry.ParentProcessID

			// Reset to beginning
			ret, _, _ = process32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
			if ret == 0 {
				return ""
			}

			// Find parent process
			for {
				if entry.ProcessID == parentPID {
					return windows.UTF16ToString(entry.ExeFile[:])
				}
				process32Next := toolhelp.NewProc("Process32NextW")
				ret, _, _ := process32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
				if ret == 0 {
					break
				}
			}
			return ""
		}

		process32Next := toolhelp.NewProc("Process32NextW")
		ret, _, _ := process32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return ""
}
