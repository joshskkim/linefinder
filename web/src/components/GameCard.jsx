import OddsTable from './OddsTable'

function GameCard({ game, onClick }) {
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
    </div>
  )
}

export default GameCard
