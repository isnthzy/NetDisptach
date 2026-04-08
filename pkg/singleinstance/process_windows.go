//go:build windows

package singleinstance

import (
	"golang.org/x/sys/windows"
)

const STILL_ACTIVE = 259

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// Try to open the process with query information access
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false // Process doesn't exist or we can't access it
	}
	defer windows.CloseHandle(handle)

	// Check if the process is still running
	var exitCode uint32
	err = windows.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}

	return exitCode == STILL_ACTIVE
}
