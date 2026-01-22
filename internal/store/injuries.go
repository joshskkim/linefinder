package store

// InjuredPlayer represents a player with an injury
type InjuredPlayer struct {
	Name         string  `json:"name"`
	Position     string  `json:"position"`
	Status       string  `json:"status"` // Injured, Doubtful, Questionable, Probable
	BodyPart     string  `json:"body_part"`
	Notes        string  `json:"notes"`
}

// TeamInjuries holds injuries for a team
type TeamInjuries struct {
	Team    string          `json:"team"`
	Players []InjuredPlayer `json:"players"`
}

// GameInjuries holds injuries for both teams in a game
type GameInjuries struct {
	GameID     string         `json:"game_id"`
	HomeTeam   TeamInjuries   `json:"home_team"`
	AwayTeam   TeamInjuries   `json:"away_team"`
}

// PlayerAverages holds a player's average stats from last 5 games
type PlayerAverages struct {
	Name           string             `json:"name"`
	Team           string             `json:"team"`
	InjuryStatus   string             `json:"injury_status,omitempty"`
	GamesPlayed    int                `json:"games_played"`
	Averages       map[string]float64 `json:"averages"` // category -> average value
}

// GetDummyInjuries returns dummy injury data for a game
func GetDummyInjuries(gameID, homeTeam, awayTeam, sport string) *GameInjuries {
	if sport == "nba" {
		return getDummyNBAInjuries(gameID, homeTeam, awayTeam)
	}
	return getDummyNFLInjuries(gameID, homeTeam, awayTeam)
}

func getDummyNBAInjuries(gameID, homeTeam, awayTeam string) *GameInjuries {
	return &GameInjuries{
		GameID: gameID,
		HomeTeam: TeamInjuries{
			Team: homeTeam,
			Players: []InjuredPlayer{
				{Name: "Player A", Position: "PG", Status: "Questionable", BodyPart: "Ankle", Notes: "Ankle sprain - game-time decision"},
				{Name: "Player B", Position: "SF", Status: "Out", BodyPart: "Knee", Notes: "Knee soreness - will miss 1-2 games"},
			},
		},
		AwayTeam: TeamInjuries{
			Team: awayTeam,
			Players: []InjuredPlayer{
				{Name: "Player C", Position: "C", Status: "Probable", BodyPart: "Back", Notes: "Back tightness - expected to play"},
				{Name: "Player D", Position: "SG", Status: "Doubtful", BodyPart: "Hamstring", Notes: "Hamstring strain - unlikely to play"},
				{Name: "Player E", Position: "PF", Status: "Out", BodyPart: "Wrist", Notes: "Wrist fracture - out indefinitely"},
			},
		},
	}
}

func getDummyNFLInjuries(gameID, homeTeam, awayTeam string) *GameInjuries {
	return &GameInjuries{
		GameID: gameID,
		HomeTeam: TeamInjuries{
			Team: homeTeam,
			Players: []InjuredPlayer{
				{Name: "Player A", Position: "WR", Status: "Questionable", BodyPart: "Hamstring", Notes: "Hamstring - limited practice"},
				{Name: "Player B", Position: "CB", Status: "Out", BodyPart: "Knee", Notes: "ACL - IR"},
				{Name: "Player C", Position: "LB", Status: "Probable", BodyPart: "Shoulder", Notes: "Shoulder - full practice"},
			},
		},
		AwayTeam: TeamInjuries{
			Team: awayTeam,
			Players: []InjuredPlayer{
				{Name: "Player D", Position: "RB", Status: "Doubtful", BodyPart: "Ankle", Notes: "High ankle sprain"},
				{Name: "Player E", Position: "TE", Status: "Questionable", BodyPart: "Concussion", Notes: "Concussion protocol"},
			},
		},
	}
}

// GetDummyPlayerAverages returns dummy player averages for last 5 games
func GetDummyPlayerAverages(sport string) []PlayerAverages {
	if sport == "nba" {
		return getDummyNBAAverages()
	}
	return getDummyNFLAverages()
}

func getDummyNBAAverages() []PlayerAverages {
	return []PlayerAverages{
		{
			Name: "Player 1", Team: "Away Team", InjuryStatus: "", GamesPlayed: 5,
			Averages: map[string]float64{"Points": 26.4, "Rebounds": 7.2, "Assists": 8.8, "Threes Made": 2.6},
		},
		{
			Name: "Player 2", Team: "Away Team", InjuryStatus: "Questionable", GamesPlayed: 5,
			Averages: map[string]float64{"Points": 28.2, "Rebounds": 12.4, "Assists": 3.2},
		},
		{
			Name: "Player 3", Team: "Home Team", InjuryStatus: "", GamesPlayed: 5,
			Averages: map[string]float64{"Points": 29.8, "Rebounds": 8.6, "Assists": 5.4, "Threes Made": 3.8},
		},
		{
			Name: "Player 4", Team: "Home Team", InjuryStatus: "Probable", GamesPlayed: 4,
			Averages: map[string]float64{"Points": 24.5, "Rebounds": 5.8, "Assists": 3.5},
		},
	}
}

func getDummyNFLAverages() []PlayerAverages {
	return []PlayerAverages{
		{
			Name: "QB 1", Team: "Home Team", InjuryStatus: "", GamesPlayed: 5,
			Averages: map[string]float64{"Passing Yards": 278.4, "Passing TDs": 2.4, "Completions": 24.2, "Rush Yards": 28.6},
		},
		{
			Name: "WR 1", Team: "Home Team", InjuryStatus: "Questionable", GamesPlayed: 4,
			Averages: map[string]float64{"Receiving Yards": 68.5, "Receptions": 5.8},
		},
		{
			Name: "QB 2", Team: "Away Team", InjuryStatus: "", GamesPlayed: 5,
			Averages: map[string]float64{"Passing Yards": 292.8, "Passing TDs": 2.2, "Rush Yards": 42.4},
		},
		{
			Name: "WR 2", Team: "Away Team", InjuryStatus: "", GamesPlayed: 5,
			Averages: map[string]float64{"Receiving Yards": 82.4, "Receptions": 6.2},
		},
	}
}
