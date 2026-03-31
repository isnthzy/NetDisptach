package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client represents a WebSocket client
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	Channels map[string]bool
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Channels: make(map[string]bool),
	}
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

// Register registers a new client (non-blocking)
func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	default:
		// Hub is not running or channel is full, close the connection
		log.Warn().Msg("Failed to register client: hub not running or channel full")
		if client != nil && client.Conn != nil {
			client.Conn.Close()
		}
	}
}

// Unregister unregisters a client (non-blocking)
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	default:
		// Hub is not running, just close the send channel
		if client != nil && client.Send != nil {
			close(client.Send)
		}
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Debug().Int("clients", len(h.clients)).Msg("Client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Debug().Int("clients", len(h.clients)).Msg("Client disconnected")

		case message := <-h.broadcast:
			h.mu.RLock()
			var toRemove []*Client
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					toRemove = append(toRemove, client)
				}
			}
			h.mu.RUnlock()

			// Remove blocked clients with proper write lock
			if len(toRemove) > 0 {
				h.mu.Lock()
				for _, client := range toRemove {
					if _, ok := h.clients[client]; ok {
						close(client.Send)
						delete(h.clients, client)
					}
				}
				h.mu.Unlock()
			}

		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				close(client.Send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	close(h.stop)
}

// Broadcast sends a message to all clients (non-blocking)
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		// Hub is not running or channel is full, drop the message
	}
}

// BroadcastJSON broadcasts a JSON message
func (h *Hub) BroadcastJSON(eventType string, data interface{}) {
	msg := map[string]interface{}{
		"type":      eventType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      data,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal broadcast message")
		return
	}

	h.Broadcast(jsonData)
}

// BroadcastTraffic broadcasts traffic update
func (h *Hub) BroadcastTraffic(bytesIn, bytesOut int64, connections int64) {
	h.BroadcastJSON("traffic", map[string]interface{}{
		"bytes_in":          bytesIn,
		"bytes_out":         bytesOut,
		"active_connections": connections,
	})
}

// BroadcastConnection broadcasts connection event
func (h *Hub) BroadcastConnection(action, id, client, target, protocol, egressID, nic string, proxyUsed bool) {
	h.BroadcastJSON("connection", map[string]interface{}{
		"action":     action,
		"id":         id,
		"client":     client,
		"target":     target,
		"protocol":   protocol,
		"egress_id":  egressID,
		"nic":        nic,
		"proxy_used": proxyUsed,
	})
}

// BroadcastLog broadcasts a log event
func (h *Hub) BroadcastLog(level, message string, fields map[string]interface{}) {
	h.BroadcastJSON("log", map[string]interface{}{
		"level":   level,
		"message": message,
		"fields":  fields,
	})
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type     string   `json:"type"`
			Channels []string `json:"channels"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Type == "subscribe" {
			for _, ch := range msg.Channels {
				c.Channels[ch] = true
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
