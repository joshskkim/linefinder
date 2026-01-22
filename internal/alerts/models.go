package alerts

import (
	"time"
)

// Confidence levels for alerts
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

// Direction indicates whether value is on over or under
const (
	DirectionOver  = "over"
	DirectionUnder = "under"
)

// PropCategory standard names
const (
	PropPoints      = "Points"
	PropRebounds    = "Rebounds"
	PropAssists     = "Assists"
	PropThrees      = "Threes"
	PropSteals      = "Steals"
	PropBlocks      = "Blocks"
	PropTurnovers   = "Turnovers"
	PropPRA         = "Pts+Reb+Ast"
	PropPR          = "Pts+Reb"
	PropPA          = "Pts+Ast"
	PropRA          = "Reb+Ast"
)

// ValueAlert represents a detected value opportunity
type ValueAlert struct {
	// Identification
	ID           string `json:"id"`
	PlayerName   string `json:"player_name"`
	Team         string `json:"team"`
	Sport        string `json:"sport"`
	GameID       string `json:"game_id"`
	GameTime     string `json:"game_time"`
	AwayTeam     string `json:"away_team"`
	HomeTeam     string `json:"home_team"`

	// Prop details
	PropCategory string  `json:"prop_category"`
	Line         float64 `json:"line"`
	Average      float64 `json:"average"`
	Difference   float64 `json:"difference"`
	AbsDifference float64 `json:"abs_difference"`

	// Analysis
	Direction  string `json:"direction"`
	Confidence string `json:"confidence"`

	// Best available odds
	BestOdds   float64 `json:"best_odds"`
	Bookmaker  string  `json:"bookmaker"`

	// Timing
	DetectedAt time.Time `json:"detected_at"`
	ExpiresAt  time.Time `json:"expires_at"` // Game start time
}

// AlertBatch represents a collection of alerts for push notification
type AlertBatch struct {
	Alerts    []ValueAlert `json:"alerts"`
	Count     int          `json:"count"`
	CreatedAt time.Time    `json:"created_at"`
	Summary   string       `json:"summary"`
}

// Thresholds holds per-prop threshold configuration
type Thresholds struct {
	Points   float64 `json:"points"`
	Rebounds float64 `json:"rebounds"`
	Assists  float64 `json:"assists"`
	Threes   float64 `json:"threes"`
	Default  float64 `json:"default"`
}

// DefaultThresholds returns the default threshold configuration
func DefaultThresholds() Thresholds {
	return Thresholds{
		Points:   2.0,
		Rebounds: 1.5,
		Assists:  1.0,
		Threes:   0.5,
		Default:  2.0,
	}
}

// GetThreshold returns the threshold for a given prop category
func (t Thresholds) GetThreshold(category string) float64 {
	switch category {
	case PropPoints:
		return t.Points
	case PropRebounds:
		return t.Rebounds
	case PropAssists:
		return t.Assists
	case PropThrees:
		return t.Threes
	default:
		return t.Default
	}
}

// CooldownDurations for different confidence levels
var CooldownDurations = map[string]time.Duration{
	ConfidenceLow:    4 * time.Hour,
	ConfidenceMedium: 2 * time.Hour,
	ConfidenceHigh:   1 * time.Hour,
}

// GetCooldownDuration returns the cooldown duration for a confidence level
func GetCooldownDuration(confidence string) time.Duration {
	if d, ok := CooldownDurations[confidence]; ok {
		return d
	}
	return 4 * time.Hour // Default
}

// GetConfidence returns confidence level based on absolute difference
func GetConfidence(absDiff float64, threshold float64) string {
	ratio := absDiff / threshold

	switch {
	case ratio >= 2.0: // 2x threshold or more
		return ConfidenceHigh
	case ratio >= 1.5: // 1.5x threshold
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}
