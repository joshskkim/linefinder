package models

import "time"

// Sport represents supported sports
type Sport string

const (
	SportNFL Sport = "americanfootball_nfl"
	SportNBA Sport = "basketball_nba"
)

// Market represents betting market types
type Market string

const (
	MarketH2H     Market = "h2h"     // Moneyline
	MarketSpreads Market = "spreads" // Point spread
	MarketTotals  Market = "totals"  // Over/under
)

// Game represents a single sporting event
type Game struct {
	ID           string    `json:"id"`
	SportKey     Sport     `json:"sport_key"`
	SportTitle   string    `json:"sport_title"`
	CommenceTime time.Time `json:"commence_time"`
	HomeTeam     string    `json:"home_team"`
	AwayTeam     string    `json:"away_team"`
	Bookmakers   []Bookmaker `json:"bookmakers,omitempty"`
}

// Bookmaker represents a sportsbook's odds for a game
type Bookmaker struct {
	Key        string    `json:"key"`
	Title      string    `json:"title"`
	LastUpdate time.Time `json:"last_update"`
	Markets    []MarketData `json:"markets"`
}

// MarketData represents odds for a specific market type
type MarketData struct {
	Key      Market    `json:"key"`
	Outcomes []Outcome `json:"outcomes"`
}

// Outcome represents a single betting option
type Outcome struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`  // American odds (e.g., -110, +150)
	Point *float64 `json:"point,omitempty"` // Spread or total line
}

// OddsComparison represents the best odds found across bookmakers
type OddsComparison struct {
	GameID       string           `json:"game_id"`
	HomeTeam     string           `json:"home_team"`
	AwayTeam     string           `json:"away_team"`
	CommenceTime time.Time        `json:"commence_time"`
	Moneyline    *MoneylineComparison `json:"moneyline,omitempty"`
	Spread       *SpreadComparison    `json:"spread,omitempty"`
	Total        *TotalComparison     `json:"total,omitempty"`
}

// MoneylineComparison shows best moneyline odds
type MoneylineComparison struct {
	BestHome      BestOdds `json:"best_home"`
	BestAway      BestOdds `json:"best_away"`
	AllBookmakers []BookmakerOdds `json:"all_bookmakers"`
}

// SpreadComparison shows best spread odds
type SpreadComparison struct {
	BestHome      BestSpreadOdds `json:"best_home"`
	BestAway      BestSpreadOdds `json:"best_away"`
	AllBookmakers []BookmakerSpreadOdds `json:"all_bookmakers"`
}

// TotalComparison shows best over/under odds
type TotalComparison struct {
	BestOver      BestTotalOdds `json:"best_over"`
	BestUnder     BestTotalOdds `json:"best_under"`
	AllBookmakers []BookmakerTotalOdds `json:"all_bookmakers"`
}

// BestOdds represents the best odds found for a moneyline
type BestOdds struct {
	Price     float64 `json:"price"`
	Bookmaker string  `json:"bookmaker"`
}

// BestSpreadOdds represents the best spread odds
type BestSpreadOdds struct {
	Price     float64 `json:"price"`
	Point     float64 `json:"point"`
	Bookmaker string  `json:"bookmaker"`
}

// BestTotalOdds represents the best total odds
type BestTotalOdds struct {
	Price     float64 `json:"price"`
	Point     float64 `json:"point"`
	Bookmaker string  `json:"bookmaker"`
}

// BookmakerOdds holds moneyline odds from a single bookmaker
type BookmakerOdds struct {
	Bookmaker string  `json:"bookmaker"`
	HomePrice float64 `json:"home_price"`
	AwayPrice float64 `json:"away_price"`
}

// BookmakerSpreadOdds holds spread odds from a single bookmaker
type BookmakerSpreadOdds struct {
	Bookmaker string  `json:"bookmaker"`
	HomePrice float64 `json:"home_price"`
	HomePoint float64 `json:"home_point"`
	AwayPrice float64 `json:"away_price"`
	AwayPoint float64 `json:"away_point"`
}

// BookmakerTotalOdds holds total odds from a single bookmaker
type BookmakerTotalOdds struct {
	Bookmaker  string  `json:"bookmaker"`
	OverPrice  float64 `json:"over_price"`
	UnderPrice float64 `json:"under_price"`
	Point      float64 `json:"point"`
}

// PlayerPropMarket represents a player prop market type
type PlayerPropMarket string

// NBA player prop markets
const (
	PlayerPoints          PlayerPropMarket = "player_points"
	PlayerRebounds        PlayerPropMarket = "player_rebounds"
	PlayerAssists         PlayerPropMarket = "player_assists"
	PlayerThrees          PlayerPropMarket = "player_threes"
	PlayerPointsRebounds  PlayerPropMarket = "player_points_rebounds"
	PlayerPointsAssists   PlayerPropMarket = "player_points_assists"
	PlayerReboundsAssists PlayerPropMarket = "player_rebounds_assists"
	PlayerPRA             PlayerPropMarket = "player_points_rebounds_assists"
)

// NFL player prop markets
const (
	PlayerPassYards      PlayerPropMarket = "player_pass_yds"
	PlayerPassTDs        PlayerPropMarket = "player_pass_tds"
	PlayerPassAttempts   PlayerPropMarket = "player_pass_attempts"
	PlayerPassCompletions PlayerPropMarket = "player_pass_completions"
	PlayerRushYards      PlayerPropMarket = "player_rush_yds"
	PlayerRushAttempts   PlayerPropMarket = "player_rush_attempts"
	PlayerReceptions     PlayerPropMarket = "player_receptions"
	PlayerReceivingYards PlayerPropMarket = "player_reception_yds"
)

// PlayerProp represents a single player prop bet
type PlayerProp struct {
	PlayerName string           `json:"player_name"`
	Market     PlayerPropMarket `json:"market"`
	Bookmakers []PropBookmaker  `json:"bookmakers"`
}

// PropBookmaker represents a bookmaker's odds for a player prop
type PropBookmaker struct {
	Key        string  `json:"key"`
	Title      string  `json:"title"`
	OverPrice  float64 `json:"over_price"`
	UnderPrice float64 `json:"under_price"`
	Point      float64 `json:"point"`
}

// GamePlayerProps holds all player props for a game
type GamePlayerProps struct {
	GameID     string            `json:"game_id"`
	HomeTeam   string            `json:"home_team"`
	AwayTeam   string            `json:"away_team"`
	Players    []PlayerWithProps `json:"players"`
}

// PlayerWithProps groups all props for a single player
type PlayerWithProps struct {
	Name  string               `json:"name"`
	Team  string               `json:"team"`
	Props []PlayerPropCategory `json:"props"`
}

// PlayerPropCategory groups props by category (points, rebounds, etc.)
type PlayerPropCategory struct {
	Category   string          `json:"category"`
	Market     PlayerPropMarket `json:"market"`
	Bookmakers []PropBookmaker `json:"bookmakers"`
}
