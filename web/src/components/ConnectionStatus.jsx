function ConnectionStatus({
  connected,
  connecting,
  lastUpdate,
  error,
  status,
  enabled,
  onToggle,
  onReconnect
}) {
  const formatTime = (date) => {
    if (!date) return null
    return date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      second: '2-digit',
      hour12: true
    })
  }

  const getStatusIndicator = () => {
    if (!enabled) {
      return { class: 'status-disabled', text: 'Live Off' }
    }
    if (connecting) {
      return { class: 'status-connecting', text: 'Connecting...' }
    }
    if (connected) {
      if (status === 'polling_degraded') {
        return { class: 'status-degraded', text: 'Degraded' }
      }
      return { class: 'status-connected', text: 'Live' }
    }
    if (error) {
      return { class: 'status-error', text: 'Error' }
    }
    return { class: 'status-disconnected', text: 'Disconnected' }
  }

  const statusInfo = getStatusIndicator()

  return (
    <div className="connection-status">
      <div className="connection-indicator">
        <span className={`status-dot ${statusInfo.class}`}></span>
        <span className="status-text">{statusInfo.text}</span>
      </div>

      {lastUpdate && connected && (
        <span className="last-update">
          Updated: {formatTime(lastUpdate)}
        </span>
      )}

      <div className="connection-controls">
        <button
          className="ws-toggle-btn"
          onClick={onToggle}
          title={enabled ? 'Disable live updates' : 'Enable live updates'}
        >
          {enabled ? 'Disable' : 'Enable'}
        </button>

        {!connected && enabled && !connecting && (
          <button
            className="ws-reconnect-btn"
            onClick={onReconnect}
            title="Reconnect to live updates"
          >
            Reconnect
          </button>
        )}
      </div>

      {error && (
        <span className="connection-error" title={error}>
          !
        </span>
      )}
    </div>
  )
}

export default ConnectionStatus
