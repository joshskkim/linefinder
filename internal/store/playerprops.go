package store

import "github.com/joshuakim/linefinder/internal/models"

// GetDummyPlayerProps returns dummy player props data for a given game ID and sport
func GetDummyPlayerProps(gameID string, sport models.Sport, homeTeam, awayTeam string) *models.GamePlayerProps {
	if sport == models.SportNBA {
		return getDummyNBAProps(gameID, homeTeam, awayTeam)
	}
	return getDummyNFLProps(gameID, homeTeam, awayTeam)
}

func getDummyNBAProps(gameID, homeTeam, awayTeam string) *models.GamePlayerProps {
	// Default teams if not provided
	if homeTeam == "" {
		homeTeam = "Home Team"
	}
	if awayTeam == "" {
		awayTeam = "Away Team"
	}

	return &models.GamePlayerProps{
		GameID:   gameID,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		Players: []models.PlayerWithProps{
			{
				Name: "Player 1",
				Team: awayTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Points",
						Market:   models.PlayerPoints,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 25.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 25.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 26.5, OverPrice: -105, UnderPrice: -115},
						},
					},
					{
						Category: "Rebounds",
						Market:   models.PlayerRebounds,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 7.5, OverPrice: -120, UnderPrice: 100},
							{Key: "fanduel", Title: "FanDuel", Point: 7.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 7.5, OverPrice: -110, UnderPrice: -110},
						},
					},
					{
						Category: "Assists",
						Market:   models.PlayerAssists,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 8.5, OverPrice: -105, UnderPrice: -115},
							{Key: "fanduel", Title: "FanDuel", Point: 8.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 8.5, OverPrice: -115, UnderPrice: -105},
						},
					},
					{
						Category: "Threes Made",
						Market:   models.PlayerThrees,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 2.5, OverPrice: -130, UnderPrice: 110},
							{Key: "fanduel", Title: "FanDuel", Point: 2.5, OverPrice: -125, UnderPrice: 105},
							{Key: "betmgm", Title: "BetMGM", Point: 2.5, OverPrice: -120, UnderPrice: 100},
						},
					},
				},
			},
			{
				Name: "Player 2",
				Team: awayTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Points",
						Market:   models.PlayerPoints,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 27.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 27.5, OverPrice: -105, UnderPrice: -115},
							{Key: "betmgm", Title: "BetMGM", Point: 28.5, OverPrice: 100, UnderPrice: -120},
						},
					},
					{
						Category: "Rebounds",
						Market:   models.PlayerRebounds,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 12.5, OverPrice: -105, UnderPrice: -115},
							{Key: "fanduel", Title: "FanDuel", Point: 12.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 12.5, OverPrice: -108, UnderPrice: -112},
						},
					},
					{
						Category: "Assists",
						Market:   models.PlayerAssists,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 3.5, OverPrice: -125, UnderPrice: 105},
							{Key: "fanduel", Title: "FanDuel", Point: 3.5, OverPrice: -120, UnderPrice: 100},
							{Key: "betmgm", Title: "BetMGM", Point: 3.5, OverPrice: -115, UnderPrice: -105},
						},
					},
				},
			},
			{
				Name: "Player 3",
				Team: homeTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Points",
						Market:   models.PlayerPoints,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 28.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 28.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 29.5, OverPrice: 100, UnderPrice: -120},
						},
					},
					{
						Category: "Rebounds",
						Market:   models.PlayerRebounds,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 8.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 8.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 8.5, OverPrice: -105, UnderPrice: -115},
						},
					},
					{
						Category: "Assists",
						Market:   models.PlayerAssists,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 5.5, OverPrice: -120, UnderPrice: 100},
							{Key: "fanduel", Title: "FanDuel", Point: 5.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 5.5, OverPrice: -110, UnderPrice: -110},
						},
					},
					{
						Category: "Threes Made",
						Market:   models.PlayerThrees,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 3.5, OverPrice: -105, UnderPrice: -115},
							{Key: "fanduel", Title: "FanDuel", Point: 3.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 3.5, OverPrice: 100, UnderPrice: -120},
						},
					},
				},
			},
			{
				Name: "Player 4",
				Team: homeTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Points",
						Market:   models.PlayerPoints,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 23.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 23.5, OverPrice: -105, UnderPrice: -115},
							{Key: "betmgm", Title: "BetMGM", Point: 24.5, OverPrice: 105, UnderPrice: -125},
						},
					},
					{
						Category: "Rebounds",
						Market:   models.PlayerRebounds,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 5.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 5.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 5.5, OverPrice: -105, UnderPrice: -115},
						},
					},
					{
						Category: "Assists",
						Market:   models.PlayerAssists,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 3.5, OverPrice: -105, UnderPrice: -115},
							{Key: "fanduel", Title: "FanDuel", Point: 3.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 3.5, OverPrice: 100, UnderPrice: -120},
						},
					},
				},
			},
		},
	}
}

func getDummyNFLProps(gameID, homeTeam, awayTeam string) *models.GamePlayerProps {
	// Default teams if not provided
	if homeTeam == "" {
		homeTeam = "Home Team"
	}
	if awayTeam == "" {
		awayTeam = "Away Team"
	}

	return &models.GamePlayerProps{
		GameID:   gameID,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		Players: []models.PlayerWithProps{
			{
				Name: "QB 1",
				Team: homeTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Passing Yards",
						Market:   models.PlayerPassYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 275.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 275.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 280.5, OverPrice: 100, UnderPrice: -120},
						},
					},
					{
						Category: "Passing TDs",
						Market:   models.PlayerPassTDs,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 2.5, OverPrice: -130, UnderPrice: 110},
							{Key: "fanduel", Title: "FanDuel", Point: 2.5, OverPrice: -125, UnderPrice: 105},
							{Key: "betmgm", Title: "BetMGM", Point: 2.5, OverPrice: -120, UnderPrice: 100},
						},
					},
					{
						Category: "Completions",
						Market:   models.PlayerPassCompletions,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 23.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 23.5, OverPrice: -105, UnderPrice: -115},
							{Key: "betmgm", Title: "BetMGM", Point: 24.5, OverPrice: 105, UnderPrice: -125},
						},
					},
					{
						Category: "Rush Yards",
						Market:   models.PlayerRushYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 25.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 25.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 24.5, OverPrice: -105, UnderPrice: -115},
						},
					},
				},
			},
			{
				Name: "WR 1",
				Team: homeTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Receiving Yards",
						Market:   models.PlayerReceivingYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 65.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 65.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 68.5, OverPrice: 105, UnderPrice: -125},
						},
					},
					{
						Category: "Receptions",
						Market:   models.PlayerReceptions,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 6.5, OverPrice: -120, UnderPrice: 100},
							{Key: "fanduel", Title: "FanDuel", Point: 6.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 6.5, OverPrice: -110, UnderPrice: -110},
						},
					},
				},
			},
			{
				Name: "QB 2",
				Team: awayTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Passing Yards",
						Market:   models.PlayerPassYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 285.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 285.5, OverPrice: -105, UnderPrice: -115},
							{Key: "betmgm", Title: "BetMGM", Point: 290.5, OverPrice: 105, UnderPrice: -125},
						},
					},
					{
						Category: "Passing TDs",
						Market:   models.PlayerPassTDs,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 2.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 2.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 2.5, OverPrice: -105, UnderPrice: -115},
						},
					},
					{
						Category: "Rush Yards",
						Market:   models.PlayerRushYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 45.5, OverPrice: -110, UnderPrice: -110},
							{Key: "fanduel", Title: "FanDuel", Point: 45.5, OverPrice: -115, UnderPrice: -105},
							{Key: "betmgm", Title: "BetMGM", Point: 42.5, OverPrice: -105, UnderPrice: -115},
						},
					},
				},
			},
			{
				Name: "WR 2",
				Team: awayTeam,
				Props: []models.PlayerPropCategory{
					{
						Category: "Receiving Yards",
						Market:   models.PlayerReceivingYards,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 75.5, OverPrice: -115, UnderPrice: -105},
							{Key: "fanduel", Title: "FanDuel", Point: 75.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 78.5, OverPrice: 100, UnderPrice: -120},
						},
					},
					{
						Category: "Receptions",
						Market:   models.PlayerReceptions,
						Bookmakers: []models.PropBookmaker{
							{Key: "draftkings", Title: "DraftKings", Point: 6.5, OverPrice: -105, UnderPrice: -115},
							{Key: "fanduel", Title: "FanDuel", Point: 6.5, OverPrice: -110, UnderPrice: -110},
							{Key: "betmgm", Title: "BetMGM", Point: 7.5, OverPrice: 120, UnderPrice: -140},
						},
					},
				},
			},
		},
	}
}
