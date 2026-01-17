function SportTabs({ selectedSport, onSportChange }) {
  const sports = [
    { key: 'nfl', label: 'NFL' },
    { key: 'nba', label: 'NBA' },
  ]

  return (
    <div className="sport-tabs">
      {sports.map((sport) => (
        <button
          key={sport.key}
          className={`sport-tab ${selectedSport === sport.key ? 'active' : ''}`}
          onClick={() => onSportChange(sport.key)}
        >
          {sport.label}
        </button>
      ))}
    </div>
  )
}

export default SportTabs
