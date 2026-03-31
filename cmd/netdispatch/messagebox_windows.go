//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// ShowMessageBox displays a Windows message box with the given title and message
// Returns the result of the MessageBox call (IDOK, IDCANCEL, etc.)
func ShowMessageBox(title, message string) int {
	user32 := windows.NewLazySystemDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")

	titlePtr, _ := windows.UTF16PtrFromString(title)
	messagePtr, _ := windows.UTF16PtrFromString(message)

	// MB_OK = 0, MB_ICONINFORMATION = 0x40
	const MB_OK = 0x00000000
	const MB_ICONINFORMATION = 0x00000040

	ret, _, _ := messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		MB_OK|MB_ICONINFORMATION,
	)

	return int(ret)
}

// focusExistingInstance tries to bring the existing instance window to front
func focusExistingInstance() {
	// Find the existing instance window by class name
	user32 := windows.NewLazySystemDLL("user32.dll")
	findWindowW := user32.NewProc("FindWindowW")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")

	// Try to find the systray hidden window
	className, _ := windows.UTF16PtrFromString("SystrayClass")
	hwnd, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(className)), 0)
	if hwnd != 0 {
		setForegroundWindow.Call(hwnd)
	}
}
