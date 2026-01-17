package store

import (
	"sync"
	"time"

	"github.com/joshuakim/linefinder/internal/models"
)

// Store holds games data in memory
type Store struct {
	mu          sync.RWMutex
	games       map[string]models.Game // keyed by game ID
	lastUpdated time.Time
}

// New creates a new in-memory store
func New() *Store {
	return &Store{
		games: make(map[string]models.Game),
	}
}

// UpdateGames replaces all games for a given sport
func (s *Store) UpdateGames(games []models.Game) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, game := range games {
		s.games[game.ID] = game
	}
	s.lastUpdated = time.Now()
}

// GetGame returns a single game by ID
func (s *Store) GetGame(id string) (models.Game, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	game, ok := s.games[id]
	return game, ok
}

// GetGamesBySport returns all games for a specific sport
func (s *Store) GetGamesBySport(sport models.Sport) []models.Game {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Game
	for _, game := range s.games {
		if game.SportKey == sport {
			result = append(result, game)
		}
	}
	return result
}

// GetAllGames returns all stored games
func (s *Store) GetAllGames() []models.Game {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.Game, 0, len(s.games))
	for _, game := range s.games {
		result = append(result, game)
	}
	return result
}

// LastUpdated returns when the store was last updated
func (s *Store) LastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdated
}

// Clear removes all games from the store
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games = make(map[string]models.Game)
}
