package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joshuakim/linefinder/internal/models"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 1024

	// Send channel buffer size
	sendBufferSize = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin properly
		// For development, allow all origins
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Subscriptions this client has
	sports map[models.Sport]bool
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type  string `json:"type"`
	Sport string `json:"sport,omitempty"`
}

// NewClient creates a new client and starts its goroutines
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		sports: make(map[models.Sport]bool),
	}
}

// ServeWs handles WebSocket requests from the peer
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Check if we can accept more connections
	if !hub.CanAccept() {
		http.Error(w, "Server at capacity", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(hub, conn)
	hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
// It runs in its own goroutine and handles:
// - Reading client messages (subscribe/unsubscribe)
// - Ping/pong for connection health
// - Detecting disconnection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
// It runs in its own goroutine and handles:
// - Writing messages to client
// - Sending periodic pings
// - Respecting write deadlines
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch pending messages for efficiency
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming client messages
func (c *Client) handleMessage(data []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("WebSocket: Invalid message format: %v", err)
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case MessageTypeSubscribe:
		c.handleSubscribe(msg.Sport)
	case MessageTypeUnsubscribe:
		c.handleUnsubscribe(msg.Sport)
	case "ping":
		c.sendPong()
	default:
		log.Printf("WebSocket: Unknown message type: %s", msg.Type)
	}
}

func (c *Client) handleSubscribe(sportStr string) {
	sport := models.Sport(sportStr)
	if sport != models.SportNFL && sport != models.SportNBA {
		c.sendError("Invalid sport: use 'nfl' or 'nba'")
		return
	}

	// Unsubscribe from previous sports (one sport at a time for simplicity)
	for s := range c.sports {
		c.hub.Unsubscribe(c, s)
	}
	c.sports = make(map[models.Sport]bool)

	// Subscribe to new sport
	c.sports[sport] = true
	c.hub.Subscribe(c, sport)

	// Send confirmation
	c.sendStatus("subscribed to " + sportStr)
}

func (c *Client) handleUnsubscribe(sportStr string) {
	sport := models.Sport(sportStr)
	if c.sports[sport] {
		delete(c.sports, sport)
		c.hub.Unsubscribe(c, sport)
		c.sendStatus("unsubscribed from " + sportStr)
	}
}

func (c *Client) sendError(errMsg string) {
	msg := Message{
		Type:      MessageTypeError,
		Error:     errMsg,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
		// Buffer full, skip
	}
}

func (c *Client) sendStatus(status string) {
	msg := Message{
		Type:      MessageTypeStatus,
		Status:    status,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
		// Buffer full, skip
	}
}

func (c *Client) sendPong() {
	msg := Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
	}
}
