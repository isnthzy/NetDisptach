package ws

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// BroadcastLogWriter wraps an io.Writer and broadcasts structured logs
type BroadcastLogWriter struct {
	underlying io.Writer
	hub        *Hub
	mu         sync.RWMutex
	bufMu      sync.Mutex
	buf        bytes.Buffer
}

// NewBroadcastLogWriter creates a writer that both writes to underlying and broadcasts
func NewBroadcastLogWriter(underlying io.Writer, hub *Hub) *BroadcastLogWriter {
	return &BroadcastLogWriter{
		underlying: underlying,
		hub:        hub,
	}
}

// SetHub updates the hub
func (w *BroadcastLogWriter) SetHub(hub *Hub) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.hub = hub
}

// Log pattern for zerolog ConsoleWriter output
// Format: "15:04:05 INF message" or "15:04:05 WRN message"
var logPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2})\s+(INF|WRN|ERR|DBG|FTL)\s+(.*)$`)

// levelMapping maps zerolog level prefixes to lowercase strings
var levelMapping = map[string]string{
	"INF": "info",
	"WRN": "warn",
	"ERR": "error",
	"DBG": "debug",
	"FTL": "fatal",
}

// Write implements io.Writer
func (w *BroadcastLogWriter) Write(p []byte) (n int, err error) {
	// Write to underlying writer first
	if w.underlying != nil {
		n, err = w.underlying.Write(p)
	}

	// Try to parse and broadcast
	w.mu.RLock()
	hub := w.hub
	w.mu.RUnlock()

	if hub == nil || len(p) == 0 {
		return n, err
	}

	// Accumulate data in buffer for multi-line handling (protected by bufMu)
	w.bufMu.Lock()
	defer w.bufMu.Unlock()

	w.buf.Write(p)

	// Process complete lines
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// No complete line yet, put back the partial line
			if line != "" {
				w.buf.WriteString(line)
			}
			break
		}

		// Parse the log line
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}

		// Try to match the log pattern
		matches := logPattern.FindStringSubmatch(line)
		if len(matches) == 4 {
			level := levelMapping[matches[2]]
			message := matches[3]

			// Extract fields from message if present
			fields := extractFields(message)

			// Clean message (remove field markers)
			cleanMessage := cleanMessage(message)

			// Broadcast to WebSocket
			hub.BroadcastLog(level, cleanMessage, fields)
		}
	}

	return n, err
}

// fieldPattern matches key=value patterns in log messages
var fieldPattern = regexp.MustCompile(`(\w+)=("[^"]*"|\S+)`)

// extractFields extracts key=value fields from log message
func extractFields(message string) map[string]interface{} {
	fields := make(map[string]interface{})
	matches := fieldPattern.FindAllStringSubmatch(message, -1)
	for _, match := range matches {
		key := match[1]
		value := match[2]
		// Remove quotes if present
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
		}
		fields[key] = value
	}
	return fields
}

// cleanMessage removes field markers from the message
func cleanMessage(message string) string {
	// Remove key=value patterns at the end of message
	idx := strings.Index(message, " path=")
	if idx == -1 {
		idx = strings.Index(message, " msg=")
	}
	if idx == -1 {
		idx = strings.Index(message, " error=")
	}
	if idx == -1 {
		idx = strings.Index(message, " level=")
	}
	if idx > 0 {
		return message[:idx]
	}
	return message
}

// LogHook is a zerolog hook that broadcasts logs to WebSocket clients
type LogHook struct {
	hub   *Hub
	level zerolog.Level
	mu    sync.RWMutex
}

var (
	globalHook *LogHook
	hubMu      sync.RWMutex
)

// SetHub sets the hub for the global log hook
func SetHub(hub *Hub) {
	hubMu.Lock()
	defer hubMu.Unlock()
	if globalHook != nil {
		globalHook.mu.Lock()
		globalHook.hub = hub
		globalHook.mu.Unlock()
	}
}

// NewLogHook creates a new log hook that forwards logs to WebSocket
func NewLogHook(level zerolog.Level) *LogHook {
	hook := &LogHook{
		level: level,
	}
	globalHook = hook
	return hook
}

// Run implements zerolog.Hook interface
func (h *LogHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if level > h.level {
		return
	}

	h.mu.RLock()
	hub := h.hub
	h.mu.RUnlock()

	if hub == nil {
		return
	}

	// Get the level name
	levelName := level.String()

	// Extract fields from the event
	fields := make(map[string]interface{})

	hub.BroadcastLog(levelName, message, fields)
}

// LogEntry represents a structured log entry for broadcasting
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// BroadcastLogEntry broadcasts a structured log entry
func (h *Hub) BroadcastLogEntry(entry LogEntry) {
	msg := map[string]interface{}{
		"type":      "log",
		"timestamp": entry.Timestamp.Format(time.RFC3339),
		"data": map[string]interface{}{
			"level":   entry.Level,
			"message": entry.Message,
			"fields":  entry.Fields,
		},
	}

	h.BroadcastJSON("log", msg["data"])
}
