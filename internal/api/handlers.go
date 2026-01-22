package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/joshuakim/linefinder/internal/alerts"
	"github.com/joshuakim/linefinder/internal/database"
	"github.com/joshuakim/linefinder/internal/metrics"
	"github.com/joshuakim/linefinder/internal/models"
	"github.com/joshuakim/linefinder/internal/notifications"
	"github.com/joshuakim/linefinder/internal/polling"
	"github.com/joshuakim/linefinder/internal/service"
	"github.com/joshuakim/linefinder/internal/sportsdata"
	"github.com/joshuakim/linefinder/internal/store"
	"github.com/joshuakim/linefinder/internal/websocket"
)

// Handler holds HTTP handlers
type Handler struct {
	oddsService      *service.OddsService
	sportsDataClient *sportsdata.Client
	hub              *websocket.Hub
	pollingSvc       *polling.Service
	metrics          *metrics.Metrics
	db               *database.DB
	alertDetector    *alerts.Detector
	notificationSvc  *notifications.Service
}

// NewHandler creates a new handler
func NewHandler(
	oddsService *service.OddsService,
	sportsDataClient *sportsdata.Client,
	hub *websocket.Hub,
	pollingSvc *polling.Service,
	m *metrics.Metrics,
	db *database.DB,
	alertDetector *alerts.Detector,
	notificationSvc *notifications.Service,
) *Handler {
	return &Handler{
		oddsService:      oddsService,
		sportsDataClient: sportsDataClient,
		hub:              hub,
		pollingSvc:       pollingSvc,
		metrics:          m,
		db:               db,
		alertDetector:    alertDetector,
		notificationSvc:  notificationSvc,
	}
}

// RegisterRoutes sets up the HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Core API endpoints
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/odds/", h.handleOdds)
	mux.HandleFunc("/api/games/", h.handleGames)
	mux.HandleFunc("/api/compare/", h.handleCompare)
	mux.HandleFunc("/api/refresh/", h.handleRefresh)
	mux.HandleFunc("/api/props/", h.handlePlayerProps)
	mux.HandleFunc("/api/injuries/", h.handleInjuries)
	mux.HandleFunc("/api/averages/", h.handlePlayerAverages)

	// WebSocket endpoint
	mux.HandleFunc("/api/ws", h.handleWebSocket)

	// Metrics and monitoring endpoints
	mux.HandleFunc("/api/metrics", h.handleMetrics)
	mux.HandleFunc("/api/polling/status", h.handlePollingStatus)
	mux.HandleFunc("/api/polling/toggle", h.handlePollingToggle)
	mux.HandleFunc("/api/polling/enable", h.handlePollingEnable)
	mux.HandleFunc("/api/polling/disable", h.handlePollingDisable)

	// Alert and notification endpoints
	mux.HandleFunc("/api/alerts/check", h.handleCheckAlerts)
	mux.HandleFunc("/api/preferences", h.handlePreferences)
	mux.HandleFunc("/api/subscribe", h.handleSubscribe)
	mux.HandleFunc("/api/unsubscribe", h.handleUnsubscribe)
	mux.HandleFunc("/api/vapid-public-key", h.handleVAPIDPublicKey)
}

// handleHealth returns service health status
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	pollingEnabled := false
	if h.pollingSvc != nil {
		pollingEnabled = h.pollingSvc.IsEnabled()
	}

	health := h.metrics.GetHealth(pollingEnabled)
	h.jsonResponse(w, http.StatusOK, health)
}

// handleWebSocket upgrades HTTP to WebSocket connection
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "WebSocket not available")
		return
	}
	websocket.ServeWs(h.hub, w, r)
}

// handleMetrics returns detailed metrics
func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pollingEnabled := false
	if h.pollingSvc != nil {
		pollingEnabled = h.pollingSvc.IsEnabled()
	}

	response := map[string]interface{}{
		"health": h.metrics.GetHealth(pollingEnabled),
	}

	if h.hub != nil {
		response["websocket"] = h.hub.GetStats()
	}

	if h.pollingSvc != nil {
		response["polling"] = h.pollingSvc.GetStatus()
	}

	h.jsonResponse(w, http.StatusOK, response)
}

// handlePollingStatus returns the current polling status
func (h *Handler) handlePollingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.pollingSvc == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "polling service not configured")
		return
	}

	h.jsonResponse(w, http.StatusOK, h.pollingSvc.GetStatus())
}

// handlePollingToggle toggles the polling state
func (h *Handler) handlePollingToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.pollingSvc == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "polling service not configured")
		return
	}

	h.pollingSvc.Toggle()

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "polling toggled",
		"status":  h.pollingSvc.GetStatus(),
	})
}

// handlePollingEnable enables polling
func (h *Handler) handlePollingEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.pollingSvc == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "polling service not configured")
		return
	}

	h.pollingSvc.Enable()

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "polling enabled",
		"status":  h.pollingSvc.GetStatus(),
	})
}

// handlePollingDisable disables polling
func (h *Handler) handlePollingDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.pollingSvc == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "polling service not configured")
		return
	}

	h.pollingSvc.Disable()

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "polling disabled",
		"status":  h.pollingSvc.GetStatus(),
	})
}

// handleCheckAlerts checks for value alerts across all games
// GET /api/alerts/check?sport=nba
func (h *Handler) handleCheckAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.alertDetector == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "alert detection not configured")
		return
	}

	sportStr := r.URL.Query().Get("sport")
	if sportStr == "" {
		sportStr = "nba"
	}

	var sport models.Sport
	switch sportStr {
	case "nfl":
		sport = models.SportNFL
	case "nba":
		sport = models.SportNBA
	default:
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	// Get all games for the sport
	games := h.oddsService.GetGamesBySport(sport)

	var allAlerts []alerts.ValueAlert

	// Check each game for value
	for _, game := range games {
		// Get player props and averages
		props := store.GetDummyPlayerProps(game.ID, sport, game.HomeTeam, game.AwayTeam)
		averages := store.GetDummyPlayerAverages(sportStr)

		// Build averages map
		avgMap := make(map[string]map[string]float64)
		for _, pa := range averages {
			avgMap[strings.ToLower(pa.Name)] = pa.Averages
		}

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

				alert := h.alertDetector.DetectValue(propData, ctx)
				if alert != nil {
					shouldNotify, _ := h.alertDetector.ShouldNotify(alert)
					if shouldNotify {
						h.alertDetector.RecordAlert(alert)
						allAlerts = append(allAlerts, *alert)
					}
				}
			}
		}
	}

	// Queue alerts for notification
	if len(allAlerts) > 0 && h.notificationSvc != nil {
		h.notificationSvc.QueueAlerts(allAlerts)
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"sport":       sportStr,
		"games":       len(games),
		"alerts":      allAlerts,
		"alert_count": len(allAlerts),
	})
}

// handlePreferences handles GET/PUT for notification preferences
func (h *Handler) handlePreferences(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		prefs, err := h.db.GetPreferences()
		if err != nil {
			h.errorResponse(w, http.StatusInternalServerError, "failed to get preferences")
			return
		}
		h.jsonResponse(w, http.StatusOK, prefs)

	case http.MethodPut:
		var prefs database.Preferences
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			h.errorResponse(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := h.db.UpdatePreferences(&prefs); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, "failed to update preferences")
			return
		}

		// Update alert detector thresholds
		if h.alertDetector != nil {
			h.alertDetector.UpdateThresholds(alerts.Thresholds{
				Points:   prefs.ThresholdPoints,
				Rebounds: prefs.ThresholdRebounds,
				Assists:  prefs.ThresholdAssists,
				Threes:   prefs.ThresholdThrees,
				Default:  prefs.ThresholdDefault,
			})
		}

		h.jsonResponse(w, http.StatusOK, map[string]string{"message": "preferences updated"})

	default:
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSubscribe handles push notification subscription
// POST /api/subscribe
func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.db == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var body struct {
		Subscription string `json:"subscription"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if body.Subscription == "" {
		h.errorResponse(w, http.StatusBadRequest, "subscription required")
		return
	}

	if err := h.db.SetPushSubscription(body.Subscription); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "failed to save subscription")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "subscribed to push notifications"})
}

// handleUnsubscribe handles unsubscribing from all notifications
// POST /api/unsubscribe
func (h *Handler) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.db == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	if err := h.db.Unsubscribe(); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "failed to unsubscribe")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"message": "unsubscribed from all notifications"})
}

// handleVAPIDPublicKey returns the VAPID public key for push subscription
// GET /api/vapid-public-key
func (h *Handler) handleVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.notificationSvc == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, "push notifications not configured")
		return
	}

	key := h.notificationSvc.GetVAPIDPublicKey()
	if key == "" {
		h.errorResponse(w, http.StatusServiceUnavailable, "VAPID keys not configured")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{"publicKey": key})
}

// handleOdds returns raw odds data for a sport
// GET /api/odds/{sport}
func (h *Handler) handleOdds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sport := h.parseSport(r.URL.Path, "/api/odds/")
	if sport == "" {
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	games := h.oddsService.GetGamesBySport(sport)
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"sport": sport,
		"count": len(games),
		"games": games,
	})
}

// handleGames returns a summary of games for a sport
// GET /api/games/{sport}
func (h *Handler) handleGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sport := h.parseSport(r.URL.Path, "/api/games/")
	if sport == "" {
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	games := h.oddsService.GetGamesBySport(sport)

	// Return simplified game list
	type gameSummary struct {
		ID             string `json:"id"`
		HomeTeam       string `json:"home_team"`
		AwayTeam       string `json:"away_team"`
		CommenceTime   string `json:"commence_time"`
		BookmakerCount int    `json:"bookmaker_count"`
	}

	summaries := make([]gameSummary, len(games))
	for i, game := range games {
		summaries[i] = gameSummary{
			ID:             game.ID,
			HomeTeam:       game.HomeTeam,
			AwayTeam:       game.AwayTeam,
			CommenceTime:   game.CommenceTime.Format("2006-01-02 15:04 MST"),
			BookmakerCount: len(game.Bookmakers),
		}
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"sport": sport,
		"count": len(summaries),
		"games": summaries,
	})
}

// handleCompare returns odds comparison for a specific game
// GET /api/compare/{gameID}
func (h *Handler) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	gameID := strings.TrimPrefix(r.URL.Path, "/api/compare/")
	if gameID == "" {
		h.errorResponse(w, http.StatusBadRequest, "game ID required")
		return
	}

	game, ok := h.oddsService.GetGame(gameID)
	if !ok {
		h.errorResponse(w, http.StatusNotFound, "game not found")
		return
	}

	comparison := h.oddsService.CompareOdds(game)
	h.jsonResponse(w, http.StatusOK, comparison)
}

// handleRefresh fetches fresh data from the Odds API
// POST /api/refresh/{sport}
func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sport := h.parseSport(r.URL.Path, "/api/refresh/")
	if sport == "" {
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	games, err := h.oddsService.FetchAndStoreOdds(sport)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, "failed to fetch odds: "+err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "data refreshed",
		"sport":   sport,
		"count":   len(games),
	})
}

// handlePlayerProps returns player props for a specific game
// GET /api/props/{sport}/{gameID}
func (h *Handler) handlePlayerProps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse path: /api/props/{sport}/{gameID}
	path := strings.TrimPrefix(r.URL.Path, "/api/props/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		h.errorResponse(w, http.StatusBadRequest, "invalid path: use /api/props/{sport}/{gameID}")
		return
	}

	sportStr := strings.ToLower(parts[0])
	gameID := parts[1]

	var sport models.Sport
	switch sportStr {
	case "nfl":
		sport = models.SportNFL
	case "nba":
		sport = models.SportNBA
	default:
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	// Get actual game data if available
	game, found := h.oddsService.GetGame(gameID)
	var homeTeam, awayTeam string
	var gameTime time.Time
	if found {
		homeTeam = game.HomeTeam
		awayTeam = game.AwayTeam
		gameTime = game.CommenceTime
	}

	// Return dummy player props data
	props := store.GetDummyPlayerProps(gameID, sport, homeTeam, awayTeam)

	// Check for value alerts if detector is available
	var valueAlerts []alerts.ValueAlert
	if h.alertDetector != nil && found {
		averages := store.GetDummyPlayerAverages(sportStr)
		avgMap := make(map[string]map[string]float64)
		for _, pa := range averages {
			avgMap[strings.ToLower(pa.Name)] = pa.Averages
		}

		ctx := alerts.GameContext{
			GameID:   gameID,
			Sport:    sportStr,
			HomeTeam: homeTeam,
			AwayTeam: awayTeam,
			GameTime: gameTime,
		}

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

				var bestLine, bestOdds float64
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

				alert := h.alertDetector.DetectValue(propData, ctx)
				if alert != nil {
					valueAlerts = append(valueAlerts, *alert)
				}
			}
		}
	}

	response := map[string]interface{}{
		"game_id":      props.GameID,
		"home_team":    props.HomeTeam,
		"away_team":    props.AwayTeam,
		"players":      props.Players,
		"value_alerts": valueAlerts,
	}

	h.jsonResponse(w, http.StatusOK, response)
}

// handleInjuries returns injury data for a specific game
// GET /api/injuries/{sport}/{gameID}
func (h *Handler) handleInjuries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse path: /api/injuries/{sport}/{gameID}
	path := strings.TrimPrefix(r.URL.Path, "/api/injuries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		h.errorResponse(w, http.StatusBadRequest, "invalid path: use /api/injuries/{sport}/{gameID}")
		return
	}

	sportStr := strings.ToLower(parts[0])
	gameID := parts[1]

	if sportStr != "nfl" && sportStr != "nba" {
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	// Get actual game data if available
	game, found := h.oddsService.GetGame(gameID)
	var homeTeam, awayTeam string
	if found {
		homeTeam = game.HomeTeam
		awayTeam = game.AwayTeam
	} else {
		homeTeam = "Home Team"
		awayTeam = "Away Team"
	}

	// Return dummy injury data
	injuries := store.GetDummyInjuries(gameID, homeTeam, awayTeam, sportStr)
	h.jsonResponse(w, http.StatusOK, injuries)
}

// handlePlayerAverages returns player averages from last 5 games
// GET /api/averages/{sport}/{gameID}
func (h *Handler) handlePlayerAverages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse path: /api/averages/{sport}/{gameID}
	path := strings.TrimPrefix(r.URL.Path, "/api/averages/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		h.errorResponse(w, http.StatusBadRequest, "invalid path: use /api/averages/{sport}/{gameID}")
		return
	}

	sportStr := strings.ToLower(parts[0])

	if sportStr != "nfl" && sportStr != "nba" {
		h.errorResponse(w, http.StatusBadRequest, "invalid sport: use 'nfl' or 'nba'")
		return
	}

	// Return dummy player averages
	averages := store.GetDummyPlayerAverages(sportStr)
	h.jsonResponse(w, http.StatusOK, averages)
}

// parseSport extracts and validates sport from URL path
func (h *Handler) parseSport(path, prefix string) models.Sport {
	sportStr := strings.TrimPrefix(path, prefix)
	sportStr = strings.ToLower(strings.TrimSuffix(sportStr, "/"))

	switch sportStr {
	case "nfl":
		return models.SportNFL
	case "nba":
		return models.SportNBA
	default:
		return ""
	}
}

func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// CORSMiddleware wraps a handler to add CORS headers for development
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}
