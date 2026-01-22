package polling

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/joshuakim/linefinder/internal/alerts"
	"github.com/joshuakim/linefinder/internal/metrics"
	"github.com/joshuakim/linefinder/internal/models"
	"github.com/joshuakim/linefinder/internal/service"
	"github.com/joshuakim/linefinder/internal/store"
	"github.com/joshuakim/linefinder/internal/websocket"
)

// Config holds polling service configuration
type Config struct {
	// Enabled controls whether polling is active
	Enabled bool

	// Interval is the time between polls
	Interval time.Duration

	// Sports to poll
	Sports []models.Sport

	// MaxRetries before giving up on a poll cycle
	MaxRetries int

	// RetryBaseDelay is the base delay for exponential backoff
	RetryBaseDelay time.Duration

	// MaxConsecutiveErrors before entering recovery mode
	MaxConsecutiveErrors int

	// RecoveryInterval is the interval when in recovery mode
	RecoveryInterval time.Duration
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		Enabled:              false, // Off by default
		Interval:             60 * time.Second,
		Sports:               []models.Sport{models.SportNBA, models.SportNFL},
		MaxRetries:           3,
		RetryBaseDelay:       2 * time.Second,
		MaxConsecutiveErrors: 5,
		RecoveryInterval:     5 * time.Minute,
	}
}

// AlertCallback is called when value alerts are detected
type AlertCallback func(alerts []alerts.ValueAlert)

// Service handles periodic polling of the Odds API
type Service struct {
	config      Config
	oddsService *service.OddsService
	hub         *websocket.Hub
	metrics     *metrics.Metrics

	// Alert detection
	alertDetector *alerts.Detector
	alertCallback AlertCallback

	// State
	mu              sync.RWMutex
	enabled         bool
	inRecoveryMode  bool
	lastData        map[models.Sport]string // Hash of last data for change detection
	lastSuccessTime map[models.Sport]time.Time

	// Control channels
	stopCh   chan struct{}
	toggleCh chan bool
}

// NewService creates a new polling service
func NewService(config Config, oddsService *service.OddsService, hub *websocket.Hub, m *metrics.Metrics) *Service {
	return &Service{
		config:          config,
		oddsService:     oddsService,
		hub:             hub,
		metrics:         m,
		enabled:         config.Enabled,
		lastData:        make(map[models.Sport]string),
		lastSuccessTime: make(map[models.Sport]time.Time),
		stopCh:          make(chan struct{}),
		toggleCh:        make(chan bool, 1),
	}
}

// SetAlertDetector sets the alert detector for value detection during polling
func (s *Service) SetAlertDetector(detector *alerts.Detector, callback AlertCallback) {
	s.alertDetector = detector
	s.alertCallback = callback
}

// Start begins the polling loop
func (s *Service) Start(ctx context.Context) {
	log.Printf("Polling service starting (enabled: %v, interval: %v)", s.enabled, s.config.Interval)

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	// Do an immediate poll if enabled
	if s.enabled {
		s.pollAllSports()
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Polling service stopped (context cancelled)")
			return

		case <-s.stopCh:
			log.Println("Polling service stopped")
			return

		case enabled := <-s.toggleCh:
			s.handleToggle(enabled)

		case <-ticker.C:
			if s.IsEnabled() {
				s.pollAllSports()
				// Adjust ticker if in recovery mode
				s.adjustTickerIfNeeded(ticker)
			}
		}
	}
}

// Stop stops the polling service
func (s *Service) Stop() {
	close(s.stopCh)
}

// Enable turns polling on
func (s *Service) Enable() {
	select {
	case s.toggleCh <- true:
	default:
		// Channel full, toggle already pending
	}
}

// Disable turns polling off
func (s *Service) Disable() {
	select {
	case s.toggleCh <- false:
	default:
	}
}

// Toggle switches the polling state
func (s *Service) Toggle() {
	s.mu.RLock()
	currentState := s.enabled
	s.mu.RUnlock()

	select {
	case s.toggleCh <- !currentState:
	default:
	}
}

// IsEnabled returns whether polling is currently enabled
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// IsInRecoveryMode returns whether the service is in recovery mode
func (s *Service) IsInRecoveryMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inRecoveryMode
}

// GetStatus returns current service status
func (s *Service) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lastSuccess := make(map[string]string)
	for sport, t := range s.lastSuccessTime {
		if !t.IsZero() {
			lastSuccess[string(sport)] = time.Since(t).Round(time.Second).String() + " ago"
		}
	}

	return map[string]interface{}{
		"enabled":        s.enabled,
		"recovery_mode":  s.inRecoveryMode,
		"interval":       s.config.Interval.String(),
		"sports":         s.config.Sports,
		"last_success":   lastSuccess,
	}
}

func (s *Service) handleToggle(enabled bool) {
	s.mu.Lock()
	wasEnabled := s.enabled
	s.enabled = enabled
	s.mu.Unlock()

	if enabled && !wasEnabled {
		log.Println("Polling service ENABLED")
		// Do an immediate poll
		go s.pollAllSports()
	} else if !enabled && wasEnabled {
		log.Println("Polling service DISABLED")
	}
}

func (s *Service) adjustTickerIfNeeded(ticker *time.Ticker) {
	s.mu.RLock()
	inRecovery := s.inRecoveryMode
	s.mu.RUnlock()

	if inRecovery {
		ticker.Reset(s.config.RecoveryInterval)
	} else {
		ticker.Reset(s.config.Interval)
	}
}

func (s *Service) pollAllSports() {
	for _, sport := range s.config.Sports {
		s.pollSport(sport)
	}
}

func (s *Service) pollSport(sport models.Sport) {
	start := s.metrics.RecordPollStart()

	games, err := s.pollWithRetry(sport)
	if err != nil {
		s.metrics.RecordPollError(start, err)
		s.handlePollError(sport)
		return
	}

	s.metrics.RecordPollSuccess(start, string(sport), len(games))
	s.handlePollSuccess(sport)

	// Check for changes
	if s.hasChanges(sport, games) {
		log.Printf("Polling: Changes detected for %s, broadcasting to clients", sport)
		s.metrics.RecordChange(string(sport))
		s.hub.Broadcast(sport, games)
		s.updateCache(sport, games)

		// Check for value alerts on changed data
		if s.alertDetector != nil && s.alertCallback != nil {
			go s.checkValueAlerts(sport, games)
		}
	}
}

// checkValueAlerts scans games for value alerts and notifies via callback
func (s *Service) checkValueAlerts(sport models.Sport, games []models.Game) {
	sportStr := string(sport)
	var detectedAlerts []alerts.ValueAlert

	// Get player averages
	averages := store.GetDummyPlayerAverages(sportStr)
	avgMap := make(map[string]map[string]float64)
	for _, pa := range averages {
		avgMap[strings.ToLower(pa.Name)] = pa.Averages
	}

	// Check each game for value
	for _, game := range games {
		props := store.GetDummyPlayerProps(game.ID, sport, game.HomeTeam, game.AwayTeam)

		ctx := alerts.GameContext{
			GameID:   game.ID,
			Sport:    sportStr,
			HomeTeam: game.HomeTeam,
			AwayTeam: game.AwayTeam,
			GameTime: game.CommenceTime,
		}

		// Process each player's props
		for _, player := range props.Players {
			playerAvg := avgMap[strings.ToLower(player.Name)]
			if playerAvg == nil {
				continue
			}

			for _, prop := range player.Props {
				avg, ok := playerAvg[prop.Category]
				if !ok {
					continue
				}

				// Find best odds
				var bestLine float64
				var bestOdds float64
				var bestBook string
				for _, bm := range prop.Bookmakers {
					if bestBook == "" || bm.OverPrice > bestOdds {
						bestLine = bm.Point
						bestOdds = bm.OverPrice
						bestBook = bm.Title
					}
				}

				propData := alerts.PropData{
					PlayerName:   player.Name,
					Team:         player.Team,
					PropCategory: prop.Category,
					Line:         bestLine,
					Average:      avg,
					BestOdds:     bestOdds,
					Bookmaker:    bestBook,
				}

				alert := s.alertDetector.DetectValue(propData, ctx)
				if alert != nil {
					shouldNotify, _ := s.alertDetector.ShouldNotify(alert)
					if shouldNotify {
						s.alertDetector.RecordAlert(alert)
						detectedAlerts = append(detectedAlerts, *alert)
					}
				}
			}
		}
	}

	// Notify via callback if we found alerts
	if len(detectedAlerts) > 0 {
		log.Printf("Polling: Found %d value alerts for %s", len(detectedAlerts), sport)
		s.alertCallback(detectedAlerts)
	}
}

func (s *Service) pollWithRetry(sport models.Sport) ([]models.Game, error) {
	var lastErr error

	for attempt := 0; attempt < s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s...
			delay := s.config.RetryBaseDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("Polling: Retry %d for %s after %v", attempt, sport, delay)
			time.Sleep(delay)
		}

		games, err := s.oddsService.FetchAndStoreOdds(sport)
		if err == nil {
			return games, nil
		}

		lastErr = err
		log.Printf("Polling: Attempt %d failed for %s: %v", attempt+1, sport, err)
	}

	return nil, fmt.Errorf("all %d retries failed: %w", s.config.MaxRetries, lastErr)
}

func (s *Service) handlePollError(sport models.Sport) {
	consecutiveErrors := s.metrics.ConsecutiveErrors.Load()

	if consecutiveErrors >= int64(s.config.MaxConsecutiveErrors) {
		s.mu.Lock()
		if !s.inRecoveryMode {
			s.inRecoveryMode = true
			log.Printf("Polling: Entering RECOVERY MODE after %d consecutive errors", consecutiveErrors)
			s.hub.BroadcastStatus("polling_degraded")
		}
		s.mu.Unlock()
	}
}

func (s *Service) handlePollSuccess(sport models.Sport) {
	s.mu.Lock()
	s.lastSuccessTime[sport] = time.Now()

	// Exit recovery mode on success
	if s.inRecoveryMode {
		s.inRecoveryMode = false
		log.Println("Polling: Exiting recovery mode - poll successful")
		s.hub.BroadcastStatus("polling_healthy")
	}
	s.mu.Unlock()
}

// hasChanges checks if the data has changed since last poll
func (s *Service) hasChanges(sport models.Sport, games []models.Game) bool {
	newHash := s.hashGames(games)

	s.mu.RLock()
	oldHash := s.lastData[sport]
	s.mu.RUnlock()

	return newHash != oldHash
}

// updateCache stores the current data hash
func (s *Service) updateCache(sport models.Sport, games []models.Game) {
	s.mu.Lock()
	s.lastData[sport] = s.hashGames(games)
	s.mu.Unlock()
}

// hashGames creates a hash of the games data for change detection
// We hash the essential fields that matter for odds comparison
func (s *Service) hashGames(games []models.Game) string {
	// Extract only the fields that matter for change detection
	type outcomeSnap struct {
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Point float64 `json:"point"`
	}

	type marketSnap struct {
		Key      string        `json:"key"`
		Outcomes []outcomeSnap `json:"outcomes"`
	}

	type bookmakerSnap struct {
		Key     string       `json:"key"`
		Markets []marketSnap `json:"markets"`
	}

	type oddsSnapshot struct {
		GameID     string          `json:"game_id"`
		Bookmakers []bookmakerSnap `json:"bookmakers"`
	}

	snapshots := make([]oddsSnapshot, len(games))
	for i, game := range games {
		snap := oddsSnapshot{GameID: game.ID}
		for _, bm := range game.Bookmakers {
			bmSnap := bookmakerSnap{Key: bm.Key}

			for _, m := range bm.Markets {
				mSnap := marketSnap{Key: string(m.Key)}

				for _, o := range m.Outcomes {
					point := 0.0
					if o.Point != nil {
						point = *o.Point
					}
					mSnap.Outcomes = append(mSnap.Outcomes, outcomeSnap{
						Name:  o.Name,
						Price: o.Price,
						Point: point,
					})
				}
				bmSnap.Markets = append(bmSnap.Markets, mSnap)
			}
			snap.Bookmakers = append(snap.Bookmakers, bmSnap)
		}
		snapshots[i] = snap
	}

	data, _ := json.Marshal(snapshots)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ForceRefresh triggers an immediate poll regardless of timing
func (s *Service) ForceRefresh(sport models.Sport) error {
	if !s.IsEnabled() {
		return fmt.Errorf("polling is disabled")
	}

	log.Printf("Polling: Force refresh requested for %s", sport)
	start := s.metrics.RecordPollStart()

	games, err := s.pollWithRetry(sport)
	if err != nil {
		s.metrics.RecordPollError(start, err)
		return err
	}

	s.metrics.RecordPollSuccess(start, string(sport), len(games))
	s.handlePollSuccess(sport)

	// Always broadcast on force refresh
	s.metrics.RecordChange(string(sport))
	s.hub.Broadcast(sport, games)
	s.updateCache(sport, games)

	return nil
}
