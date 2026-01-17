function PlayerPropsTable({ category, bookmakers }) {
  if (!bookmakers || bookmakers.length === 0) {
    return null
  }

  const formatOdds = (price) => {
    if (price === undefined || price === null) return '-'
    return price > 0 ? `+${price}` : price.toString()
  }

  return (
    <div className="prop-category">
      <div className="prop-category-title">{category}</div>
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
          {bookmakers.map((bm, idx) => (
            <tr key={idx}>
              <td>{bm.title}</td>
              <td className="line-value">{bm.point}</td>
              <td>{formatOdds(bm.over_price)}</td>
              <td>{formatOdds(bm.under_price)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default PlayerPropsTable
