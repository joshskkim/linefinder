package metrics

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks system health and performance metrics
type Metrics struct {
	// Polling metrics
	PollCount          atomic.Int64 // Total polls executed
	PollSuccessCount   atomic.Int64 // Successful polls
	PollErrorCount     atomic.Int64 // Failed polls
	LastPollTime       atomic.Value // time.Time of last poll
	LastPollDuration   atomic.Int64 // Duration in milliseconds
	LastPollError      atomic.Value // Last error message (string)
	ConsecutiveErrors  atomic.Int64 // Consecutive poll failures

	// WebSocket metrics
	ConnectionsTotal   atomic.Int64 // Total connections ever made
	ConnectionsCurrent atomic.Int64 // Current active connections
	ConnectionsPeak    atomic.Int64 // Peak concurrent connections
	MessagesOut        atomic.Int64 // Messages sent to clients
	MessagesFailed     atomic.Int64 // Failed message sends
	BytesOut           atomic.Int64 // Total bytes sent

	// Change detection metrics
	ChangesDetected    atomic.Int64 // Number of times odds changed
	BroadcastCount     atomic.Int64 // Number of broadcasts sent
	LastChangeTime     atomic.Value // time.Time of last detected change

	// API usage tracking
	APIRequestsToday   atomic.Int64 // Requests made today
	APIRequestsTotal   atomic.Int64 // Total requests ever
	APIQuotaLimit      int64        // Daily quota limit
	APIQuotaResetTime  atomic.Value // time.Time when quota resets

	// System health
	StartTime          time.Time
	mu                 sync.RWMutex
	sportMetrics       map[string]*SportMetrics
}

// SportMetrics tracks per-sport metrics
type SportMetrics struct {
	Sport            string    `json:"sport"`
	LastPollTime     time.Time `json:"last_poll_time"`
	LastChangeTime   time.Time `json:"last_change_time"`
	GamesTracked     int       `json:"games_tracked"`
	PollCount        int64     `json:"poll_count"`
	ChangeCount      int64     `json:"change_count"`
	SubscriberCount  int64     `json:"subscriber_count"`
}

// New creates a new Metrics instance
func New() *Metrics {
	m := &Metrics{
		StartTime:    time.Now(),
		sportMetrics: make(map[string]*SportMetrics),
	}
	m.LastPollTime.Store(time.Time{})
	m.LastChangeTime.Store(time.Time{})
	m.LastPollError.Store("")
	m.APIQuotaResetTime.Store(time.Now().Add(24 * time.Hour))
	return m
}

// RecordPollStart records the start of a poll
func (m *Metrics) RecordPollStart() time.Time {
	return time.Now()
}

// RecordPollSuccess records a successful poll
func (m *Metrics) RecordPollSuccess(start time.Time, sport string, gamesCount int) {
	duration := time.Since(start)

	m.PollCount.Add(1)
	m.PollSuccessCount.Add(1)
	m.LastPollTime.Store(time.Now())
	m.LastPollDuration.Store(duration.Milliseconds())
	m.ConsecutiveErrors.Store(0)
	m.LastPollError.Store("")
	m.APIRequestsToday.Add(1)
	m.APIRequestsTotal.Add(1)

	m.mu.Lock()
	if m.sportMetrics[sport] == nil {
		m.sportMetrics[sport] = &SportMetrics{Sport: sport}
	}
	m.sportMetrics[sport].LastPollTime = time.Now()
	m.sportMetrics[sport].GamesTracked = gamesCount
	m.sportMetrics[sport].PollCount++
	m.mu.Unlock()
}

// RecordPollError records a failed poll
func (m *Metrics) RecordPollError(start time.Time, err error) {
	m.PollCount.Add(1)
	m.PollErrorCount.Add(1)
	m.LastPollTime.Store(time.Now())
	m.LastPollDuration.Store(time.Since(start).Milliseconds())
	m.ConsecutiveErrors.Add(1)
	m.LastPollError.Store(err.Error())
}

// RecordChange records when odds changes are detected
func (m *Metrics) RecordChange(sport string) {
	m.ChangesDetected.Add(1)
	m.LastChangeTime.Store(time.Now())

	m.mu.Lock()
	if m.sportMetrics[sport] != nil {
		m.sportMetrics[sport].LastChangeTime = time.Now()
		m.sportMetrics[sport].ChangeCount++
	}
	m.mu.Unlock()
}

// RecordBroadcast records a broadcast to clients
func (m *Metrics) RecordBroadcast(messageSize int, clientCount int) {
	m.BroadcastCount.Add(1)
	m.MessagesOut.Add(int64(clientCount))
	m.BytesOut.Add(int64(messageSize * clientCount))
}

// RecordMessageFailed records a failed message send
func (m *Metrics) RecordMessageFailed() {
	m.MessagesFailed.Add(1)
}

// RecordConnection records a new WebSocket connection
func (m *Metrics) RecordConnection() {
	m.ConnectionsTotal.Add(1)
	current := m.ConnectionsCurrent.Add(1)

	// Update peak if necessary
	for {
		peak := m.ConnectionsPeak.Load()
		if current <= peak {
			break
		}
		if m.ConnectionsPeak.CompareAndSwap(peak, current) {
			break
		}
	}
}

// RecordDisconnection records a WebSocket disconnection
func (m *Metrics) RecordDisconnection() {
	m.ConnectionsCurrent.Add(-1)
}

// UpdateSubscriberCount updates subscriber count for a sport
func (m *Metrics) UpdateSubscriberCount(sport string, count int64) {
	m.mu.Lock()
	if m.sportMetrics[sport] == nil {
		m.sportMetrics[sport] = &SportMetrics{Sport: sport}
	}
	m.sportMetrics[sport].SubscriberCount = count
	m.mu.Unlock()
}

// ResetDailyQuota resets daily API quota counter
func (m *Metrics) ResetDailyQuota() {
	m.APIRequestsToday.Store(0)
	m.APIQuotaResetTime.Store(time.Now().Add(24 * time.Hour))
}

// HealthStatus represents the system health
type HealthStatus struct {
	Status             string                   `json:"status"` // "healthy", "degraded", "unhealthy"
	Uptime             string                   `json:"uptime"`
	UptimeSeconds      int64                    `json:"uptime_seconds"`
	Polling            PollingHealth            `json:"polling"`
	WebSocket          WebSocketHealth          `json:"websocket"`
	API                APIHealth                `json:"api"`
	Sports             map[string]*SportMetrics `json:"sports"`
	Warnings           []string                 `json:"warnings,omitempty"`
}

type PollingHealth struct {
	Enabled            bool      `json:"enabled"`
	TotalPolls         int64     `json:"total_polls"`
	SuccessfulPolls    int64     `json:"successful_polls"`
	FailedPolls        int64     `json:"failed_polls"`
	SuccessRate        float64   `json:"success_rate_percent"`
	LastPollTime       time.Time `json:"last_poll_time"`
	LastPollAgo        string    `json:"last_poll_ago"`
	LastPollDurationMs int64     `json:"last_poll_duration_ms"`
	ConsecutiveErrors  int64     `json:"consecutive_errors"`
	LastError          string    `json:"last_error,omitempty"`
	ChangesDetected    int64     `json:"changes_detected"`
	LastChangeTime     time.Time `json:"last_change_time,omitempty"`
	LastChangeAgo      string    `json:"last_change_ago,omitempty"`
}

type WebSocketHealth struct {
	CurrentConnections int64   `json:"current_connections"`
	PeakConnections    int64   `json:"peak_connections"`
	TotalConnections   int64   `json:"total_connections"`
	MessagesSent       int64   `json:"messages_sent"`
	MessagesFailed     int64   `json:"messages_failed"`
	DeliveryRate       float64 `json:"delivery_rate_percent"`
	BytesSent          int64   `json:"bytes_sent"`
	BroadcastCount     int64   `json:"broadcast_count"`
}

type APIHealth struct {
	RequestsToday  int64     `json:"requests_today"`
	RequestsTotal  int64     `json:"requests_total"`
	QuotaLimit     int64     `json:"quota_limit"`
	QuotaRemaining int64     `json:"quota_remaining"`
	QuotaUsedPct   float64   `json:"quota_used_percent"`
	QuotaResetTime time.Time `json:"quota_reset_time"`
}

// GetHealth returns current health status
func (m *Metrics) GetHealth(pollingEnabled bool) HealthStatus {
	uptime := time.Since(m.StartTime)

	totalPolls := m.PollCount.Load()
	successPolls := m.PollSuccessCount.Load()
	failedPolls := m.PollErrorCount.Load()

	var successRate float64
	if totalPolls > 0 {
		successRate = float64(successPolls) / float64(totalPolls) * 100
	}

	messagesSent := m.MessagesOut.Load()
	messagesFailed := m.MessagesFailed.Load()
	var deliveryRate float64
	if messagesSent+messagesFailed > 0 {
		deliveryRate = float64(messagesSent) / float64(messagesSent+messagesFailed) * 100
	}

	lastPollTime := m.LastPollTime.Load().(time.Time)
	lastChangeTime := m.LastChangeTime.Load().(time.Time)
	lastPollError := m.LastPollError.Load().(string)
	quotaResetTime := m.APIQuotaResetTime.Load().(time.Time)

	requestsToday := m.APIRequestsToday.Load()
	quotaRemaining := m.APIQuotaLimit - requestsToday
	if quotaRemaining < 0 {
		quotaRemaining = 0
	}

	var quotaUsedPct float64
	if m.APIQuotaLimit > 0 {
		quotaUsedPct = float64(requestsToday) / float64(m.APIQuotaLimit) * 100
	}

	// Determine overall health status
	status := "healthy"
	var warnings []string

	consecutiveErrors := m.ConsecutiveErrors.Load()
	if consecutiveErrors >= 5 {
		status = "unhealthy"
		warnings = append(warnings, "High consecutive poll errors")
	} else if consecutiveErrors >= 3 {
		status = "degraded"
		warnings = append(warnings, "Multiple consecutive poll errors")
	}

	if pollingEnabled && !lastPollTime.IsZero() && time.Since(lastPollTime) > 5*time.Minute {
		status = "degraded"
		warnings = append(warnings, "Polling appears stale (>5 min since last poll)")
	}

	if quotaUsedPct > 90 {
		warnings = append(warnings, "API quota nearly exhausted (>90%)")
		if status == "healthy" {
			status = "degraded"
		}
	}

	if deliveryRate < 95 && messagesSent > 100 {
		warnings = append(warnings, "Message delivery rate below 95%")
	}

	// Build sport metrics snapshot
	m.mu.RLock()
	sports := make(map[string]*SportMetrics)
	for k, v := range m.sportMetrics {
		sportCopy := *v
		sports[k] = &sportCopy
	}
	m.mu.RUnlock()

	var lastPollAgo, lastChangeAgo string
	if !lastPollTime.IsZero() {
		lastPollAgo = time.Since(lastPollTime).Round(time.Second).String()
	}
	if !lastChangeTime.IsZero() {
		lastChangeAgo = time.Since(lastChangeTime).Round(time.Second).String()
	}

	return HealthStatus{
		Status:        status,
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: int64(uptime.Seconds()),
		Polling: PollingHealth{
			Enabled:            pollingEnabled,
			TotalPolls:         totalPolls,
			SuccessfulPolls:    successPolls,
			FailedPolls:        failedPolls,
			SuccessRate:        successRate,
			LastPollTime:       lastPollTime,
			LastPollAgo:        lastPollAgo,
			LastPollDurationMs: m.LastPollDuration.Load(),
			ConsecutiveErrors:  consecutiveErrors,
			LastError:          lastPollError,
			ChangesDetected:    m.ChangesDetected.Load(),
			LastChangeTime:     lastChangeTime,
			LastChangeAgo:      lastChangeAgo,
		},
		WebSocket: WebSocketHealth{
			CurrentConnections: m.ConnectionsCurrent.Load(),
			PeakConnections:    m.ConnectionsPeak.Load(),
			TotalConnections:   m.ConnectionsTotal.Load(),
			MessagesSent:       messagesSent,
			MessagesFailed:     messagesFailed,
			DeliveryRate:       deliveryRate,
			BytesSent:          m.BytesOut.Load(),
			BroadcastCount:     m.BroadcastCount.Load(),
		},
		API: APIHealth{
			RequestsToday:  requestsToday,
			RequestsTotal:  m.APIRequestsTotal.Load(),
			QuotaLimit:     m.APIQuotaLimit,
			QuotaRemaining: quotaRemaining,
			QuotaUsedPct:   quotaUsedPct,
			QuotaResetTime: quotaResetTime,
		},
		Sports:   sports,
		Warnings: warnings,
	}
}

// JSON returns metrics as JSON
func (m *Metrics) JSON(pollingEnabled bool) ([]byte, error) {
	return json.Marshal(m.GetHealth(pollingEnabled))
}
