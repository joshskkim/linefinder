import { useState, useEffect } from 'react'
import PlayerPropsTable from './PlayerPropsTable'

function GameDetail({ game, sport, onBack }) {
  const [playerProps, setPlayerProps] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const fetchPlayerProps = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await fetch(`/api/props/${sport}/${game.id}`)
        if (!response.ok) {
          throw new Error('Failed to fetch player props')
        }
        const data = await response.json()
        setPlayerProps(data)
      } catch (err) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    fetchPlayerProps()
  }, [game.id, sport])

  const formatGameTime = (timeStr) => {
    const date = new Date(timeStr)
    return date.toLocaleString('en-US', {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    })
  }

  return (
    <div className="game-detail">
      <button className="back-button" onClick={onBack}>
        &larr; Back to Games
      </button>

      <div className="game-detail-header">
        <div className="game-detail-teams">
          {game.away_team} @ {game.home_team}
        </div>
        <div className="game-detail-time">
          {formatGameTime(game.commence_time)}
        </div>
      </div>

      {loading && <div className="loading">Loading player props...</div>}

      {error && <div className="error">{error}</div>}

      {!loading && !error && playerProps && (
        <div className="players-list">
          {playerProps.players?.map((player, idx) => (
            <div key={idx} className="player-section">
              <div className="player-name">
                {player.name}
                <span className="player-team"> - {player.team}</span>
              </div>
              <div className="player-props">
                {player.props?.map((prop, propIdx) => (
                  <PlayerPropsTable
                    key={propIdx}
                    category={prop.category}
                    bookmakers={prop.bookmakers}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {!loading && !error && (!playerProps || playerProps.players?.length === 0) && (
        <div className="empty">No player props available for this game.</div>
      )}
    </div>
  )
}

export default GameDetail
