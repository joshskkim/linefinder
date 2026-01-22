package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/joshuakim/linefinder/internal/alerts"
	"github.com/joshuakim/linefinder/internal/database"
	"github.com/joshuakim/linefinder/internal/websocket"
)

// Config holds notification service configuration
type Config struct {
	// VAPID keys for Web Push
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string // mailto: or https:// URL

	// Batching
	BatchInterval time.Duration

	// Enable/disable
	Enabled bool
}

// DefaultConfig returns default notification configuration
func DefaultConfig() Config {
	return Config{
		BatchInterval: 60 * time.Second,
		Enabled:       true,
	}
}

// Service handles notification dispatch
type Service struct {
	config Config
	db     *database.DB
	hub    *websocket.Hub

	// Pending alerts for batching
	mu            sync.Mutex
	pendingAlerts []alerts.ValueAlert

	// Control
	stopCh chan struct{}
}

// NewService creates a new notification service
func NewService(config Config, db *database.DB, hub *websocket.Hub) *Service {
	return &Service{
		config:        config,
		db:            db,
		hub:           hub,
		pendingAlerts: make([]alerts.ValueAlert, 0),
		stopCh:        make(chan struct{}),
	}
}

// Start starts the batch processing loop
func (s *Service) Start(ctx context.Context) {
	if s.config.BatchInterval <= 0 {
		s.config.BatchInterval = 60 * time.Second
	}

	ticker := time.NewTicker(s.config.BatchInterval)
	defer ticker.Stop()

	log.Printf("Notification service started (batch interval: %v)", s.config.BatchInterval)

	for {
		select {
		case <-ctx.Done():
			s.processBatch() // Process any remaining alerts
			log.Println("Notification service stopped")
			return
		case <-s.stopCh:
			s.processBatch()
			return
		case <-ticker.C:
			s.processBatch()
		}
	}
}

// Stop stops the notification service
func (s *Service) Stop() {
	close(s.stopCh)
}

// QueueAlert adds an alert to the pending batch
func (s *Service) QueueAlert(alert alerts.ValueAlert) {
	if !s.config.Enabled {
		return
	}

	s.mu.Lock()
	s.pendingAlerts = append(s.pendingAlerts, alert)
	s.mu.Unlock()

	log.Printf("Alert queued: %s %s %s", alert.PlayerName, alert.PropCategory, alert.Direction)

	// Send immediately via WebSocket
	s.sendWebSocket(alert)
}

// QueueAlerts adds multiple alerts to the pending batch
func (s *Service) QueueAlerts(alertsList []alerts.ValueAlert) {
	for _, alert := range alertsList {
		s.QueueAlert(alert)
	}
}

// processBatch processes pending alerts and sends push notification
func (s *Service) processBatch() {
	s.mu.Lock()
	if len(s.pendingAlerts) == 0 {
		s.mu.Unlock()
		return
	}

	// Take the pending alerts
	batch := s.pendingAlerts
	s.pendingAlerts = make([]alerts.ValueAlert, 0)
	s.mu.Unlock()

	// Check if we're in quiet hours
	if s.isQuietHours() {
		log.Printf("Quiet hours - skipping push for %d alerts", len(batch))
		return
	}

	// Check rate limit
	if !s.checkRateLimit("push") {
		log.Printf("Rate limit exceeded - skipping push for %d alerts", len(batch))
		return
	}

	// Send push notification
	if err := s.sendPush(batch); err != nil {
		log.Printf("Failed to send push notification: %v", err)
	}
}

// sendWebSocket sends an alert via WebSocket
func (s *Service) sendWebSocket(alert alerts.ValueAlert) {
	if s.hub == nil {
		return
	}

	prefs, err := s.db.GetPreferences()
	if err != nil || !prefs.EnableWebsocket {
		return
	}

	// Create WebSocket message
	msg := websocket.Message{
		Type:      "value_alert",
		Timestamp: time.Now(),
	}

	// Marshal alert data
	alertData, _ := json.Marshal(alert)
	msg.Status = string(alertData) // Using Status field to carry alert data

	// Broadcast to all connected clients
	// Since we have single user, broadcast to all sports
	s.hub.BroadcastStatus(fmt.Sprintf("value_alert:%s", string(alertData)))
}

// sendPush sends a batched push notification
func (s *Service) sendPush(batch []alerts.ValueAlert) error {
	if s.config.VAPIDPrivateKey == "" || s.config.VAPIDPublicKey == "" {
		log.Println("VAPID keys not configured - skipping push")
		return nil
	}

	prefs, err := s.db.GetPreferences()
	if err != nil {
		return fmt.Errorf("failed to get preferences: %w", err)
	}

	if !prefs.EnablePush || prefs.PushSubscription == "" {
		return nil
	}

	// Create notification payload
	payload := PushPayload{
		Title: s.formatTitle(batch),
		Body:  s.formatBody(batch),
		Icon:  "/icon-192.png",
		Badge: "/badge-72.png",
		Tag:   "value-alerts",
		Data: PushData{
			URL:    "/",
			Alerts: batch,
			Count:  len(batch),
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Parse subscription
	sub := &webpush.Subscription{}
	if err := json.Unmarshal([]byte(prefs.PushSubscription), sub); err != nil {
		return fmt.Errorf("failed to parse subscription: %w", err)
	}

	// Send push notification
	resp, err := webpush.SendNotification(payloadJSON, sub, &webpush.Options{
		Subscriber:      s.config.VAPIDSubject,
		VAPIDPublicKey:  s.config.VAPIDPublicKey,
		VAPIDPrivateKey: s.config.VAPIDPrivateKey,
		TTL:             3600, // 1 hour
	})
	if err != nil {
		return fmt.Errorf("failed to send push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Subscription might be invalid
		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			log.Println("Push subscription expired/invalid - disabling")
			s.db.UpdatePreferences(&database.Preferences{
				EnablePush:       false,
				PushSubscription: "",
			})
		}
		return fmt.Errorf("push failed with status %d", resp.StatusCode)
	}

	// Increment rate limit
	s.db.IncrementRateLimit("push")

	log.Printf("Push notification sent: %d alerts", len(batch))
	return nil
}

// formatTitle creates the push notification title
func (s *Service) formatTitle(batch []alerts.ValueAlert) string {
	if len(batch) == 1 {
		a := batch[0]
		return fmt.Sprintf("Value Alert: %s %s", a.PlayerName, a.PropCategory)
	}

	highCount := 0
	for _, a := range batch {
		if a.Confidence == alerts.ConfidenceHigh {
			highCount++
		}
	}

	if highCount > 0 {
		return fmt.Sprintf("%d Value Alerts (%d High Confidence)", len(batch), highCount)
	}
	return fmt.Sprintf("%d Value Alerts", len(batch))
}

// formatBody creates the push notification body
func (s *Service) formatBody(batch []alerts.ValueAlert) string {
	if len(batch) == 1 {
		a := batch[0]
		dir := "OVER"
		if a.Direction == alerts.DirectionUnder {
			dir = "UNDER"
		}
		return fmt.Sprintf("%s %.1f (avg %.1f, diff %.1f). Best: %+.0f @ %s",
			dir, a.Line, a.Average, a.AbsDifference, a.BestOdds, a.Bookmaker)
	}

	// Summary for multiple alerts
	lines := make([]string, 0, 3)
	for i, a := range batch {
		if i >= 3 {
			break
		}
		dir := "O"
		if a.Direction == alerts.DirectionUnder {
			dir = "U"
		}
		lines = append(lines, fmt.Sprintf("%s %s %.1f (%s)", a.PlayerName, a.PropCategory, a.Line, dir))
	}

	body := ""
	for i, line := range lines {
		if i > 0 {
			body += " | "
		}
		body += line
	}

	if len(batch) > 3 {
		body += fmt.Sprintf(" +%d more", len(batch)-3)
	}

	return body
}

// isQuietHours checks if current time is within quiet hours
func (s *Service) isQuietHours() bool {
	prefs, err := s.db.GetPreferences()
	if err != nil {
		return false
	}

	loc, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	currentMinutes := now.Hour()*60 + now.Minute()

	// Parse quiet start
	startHour, startMin := 23, 0
	fmt.Sscanf(prefs.QuietStart, "%d:%d", &startHour, &startMin)
	startMinutes := startHour*60 + startMin

	// Parse quiet end
	endHour, endMin := 8, 0
	fmt.Sscanf(prefs.QuietEnd, "%d:%d", &endHour, &endMin)
	endMinutes := endHour*60 + endMin

	// Handle overnight quiet hours (e.g., 23:00 - 08:00)
	if startMinutes > endMinutes {
		// Quiet hours span midnight
		return currentMinutes >= startMinutes || currentMinutes < endMinutes
	}

	// Normal case (e.g., 02:00 - 06:00)
	return currentMinutes >= startMinutes && currentMinutes < endMinutes
}

// checkRateLimit checks if we can send on a channel
func (s *Service) checkRateLimit(channel string) bool {
	prefs, err := s.db.GetPreferences()
	if err != nil {
		return true
	}

	limit := prefs.RateLimitPush
	canSend, remaining, err := s.db.CheckRateLimit(channel, limit)
	if err != nil {
		log.Printf("Rate limit check error: %v", err)
		return true
	}

	if !canSend {
		log.Printf("Rate limit exceeded for %s (0 remaining)", channel)
	} else {
		log.Printf("Rate limit OK for %s (%d remaining)", channel, remaining)
	}

	return canSend
}

// GetVAPIDPublicKey returns the public key for client subscription
func (s *Service) GetVAPIDPublicKey() string {
	return s.config.VAPIDPublicKey
}

// PushPayload represents the push notification payload
type PushPayload struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Icon  string   `json:"icon,omitempty"`
	Badge string   `json:"badge,omitempty"`
	Tag   string   `json:"tag,omitempty"`
	Data  PushData `json:"data,omitempty"`
}

// PushData represents custom data in push notification
type PushData struct {
	URL    string              `json:"url,omitempty"`
	Alerts []alerts.ValueAlert `json:"alerts,omitempty"`
	Count  int                 `json:"count"`
}
