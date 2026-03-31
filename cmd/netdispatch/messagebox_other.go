//go:build !windows

package main

import (
	"fmt"
	"os"
)

// ShowMessageBox displays a message on non-Windows platforms
// On Linux/macOS, we just print to stderr (no popup as per user request)
func ShowMessageBox(title, message string) int {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", title, message)
	return 0
}

// focusExistingInstance is a no-op on non-Windows platforms
func focusExistingInstance() {
	// No-op on non-Windows
}
