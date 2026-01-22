package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/joshuakim/linefinder/internal/alerts"
	"github.com/joshuakim/linefinder/internal/api"
	"github.com/joshuakim/linefinder/internal/database"
	"github.com/joshuakim/linefinder/internal/metrics"
	"github.com/joshuakim/linefinder/internal/models"
	"github.com/joshuakim/linefinder/internal/notifications"
	"github.com/joshuakim/linefinder/internal/oddsapi"
	"github.com/joshuakim/linefinder/internal/polling"
	"github.com/joshuakim/linefinder/internal/service"
	"github.com/joshuakim/linefinder/internal/sportsdata"
	"github.com/joshuakim/linefinder/internal/store"
	"github.com/joshuakim/linefinder/internal/websocket"
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

	// Get SportsDataIO API key (optional)
	sportsDataKey := os.Getenv("SPORTSDATA_API_KEY")
	var sportsDataClient *sportsdata.Client
	if sportsDataKey != "" {
		sportsDataClient = sportsdata.NewClient(sportsDataKey)
		log.Println("SportsDataIO client initialized")
	} else {
		log.Println("SPORTSDATA_API_KEY not set - using dummy data for injuries/stats")
	}

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		homeDir, _ := os.UserHomeDir()
		dbPath = filepath.Join(homeDir, ".linefinder", "linefinder.db")
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Printf("Database initialized at %s", dbPath)

	// Initialize metrics
	m := metrics.New()

	// Set API quota limit from environment (default: 500 for free tier)
	if quotaStr := os.Getenv("API_QUOTA_LIMIT"); quotaStr != "" {
		if quota, err := strconv.ParseInt(quotaStr, 10, 64); err == nil {
			m.APIQuotaLimit = quota
		}
	} else {
		m.APIQuotaLimit = 500 // Default free tier
	}

	// Initialize core components
	client := oddsapi.NewClient(apiKey)
	dataStore := store.New()
	oddsService := service.NewOddsService(client, dataStore)

	// Initialize WebSocket hub
	maxConnections := 1000
	if maxConnStr := os.Getenv("WS_MAX_CONNECTIONS"); maxConnStr != "" {
		if maxConn, err := strconv.Atoi(maxConnStr); err == nil {
			maxConnections = maxConn
		}
	}
	hub := websocket.NewHub(m, maxConnections)
	go hub.Run()

	// Initialize alert detector
	alertDetector := alerts.NewDetector(db)

	// Load thresholds from database
	prefs, err := db.GetPreferences()
	if err == nil {
		alertDetector.UpdateThresholds(alerts.Thresholds{
			Points:   prefs.ThresholdPoints,
			Rebounds: prefs.ThresholdRebounds,
			Assists:  prefs.ThresholdAssists,
			Threes:   prefs.ThresholdThrees,
			Default:  prefs.ThresholdDefault,
		})
	}

	// Initialize notification service
	notifConfig := notifications.DefaultConfig()
	notifConfig.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	notifConfig.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	notifConfig.VAPIDSubject = os.Getenv("VAPID_SUBJECT")
	if notifConfig.VAPIDSubject == "" {
		notifConfig.VAPIDSubject = "mailto:alerts@linefinder.app"
	}

	if batchStr := os.Getenv("NOTIFICATION_BATCH_SECONDS"); batchStr != "" {
		if batch, err := strconv.Atoi(batchStr); err == nil {
			notifConfig.BatchInterval = time.Duration(batch) * time.Second
		}
	}

	notificationSvc := notifications.NewService(notifConfig, db, hub)

	// Initialize polling service
	pollConfig := polling.DefaultConfig()

	// Read polling configuration from environment
	if enabled := os.Getenv("POLL_ENABLED"); enabled == "true" {
		pollConfig.Enabled = true
	}
	if intervalStr := os.Getenv("POLL_INTERVAL_SECONDS"); intervalStr != "" {
		if interval, err := strconv.Atoi(intervalStr); err == nil {
			pollConfig.Interval = time.Duration(interval) * time.Second
		}
	}
	if sportsStr := os.Getenv("POLL_SPORTS"); sportsStr != "" {
		pollConfig.Sports = []models.Sport{}
		if sportsStr == "nba" || sportsStr == "nba,nfl" || sportsStr == "nfl,nba" {
			pollConfig.Sports = append(pollConfig.Sports, models.SportNBA)
		}
		if sportsStr == "nfl" || sportsStr == "nba,nfl" || sportsStr == "nfl,nba" {
			pollConfig.Sports = append(pollConfig.Sports, models.SportNFL)
		}
		if len(pollConfig.Sports) == 0 {
			pollConfig.Sports = []models.Sport{models.SportNBA, models.SportNFL}
		}
	}

	pollingSvc := polling.NewService(pollConfig, oddsService, hub, m)

	// Wire alert detection to polling service
	pollingSvc.SetAlertDetector(alertDetector, func(valueAlerts []alerts.ValueAlert) {
		notificationSvc.QueueAlerts(valueAlerts)
	})

	// Start services in background
	ctx, cancel := context.WithCancel(context.Background())
	go pollingSvc.Start(ctx)
	go notificationSvc.Start(ctx)

	// Initialize HTTP handler
	handler := api.NewHandler(
		oddsService,
		sportsDataClient,
		hub,
		pollingSvc,
		m,
		db,
		alertDetector,
		notificationSvc,
	)

	// Setup routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Wrap with CORS middleware for development
	corsHandler := api.CORSMiddleware(mux)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: corsHandler,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("LineFinder API starting on http://localhost%s\n", server.Addr)
		fmt.Println("\nCore Endpoints:")
		fmt.Println("  GET  /api/health           - Health check with metrics")
		fmt.Println("  GET  /api/games/{sport}    - List games (nfl/nba)")
		fmt.Println("  GET  /api/odds/{sport}     - Get raw odds data")
		fmt.Println("  POST /api/refresh/{sport}  - Fetch fresh data from Odds API")
		fmt.Println("\nPlayer Data Endpoints:")
		fmt.Println("  GET  /api/props/{sport}/{id}    - Player props for a game")
		fmt.Println("  GET  /api/injuries/{sport}/{id} - Injuries for a game")
		fmt.Println("  GET  /api/averages/{sport}/{id} - Player averages")
		fmt.Println("\nReal-time Endpoints:")
		fmt.Println("  WS   /api/ws                - WebSocket for live updates")
		fmt.Println("  GET  /api/metrics           - Detailed system metrics")
		fmt.Println("  POST /api/polling/toggle    - Toggle polling on/off")
		fmt.Println("\nAlert & Notification Endpoints:")
		fmt.Println("  GET  /api/alerts/check      - Check for value alerts")
		fmt.Println("  GET  /api/preferences       - Get notification preferences")
		fmt.Println("  PUT  /api/preferences       - Update preferences")
		fmt.Println("  POST /api/subscribe         - Subscribe to push notifications")
		fmt.Println("  POST /api/unsubscribe       - Unsubscribe from all notifications")
		fmt.Println("  GET  /api/vapid-public-key  - Get VAPID public key")
		fmt.Printf("\nPolling: %v (interval: %v)\n", pollConfig.Enabled, pollConfig.Interval)
		fmt.Printf("Database: %s\n", dbPath)

		if notifConfig.VAPIDPublicKey != "" {
			fmt.Println("Push notifications: ENABLED")
		} else {
			fmt.Println("Push notifications: DISABLED (set VAPID keys to enable)")
		}
		fmt.Println()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel background services
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
