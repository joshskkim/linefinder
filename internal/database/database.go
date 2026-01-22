package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes schema
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}

	log.Printf("Database initialized at %s", dbPath)
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) initSchema() error {
	schema := `
	-- Notification preferences (single user for now)
	CREATE TABLE IF NOT EXISTS preferences (
		id INTEGER PRIMARY KEY CHECK (id = 1),

		-- Channel settings
		enable_websocket BOOLEAN DEFAULT true,
		enable_push BOOLEAN DEFAULT false,
		push_subscription TEXT,

		-- Alert thresholds per prop type
		threshold_points REAL DEFAULT 2.0,
		threshold_rebounds REAL DEFAULT 1.5,
		threshold_assists REAL DEFAULT 1.0,
		threshold_threes REAL DEFAULT 0.5,
		threshold_default REAL DEFAULT 2.0,

		-- Filters
		sports TEXT DEFAULT 'nba,nfl',

		-- Quiet hours
		quiet_start TEXT DEFAULT '23:00',
		quiet_end TEXT DEFAULT '08:00',
		timezone TEXT DEFAULT 'America/New_York',

		-- Rate limits (per hour)
		rate_limit_push INTEGER DEFAULT 20,

		-- Batching
		batch_interval_seconds INTEGER DEFAULT 60,

		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert default preferences if not exists
	INSERT OR IGNORE INTO preferences (id) VALUES (1);

	-- Alert history for deduplication
	CREATE TABLE IF NOT EXISTS alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,

		-- Alert identification
		player_name TEXT NOT NULL,
		prop_category TEXT NOT NULL,
		direction TEXT NOT NULL,
		game_id TEXT NOT NULL,

		-- Alert details
		line_value REAL NOT NULL,
		average_value REAL NOT NULL,
		difference REAL NOT NULL,
		confidence TEXT NOT NULL,

		-- Timing
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		cooldown_until TIMESTAMP NOT NULL,

		-- Notification tracking
		notified_websocket BOOLEAN DEFAULT false,
		notified_push BOOLEAN DEFAULT false,

		UNIQUE(player_name, prop_category, direction, game_id)
	);

	-- Rate limit tracking
	CREATE TABLE IF NOT EXISTS rate_limits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		window_start TIMESTAMP NOT NULL,
		count INTEGER DEFAULT 0,
		UNIQUE(channel, window_start)
	);

	-- Pending notifications for batching
	CREATE TABLE IF NOT EXISTS pending_notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alert_json TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		batch_id TEXT
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_alert_history_lookup
		ON alert_history(player_name, prop_category, direction, game_id);
	CREATE INDEX IF NOT EXISTS idx_alert_history_cooldown
		ON alert_history(cooldown_until);
	CREATE INDEX IF NOT EXISTS idx_pending_batch
		ON pending_notifications(batch_id);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Preferences represents user notification preferences
type Preferences struct {
	EnableWebsocket bool    `json:"enable_websocket"`
	EnablePush      bool    `json:"enable_push"`
	PushSubscription string `json:"push_subscription,omitempty"`

	// Per-prop thresholds
	ThresholdPoints   float64 `json:"threshold_points"`
	ThresholdRebounds float64 `json:"threshold_rebounds"`
	ThresholdAssists  float64 `json:"threshold_assists"`
	ThresholdThrees   float64 `json:"threshold_threes"`
	ThresholdDefault  float64 `json:"threshold_default"`

	// Filters
	Sports []string `json:"sports"`

	// Quiet hours
	QuietStart string `json:"quiet_start"`
	QuietEnd   string `json:"quiet_end"`
	Timezone   string `json:"timezone"`

	// Rate limits
	RateLimitPush int `json:"rate_limit_push"`

	// Batching
	BatchIntervalSeconds int `json:"batch_interval_seconds"`

	UpdatedAt time.Time `json:"updated_at"`
}

// GetPreferences retrieves user preferences
func (db *DB) GetPreferences() (*Preferences, error) {
	row := db.conn.QueryRow(`
		SELECT
			enable_websocket, enable_push, push_subscription,
			threshold_points, threshold_rebounds, threshold_assists,
			threshold_threes, threshold_default,
			sports, quiet_start, quiet_end, timezone,
			rate_limit_push, batch_interval_seconds, updated_at
		FROM preferences WHERE id = 1
	`)

	var p Preferences
	var sportsStr string
	var pushSub sql.NullString

	err := row.Scan(
		&p.EnableWebsocket, &p.EnablePush, &pushSub,
		&p.ThresholdPoints, &p.ThresholdRebounds, &p.ThresholdAssists,
		&p.ThresholdThrees, &p.ThresholdDefault,
		&sportsStr, &p.QuietStart, &p.QuietEnd, &p.Timezone,
		&p.RateLimitPush, &p.BatchIntervalSeconds, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if pushSub.Valid {
		p.PushSubscription = pushSub.String
	}

	// Parse sports
	if sportsStr != "" {
		for _, s := range splitAndTrim(sportsStr, ",") {
			if s != "" {
				p.Sports = append(p.Sports, s)
			}
		}
	}

	return &p, nil
}

// UpdatePreferences updates user preferences
func (db *DB) UpdatePreferences(p *Preferences) error {
	sportsStr := joinStrings(p.Sports, ",")

	_, err := db.conn.Exec(`
		UPDATE preferences SET
			enable_websocket = ?,
			enable_push = ?,
			push_subscription = ?,
			threshold_points = ?,
			threshold_rebounds = ?,
			threshold_assists = ?,
			threshold_threes = ?,
			threshold_default = ?,
			sports = ?,
			quiet_start = ?,
			quiet_end = ?,
			timezone = ?,
			rate_limit_push = ?,
			batch_interval_seconds = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`,
		p.EnableWebsocket, p.EnablePush, p.PushSubscription,
		p.ThresholdPoints, p.ThresholdRebounds, p.ThresholdAssists,
		p.ThresholdThrees, p.ThresholdDefault,
		sportsStr, p.QuietStart, p.QuietEnd, p.Timezone,
		p.RateLimitPush, p.BatchIntervalSeconds,
	)
	return err
}

// SetPushSubscription updates the push subscription
func (db *DB) SetPushSubscription(subscription string) error {
	_, err := db.conn.Exec(`
		UPDATE preferences SET
			push_subscription = ?,
			enable_push = true,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, subscription)
	return err
}

// Unsubscribe disables all notifications
func (db *DB) Unsubscribe() error {
	_, err := db.conn.Exec(`
		UPDATE preferences SET
			enable_websocket = false,
			enable_push = false,
			push_subscription = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	return err
}

// AlertHistory represents a historical alert record
type AlertHistory struct {
	ID            int64     `json:"id"`
	PlayerName    string    `json:"player_name"`
	PropCategory  string    `json:"prop_category"`
	Direction     string    `json:"direction"`
	GameID        string    `json:"game_id"`
	LineValue     float64   `json:"line_value"`
	AverageValue  float64   `json:"average_value"`
	Difference    float64   `json:"difference"`
	Confidence    string    `json:"confidence"`
	CreatedAt     time.Time `json:"created_at"`
	CooldownUntil time.Time `json:"cooldown_until"`
}

// GetAlertHistory retrieves alert history for deduplication check
func (db *DB) GetAlertHistory(playerName, propCategory, direction, gameID string) (*AlertHistory, error) {
	row := db.conn.QueryRow(`
		SELECT id, player_name, prop_category, direction, game_id,
			   line_value, average_value, difference, confidence,
			   created_at, cooldown_until
		FROM alert_history
		WHERE player_name = ? AND prop_category = ? AND direction = ? AND game_id = ?
	`, playerName, propCategory, direction, gameID)

	var h AlertHistory
	err := row.Scan(
		&h.ID, &h.PlayerName, &h.PropCategory, &h.Direction, &h.GameID,
		&h.LineValue, &h.AverageValue, &h.Difference, &h.Confidence,
		&h.CreatedAt, &h.CooldownUntil,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// SaveAlertHistory saves or updates alert history
func (db *DB) SaveAlertHistory(h *AlertHistory) error {
	_, err := db.conn.Exec(`
		INSERT INTO alert_history
			(player_name, prop_category, direction, game_id,
			 line_value, average_value, difference, confidence, cooldown_until)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(player_name, prop_category, direction, game_id)
		DO UPDATE SET
			line_value = excluded.line_value,
			average_value = excluded.average_value,
			difference = excluded.difference,
			confidence = excluded.confidence,
			cooldown_until = excluded.cooldown_until,
			created_at = CURRENT_TIMESTAMP
	`, h.PlayerName, h.PropCategory, h.Direction, h.GameID,
		h.LineValue, h.AverageValue, h.Difference, h.Confidence, h.CooldownUntil)
	return err
}

// CleanupExpiredHistory removes old alert history
func (db *DB) CleanupExpiredHistory() error {
	_, err := db.conn.Exec(`
		DELETE FROM alert_history
		WHERE cooldown_until < datetime('now', '-24 hours')
	`)
	return err
}

// CheckRateLimit checks if we can send on a channel
func (db *DB) CheckRateLimit(channel string, limit int) (bool, int, error) {
	windowStart := time.Now().Truncate(time.Hour)

	// Get or create rate limit record
	row := db.conn.QueryRow(`
		SELECT count FROM rate_limits
		WHERE channel = ? AND window_start = ?
	`, channel, windowStart)

	var count int
	err := row.Scan(&count)
	if err == sql.ErrNoRows {
		count = 0
	} else if err != nil {
		return false, 0, err
	}

	remaining := limit - count
	return count < limit, remaining, nil
}

// IncrementRateLimit increments the rate limit counter
func (db *DB) IncrementRateLimit(channel string) error {
	windowStart := time.Now().Truncate(time.Hour)

	_, err := db.conn.Exec(`
		INSERT INTO rate_limits (channel, window_start, count)
		VALUES (?, ?, 1)
		ON CONFLICT(channel, window_start)
		DO UPDATE SET count = count + 1
	`, channel, windowStart)
	return err
}

// CleanupOldRateLimits removes old rate limit records
func (db *DB) CleanupOldRateLimits() error {
	_, err := db.conn.Exec(`
		DELETE FROM rate_limits
		WHERE window_start < datetime('now', '-2 hours')
	`)
	return err
}

// PendingNotification represents a batched notification
type PendingNotification struct {
	ID        int64     `json:"id"`
	AlertJSON string    `json:"alert_json"`
	CreatedAt time.Time `json:"created_at"`
	BatchID   string    `json:"batch_id"`
}

// AddPendingNotification adds a notification to the batch queue
func (db *DB) AddPendingNotification(alertJSON string) error {
	_, err := db.conn.Exec(`
		INSERT INTO pending_notifications (alert_json)
		VALUES (?)
	`, alertJSON)
	return err
}

// GetPendingNotifications retrieves all pending notifications
func (db *DB) GetPendingNotifications() ([]PendingNotification, error) {
	rows, err := db.conn.Query(`
		SELECT id, alert_json, created_at, COALESCE(batch_id, '')
		FROM pending_notifications
		WHERE batch_id IS NULL
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []PendingNotification
	for rows.Next() {
		var n PendingNotification
		if err := rows.Scan(&n.ID, &n.AlertJSON, &n.CreatedAt, &n.BatchID); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

// ClearPendingNotifications removes processed notifications
func (db *DB) ClearPendingNotifications(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := "DELETE FROM pending_notifications WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	_, err := db.conn.Exec(query, args...)
	return err
}

// Helper functions
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
