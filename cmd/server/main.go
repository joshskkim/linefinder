package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joshuakim/linefinder/internal/api"
	"github.com/joshuakim/linefinder/internal/oddsapi"
	"github.com/joshuakim/linefinder/internal/service"
	"github.com/joshuakim/linefinder/internal/store"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ODDS_API_KEY")
	if apiKey == "" {
		log.Fatal("ODDS_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize components
	client := oddsapi.NewClient(apiKey)
	dataStore := store.New()
	oddsService := service.NewOddsService(client, dataStore)
	handler := api.NewHandler(oddsService)

	// Setup routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Wrap with CORS middleware for development
	corsHandler := api.CORSMiddleware(mux)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("LineFinder API starting on http://localhost%s\n", addr)
	fmt.Println("\nEndpoints:")
	fmt.Println("  GET  /api/health          - Health check")
	fmt.Println("  GET  /api/games/{sport}   - List games (nfl/nba)")
	fmt.Println("  GET  /api/odds/{sport}    - Get raw odds data")
	fmt.Println("  GET  /api/compare/{id}    - Compare odds for a game")
	fmt.Println("  POST /api/refresh/{sport} - Fetch fresh data from Odds API")
	fmt.Println()

	if err := http.ListenAndServe(addr, corsHandler); err != nil {
		log.Fatal(err)
	}
}
