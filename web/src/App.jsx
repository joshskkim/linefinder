import { useState, useEffect, useCallback } from 'react'
import SportTabs from './components/SportTabs'
import GameCard from './components/GameCard'
import GameDetail from './components/GameDetail'
import ConnectionStatus from './components/ConnectionStatus'
import Settings from './components/Settings'
import { useOddsWebSocket } from './hooks/useOddsWebSocket'

function App() {
  const [selectedSport, setSelectedSport] = useState('nba')
  const [games, setGames] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [refreshing, setRefreshing] = useState(false)
  const [selectedGame, setSelectedGame] = useState(null)
  const [wsEnabled, setWsEnabled] = useState(true)
  const [settingsOpen, setSettingsOpen] = useState(false)

  // WebSocket handler for real-time updates
  const handleOddsUpdate = useCallback((newGames) => {
    setGames(newGames)
    setError(null)
  }, [])

  // WebSocket connection
  const {
    connected,
    connecting,
    lastUpdate,
    error: wsError,
    status: wsStatus,
    reconnect
  } = useOddsWebSocket(selectedSport, handleOddsUpdate, wsEnabled)

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

  const toggleWebSocket = () => {
    setWsEnabled(!wsEnabled)
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
        <div className="header-left">
          <button
            className="refresh-button"
            onClick={handleRefresh}
            disabled={refreshing || loading}
          >
            {refreshing ? 'Refreshing...' : 'Refresh'}
          </button>
          <ConnectionStatus
            connected={connected}
            connecting={connecting}
            lastUpdate={lastUpdate}
            error={wsError}
            status={wsStatus}
            enabled={wsEnabled}
            onToggle={toggleWebSocket}
            onReconnect={reconnect}
          />
          <button
            className="settings-button"
            onClick={() => setSettingsOpen(true)}
          >
            Settings
          </button>
        </div>
        <SportTabs
          selectedSport={selectedSport}
          onSportChange={handleSportChange}
        />
      </header>

      <Settings isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />

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
            sport={selectedSport}
            onClick={() => handleGameClick(game)}
          />
        ))}
      </main>
    </div>
  )
}

export default App
