function InjuryTable({ injuries }) {
  if (!injuries) {
    return null
  }

  const getStatusColor = (status) => {
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

  const renderTeamInjuries = (teamInjuries) => {
    if (!teamInjuries?.players || teamInjuries.players.length === 0) {
      return <div className="no-injuries">No injuries reported</div>
    }

    return (
      <div className="injury-list">
        {teamInjuries.players.map((player, idx) => (
          <div key={idx} className="injury-row">
            <span className="injury-player">
              {player.name}
              <span className="injury-position">{player.position}</span>
            </span>
            <span className={`injury-status ${getStatusColor(player.status)}`}>
              {player.status}
            </span>
            <span className="injury-detail">
              {player.body_part}
            </span>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="injury-table">
      <div className="injury-team-section">
        <div className="injury-team-header">{injuries.away_team?.team}</div>
        {renderTeamInjuries(injuries.away_team)}
      </div>
      <div className="injury-team-section">
        <div className="injury-team-header">{injuries.home_team?.team}</div>
        {renderTeamInjuries(injuries.home_team)}
      </div>
    </div>
  )
}

export default InjuryTable
