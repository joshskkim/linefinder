import { useState, useEffect } from 'react'
import SportTabs from './components/SportTabs'
import GameCard from './components/GameCard'
import GameDetail from './components/GameDetail'

function App() {
  const [selectedSport, setSelectedSport] = useState('nba')
  const [games, setGames] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [refreshing, setRefreshing] = useState(false)
  const [selectedGame, setSelectedGame] = useState(null)

  const fetchOdds = async (sport) => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(`/api/odds/${sport}`)
      if (!response.ok) {
        throw new Error('Failed to fetch odds')
      }
      const data = await response.json()
      setGames(data.games || [])
    } catch (err) {
      setError(err.message)
      setGames([])
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    setError(null)
    try {
      const response = await fetch(`/api/refresh/${selectedSport}`, {
        method: 'POST',
      })
      if (!response.ok) {
        throw new Error('Failed to refresh odds')
      }
      await fetchOdds(selectedSport)
    } catch (err) {
      setError(err.message)
    } finally {
      setRefreshing(false)
    }
  }

  const handleSportChange = (sport) => {
    setSelectedSport(sport)
    setSelectedGame(null)
    fetchOdds(sport)
  }

  const handleGameClick = (game) => {
    setSelectedGame(game)
  }

  const handleBackToGames = () => {
    setSelectedGame(null)
  }

  useEffect(() => {
    fetchOdds(selectedSport)
  }, [])

  // Show game detail page if a game is selected
  if (selectedGame) {
    return (
      <div className="app">
        <GameDetail
          game={selectedGame}
          sport={selectedSport}
          onBack={handleBackToGames}
        />
      </div>
    )
  }

  return (
    <div className="app">
      <header className="header">
        <button
          className="refresh-button"
          onClick={handleRefresh}
          disabled={refreshing || loading}
        >
          {refreshing ? 'Refreshing...' : 'Refresh'}
        </button>
        <SportTabs
          selectedSport={selectedSport}
          onSportChange={handleSportChange}
        />
      </header>

      <main className="main">
        {error && <div className="error">{error}</div>}

        {loading && <div className="loading">Loading odds...</div>}

        {!loading && !error && games.length === 0 && (
          <div className="empty">
            No games found. Click Refresh to fetch odds.
          </div>
        )}

        {!loading && games.map((game) => (
          <GameCard
            key={game.id}
            game={game}
            onClick={() => handleGameClick(game)}
          />
        ))}
      </main>
    </div>
  )
}

export default App
