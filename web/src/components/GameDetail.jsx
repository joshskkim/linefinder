import { useState, useEffect } from 'react'
import PlayerPropsTable from './PlayerPropsTable'

function GameDetail({ game, sport, onBack }) {
  const [playerProps, setPlayerProps] = useState(null)
  const [playerAverages, setPlayerAverages] = useState([])
  const [injuries, setInjuries] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      setError(null)
      try {
        // Fetch all data in parallel
        const [propsRes, avgRes, injRes] = await Promise.all([
          fetch(`/api/props/${sport}/${game.id}`),
          fetch(`/api/averages/${sport}/${game.id}`),
          fetch(`/api/injuries/${sport}/${game.id}`),
        ])

        if (!propsRes.ok) {
          throw new Error('Failed to fetch player props')
        }
        const propsData = await propsRes.json()
        setPlayerProps(propsData)

        if (avgRes.ok) {
          const avgData = await avgRes.json()
          setPlayerAverages(avgData)
        }

        if (injRes.ok) {
          const injData = await injRes.json()
          setInjuries(injData)
        }
      } catch (err) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
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

  const getPlayerInjuryStatus = (playerName) => {
    if (!injuries) return null

    const allInjured = [
      ...(injuries.home_team?.players || []),
      ...(injuries.away_team?.players || []),
    ]

    const injured = allInjured.find(
      (p) => p.name.toLowerCase() === playerName.toLowerCase()
    )
    return injured?.status || null
  }

  const getPlayerAverages = (playerName) => {
    const player = playerAverages.find(
      (p) => p.name.toLowerCase() === playerName.toLowerCase()
    )
    return player?.averages || null
  }

  const getStatusClass = (status) => {
    switch (status?.toLowerCase()) {
      case 'out':
      case 'injured':
        return 'status-out'
      case 'doubtful':
        return 'status-doubtful'
      case 'questionable':
        return 'status-questionable'
      case 'probable':
        return 'status-probable'
      default:
        return ''
    }
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
          {playerProps.players?.map((player, idx) => {
            const injuryStatus = getPlayerInjuryStatus(player.name)
            const averages = getPlayerAverages(player.name)

            return (
              <div key={idx} className="player-section">
                <div className="player-name">
                  {player.name}
                  {injuryStatus && (
                    <span className={`player-injury-badge ${getStatusClass(injuryStatus)}`}>
                      {injuryStatus}
                    </span>
                  )}
                  <span className="player-team"> - {player.team}</span>
                </div>
                {averages && (
                  <div className="player-averages">
                    <strong>Last 5 Games Avg:</strong>{' '}
                    {Object.entries(averages).map(([key, val], i) => (
                      <span key={key}>
                        {key}: {val.toFixed(1)}
                        {i < Object.entries(averages).length - 1 ? ' | ' : ''}
                      </span>
                    ))}
                  </div>
                )}
                <div className="player-props">
                  {player.props?.map((prop, propIdx) => (
                    <PlayerPropsTable
                      key={propIdx}
                      category={prop.category}
                      bookmakers={prop.bookmakers}
                      average={averages?.[prop.category]}
                    />
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {!loading && !error && (!playerProps || playerProps.players?.length === 0) && (
        <div className="empty">No player props available for this game.</div>
      )}
    </div>
  )
}

export default GameDetail
