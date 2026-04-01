//go:build windows

package tray

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	moduser32   = syscall.NewLazyDLL("user32.dll")
	modshell32  = syscall.NewLazyDLL("shell32.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW       = moduser32.NewProc("RegisterClassExW")
	procCreateWindowExW        = moduser32.NewProc("CreateWindowExW")
	procDefWindowProcW         = moduser32.NewProc("DefWindowProcW")
	procGetMessageW            = moduser32.NewProc("GetMessageW")
	procTranslateMessage       = moduser32.NewProc("TranslateMessage")
	procDispatchMessageW       = moduser32.NewProc("DispatchMessageW")
	procPostQuitMessage        = moduser32.NewProc("PostQuitMessage")
	procPostThreadMessageW     = moduser32.NewProc("PostThreadMessageW")
	procCreatePopupMenu        = moduser32.NewProc("CreatePopupMenu")
	procAppendMenuW            = moduser32.NewProc("AppendMenuW")
	procTrackPopupMenu         = moduser32.NewProc("TrackPopupMenu")
	procDestroyMenu            = moduser32.NewProc("DestroyMenu")
	procLoadImageW             = moduser32.NewProc("LoadImageW")
	procDestroyIcon            = moduser32.NewProc("DestroyIcon")
	procSetForegroundWindow    = moduser32.NewProc("SetForegroundWindow")
	procGetCurrentThreadId     = modkernel32.NewProc("GetCurrentThreadId")
	procGetCursorPos           = moduser32.NewProc("GetCursorPos")
	procSetForegroundWindowEx  = moduser32.NewProc("SetForegroundWindow")
	procFindWindowW            = moduser32.NewProc("FindWindowW")
	procPostMessageW           = moduser32.NewProc("PostMessageW")
	procGetForegroundWindow    = moduser32.NewProc("GetForegroundWindow")
	procSetWindowPos           = moduser32.NewProc("SetWindowPos")
	procSendMessageW           = moduser32.NewProc("SendMessageW")

	procShellNotifyIconW = modshell32.NewProc("Shell_NotifyIconW")
)

// Constants
const (
	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 1
	WM_COMMAND       = 0x0111
	WM_INITMENUPOPUP = 0x0117
	WM_NULL          = 0x0000

	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIF_MESSAGE    = 0x00000001
	NIF_ICON       = 0x00000002
	NIF_TIP        = 0x00000004
	NIF_SHOWTIP    = 0x00000080

	MF_STRING      = 0x00000000
	MF_SEPARATOR   = 0x00000800
	MF_ENABLED     = 0x00000000
	MF_DISABLED    = 0x00000002
	MF_GRAYED      = 0x00000001

	TPM_RIGHTALIGN   = 0x0008
	TPM_BOTTOMALIGN  = 0x0020
	TPM_LEFTBUTTON   = 0x0000
	TPM_RIGHTBUTTON  = 0x0002
	TPM_VERNEGANIMATION = 0x2000
	TPM_VERPOSANIMATION = 0x1000

	LR_LOADFROMFILE = 0x00000010
	IMAGE_ICON      = 1

	// CW_USEDEFAULT for window position/size
	// Use uintptr directly to avoid overflow
	SWP_NOACTIVATE     = 0x0010
	SWP_NOMOVE         = 0x0002
	SWP_NOSIZE         = 0x0001
	SWP_SHOWWINDOW     = 0x0040
)

// Menu item IDs
const (
	IDM_OPEN   = 1001
	IDM_TOGGLE = 1002
	IDM_QUIT   = 1003
)

// NOTIFYICONDATAW structure
type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         syscall.GUID
	HBalloonIcon     uintptr
}

// WNDCLASSEXW structure
type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

// POINT structure
type POINT struct {
	X int32
	Y int32
}

// MSG structure
type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// Tray state
var (
	currentStatus    int32 = 1 // 1 = running, 0 = stopped
	hwnd             uintptr
	hIconRunning     uintptr
	hIconStopped     uintptr
	className        = syscall.StringToUTF16Ptr("NetDispatchTrayClass")
	windowTitle      = syscall.StringToUTF16Ptr("NetDispatch")
	quitChan         chan struct{}
	running          atomic.Bool
	tempIconPath     string
	tempIconDisabled string
)

// SetStatus sets the tray icon status ("running" or "stopped")
func SetStatus(status string) {
	if status == "running" {
		atomic.StoreInt32(&currentStatus, 1)
	} else {
		atomic.StoreInt32(&currentStatus, 0)
	}
	updateTrayIcon()
}

// isRunning returns true if the current status is "running"
func isRunning() bool {
	return atomic.LoadInt32(&currentStatus) == 1
}

// Quit quits the system tray
func Quit() {
	if running.Load() {
		// Post WM_CLOSE to our hidden window
		if hwnd != 0 {
			procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
		}
	}
}

// Run starts the system tray (blocking call, should run on main thread)
func Run() {
	initializeTray()
	messageLoop()
	cleanup()
}

// RunExternalLoop starts the system tray with external loop control
func RunExternalLoop() (start, stop func()) {
	quitChan = make(chan struct{})

	start = func() {
		initializeTray()
		running.Store(true)
		// Run message loop in a separate goroutine
		go func() {
			messageLoop()
			cleanup()
		}()
	}

	stop = func() {
		if running.Load() {
			Quit()
			running.Store(false)
		}
	}

	return start, stop
}

// initializeTray sets up the window and tray icon
func initializeTray() {
	// Get module handle
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	// Write icons to temp files
	tempIconPath = writeIconToTempFile(IconICO, "netdispatch_icon.ico")
	tempIconDisabled = writeIconToTempFile(IconDisabledICO, "netdispatch_icon_disabled.ico")

	// Load icons
	hIconRunning = loadIconFromFile(tempIconPath)
	hIconStopped = loadIconFromFile(tempIconDisabled)

	// Register window class
	registerWindowClass(hInstance)

	// Create hidden window
	createHiddenWindow(hInstance)

	// Add tray icon
	addTrayIcon()
}

// writeIconToTempFile writes icon bytes to a temp file and returns the path
func writeIconToTempFile(iconData []byte, filename string) string {
	tempDir := os.TempDir()
	path := filepath.Join(tempDir, filename)

	// Check if file already exists with same content
	if existingData, err := os.ReadFile(path); err == nil {
		if len(existingData) == len(iconData) {
			return path
		}
	}

	if err := os.WriteFile(path, iconData, 0644); err != nil {
		return ""
	}
	return path
}

// loadIconFromFile loads an icon from a file
func loadIconFromFile(path string) uintptr {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := procLoadImageW.Call(
		0,                              // hInst
		uintptr(unsafe.Pointer(pathPtr)), // lpszName
		IMAGE_ICON,                     // uType
		0,                              // cxDesired (0 = actual size)
		0,                              // cyDesired
		LR_LOADFROMFILE,                // fuLoad
	)
	return ret
}

// registerWindowClass registers the window class for our hidden window
func registerWindowClass(hInstance uintptr) {
	wndProc := syscall.NewCallback(windowProc)

	wcx := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         0,
		LpfnWndProc:   wndProc,
		CbClsExtra:    0,
		CbWndExtra:    0,
		HInstance:     hInstance,
		HIcon:         0,
		HCursor:       0,
		HbrBackground: 0,
		LpszMenuName:  nil,
		LpszClassName: className,
		HIconSm:       0,
	}

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcx)))
}

// createHiddenWindow creates the hidden window for message handling
func createHiddenWindow(hInstance uintptr) {
	// CW_USEDEFAULT = 0x80000000 as uintptr
	cwUseDefault := uintptr(0x80000000)

	ret, _, _ := procCreateWindowExW.Call(
		0,                               // dwExStyle
		uintptr(unsafe.Pointer(className)), // lpClassName
		uintptr(unsafe.Pointer(windowTitle)), // lpWindowName
		0,                               // dwStyle
		cwUseDefault,                    // x
		cwUseDefault,                    // y
		cwUseDefault,                    // nWidth
		cwUseDefault,                    // nHeight
		0,                               // hWndParent
		0,                               // hMenu
		hInstance,                       // hInstance
		0,                               // lpParam
	)
	hwnd = ret
}

// addTrayIcon adds the tray icon to the notification area
func addTrayIcon() {
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP | NIF_SHOWTIP
	nid.UCallbackMessage = WM_TRAYICON

	if isRunning() {
		nid.HIcon = hIconRunning
		copy(nid.SzTip[:], syscall.StringToUTF16("NetDispatch 网络调度器 - 运行中"))
	} else {
		nid.HIcon = hIconStopped
		copy(nid.SzTip[:], syscall.StringToUTF16("NetDispatch 网络调度器 - 已停止"))
	}

	procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
}

// updateTrayIcon updates the tray icon and tooltip
func updateTrayIcon() {
	if hwnd == 0 {
		return
	}

	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = NIF_ICON | NIF_TIP
	nid.UCallbackMessage = WM_TRAYICON

	if isRunning() {
		nid.HIcon = hIconRunning
		copy(nid.SzTip[:], syscall.StringToUTF16("NetDispatch 网络调度器 - 运行中"))
	} else {
		nid.HIcon = hIconStopped
		copy(nid.SzTip[:], syscall.StringToUTF16("NetDispatch 网络调度器 - 已停止"))
	}

	procShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
}

// windowProc is the window procedure for our hidden window
func windowProc(hWnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case WM_TRAYICON:
		// Handle tray icon events
		switch lParam {
		case 0x0205: // WM_RBUTTONUP
			showPopupMenu(hWnd)
		case 0x0203: // WM_LBUTTONDBLCLK
			// Double-click to open browser
			if onOpenBrowser != nil {
				go onOpenBrowser()
			}
		case 0x0202: // WM_RBUTTONDBLCLK
			// Right double-click also shows menu
			showPopupMenu(hWnd)
		}
		return 0

	case WM_COMMAND:
		// Handle menu item clicks
		switch wParam {
		case IDM_OPEN:
			if onOpenBrowser != nil {
				go onOpenBrowser()
			}
		case IDM_TOGGLE:
			if isRunning() {
				SetStatus("stopped")
				if onStatusChange != nil {
					go onStatusChange(false)
				}
			} else {
				SetStatus("running")
				if onStatusChange != nil {
					go onStatusChange(true)
				}
			}
		case IDM_QUIT:
			if onQuit != nil {
				go onQuit()
			}
			// Remove tray icon and quit
			removeTrayIcon()
			procPostQuitMessage.Call(0)
		}
		return 0

	case WM_CLOSE, WM_DESTROY:
		removeTrayIcon()
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hWnd, uintptr(message), wParam, lParam)
	return ret
}

// showPopupMenu shows the popup menu at the cursor position
func showPopupMenu(hWnd uintptr) {
	// Create popup menu
	hMenu, _, _ := procCreatePopupMenu.Call()

	// Add menu items (Chinese)
	openText, _ := syscall.UTF16PtrFromString("打开网页")
	toggleText, _ := syscall.UTF16PtrFromString("停止代理")
	quitText, _ := syscall.UTF16PtrFromString("退出")

	if !isRunning() {
		toggleText, _ = syscall.UTF16PtrFromString("启动代理")
	}

	procAppendMenuW.Call(hMenu, MF_STRING, IDM_OPEN, uintptr(unsafe.Pointer(openText)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_TOGGLE, uintptr(unsafe.Pointer(toggleText)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_QUIT, uintptr(unsafe.Pointer(quitText)))

	// Get cursor position
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Set foreground window to ensure menu closes when clicking outside
	procSetForegroundWindow.Call(hWnd)

	// Show popup menu
	// TrackPopupMenu blocks until a menu item is selected or the menu is cancelled
	// This runs on the message loop thread, so it doesn't freeze
	procTrackPopupMenu.Call(
		hMenu,
		TPM_RIGHTALIGN|TPM_BOTTOMALIGN|TPM_LEFTBUTTON,
		uintptr(pt.X),
		uintptr(pt.Y),
		0,
		hWnd,
		0,
	)

	// Destroy menu
	procDestroyMenu.Call(hMenu)
}

// removeTrayIcon removes the tray icon
func removeTrayIcon() {
	if hwnd == 0 {
		return
	}

	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1

	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}

// messageLoop runs the Windows message loop
func messageLoop() {
	var msg MSG

	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)

		if ret == 0 { // WM_QUIT
			break
		}

		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// cleanup cleans up resources
func cleanup() {
	// Destroy icons
	if hIconRunning != 0 {
		procDestroyIcon.Call(hIconRunning)
		hIconRunning = 0
	}
	if hIconStopped != 0 {
		procDestroyIcon.Call(hIconStopped)
		hIconStopped = 0
	}

	// Clean up temp icon files (optional)
	if tempIconPath != "" {
		os.Remove(tempIconPath)
	}
	if tempIconDisabled != "" {
		os.Remove(tempIconDisabled)
	}

	// Signal quit channel if set
	if quitChan != nil {
		close(quitChan)
	}
}

// GetModuleHandleW wrapper
var procGetModuleHandleW = modkernel32.NewProc("GetModuleHandleW")

func init() {
	// Initialize the proc
	procGetModuleHandleW.Find()
}
