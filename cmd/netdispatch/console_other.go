//go:build !windows

package main

// hideConsole is a no-op on non-Windows platforms
func hideConsole() {
	// No-op - console management is Windows-specific
}

// shouldHideConsole always returns false on non-Windows platforms
func shouldHideConsole() bool {
	return false
}

// getParentProcessName returns empty string on non-Windows platforms
func getParentProcessName() string {
	return ""
}
