---
name: windows-process-detection-fix
description: Fix Windows process detection for single instance check and add debug logging
type: design
created: 2026-04-08
---

# Windows Process Detection Fix

## Problem Statement

Windows users report seeing 3 netdispatch.exe processes in Task Manager. Investigation reveals that `isProcessRunning()` in `process_windows.go` has a flawed implementation that doesn't correctly detect if a process is running.

## Root Cause Analysis

### Current Implementation (BROKEN)

```go
func isProcessRunning(pid int) bool {
    process, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    err = process.Release()
    return err == nil
}
```

### Why It's Wrong

1. On Windows, `os.FindProcess(pid)` **always succeeds** - it returns a Process object regardless of whether the PID exists
2. `process.Release()` only releases the handle, it does **not** check if the process is running
3. Result: This function almost always returns `true`

### Impact

- If the lock file contains a stale PID, the function incorrectly thinks the process is still running
- This can lead to confusion in the single instance detection logic
- While file locking (`tryLock`) is the primary protection, this secondary check should also be correct

## Solution

### 1. Fix `isProcessRunning()` using Windows API

Use `OpenProcess` and `GetExitCodeProcess` to correctly detect if a process is running:

```go
//go:build windows

package singleinstance

import (
    "golang.org/x/sys/windows"
)

const STILL_ACTIVE = 259

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
```

### 2. Add Debug Logging

Add logging to `singleinstance.Acquire()` to help diagnose issues:

```go
func Acquire(appName string, apiPort int) (func(), error) {
    lockDir := getUserLockDir(appName)
    if err := os.MkdirAll(lockDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create lock directory: %w", err)
    }

    lockPath = filepath.Join(lockDir, appName+".lock")

    // Check if lock file exists
    if pid, err := readPID(lockPath); err == nil {
        log.Debug().Int("pid", pid).Msg("Found existing lock file")
        if isProcessRunning(pid) {
            log.Warn().Int("pid", pid).Msg("Process is still running")
            return nil, fmt.Errorf("another instance of %s is already running (PID: %d)", appName, pid)
        }
        log.Info().Int("pid", pid).Msg("Stale lock file found, cleaning up")
        os.Remove(lockPath)
    }

    // Create lock file
    f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
    if err != nil {
        return nil, fmt.Errorf("failed to create lock file: %w", err)
    }

    // Try exclusive lock
    err = tryLock(f)
    if err != nil {
        f.Close()
        log.Warn().Msg("Failed to acquire file lock - another instance is running")
        return nil, fmt.Errorf("another instance of %s is already running", appName)
    }

    // Write PID
    pid := os.Getpid()
    f.Truncate(0)
    f.Seek(0, 0)
    fmt.Fprintf(f, "%d\n", pid)

    lockFile = f
    log.Info().Int("pid", pid).Str("path", lockPath).Msg("Acquired single instance lock")
    return release, nil
}
```

## Files Changed

1. `pkg/singleinstance/process_windows.go` - Fix `isProcessRunning()` implementation
2. `pkg/singleinstance/singleinstance.go` - Add debug logging

## Testing

1. Start the program, verify single instance log message
2. Try to start another instance, verify it's blocked
3. Kill the process, verify lock file is cleaned up on next start
4. Test stale lock file scenario (process killed without cleanup)

## Risk Assessment

- **Low risk**: Changes are isolated to single instance detection
- **Backward compatible**: Behavior should be the same for normal use cases
- **Improves reliability**: Correct detection prevents confusion
