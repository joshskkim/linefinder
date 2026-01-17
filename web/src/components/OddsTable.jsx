function OddsTable({ title, marketKey, data, homeTeam, awayTeam }) {
  if (!data || data.length === 0) {
    return null
  }

  const formatOdds = (price) => {
    if (price === undefined || price === null) return '-'
    return price > 0 ? `+${price}` : price.toString()
  }

  const formatPoint = (point) => {
    if (point === undefined || point === null) return ''
    return point > 0 ? `+${point}` : point.toString()
  }

  const getOutcome = (outcomes, name) => {
    return outcomes?.find((o) => o.name === name)
  }

  const renderH2HTable = () => {
    return (
      <table className="odds-table">
        <thead>
          <tr>
            <th>Sportsbook</th>
            <th>{awayTeam}</th>
            <th>{homeTeam}</th>
          </tr>
        </thead>
        <tbody>
          {data.map((item, idx) => {
            const homeOdds = getOutcome(item.outcomes, homeTeam)
            const awayOdds = getOutcome(item.outcomes, awayTeam)
            return (
              <tr key={idx}>
                <td>{item.bookmaker}</td>
                <td>{formatOdds(awayOdds?.price)}</td>
                <td>{formatOdds(homeOdds?.price)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    )
  }

  const renderSpreadsTable = () => {
    return (
      <table className="odds-table">
        <thead>
          <tr>
            <th>Sportsbook</th>
            <th>{awayTeam}</th>
            <th>{homeTeam}</th>
          </tr>
        </thead>
        <tbody>
          {data.map((item, idx) => {
            const homeOdds = getOutcome(item.outcomes, homeTeam)
            const awayOdds = getOutcome(item.outcomes, awayTeam)
            return (
              <tr key={idx}>
                <td>{item.bookmaker}</td>
                <td>
                  <span className="spread-point">{formatPoint(awayOdds?.point)}</span>
                  <span className="spread-odds">{formatOdds(awayOdds?.price)}</span>
                </td>
                <td>
                  <span className="spread-point">{formatPoint(homeOdds?.point)}</span>
                  <span className="spread-odds">{formatOdds(homeOdds?.price)}</span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    )
  }

  const renderTotalsTable = () => {
    return (
      <>
        <div className="odds-table-title">{title}</div>
        <table className="odds-table">
          <thead>
            <tr>
              <th>Sportsbook</th>
              <th>Line</th>
              <th>Over</th>
              <th>Under</th>
            </tr>
          </thead>
          <tbody>
            {data.map((item, idx) => {
              const overOdds = getOutcome(item.outcomes, 'Over')
              const underOdds = getOutcome(item.outcomes, 'Under')
              const line = overOdds?.point ?? underOdds?.point
              return (
                <tr key={idx}>
                  <td>{item.bookmaker}</td>
                  <td className="line-value">{line !== undefined && line !== null ? line : '-'}</td>
                  <td>{formatOdds(overOdds?.price)}</td>
                  <td>{formatOdds(underOdds?.price)}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </>
    )
  }

  if (marketKey === 'totals') {
    return <div className="odds-section">{renderTotalsTable()}</div>
  }

  return (
    <div className="odds-section">
      <div className="odds-table-title">{title}</div>
      {marketKey === 'h2h' ? renderH2HTable() : renderSpreadsTable()}
    </div>
  )
}

export default OddsTable
