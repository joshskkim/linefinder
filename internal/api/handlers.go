package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/joshuakim/linefinder/internal/models"
	"github.com/joshuakim/linefinder/internal/service"
	"github.com/joshuakim/linefinder/internal/store"
)

// Handler holds HTTP handlers
type Handler struct {
	oddsService *service.OddsService
}

// NewHandler creates a new handler
func NewHandler(oddsService *service.OddsService) *Handler {
	return &Handler{
		oddsService: oddsService,
	}
}

// RegisterRoutes sets up the HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/odds/", h.handleOdds)
	mux.HandleFunc("/api/games/", h.handleGames)
	mux.HandleFunc("/api/compare/", h.handleCompare)
	mux.HandleFunc("/api/refresh/", h.handleRefresh)
	mux.HandleFunc("/api/props/", h.handlePlayerProps)
}

// handleHealth returns service health status
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
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
		ID           string `json:"id"`
		HomeTeam     string `json:"home_team"`
		AwayTeam     string `json:"away_team"`
		CommenceTime string `json:"commence_time"`
		BookmakerCount int  `json:"bookmaker_count"`
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
	if found {
		homeTeam = game.HomeTeam
		awayTeam = game.AwayTeam
	}

	// Return dummy player props data
	props := store.GetDummyPlayerProps(gameID, sport, homeTeam, awayTeam)
	h.jsonResponse(w, http.StatusOK, props)
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
