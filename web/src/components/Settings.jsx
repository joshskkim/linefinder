import { useState, useEffect } from 'react'
import {
  isPushSupported,
  getNotificationPermission,
  setupPushNotifications,
  unsubscribeFromPush,
  getServiceWorkerRegistration,
  isSubscribedToPush,
  sendTestNotification
} from '../utils/pushNotifications'

function Settings({ isOpen, onClose }) {
  const [preferences, setPreferences] = useState(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(null)

  // Push notification state
  const [pushSupported] = useState(isPushSupported())
  const [pushPermission, setPushPermission] = useState(getNotificationPermission())
  const [pushSubscribed, setPushSubscribed] = useState(false)
  const [pushLoading, setPushLoading] = useState(false)

  // Load preferences
  useEffect(() => {
    if (isOpen) {
      loadPreferences()
      checkPushStatus()
    }
  }, [isOpen])

  const loadPreferences = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch('/api/preferences')
      if (!response.ok) throw new Error('Failed to load preferences')
      const data = await response.json()
      setPreferences(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const checkPushStatus = async () => {
    if (!pushSupported) return
    const registration = await getServiceWorkerRegistration()
    if (registration) {
      const subscribed = await isSubscribedToPush(registration)
      setPushSubscribed(subscribed)
    }
    setPushPermission(getNotificationPermission())
  }

  const savePreferences = async (updates) => {
    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      const response = await fetch('/api/preferences', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...preferences, ...updates })
      })
      if (!response.ok) throw new Error('Failed to save preferences')
      setPreferences(prev => ({ ...prev, ...updates }))
      setSuccess('Preferences saved')
      setTimeout(() => setSuccess(null), 2000)
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleEnablePush = async () => {
    setPushLoading(true)
    setError(null)
    try {
      await setupPushNotifications()
      await savePreferences({ enable_push: true })
      setPushSubscribed(true)
      setPushPermission('granted')
      setSuccess('Push notifications enabled')
    } catch (err) {
      setError(err.message)
    } finally {
      setPushLoading(false)
    }
  }

  const handleDisablePush = async () => {
    setPushLoading(true)
    setError(null)
    try {
      const registration = await getServiceWorkerRegistration()
      if (registration) {
        await unsubscribeFromPush(registration)
      }
      await savePreferences({ enable_push: false })
      setPushSubscribed(false)
      setSuccess('Push notifications disabled')
    } catch (err) {
      setError(err.message)
    } finally {
      setPushLoading(false)
    }
  }

  const handleTestNotification = async () => {
    try {
      await sendTestNotification()
    } catch (err) {
      setError(err.message)
    }
  }

  const handleUnsubscribeAll = async () => {
    if (!confirm('Are you sure you want to unsubscribe from all notifications?')) {
      return
    }
    setPushLoading(true)
    setError(null)
    try {
      const registration = await getServiceWorkerRegistration()
      if (registration) {
        await unsubscribeFromPush(registration)
      }
      setPreferences(prev => ({
        ...prev,
        enable_push: false,
        enable_websocket: false
      }))
      setPushSubscribed(false)
      setSuccess('Unsubscribed from all notifications')
    } catch (err) {
      setError(err.message)
    } finally {
      setPushLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="settings-overlay" onClick={onClose}>
      <div className="settings-modal" onClick={e => e.stopPropagation()}>
        <div className="settings-header">
          <h2>Settings</h2>
          <button className="close-button" onClick={onClose}>&times;</button>
        </div>

        {loading && <div className="settings-loading">Loading...</div>}

        {error && <div className="settings-error">{error}</div>}
        {success && <div className="settings-success">{success}</div>}

        {!loading && preferences && (
          <div className="settings-content">
            {/* Push Notifications */}
            <section className="settings-section">
              <h3>Push Notifications</h3>
              {!pushSupported ? (
                <p className="settings-note">Push notifications are not supported in this browser.</p>
              ) : pushPermission === 'denied' ? (
                <p className="settings-note">
                  Notification permission was denied. Please enable notifications in your browser settings.
                </p>
              ) : (
                <div className="settings-row">
                  <div className="settings-label">
                    <span>Enable Push Notifications</span>
                    <span className="settings-desc">Get alerts when value opportunities are detected</span>
                  </div>
                  <div className="settings-control">
                    {pushSubscribed ? (
                      <div className="push-controls">
                        <button
                          className="btn-secondary"
                          onClick={handleTestNotification}
                          disabled={pushLoading}
                        >
                          Test
                        </button>
                        <button
                          className="btn-danger"
                          onClick={handleDisablePush}
                          disabled={pushLoading}
                        >
                          {pushLoading ? 'Disabling...' : 'Disable'}
                        </button>
                      </div>
                    ) : (
                      <button
                        className="btn-primary"
                        onClick={handleEnablePush}
                        disabled={pushLoading}
                      >
                        {pushLoading ? 'Enabling...' : 'Enable'}
                      </button>
                    )}
                  </div>
                </div>
              )}
            </section>

            {/* WebSocket Alerts */}
            <section className="settings-section">
              <h3>Real-time Alerts</h3>
              <div className="settings-row">
                <div className="settings-label">
                  <span>Enable WebSocket Alerts</span>
                  <span className="settings-desc">Show in-app alerts in real-time</span>
                </div>
                <div className="settings-control">
                  <label className="toggle">
                    <input
                      type="checkbox"
                      checked={preferences.enable_websocket}
                      onChange={e => savePreferences({ enable_websocket: e.target.checked })}
                      disabled={saving}
                    />
                    <span className="toggle-slider"></span>
                  </label>
                </div>
              </div>
            </section>

            {/* Alert Thresholds */}
            <section className="settings-section">
              <h3>Value Alert Thresholds</h3>
              <p className="settings-note">
                Alert when line differs from average by more than this many units
              </p>

              <div className="settings-row">
                <div className="settings-label">Points</div>
                <div className="settings-control">
                  <input
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="10"
                    value={preferences.threshold_points}
                    onChange={e => savePreferences({ threshold_points: parseFloat(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="settings-row">
                <div className="settings-label">Rebounds</div>
                <div className="settings-control">
                  <input
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="10"
                    value={preferences.threshold_rebounds}
                    onChange={e => savePreferences({ threshold_rebounds: parseFloat(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="settings-row">
                <div className="settings-label">Assists</div>
                <div className="settings-control">
                  <input
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="10"
                    value={preferences.threshold_assists}
                    onChange={e => savePreferences({ threshold_assists: parseFloat(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="settings-row">
                <div className="settings-label">Three Pointers</div>
                <div className="settings-control">
                  <input
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="10"
                    value={preferences.threshold_threes}
                    onChange={e => savePreferences({ threshold_threes: parseFloat(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="settings-row">
                <div className="settings-label">Other Props</div>
                <div className="settings-control">
                  <input
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="10"
                    value={preferences.threshold_default}
                    onChange={e => savePreferences({ threshold_default: parseFloat(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>
            </section>

            {/* Quiet Hours */}
            <section className="settings-section">
              <h3>Quiet Hours</h3>
              <p className="settings-note">
                Pause push notifications during these hours
              </p>

              <div className="settings-row">
                <div className="settings-label">Start</div>
                <div className="settings-control">
                  <input
                    type="time"
                    value={preferences.quiet_start || '23:00'}
                    onChange={e => savePreferences({ quiet_start: e.target.value })}
                    disabled={saving}
                  />
                </div>
              </div>

              <div className="settings-row">
                <div className="settings-label">End</div>
                <div className="settings-control">
                  <input
                    type="time"
                    value={preferences.quiet_end || '08:00'}
                    onChange={e => savePreferences({ quiet_end: e.target.value })}
                    disabled={saving}
                  />
                </div>
              </div>
            </section>

            {/* Rate Limits */}
            <section className="settings-section">
              <h3>Rate Limits</h3>
              <div className="settings-row">
                <div className="settings-label">
                  <span>Max Push Notifications per Hour</span>
                </div>
                <div className="settings-control">
                  <input
                    type="number"
                    min="1"
                    max="100"
                    value={preferences.rate_limit_push || 20}
                    onChange={e => savePreferences({ rate_limit_push: parseInt(e.target.value) })}
                    disabled={saving}
                  />
                </div>
              </div>
            </section>

            {/* Unsubscribe */}
            <section className="settings-section settings-danger">
              <h3>Unsubscribe</h3>
              <div className="settings-row">
                <div className="settings-label">
                  <span>Unsubscribe from All</span>
                  <span className="settings-desc">Disable all notifications</span>
                </div>
                <div className="settings-control">
                  <button
                    className="btn-danger"
                    onClick={handleUnsubscribeAll}
                    disabled={pushLoading}
                  >
                    Unsubscribe
                  </button>
                </div>
              </div>
            </section>
          </div>
        )}
      </div>
    </div>
  )
}

export default Settings
