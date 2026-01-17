package service

import (
	"math"

	"github.com/joshuakim/linefinder/internal/models"
	"github.com/joshuakim/linefinder/internal/oddsapi"
	"github.com/joshuakim/linefinder/internal/store"
)

// Allowed bookmakers
var allowedBookmakers = map[string]bool{
	"draftkings": true,
	"fanduel":    true,
	"betmgm":     true,
}

// OddsService handles odds-related business logic
type OddsService struct {
	client *oddsapi.Client
	store  *store.Store
}

// NewOddsService creates a new odds service
func NewOddsService(client *oddsapi.Client, store *store.Store) *OddsService {
	return &OddsService{
		client: client,
		store:  store,
	}
}

// filterBookmakers removes bookmakers that aren't in the allowed list
func filterBookmakers(games []models.Game) []models.Game {
	for i := range games {
		var filtered []models.Bookmaker
		for _, bm := range games[i].Bookmakers {
			if allowedBookmakers[bm.Key] {
				filtered = append(filtered, bm)
			}
		}
		games[i].Bookmakers = filtered
	}
	return games
}

// FetchAndStoreOdds fetches odds from API and stores them
func (s *OddsService) FetchAndStoreOdds(sport models.Sport) ([]models.Game, error) {
	games, err := s.client.GetOdds(sport)
	if err != nil {
		return nil, err
	}
	games = filterBookmakers(games)
	s.store.UpdateGames(games)
	return games, nil
}

// GetGamesBySport returns games for a sport from the store
func (s *OddsService) GetGamesBySport(sport models.Sport) []models.Game {
	games := s.store.GetGamesBySport(sport)
	return filterBookmakers(games)
}

// GetGame returns a single game
func (s *OddsService) GetGame(id string) (models.Game, bool) {
	game, found := s.store.GetGame(id)
	if found {
		filtered := filterBookmakers([]models.Game{game})
		return filtered[0], true
	}
	return game, false
}

// CompareOdds analyzes a game and returns the best odds across bookmakers
func (s *OddsService) CompareOdds(game models.Game) models.OddsComparison {
	comparison := models.OddsComparison{
		GameID:       game.ID,
		HomeTeam:     game.HomeTeam,
		AwayTeam:     game.AwayTeam,
		CommenceTime: game.CommenceTime,
	}

	comparison.Moneyline = s.compareMoneyline(game)
	comparison.Spread = s.compareSpreads(game)
	comparison.Total = s.compareTotals(game)

	return comparison
}

func (s *OddsService) compareMoneyline(game models.Game) *models.MoneylineComparison {
	var allBookmakers []models.BookmakerOdds
	bestHome := models.BestOdds{Price: math.Inf(-1)}
	bestAway := models.BestOdds{Price: math.Inf(-1)}

	for _, bookmaker := range game.Bookmakers {
		for _, market := range bookmaker.Markets {
			if market.Key != models.MarketH2H {
				continue
			}

			var homePrice, awayPrice float64
			for _, outcome := range market.Outcomes {
				if outcome.Name == game.HomeTeam {
					homePrice = outcome.Price
				} else if outcome.Name == game.AwayTeam {
					awayPrice = outcome.Price
				}
			}

			if homePrice != 0 && awayPrice != 0 {
				allBookmakers = append(allBookmakers, models.BookmakerOdds{
					Bookmaker: bookmaker.Title,
					HomePrice: homePrice,
					AwayPrice: awayPrice,
				})

				if homePrice > bestHome.Price {
					bestHome.Price = homePrice
					bestHome.Bookmaker = bookmaker.Title
				}
				if awayPrice > bestAway.Price {
					bestAway.Price = awayPrice
					bestAway.Bookmaker = bookmaker.Title
				}
			}
		}
	}

	if len(allBookmakers) == 0 {
		return nil
	}

	return &models.MoneylineComparison{
		BestHome:      bestHome,
		BestAway:      bestAway,
		AllBookmakers: allBookmakers,
	}
}

func (s *OddsService) compareSpreads(game models.Game) *models.SpreadComparison {
	var allBookmakers []models.BookmakerSpreadOdds
	bestHome := models.BestSpreadOdds{Price: math.Inf(-1)}
	bestAway := models.BestSpreadOdds{Price: math.Inf(-1)}

	for _, bookmaker := range game.Bookmakers {
		for _, market := range bookmaker.Markets {
			if market.Key != models.MarketSpreads {
				continue
			}

			var homePrice, homePoint, awayPrice, awayPoint float64
			for _, outcome := range market.Outcomes {
				if outcome.Name == game.HomeTeam && outcome.Point != nil {
					homePrice = outcome.Price
					homePoint = *outcome.Point
				} else if outcome.Name == game.AwayTeam && outcome.Point != nil {
					awayPrice = outcome.Price
					awayPoint = *outcome.Point
				}
			}

			if homePrice != 0 && awayPrice != 0 {
				allBookmakers = append(allBookmakers, models.BookmakerSpreadOdds{
					Bookmaker: bookmaker.Title,
					HomePrice: homePrice,
					HomePoint: homePoint,
					AwayPrice: awayPrice,
					AwayPoint: awayPoint,
				})

				// For spreads, better odds = higher price at same or better point
				if homePrice > bestHome.Price {
					bestHome.Price = homePrice
					bestHome.Point = homePoint
					bestHome.Bookmaker = bookmaker.Title
				}
				if awayPrice > bestAway.Price {
					bestAway.Price = awayPrice
					bestAway.Point = awayPoint
					bestAway.Bookmaker = bookmaker.Title
				}
			}
		}
	}

	if len(allBookmakers) == 0 {
		return nil
	}

	return &models.SpreadComparison{
		BestHome:      bestHome,
		BestAway:      bestAway,
		AllBookmakers: allBookmakers,
	}
}

func (s *OddsService) compareTotals(game models.Game) *models.TotalComparison {
	var allBookmakers []models.BookmakerTotalOdds
	bestOver := models.BestTotalOdds{Price: math.Inf(-1)}
	bestUnder := models.BestTotalOdds{Price: math.Inf(-1)}

	for _, bookmaker := range game.Bookmakers {
		for _, market := range bookmaker.Markets {
			if market.Key != models.MarketTotals {
				continue
			}

			var overPrice, underPrice, point float64
			for _, outcome := range market.Outcomes {
				if outcome.Name == "Over" && outcome.Point != nil {
					overPrice = outcome.Price
					point = *outcome.Point
				} else if outcome.Name == "Under" && outcome.Point != nil {
					underPrice = outcome.Price
				}
			}

			if overPrice != 0 && underPrice != 0 {
				allBookmakers = append(allBookmakers, models.BookmakerTotalOdds{
					Bookmaker:  bookmaker.Title,
					OverPrice:  overPrice,
					UnderPrice: underPrice,
					Point:      point,
				})

				if overPrice > bestOver.Price {
					bestOver.Price = overPrice
					bestOver.Point = point
					bestOver.Bookmaker = bookmaker.Title
				}
				if underPrice > bestUnder.Price {
					bestUnder.Price = underPrice
					bestUnder.Point = point
					bestUnder.Bookmaker = bookmaker.Title
				}
			}
		}
	}

	if len(allBookmakers) == 0 {
		return nil
	}

	return &models.TotalComparison{
		BestOver:      bestOver,
		BestUnder:     bestUnder,
		AllBookmakers: allBookmakers,
	}
}
