function PlayerPropsTable({ category, bookmakers, average }) {
  if (!bookmakers || bookmakers.length === 0) {
    return null
  }

  const formatOdds = (price) => {
    if (price === undefined || price === null) return '-'
    return price > 0 ? `+${price}` : price.toString()
  }

  return (
    <div className="prop-category">
      <div className="prop-category-title">
        <span>{category}</span>
        {average !== undefined && average !== null && (
          <span className="prop-average">
            L5 Avg: <strong>{average.toFixed(1)}</strong>
          </span>
        )}
      </div>
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
          {bookmakers.map((bm, idx) => {
            const lineVsAvg = average !== undefined && average !== null
              ? bm.point - average
              : null
            return (
              <tr key={idx}>
                <td>{bm.title}</td>
                <td className="line-value">
                  {bm.point}
                  {lineVsAvg !== null && (
                    <span className={`line-diff ${lineVsAvg > 0 ? 'above-avg' : lineVsAvg < 0 ? 'below-avg' : ''}`}>
                      ({lineVsAvg > 0 ? '+' : ''}{lineVsAvg.toFixed(1)})
                    </span>
                  )}
                </td>
                <td>{formatOdds(bm.over_price)}</td>
                <td>{formatOdds(bm.under_price)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export default PlayerPropsTable
