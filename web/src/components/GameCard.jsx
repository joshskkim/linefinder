import { useState, useEffect } from 'react'
import OddsTable from './OddsTable'
import InjuryTable from './InjuryTable'

function GameCard({ game, sport, onClick }) {
  const [injuries, setInjuries] = useState(null)
  const [showInjuries, setShowInjuries] = useState(false)

  useEffect(() => {
    const fetchInjuries = async () => {
      try {
        const response = await fetch(`/api/injuries/${sport}/${game.id}`)
        if (response.ok) {
          const data = await response.json()
          setInjuries(data)
        }
      } catch (err) {
        console.error('Failed to fetch injuries:', err)
      }
    }

    fetchInjuries()
  }, [game.id, sport])

  const formatGameTime = (timeStr) => {
    const date = new Date(timeStr)
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    })
  }

  const getMarketData = (marketKey) => {
    const marketData = []
    for (const bookmaker of game.bookmakers || []) {
      const market = bookmaker.markets?.find((m) => m.key === marketKey)
      if (market) {
        marketData.push({
          bookmaker: bookmaker.title,
          outcomes: market.outcomes,
        })
      }
    }
    return marketData
  }

  const injuryCount = (injuries?.home_team?.players?.length || 0) + (injuries?.away_team?.players?.length || 0)

  return (
    <div className="game-card">
      <div className="game-header" onClick={onClick}>
        <div>
          <span className="game-teams">
            {game.away_team} @ {game.home_team}
          </span>
          <span className="game-header-hint">Click for player props</span>
        </div>
        <span className="game-time">{formatGameTime(game.commence_time)}</span>
      </div>

      <div className="game-content">
        <div className="game-odds">
          <OddsTable
            title="MONEYLINE"
            marketKey="h2h"
            data={getMarketData('h2h')}
            homeTeam={game.home_team}
            awayTeam={game.away_team}
          />

          <OddsTable
            title="SPREAD"
            marketKey="spreads"
            data={getMarketData('spreads')}
            homeTeam={game.home_team}
            awayTeam={game.away_team}
          />

          <OddsTable
            title="TOTAL"
            marketKey="totals"
            data={getMarketData('totals')}
            homeTeam={game.home_team}
            awayTeam={game.away_team}
          />
        </div>

        <div className="game-injuries">
          <div
            className="injuries-header"
            onClick={(e) => {
              e.stopPropagation()
              setShowInjuries(!showInjuries)
            }}
          >
            <span className="injuries-title">
              Injuries {injuryCount > 0 && <span className="injury-count">{injuryCount}</span>}
            </span>
            <span className="injuries-toggle">{showInjuries ? '−' : '+'}</span>
          </div>
          {showInjuries && <InjuryTable injuries={injuries} />}
        </div>
      </div>
    </div>
  )
}

export default GameCard
