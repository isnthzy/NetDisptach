package singleinstance

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// LockFile holds the lock file handle
var lockFile *os.File
var lockPath string

// Acquire tries to acquire a single instance lock
// Returns a release function on success, or error if another instance is running
// apiPort is the port to check for an existing instance
func Acquire(appName string, apiPort int) (func(), error) {
	// Try to acquire lock via port check first
	// If API port is already in use, another instance is running
	addr := fmt.Sprintf("127.0.0.1:%d", apiPort)
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
		return nil, fmt.Errorf("another instance of %s is already running (port %d is in use)", appName, apiPort)
	}

	// Also try file lock as backup
	lockDir := os.TempDir()
	lockPath = filepath.Join(lockDir, appName+".lock")

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

	lockFile = f
	return release, nil
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

// IsRunning checks if another instance is running on the given port
func IsRunning(apiPort int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", apiPort)
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}
