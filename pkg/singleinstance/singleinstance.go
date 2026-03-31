package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LockFile holds the lock file handle
var lockFile *os.File
var lockPath string

// Acquire tries to acquire a single instance lock
// Returns a release function on success, or error if another instance is running
// apiPort is used only for informational purposes in error messages
func Acquire(appName string, apiPort int) (func(), error) {
	// Get user-specific lock directory
	// Different users (e.g., root vs normal user) should have separate locks
	lockDir := getUserLockDir(appName)
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockPath = filepath.Join(lockDir, appName+".lock")

	// Check if lock file exists and has a running process
	if pid, err := readPID(lockPath); err == nil {
		if isProcessRunning(pid) {
			return nil, fmt.Errorf("another instance of %s is already running (PID: %d)", appName, pid)
		}
		// Process is not running, clean up stale lock file
		os.Remove(lockPath)
	}

	// Try to create and lock the file
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock
	err = tryLock(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance of %s is already running", appName)
	}

	// Write our PID to the lock file
	pid := os.Getpid()
	f.Truncate(0)
	f.Seek(0, 0)
	fmt.Fprintf(f, "%d\n", pid)

	lockFile = f
	return release, nil
}

// getUserLockDir returns a user-specific lock directory
// This ensures different users don't conflict with each other
func getUserLockDir(appName string) string {
	// Try to use user-specific directory first
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		return filepath.Join(homeDir, ".local", "run", appName)
	}

	// Fallback to temp directory with user ID
	uid := os.Getuid()
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", appName, uid))
}

// readPID reads the PID from a lock file
func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// release releases the single instance lock
func release() {
	if lockFile != nil {
		unlock(lockFile)
		lockFile.Close()
		os.Remove(lockPath)
		lockFile = nil
	}
}

// IsRunning checks if another instance is running
func IsRunning(apiPort int) bool {
	// This is now a simpler check - just see if we can connect
	// Used for informational purposes only
	return false
}
