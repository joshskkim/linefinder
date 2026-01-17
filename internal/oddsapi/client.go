package oddsapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/joshuakim/linefinder/internal/models"
)

const baseURL = "https://api.the-odds-api.com/v4"

// Client handles communication with The Odds API
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Odds API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// GetOdds fetches odds for a sport with all markets
func (c *Client) GetOdds(sport models.Sport) ([]models.Game, error) {
	endpoint := fmt.Sprintf("%s/sports/%s/odds/", c.baseURL, sport)

	params := url.Values{}
	params.Add("apiKey", c.apiKey)
	params.Add("regions", "us")
	params.Add("markets", "h2h,spreads,totals")
	params.Add("oddsFormat", "american")
	params.Add("bookmakers", "draftkings,fanduel,betmgm")

	fullURL := endpoint + "?" + params.Encode()

	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch odds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Log remaining requests from headers
	remaining := resp.Header.Get("X-Requests-Remaining")
	used := resp.Header.Get("X-Requests-Used")
	if remaining != "" {
		fmt.Printf("[OddsAPI] Requests remaining: %s, used: %s\n", remaining, used)
	}

	var games []models.Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return games, nil
}

// GetNFLOdds fetches NFL odds
func (c *Client) GetNFLOdds() ([]models.Game, error) {
	return c.GetOdds(models.SportNFL)
}

// GetNBAOdds fetches NBA odds
func (c *Client) GetNBAOdds() ([]models.Game, error) {
	return c.GetOdds(models.SportNBA)
}
