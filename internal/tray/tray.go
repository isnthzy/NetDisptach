package tray

// Public API - platform independent

var (
	onOpenBrowser   func()
	onQuit          func()
	onStatusChange  func(running bool)
)

// SetOnOpenBrowser sets the callback for opening browser
func SetOnOpenBrowser(fn func()) {
	onOpenBrowser = fn
}

// SetOnQuit sets the callback for quit
func SetOnQuit(fn func()) {
	onQuit = fn
}

// SetStatusChangeCallback sets the callback for status change
func SetStatusChangeCallback(fn func(running bool)) {
	onStatusChange = fn
}
