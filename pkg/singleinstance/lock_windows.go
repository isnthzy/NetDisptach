//go:build windows

package singleinstance

import (
	"golang.org/x/sys/windows"
	"os"
)

func tryLock(f *os.File) error {
	// Try exclusive lock
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &windows.Overlapped{})
	return err
}

func unlock(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
}
