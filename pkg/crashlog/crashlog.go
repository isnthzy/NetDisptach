package crashlog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// WriteCrashLog writes a crash log to a file and returns the file path
func WriteCrashLog(panicValue interface{}, stack []byte) string {
	// Get crash log directory
	crashDir := getCrashLogDir()
	if err := os.MkdirAll(crashDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create crash log directory: %v\n", err)
		crashDir = os.TempDir()
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("crash-%s.log", timestamp)
	filePath := filepath.Join(crashDir, filename)

	// Build crash log content
	content := buildCrashLogContent(panicValue, stack)

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write crash log: %v\n", err)
		return ""
	}

	return filePath
}

func getCrashLogDir() string {
	// Use user config directory based on OS
	var baseDir string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("LOCALAPPDATA")
		}
		baseDir = filepath.Join(appData, "NetDispatch")
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".config", "NetDispatch")
	default:
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, ".config", "NetDispatch")
	}
	return filepath.Join(baseDir, "crash_logs")
}

func buildCrashLogContent(panicValue interface{}, stack []byte) string {
	var content string

	content += "========================================\n"
	content += "NetDispatch Crash Log\n"
	content += "========================================\n\n"
	content += fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("OS: %s\n", runtime.GOOS)
	content += fmt.Sprintf("Arch: %s\n", runtime.GOARCH)
	content += fmt.Sprintf("Go Version: %s\n\n", runtime.Version())

	content += "----------------------------------------\n"
	content += "Panic Value:\n"
	content += "----------------------------------------\n"
	content += fmt.Sprintf("%v\n\n", panicValue)

	content += "----------------------------------------\n"
	content += "Stack Trace:\n"
	content += "----------------------------------------\n"
	content += string(stack)
	content += "\n"

	return content
}

// GetCrashLogDir returns the crash log directory path
func GetCrashLogDir() string {
	return getCrashLogDir()
}
