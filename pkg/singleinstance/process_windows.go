//go:build windows

package singleinstance

import (
	"os"
)

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, FindProcess returns an error if the process doesn't exist
	// But we should also check if we can get process info
	err = process.Release()
	return err == nil
}
