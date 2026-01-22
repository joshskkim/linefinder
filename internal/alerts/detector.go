package alerts

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/joshuakim/linefinder/internal/database"
)

// Detector detects value opportunities in player props
type Detector struct {
	db         *database.DB
	thresholds Thresholds
}

// NewDetector creates a new alert detector
func NewDetector(db *database.DB) *Detector {
	return &Detector{
		db:         db,
		thresholds: DefaultThresholds(),
	}
}

// UpdateThresholds updates the detection thresholds
func (d *Detector) UpdateThresholds(t Thresholds) {
	d.thresholds = t
}

// PropData represents a single prop with its line and average
type PropData struct {
	PlayerName   string
	Team         string
	PropCategory string
	Line         float64
	Average      float64
	BestOdds     float64
	BestOddsDir  string // "over" or "under"
	Bookmaker    string
}

// GameContext provides game context for alerts
type GameContext struct {
	GameID    string
	Sport     string
	HomeTeam  string
	AwayTeam  string
	GameTime  time.Time
}

// DetectValue checks a prop for value and returns an alert if found
func (d *Detector) DetectValue(prop PropData, ctx GameContext) *ValueAlert {
	threshold := d.thresholds.GetThreshold(prop.PropCategory)
	diff := prop.Line - prop.Average
	absDiff := math.Abs(diff)

	// No alert if within threshold
	if absDiff < threshold {
		return nil
	}

	// Determine direction
	direction := DirectionOver
	if diff > 0 {
		// Line is ABOVE average → value on UNDER
		direction = DirectionUnder
	}

	// Get confidence
	confidence := GetConfidence(absDiff, threshold)

	// Create alert
	alert := &ValueAlert{
		ID:            fmt.Sprintf("%s-%s-%s-%s", ctx.GameID, prop.PlayerName, prop.PropCategory, direction),
		PlayerName:    prop.PlayerName,
		Team:          prop.Team,
		Sport:         ctx.Sport,
		GameID:        ctx.GameID,
		GameTime:      ctx.GameTime.Format(time.RFC3339),
		HomeTeam:      ctx.HomeTeam,
		AwayTeam:      ctx.AwayTeam,
		PropCategory:  prop.PropCategory,
		Line:          prop.Line,
		Average:       prop.Average,
		Difference:    diff,
		AbsDifference: absDiff,
		Direction:     direction,
		Confidence:    confidence,
		BestOdds:      prop.BestOdds,
		Bookmaker:     prop.Bookmaker,
		DetectedAt:    time.Now(),
		ExpiresAt:     ctx.GameTime,
	}

	return alert
}

// ShouldNotify checks if an alert should trigger a notification
// considering deduplication and cooldown
func (d *Detector) ShouldNotify(alert *ValueAlert) (bool, string) {
	if d.db == nil {
		return true, "no database configured"
	}

	// Check alert history
	history, err := d.db.GetAlertHistory(
		alert.PlayerName,
		alert.PropCategory,
		alert.Direction,
		alert.GameID,
	)
	if err != nil {
		log.Printf("Error checking alert history: %v", err)
		return true, "error checking history"
	}

	// Never alerted before
	if history == nil {
		return true, "new alert"
	}

	// Check if still in cooldown
	if time.Now().Before(history.CooldownUntil) {
		// Only re-alert if line moved significantly (>0.5 units)
		lineDiff := math.Abs(alert.Line - history.LineValue)
		if lineDiff < 0.5 {
			return false, fmt.Sprintf("in cooldown until %s", history.CooldownUntil.Format("15:04"))
		}
		return true, fmt.Sprintf("line moved %.1f units", lineDiff)
	}

	return true, "cooldown expired"
}

// RecordAlert saves an alert to history
func (d *Detector) RecordAlert(alert *ValueAlert) error {
	if d.db == nil {
		return nil
	}

	cooldownDuration := GetCooldownDuration(alert.Confidence)

	history := &database.AlertHistory{
		PlayerName:    alert.PlayerName,
		PropCategory:  alert.PropCategory,
		Direction:     alert.Direction,
		GameID:        alert.GameID,
		LineValue:     alert.Line,
		AverageValue:  alert.Average,
		Difference:    alert.Difference,
		Confidence:    alert.Confidence,
		CooldownUntil: time.Now().Add(cooldownDuration),
	}

	return d.db.SaveAlertHistory(history)
}

// DetectAllValue processes multiple props and returns all value alerts
func (d *Detector) DetectAllValue(props []PropData, ctx GameContext) []ValueAlert {
	var alerts []ValueAlert

	for _, prop := range props {
		alert := d.DetectValue(prop, ctx)
		if alert == nil {
			continue
		}

		shouldNotify, reason := d.ShouldNotify(alert)
		if !shouldNotify {
			log.Printf("Skipping alert for %s %s: %s", prop.PlayerName, prop.PropCategory, reason)
			continue
		}

		log.Printf("Value detected: %s %s %.1f (avg %.1f, diff %.1f) → %s [%s]",
			prop.PlayerName, prop.PropCategory, prop.Line, prop.Average,
			alert.Difference, alert.Direction, alert.Confidence)

		// Record the alert
		if err := d.RecordAlert(alert); err != nil {
			log.Printf("Error recording alert: %v", err)
		}

		alerts = append(alerts, *alert)
	}

	return alerts
}

// FormatAlertMessage creates a human-readable alert message
func FormatAlertMessage(alert *ValueAlert) string {
	dirSymbol := "↓"
	if alert.Direction == DirectionUnder {
		dirSymbol = "↑"
	}

	return fmt.Sprintf("%s %s %s %.1f (avg %.1f %s%.1f)",
		alert.PlayerName,
		alert.Direction,
		alert.PropCategory,
		alert.Line,
		alert.Average,
		dirSymbol,
		alert.AbsDifference,
	)
}

// FormatBatchSummary creates a summary for a batch of alerts
func FormatBatchSummary(alerts []ValueAlert) string {
	if len(alerts) == 0 {
		return "No value alerts"
	}

	if len(alerts) == 1 {
		return FormatAlertMessage(&alerts[0])
	}

	// Count by confidence
	high, medium, low := 0, 0, 0
	for _, a := range alerts {
		switch a.Confidence {
		case ConfidenceHigh:
			high++
		case ConfidenceMedium:
			medium++
		default:
			low++
		}
	}

	summary := fmt.Sprintf("%d value alerts", len(alerts))
	if high > 0 {
		summary += fmt.Sprintf(" (%d high confidence)", high)
	}

	return summary
}
