package connmgr

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Buffer pool for traffic copy operations
var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

func getBuffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

func putBuffer(buf *[]byte) {
	bufferPool.Put(buf)
}

// Connection represents an active proxy connection
type Connection struct {
	ID          string    `json:"id"`
	ClientAddr  string    `json:"client_addr"`
	TargetAddr  string    `json:"target_addr"`
	Protocol    string    `json:"protocol"`
	EgressID    string    `json:"egress_id"`
	NIC         string    `json:"nic"`
	ProxyUsed   bool      `json:"proxy_used"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	RuleMatched string    `json:"rule_matched"`
	active      atomic.Bool
}

// IsActive returns whether the connection is still active
func (c *Connection) IsActive() bool {
	return c.active.Load()
}

// Duration returns the connection duration
func (c *Connection) Duration() time.Duration {
	if c.IsActive() {
		return time.Since(c.StartTime)
	}
	return c.EndTime.Sub(c.StartTime)
}

// Stats represents aggregated statistics
type Stats struct {
	TotalConnections  int64            `json:"total_connections"`
	ActiveConnections int64            `json:"active_connections"`
	BytesIn           int64            `json:"bytes_in"`
	BytesOut          int64            `json:"bytes_out"`
	ByNIC             map[string]int64 `json:"by_nic"`
	ByEgress          map[string]int64 `json:"by_egress"`
	ByProtocol        map[string]int64 `json:"by_protocol"`
}

// Manager manages active connections
type Manager struct {
	mu             sync.RWMutex
	connections    map[string]*Connection
	idCounter      atomic.Uint64
	stats          Stats
	recentConns    []*Connection // Recent closed connections (last 100)
	trafficHistory []TrafficPoint
	maxRecent      int
	maxHistory     int
	// For rate calculation
	lastRecordTime   time.Time
	lastBytesIn      int64
	lastBytesOut     int64
}

// TrafficPoint represents a point in traffic history (rate in bytes/sec)
type TrafficPoint struct {
	Timestamp   int64 `json:"timestamp"`
	BytesInRate int64 `json:"bytes_in_rate"`
	BytesOutRate int64 `json:"bytes_out_rate"`
}

// NewManager creates a new connection manager
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		stats: Stats{
			ByNIC:      make(map[string]int64),
			ByEgress:   make(map[string]int64),
			ByProtocol: make(map[string]int64),
		},
		recentConns:    make([]*Connection, 0, 100),
		trafficHistory: make([]TrafficPoint, 0, 60),
		maxRecent:      100,
		maxHistory:     60,
		lastRecordTime: time.Now(),
	}
}

// Create creates a new connection
func (m *Manager) Create(clientAddr, targetAddr, protocol string) *Connection {
	id := m.generateID()
	conn := &Connection{
		ID:         id,
		ClientAddr: clientAddr,
		TargetAddr: targetAddr,
		Protocol:   protocol,
		StartTime:  time.Now(),
	}
	conn.active.Store(true)

	m.mu.Lock()
	m.connections[id] = conn
	m.stats.TotalConnections++
	m.stats.ActiveConnections++
	m.stats.ByProtocol[protocol]++
	m.mu.Unlock()

	return conn
}

// Close closes a connection
func (m *Manager) Close(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return
	}

	conn.active.Store(false)
	conn.EndTime = time.Now()
	m.stats.ActiveConnections--

	if conn.BytesIn > 0 || conn.BytesOut > 0 {
		m.stats.BytesIn += conn.BytesIn
		m.stats.BytesOut += conn.BytesOut
		m.stats.ByNIC[conn.NIC] += conn.BytesIn + conn.BytesOut
		m.stats.ByEgress[conn.EgressID]++
	}

	// Add to recent connections
	m.recentConns = append(m.recentConns, conn)
	if len(m.recentConns) > m.maxRecent {
		m.recentConns = m.recentConns[1:]
	}

	delete(m.connections, id)
}

// Update updates connection information
func (m *Manager) Update(id string, egressID, nic string, proxyUsed bool, ruleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[id]; exists {
		conn.EgressID = egressID
		conn.NIC = nic
		conn.ProxyUsed = proxyUsed
		conn.RuleMatched = ruleID
	}
}

// AddBytes adds transferred bytes to a connection
func (m *Manager) AddBytes(id string, bytesIn, bytesOut int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[id]; exists {
		conn.BytesIn += bytesIn
		conn.BytesOut += bytesOut
	}
}

// Get returns a connection by ID
func (m *Manager) Get(id string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[id]
}

// List returns all active connections
func (m *Manager) List() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		result = append(result, conn)
	}
	return result
}

// GetStats returns current statistics
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		TotalConnections:  m.stats.TotalConnections,
		ActiveConnections: m.stats.ActiveConnections,
		BytesIn:           m.stats.BytesIn,
		BytesOut:          m.stats.BytesOut,
		ByNIC:             make(map[string]int64),
		ByEgress:          make(map[string]int64),
		ByProtocol:        make(map[string]int64),
	}

	for k, v := range m.stats.ByNIC {
		stats.ByNIC[k] = v
	}
	for k, v := range m.stats.ByEgress {
		stats.ByEgress[k] = v
	}
	for k, v := range m.stats.ByProtocol {
		stats.ByProtocol[k] = v
	}

	return stats
}

// GetRecentConnections returns recent closed connections
func (m *Manager) GetRecentConnections() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return recent connections in reverse order (newest first)
	result := make([]*Connection, 0, len(m.recentConns))
	for i := len(m.recentConns) - 1; i >= 0; i-- {
		result = append(result, m.recentConns[i])
	}
	return result
}

// RecordTraffic records current traffic rate to history
func (m *Manager) RecordTraffic() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastRecordTime).Seconds()

	var bytesInRate, bytesOutRate int64
	if elapsed > 0 {
		bytesInRate = int64(float64(m.stats.BytesIn-m.lastBytesIn) / elapsed)
		bytesOutRate = int64(float64(m.stats.BytesOut-m.lastBytesOut) / elapsed)
	}

	// Ensure non-negative (in case of counter reset)
	if bytesInRate < 0 {
		bytesInRate = 0
	}
	if bytesOutRate < 0 {
		bytesOutRate = 0
	}

	point := TrafficPoint{
		Timestamp:    now.Unix(),
		BytesInRate:  bytesInRate,
		BytesOutRate: bytesOutRate,
	}

	m.trafficHistory = append(m.trafficHistory, point)
	if len(m.trafficHistory) > m.maxHistory {
		m.trafficHistory = m.trafficHistory[1:]
	}

	// Update last values
	m.lastRecordTime = now
	m.lastBytesIn = m.stats.BytesIn
	m.lastBytesOut = m.stats.BytesOut
}

// GetTrafficHistory returns traffic history
func (m *Manager) GetTrafficHistory() []TrafficPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]TrafficPoint, len(m.trafficHistory))
	copy(result, m.trafficHistory)
	return result
}

func (m *Manager) generateID() string {
	id := m.idCounter.Add(1)
	return fmt.Sprintf("conn-%d", id)
}

// CopyTraffic copies data bidirectionally between two connections
// It ensures both goroutines exit when one direction finishes
func CopyTraffic(client, target net.Conn) (int64, int64) {
	var bytesIn, bytesOut int64
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	// When one direction finishes, close both connections to unblock the other
	closeConn := func() {
		// Close connections to unblock any pending reads/writes
		if client != nil {
			client.Close()
		}
		if target != nil {
			target.Close()
		}
	}

	go func() {
		defer closeConn()
		n, _ := ioCopy(target, client)
		mu.Lock()
		bytesOut = n
		mu.Unlock()
		done <- struct{}{}
	}()

	go func() {
		defer closeConn()
		n, _ := ioCopy(client, target)
		mu.Lock()
		bytesIn = n
		mu.Unlock()
		done <- struct{}{}
	}()

	// Wait for first goroutine to finish (connection closed one way)
	<-done

	// Give the second goroutine a chance to finish gracefully
	// If it doesn't finish within 1 second, proceed anyway (connections are already closed)
	select {
	case <-done:
		// Both directions finished cleanly
	case <-time.After(1 * time.Second):
		// Force close connections and continue
		// The goroutine should exit due to the closed connection
	}

	return bytesIn, bytesOut
}

func ioCopy(dst, src net.Conn) (int64, error) {
	buf := getBuffer()
	defer putBuffer(buf)
	return io.CopyBuffer(dst, src, *buf)
}
