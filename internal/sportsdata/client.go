package sportsdata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.sportsdata.io/v3"

// Client handles communication with SportsDataIO API
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new SportsDataIO client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Player represents a player from SportsDataIO
type Player struct {
	PlayerID        int     `json:"PlayerID"`
	SportsDataID    string  `json:"SportsDataID"`
	FirstName       string  `json:"FirstName"`
	LastName        string  `json:"LastName"`
	Team            string  `json:"Team"`
	TeamID          int     `json:"TeamID"`
	Position        string  `json:"Position"`
	InjuryStatus    *string `json:"InjuryStatus"`
	InjuryBodyPart  *string `json:"InjuryBodyPart"`
	InjuryStartDate *string `json:"InjuryStartDate"`
	InjuryNotes     *string `json:"InjuryNotes"`
}

// PlayerGameStats represents a player's stats for a single game
type PlayerGameStats struct {
	PlayerID          int     `json:"PlayerID"`
	Name              string  `json:"Name"`
	Team              string  `json:"Team"`
	Position          string  `json:"Position"`
	GameID            int     `json:"GameID"`
	DateTime          string  `json:"DateTime"`
	// NBA Stats
	Points            float64 `json:"Points"`
	Rebounds          float64 `json:"Rebounds"`
	Assists           float64 `json:"Assists"`
	Steals            float64 `json:"Steals"`
	BlockedShots      float64 `json:"BlockedShots"`
	ThreePointersMade float64 `json:"ThreePointersMade"`
	Minutes           int     `json:"Minutes"`
	// NFL Stats
	PassingYards      float64 `json:"PassingYards"`
	PassingTouchdowns float64 `json:"PassingTouchdowns"`
	PassingAttempts   float64 `json:"PassingAttempts"`
	PassingCompletions float64 `json:"PassingCompletions"`
	RushingYards      float64 `json:"RushingYards"`
	RushingAttempts   float64 `json:"RushingAttempts"`
	ReceivingYards    float64 `json:"ReceivingYards"`
	Receptions        float64 `json:"Receptions"`
}

// GetNBAPlayers fetches all NBA players with injury info
func (c *Client) GetNBAPlayers() ([]Player, error) {
	url := fmt.Sprintf("%s/nba/scores/json/Players?key=%s", baseURL, c.apiKey)
	return c.fetchPlayers(url)
}

// GetNFLPlayers fetches all NFL players with injury info
func (c *Client) GetNFLPlayers() ([]Player, error) {
	url := fmt.Sprintf("%s/nfl/scores/json/Players?key=%s", baseURL, c.apiKey)
	return c.fetchPlayers(url)
}

// GetNBAPlayerGameStats fetches NBA player game stats for a season
func (c *Client) GetNBAPlayerGameStats(season string, playerID int) ([]PlayerGameStats, error) {
	url := fmt.Sprintf("%s/nba/stats/json/PlayerGameStatsByPlayer/%s/%d?key=%s", baseURL, season, playerID, c.apiKey)
	return c.fetchPlayerGameStats(url)
}

// GetNFLPlayerGameStats fetches NFL player game stats for a season
func (c *Client) GetNFLPlayerGameStats(season string, playerID int) ([]PlayerGameStats, error) {
	url := fmt.Sprintf("%s/nfl/stats/json/PlayerGameStatsByPlayerID/%s/%d?key=%s", baseURL, season, playerID, c.apiKey)
	return c.fetchPlayerGameStats(url)
}

func (c *Client) fetchPlayers(url string) ([]Player, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch players: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var players []Player
	if err := json.NewDecoder(resp.Body).Decode(&players); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return players, nil
}

func (c *Client) fetchPlayerGameStats(url string) ([]PlayerGameStats, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch player game stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var stats []PlayerGameStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return stats, nil
}
