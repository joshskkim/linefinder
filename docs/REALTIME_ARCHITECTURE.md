# Real-time Odds Polling Architecture

## Overview

This document outlines the architecture for adding real-time odds updates to LineFinder. The goal is to keep odds data fresh without requiring users to manually click "Refresh".

---

## Architecture Options

### Option 1: Client-Side Polling (Simple)

```
┌─────────┐    every 30s     ┌─────────┐    on-demand    ┌──────────┐
│ Browser │ ───────────────► │ Go API  │ ──────────────► │ Odds API │
└─────────┘                  └─────────┘                 └──────────┘
```

**How it works:**
- React frontend sets up a `setInterval` to call `/api/odds/{sport}` every N seconds
- Each request triggers a fresh fetch from the Odds API

**Pros:**
- Dead simple to implement (5 lines of frontend code)
- No backend changes needed

**Cons:**
- Wasteful: 10 users = 10x API calls for identical data
- API credits burn fast
- Not scalable

---

### Option 2: Server-Side Polling + WebSocket Push (Recommended)

```
                                    ┌──────────┐
                         poll       │          │
                    ┌──────────────►│ Odds API │
                    │   every 60s   │          │
                    │               └──────────┘
                    │
              ┌─────┴─────┐
              │  Polling  │
              │  Service  │
              └─────┬─────┘
                    │ new data?
                    ▼
              ┌───────────┐
              │  Broadcast │
              │   Hub      │
              └─────┬─────┘
                    │ push via WebSocket
        ┌───────────┼───────────┐
        ▼           ▼           ▼
    ┌───────┐   ┌───────┐   ┌───────┐
    │Client1│   │Client2│   │Client3│
    └───────┘   └───────┘   └───────┘
```

**How it works:**
1. A background goroutine polls the Odds API on a fixed interval (e.g., 60 seconds)
2. When new data arrives, it compares with previous data to detect changes
3. If changes detected, broadcast to all connected WebSocket clients
4. Clients receive push updates and update their UI

**Pros:**
- Efficient: 1 API call serves ALL connected users
- Truly real-time feel
- Scalable

**Cons:**
- More complex implementation
- Need to manage WebSocket connections
- Need reconnection logic on frontend

---

### Option 3: Server-Sent Events (SSE) - Simpler Alternative to WebSocket

```
Browser ◄──────── SSE (one-way) ──────── Server
```

**How it works:**
- Similar to WebSocket but uses standard HTTP
- Server pushes events to client over a long-lived connection
- Client cannot send messages back (which is fine - we only need server→client)

**Pros:**
- Simpler than WebSocket (built on HTTP, auto-reconnect in browsers)
- No special protocol handling
- Works through most proxies/firewalls

**Cons:**
- One-way only (fine for our use case)
- Limited browser connection pool (6 connections per domain)

---

## Recommended Approach: Option 2 (WebSocket)

WebSocket gives us bidirectional capability if we need it later (e.g., user subscribes to specific games).

---

## Implementation Plan

### Phase 1: Backend Polling Service

**New file: `internal/polling/service.go`**

```go
package polling

import (
    "context"
    "log"
    "sync"
    "time"
)

type PollingService struct {
    oddsService  *service.OddsService
    hub          *Hub           // WebSocket hub
    interval     time.Duration
    sports       []models.Sport

    mu           sync.RWMutex
    lastData     map[models.Sport][]models.Game  // Cache for change detection
    lastFetch    time.Time
}

// Start begins the polling loop
func (p *PollingService) Start(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Println("Polling service stopped")
            return
        case <-ticker.C:
            p.pollAll()
        }
    }
}

func (p *PollingService) pollAll() {
    for _, sport := range p.sports {
        games, err := p.oddsService.FetchAndStoreOdds(sport)
        if err != nil {
            log.Printf("Poll error for %s: %v", sport, err)
            continue
        }

        // Check if data changed
        if p.hasChanges(sport, games) {
            p.hub.Broadcast(sport, games)
            p.updateCache(sport, games)
        }
    }

    p.lastFetch = time.Now()
}

func (p *PollingService) hasChanges(sport models.Sport, newGames []models.Game) bool {
    // Compare checksums, or specific fields like odds values
    // This prevents unnecessary broadcasts when data hasn't changed
}
```

**Key concepts:**
- `context.Context` allows graceful shutdown
- `ticker` fires at regular intervals
- `lastData` cache enables change detection
- `hub.Broadcast` pushes to all connected clients

---

### Phase 2: WebSocket Hub (Connection Manager)

**New file: `internal/websocket/hub.go`**

```go
package websocket

import (
    "sync"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
    // Registered clients by sport they're watching
    clients map[*Client]map[models.Sport]bool

    // Register requests from clients
    register chan *Client

    // Unregister requests from clients
    unregister chan *Client

    // Mutex for thread-safe client access
    mu sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]map[models.Sport]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

// Run starts the hub's main loop
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = make(map[models.Sport]bool)
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
        }
    }
}

// Broadcast sends a message to all clients watching a sport
func (h *Hub) Broadcast(sport models.Sport, games []models.Game) {
    message := Message{
        Type:  "odds_update",
        Sport: sport,
        Games: games,
        Time:  time.Now(),
    }

    h.mu.RLock()
    defer h.mu.RUnlock()

    for client, sports := range h.clients {
        if sports[sport] {
            select {
            case client.send <- message:
            default:
                // Client's buffer is full, they're too slow
                // Close connection to prevent memory buildup
                close(client.send)
                delete(h.clients, client)
            }
        }
    }
}
```

**Key concepts:**
- `Hub` is the central coordinator for all WebSocket connections
- Clients register/unregister through channels (thread-safe)
- `Broadcast` sends to all clients watching a specific sport
- The `select default` pattern prevents slow clients from blocking

---

### Phase 3: WebSocket Client Handler

**New file: `internal/websocket/client.go`**

```go
package websocket

import (
    "time"

    "github.com/gorilla/websocket"
)

const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second

    // Send pings to peer with this period (must be less than pongWait)
    pingPeriod = (pongWait * 9) / 10

    // Maximum message size allowed from peer
    maxMessageSize = 512
)

type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan Message
}

// readPump pumps messages from the WebSocket connection to the hub
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
            break
        }
        // Handle incoming messages (e.g., subscribe to sport)
        c.handleMessage(message)
    }
}

// writePump pumps messages from the hub to the WebSocket connection
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

            if err := c.conn.WriteJSON(message); err != nil {
                return
            }

        case <-ticker.C:
            // Send ping to keep connection alive
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

**Key concepts:**
- Two goroutines per client: `readPump` (receive) and `writePump` (send)
- Ping/pong keeps connections alive and detects dead clients
- Timeouts prevent resource leaks from zombie connections

---

### Phase 4: HTTP Handler for WebSocket Upgrade

**Add to `internal/api/handlers.go`:**

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // In production, validate origin properly
        return true
    },
}

// handleWebSocket upgrades HTTP to WebSocket
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("WebSocket upgrade failed:", err)
        return
    }

    client := &websocket.Client{
        hub:  h.hub,
        conn: conn,
        send: make(chan Message, 256),
    }

    h.hub.register <- client

    // Start client goroutines
    go client.writePump()
    go client.readPump()
}
```

---

### Phase 5: Frontend WebSocket Client

**New file: `web/src/hooks/useOddsWebSocket.js`**

```javascript
import { useEffect, useRef, useCallback, useState } from 'react'

export function useOddsWebSocket(sport, onUpdate) {
  const ws = useRef(null)
  const [connected, setConnected] = useState(false)
  const [lastUpdate, setLastUpdate] = useState(null)
  const reconnectAttempts = useRef(0)
  const maxReconnectAttempts = 5

  const connect = useCallback(() => {
    // Use wss:// in production, ws:// in development
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/ws`

    ws.current = new WebSocket(wsUrl)

    ws.current.onopen = () => {
      console.log('WebSocket connected')
      setConnected(true)
      reconnectAttempts.current = 0

      // Subscribe to the sport we're watching
      ws.current.send(JSON.stringify({
        type: 'subscribe',
        sport: sport
      }))
    }

    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data)

      if (data.type === 'odds_update' && data.sport === sport) {
        setLastUpdate(new Date(data.time))
        onUpdate(data.games)
      }
    }

    ws.current.onclose = () => {
      console.log('WebSocket disconnected')
      setConnected(false)

      // Attempt reconnect with exponential backoff
      if (reconnectAttempts.current < maxReconnectAttempts) {
        const delay = Math.pow(2, reconnectAttempts.current) * 1000
        reconnectAttempts.current++
        console.log(`Reconnecting in ${delay}ms...`)
        setTimeout(connect, delay)
      }
    }

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }, [sport, onUpdate])

  useEffect(() => {
    connect()

    return () => {
      if (ws.current) {
        ws.current.close()
      }
    }
  }, [connect])

  return { connected, lastUpdate }
}
```

**Usage in App.jsx:**

```javascript
import { useOddsWebSocket } from './hooks/useOddsWebSocket'

function App() {
  const [games, setGames] = useState([])

  const handleOddsUpdate = useCallback((newGames) => {
    setGames(newGames)
  }, [])

  const { connected, lastUpdate } = useOddsWebSocket(
    selectedSport,
    handleOddsUpdate
  )

  return (
    <div className="app">
      <header className="header">
        <div className="connection-status">
          {connected ? '🟢 Live' : '🔴 Disconnected'}
          {lastUpdate && <span>Updated: {lastUpdate.toLocaleTimeString()}</span>}
        </div>
        {/* ... rest of header */}
      </header>
      {/* ... */}
    </div>
  )
}
```

---

## Bottlenecks & Mitigation

### 1. API Rate Limits

**Problem:** Odds API has request limits (e.g., 500 requests/month on free tier)

**Mitigation:**
```go
type RateLimiter struct {
    requests    int
    maxRequests int
    resetTime   time.Time
}

func (r *RateLimiter) CanRequest() bool {
    if time.Now().After(r.resetTime) {
        r.requests = 0
        r.resetTime = time.Now().Add(24 * time.Hour)
    }
    return r.requests < r.maxRequests
}
```

Also consider:
- Adaptive polling: Poll more frequently during game times, less during off-hours
- Request budgeting: Reserve some requests for manual refreshes

### 2. Memory Usage from Many Connections

**Problem:** Each WebSocket connection consumes memory (~10-50KB per connection)

**Mitigation:**
- Set maximum connections limit
- Use connection pooling
- Monitor memory and shed connections if needed

```go
const maxConnections = 10000

func (h *Hub) canAccept() bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients) < maxConnections
}
```

### 3. Bandwidth for Large Payloads

**Problem:** Sending full game data on every update is wasteful

**Mitigation:** Send diffs instead of full data

```go
type OddsDiff struct {
    GameID     string             `json:"game_id"`
    Changes    map[string]Change  `json:"changes"`  // bookmaker -> change
}

type Change struct {
    Field    string  `json:"field"`   // "spread", "moneyline", "total"
    OldValue float64 `json:"old"`
    NewValue float64 `json:"new"`
}
```

---

## Points of Failure & Handling

### 1. Odds API Downtime

**Problem:** API is unavailable

**Handling:**
```go
func (p *PollingService) pollWithRetry(sport models.Sport) error {
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        games, err := p.oddsService.FetchAndStoreOdds(sport)
        if err == nil {
            return nil
        }
        lastErr = err
        time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
    }

    // Notify clients that data may be stale
    p.hub.BroadcastStatus(sport, StatusStale)
    return lastErr
}
```

Frontend should show stale indicator:
```javascript
{isStale && <div className="stale-warning">Data may be outdated</div>}
```

### 2. Client Disconnects

**Problem:** Network issues, browser tab closed, etc.

**Handling:**
- Ping/pong mechanism detects dead connections
- Exponential backoff reconnection on frontend
- Show connection status to user

### 3. Server Restart

**Problem:** All WebSocket connections drop

**Handling:**
- Frontend auto-reconnects (already built into the hook)
- On reconnect, client gets fresh data immediately
- Consider sticky sessions if load balancing

### 4. Slow Clients

**Problem:** Client can't keep up with updates, channel fills up

**Handling:**
```go
select {
case client.send <- message:
    // Sent successfully
default:
    // Buffer full - client is too slow
    // Close and let them reconnect
    close(client.send)
    delete(h.clients, client)
}
```

### 5. Data Inconsistency

**Problem:** Race conditions between HTTP requests and WebSocket updates

**Handling:**
- Include version/timestamp with each update
- Frontend ignores updates older than current data

```javascript
const handleOddsUpdate = (newGames, timestamp) => {
  if (timestamp > lastUpdateTime) {
    setGames(newGames)
    setLastUpdateTime(timestamp)
  }
}
```

---

## Configuration Options

Add to your environment/config:

```env
# Polling configuration
POLL_INTERVAL_SECONDS=60        # How often to poll Odds API
POLL_ENABLED=false              # Master switch for polling
POLL_SPORTS=nba,nfl             # Which sports to poll

# WebSocket configuration
WS_MAX_CONNECTIONS=1000         # Maximum concurrent connections
WS_PING_INTERVAL_SECONDS=30     # Keepalive ping interval
WS_WRITE_TIMEOUT_SECONDS=10     # Write deadline
```

---

## Files to Create/Modify

**New files:**
- `internal/polling/service.go` - Background polling goroutine
- `internal/websocket/hub.go` - Connection manager
- `internal/websocket/client.go` - Individual client handler
- `internal/websocket/message.go` - Message types
- `web/src/hooks/useOddsWebSocket.js` - React WebSocket hook

**Modified files:**
- `cmd/server/main.go` - Initialize hub and polling service
- `internal/api/handlers.go` - Add WebSocket upgrade endpoint
- `web/src/App.jsx` - Use WebSocket hook
- `web/src/index.css` - Connection status styles
- `go.mod` - Add `github.com/gorilla/websocket` dependency

---

## Testing Strategy

1. **Unit tests:** Test change detection logic
2. **Integration tests:** Test WebSocket connection/disconnection
3. **Load tests:** Simulate many concurrent connections
4. **Chaos tests:** Kill API, kill server, simulate network partitions

---

## Rollout Plan

1. **Phase 1:** Implement backend polling (disabled by default)
2. **Phase 2:** Add WebSocket infrastructure
3. **Phase 3:** Frontend integration with fallback to manual refresh
4. **Phase 4:** Enable polling in staging, monitor API usage
5. **Phase 5:** Gradual rollout to production

---

## Summary

The recommended approach uses **server-side polling + WebSocket push**:

- One API call serves all users (efficient)
- Real-time feel without manual refresh
- Change detection prevents unnecessary updates
- Graceful degradation when things fail

Start with a conservative 60-second poll interval and adjust based on API budget and user needs.
