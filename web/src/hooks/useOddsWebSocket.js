import { useEffect, useRef, useCallback, useState } from 'react'

/**
 * Custom hook for WebSocket connection to receive real-time odds updates
 *
 * Features:
 * - Auto-reconnect with exponential backoff
 * - Connection state tracking
 * - Subscription management per sport
 * - Ping/pong for keepalive
 *
 * @param {string} sport - The sport to subscribe to ('nba' or 'nfl')
 * @param {function} onUpdate - Callback when new odds data arrives
 * @param {boolean} enabled - Whether WebSocket should be connected
 * @returns {object} - { connected, connecting, lastUpdate, error, reconnectAttempts }
 */
export function useOddsWebSocket(sport, onUpdate, enabled = true) {
  const ws = useRef(null)
  const reconnectTimeout = useRef(null)
  const pingInterval = useRef(null)

  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [lastUpdate, setLastUpdate] = useState(null)
  const [error, setError] = useState(null)
  const [status, setStatus] = useState(null) // 'polling_healthy', 'polling_degraded', etc.

  const reconnectAttempts = useRef(0)
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000 // 1 second

  // Calculate reconnect delay with exponential backoff and jitter
  const getReconnectDelay = useCallback(() => {
    const exponentialDelay = baseReconnectDelay * Math.pow(2, reconnectAttempts.current)
    const maxDelay = 30000 // Cap at 30 seconds
    const delay = Math.min(exponentialDelay, maxDelay)
    // Add jitter (±20%)
    const jitter = delay * 0.2 * (Math.random() - 0.5)
    return delay + jitter
  }, [])

  const connect = useCallback(() => {
    if (!enabled) return
    if (ws.current?.readyState === WebSocket.OPEN) return
    if (ws.current?.readyState === WebSocket.CONNECTING) return

    setConnecting(true)
    setError(null)

    // Determine WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/api/ws`

    console.log(`[WebSocket] Connecting to ${wsUrl}...`)
    ws.current = new WebSocket(wsUrl)

    ws.current.onopen = () => {
      console.log('[WebSocket] Connected')
      setConnected(true)
      setConnecting(false)
      setError(null)
      reconnectAttempts.current = 0

      // Subscribe to the current sport
      if (sport) {
        console.log(`[WebSocket] Subscribing to ${sport}`)
        ws.current.send(JSON.stringify({
          type: 'subscribe',
          sport: sport
        }))
      }

      // Start ping interval for keepalive
      pingInterval.current = setInterval(() => {
        if (ws.current?.readyState === WebSocket.OPEN) {
          ws.current.send(JSON.stringify({ type: 'ping' }))
        }
      }, 30000) // Ping every 30 seconds
    }

    ws.current.onmessage = (event) => {
      try {
        // Handle multiple messages (batched)
        const messages = event.data.split('\n').filter(Boolean)

        for (const msgStr of messages) {
          const data = JSON.parse(msgStr)

          switch (data.type) {
            case 'odds_update':
              if (data.sport === sport && data.games) {
                console.log(`[WebSocket] Received odds update for ${sport}: ${data.games.length} games`)
                setLastUpdate(new Date(data.timestamp))
                onUpdate(data.games)
              }
              break

            case 'status':
              console.log(`[WebSocket] Status: ${data.status}`)
              setStatus(data.status)
              break

            case 'pong':
              // Keepalive response, no action needed
              break

            case 'error':
              console.error(`[WebSocket] Server error: ${data.error}`)
              setError(data.error)
              break

            default:
              console.log(`[WebSocket] Unknown message type: ${data.type}`)
          }
        }
      } catch (err) {
        console.error('[WebSocket] Failed to parse message:', err)
      }
    }

    ws.current.onclose = (event) => {
      console.log(`[WebSocket] Disconnected (code: ${event.code}, reason: ${event.reason})`)
      setConnected(false)
      setConnecting(false)

      // Clear ping interval
      if (pingInterval.current) {
        clearInterval(pingInterval.current)
        pingInterval.current = null
      }

      // Attempt reconnect if enabled and not at max attempts
      if (enabled && reconnectAttempts.current < maxReconnectAttempts) {
        const delay = getReconnectDelay()
        reconnectAttempts.current++
        console.log(`[WebSocket] Reconnecting in ${Math.round(delay)}ms (attempt ${reconnectAttempts.current}/${maxReconnectAttempts})`)

        reconnectTimeout.current = setTimeout(connect, delay)
      } else if (reconnectAttempts.current >= maxReconnectAttempts) {
        setError('Max reconnection attempts reached. Please refresh the page.')
      }
    }

    ws.current.onerror = (event) => {
      console.error('[WebSocket] Error:', event)
      setError('WebSocket connection error')
    }
  }, [sport, onUpdate, enabled, getReconnectDelay])

  // Connect when enabled changes
  useEffect(() => {
    if (enabled) {
      connect()
    } else {
      // Disconnect if disabled
      if (ws.current) {
        ws.current.close()
        ws.current = null
      }
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current)
        reconnectTimeout.current = null
      }
      if (pingInterval.current) {
        clearInterval(pingInterval.current)
        pingInterval.current = null
      }
      setConnected(false)
      setConnecting(false)
      reconnectAttempts.current = 0
    }

    return () => {
      if (ws.current) {
        ws.current.close()
        ws.current = null
      }
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current)
      }
      if (pingInterval.current) {
        clearInterval(pingInterval.current)
      }
    }
  }, [enabled, connect])

  // Re-subscribe when sport changes
  useEffect(() => {
    if (connected && ws.current?.readyState === WebSocket.OPEN && sport) {
      console.log(`[WebSocket] Switching subscription to ${sport}`)
      ws.current.send(JSON.stringify({
        type: 'subscribe',
        sport: sport
      }))
    }
  }, [sport, connected])

  // Manual reconnect function
  const reconnect = useCallback(() => {
    reconnectAttempts.current = 0
    if (ws.current) {
      ws.current.close()
    }
    connect()
  }, [connect])

  return {
    connected,
    connecting,
    lastUpdate,
    error,
    status,
    reconnectAttempts: reconnectAttempts.current,
    reconnect
  }
}

export default useOddsWebSocket
