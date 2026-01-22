package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/joshuakim/linefinder/internal/metrics"
	"github.com/joshuakim/linefinder/internal/models"
)

// Message types
const (
	MessageTypeOddsUpdate   = "odds_update"
	MessageTypeSubscribe    = "subscribe"
	MessageTypeUnsubscribe  = "unsubscribe"
	MessageTypeError        = "error"
	MessageTypeStatus       = "status"
	MessageTypePong         = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type      string          `json:"type"`
	Sport     string          `json:"sport,omitempty"`
	Games     []models.Game   `json:"games,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Error     string          `json:"error,omitempty"`
	Status    string          `json:"status,omitempty"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Client subscriptions by sport
	subscriptions map[models.Sport]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Metrics
	metrics *metrics.Metrics

	// Configuration
	maxConnections int
}

// NewHub creates a new Hub
func NewHub(m *metrics.Metrics, maxConnections int) *Hub {
	if maxConnections <= 0 {
		maxConnections = 1000
	}
	return &Hub{
		clients:        make(map[*Client]bool),
		subscriptions:  make(map[models.Sport]map[*Client]bool),
		register:       make(chan *Client, 256),
		unregister:     make(chan *Client, 256),
		metrics:        m,
		maxConnections: maxConnections,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check connection limit
	if len(h.clients) >= h.maxConnections {
		log.Printf("WebSocket: Connection rejected - at capacity (%d)", h.maxConnections)
		// Send error and close
		errMsg := Message{
			Type:      MessageTypeError,
			Error:     "Server at capacity, please try again later",
			Timestamp: time.Now(),
		}
		data, _ := json.Marshal(errMsg)
		client.send <- data
		close(client.send)
		return
	}

	h.clients[client] = true
	h.metrics.RecordConnection()
	log.Printf("WebSocket: Client connected (total: %d)", len(h.clients))
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		// Remove from all subscriptions
		for sport := range h.subscriptions {
			delete(h.subscriptions[sport], client)
			// Update subscriber count metric
			h.metrics.UpdateSubscriberCount(string(sport), int64(len(h.subscriptions[sport])))
		}

		close(client.send)
		h.metrics.RecordDisconnection()
		log.Printf("WebSocket: Client disconnected (total: %d)", len(h.clients))
	}
}

// Subscribe adds a client to a sport's subscription list
func (h *Hub) Subscribe(client *Client, sport models.Sport) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscriptions[sport] == nil {
		h.subscriptions[sport] = make(map[*Client]bool)
	}
	h.subscriptions[sport][client] = true
	h.metrics.UpdateSubscriberCount(string(sport), int64(len(h.subscriptions[sport])))
	log.Printf("WebSocket: Client subscribed to %s (subscribers: %d)", sport, len(h.subscriptions[sport]))
}

// Unsubscribe removes a client from a sport's subscription list
func (h *Hub) Unsubscribe(client *Client, sport models.Sport) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscriptions[sport] != nil {
		delete(h.subscriptions[sport], client)
		h.metrics.UpdateSubscriberCount(string(sport), int64(len(h.subscriptions[sport])))
	}
}

// Broadcast sends a message to all clients subscribed to a sport
func (h *Hub) Broadcast(sport models.Sport, games []models.Game) {
	message := Message{
		Type:      MessageTypeOddsUpdate,
		Sport:     string(sport),
		Games:     games,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal broadcast message: %v", err)
		return
	}

	h.mu.RLock()
	subscribers := h.subscriptions[sport]
	clientCount := len(subscribers)
	h.mu.RUnlock()

	if clientCount == 0 {
		return
	}

	h.metrics.RecordBroadcast(len(data), clientCount)

	// Send to all subscribers
	var failedClients []*Client

	h.mu.RLock()
	for client := range subscribers {
		select {
		case client.send <- data:
			// Sent successfully
		default:
			// Client's buffer is full - mark for removal
			failedClients = append(failedClients, client)
			h.metrics.RecordMessageFailed()
		}
	}
	h.mu.RUnlock()

	// Remove failed clients
	for _, client := range failedClients {
		log.Printf("WebSocket: Removing slow client")
		h.unregister <- client
	}

	log.Printf("WebSocket: Broadcast %s to %d clients (%d bytes)", sport, clientCount-len(failedClients), len(data))
}

// BroadcastStatus sends a status message to all clients
func (h *Hub) BroadcastStatus(status string) {
	message := Message{
		Type:      MessageTypeStatus,
		Status:    status,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			// Skip slow clients for status messages
		}
	}
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sportSubs := make(map[string]int)
	for sport, clients := range h.subscriptions {
		sportSubs[string(sport)] = len(clients)
	}

	return map[string]interface{}{
		"total_clients":  len(h.clients),
		"max_connections": h.maxConnections,
		"subscriptions":  sportSubs,
	}
}

// CanAccept returns whether the hub can accept new connections
func (h *Hub) CanAccept() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients) < h.maxConnections
}

// ClientCount returns the current number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
