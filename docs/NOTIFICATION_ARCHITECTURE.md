# Value Alert Notification System Architecture

## Overview

Alert users when betting lines present potential value opportunities based on player performance averages.

**Trigger Condition**: Line is > N units away from player's L5 average (default: 2 units)

Example:
- Player averages 25.5 points over last 5 games
- Line is set at 22.5 points
- Difference: 3.0 units → **ALERT: Potential value on OVER**

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              BACKEND                                         │
│                                                                              │
│  ┌──────────────┐      ┌──────────────────┐      ┌───────────────────┐     │
│  │   Polling    │─────▶│  Value Detection │─────▶│  Notification     │     │
│  │   Service    │      │     Service      │      │     Queue         │     │
│  └──────────────┘      └──────────────────┘      └─────────┬─────────┘     │
│         │                      │                           │               │
│         │                      │                           ▼               │
│         │              ┌───────▼───────┐         ┌─────────────────┐       │
│         │              │    Player     │         │  Notification   │       │
│         │              │   Averages    │         │   Dispatcher    │       │
│         │              │    Store      │         └────────┬────────┘       │
│         │              └───────────────┘                  │               │
│         │                                    ┌────────────┼────────────┐   │
│         │                                    ▼            ▼            ▼   │
│         │                               ┌────────┐  ┌─────────┐  ┌─────┐  │
│         │                               │  Push  │  │  Email  │  │ SMS │  │
│         │                               └────────┘  └─────────┘  └─────┘  │
│         │                                                                  │
│         ▼                                                                  │
│  ┌──────────────┐      ┌──────────────────┐                               │
│  │  WebSocket   │─────▶│  Frontend        │  (real-time value alerts)     │
│  │     Hub      │      │  Notification    │                               │
│  └──────────────┘      └──────────────────┘                               │
│                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           DATA STORES                                        │
│                                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │ User Preferences │  │  Alert History   │  │   Rate Limits    │          │
│  │                  │  │  (deduplication) │  │   (per channel)  │          │
│  │ - channels       │  │                  │  │                  │          │
│  │ - threshold      │  │ - last alerted   │  │ - email: 10/hr   │          │
│  │ - sports         │  │ - player+prop    │  │ - sms: 5/hr      │          │
│  │ - quiet hours    │  │ - cooldown       │  │ - push: 20/hr    │          │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Value Detection Logic

```go
type ValueAlert struct {
    PlayerName    string
    Team          string
    Sport         string
    GameID        string
    GameTime      time.Time

    PropCategory  string    // "Points", "Rebounds", etc.
    Line          float64   // Current betting line
    Average       float64   // L5 average
    Difference    float64   // Line - Average

    Direction     string    // "over" or "under"
    Confidence    string    // "high", "medium" based on difference magnitude

    BestOdds      float64   // Best available odds for this bet
    Bookmaker     string    // Which book has best odds

    DetectedAt    time.Time
}

func DetectValue(prop PlayerProp, average float64, threshold float64) *ValueAlert {
    diff := prop.Line - average
    absDiff := math.Abs(diff)

    // No alert if within threshold
    if absDiff < threshold {
        return nil
    }

    alert := &ValueAlert{
        Line:       prop.Line,
        Average:    average,
        Difference: diff,
    }

    // Determine direction and confidence
    if diff < 0 {
        // Line is BELOW average → value on OVER
        alert.Direction = "over"
    } else {
        // Line is ABOVE average → value on UNDER
        alert.Direction = "under"
    }

    // Confidence based on magnitude
    switch {
    case absDiff >= 4:
        alert.Confidence = "high"
    case absDiff >= 3:
        alert.Confidence = "medium"
    default:
        alert.Confidence = "low"
    }

    return alert
}
```

### Confidence Levels

| Difference | Confidence | Notification Priority |
|------------|------------|----------------------|
| 2.0 - 2.9  | Low        | WebSocket only (real-time UI) |
| 3.0 - 3.9  | Medium     | WebSocket + Push |
| 4.0+       | High       | All channels (Push + Email + SMS) |

---

## Notification Channels

### 1. WebSocket (Real-time UI)
- **Always on** for connected users
- No external service needed
- Instant delivery
- Shows alert badge/toast in UI

### 2. Web Push Notifications
- Uses Web Push API with service worker
- Works even when browser tab is closed
- Free (no external service needed)
- Requires user permission

### 3. Email
- **Provider Options**:
  - SendGrid (100 emails/day free)
  - AWS SES ($0.10 per 1000 emails)
  - Mailgun (5000 emails/month free)
  - Resend (3000 emails/month free)
- Best for digest/summary alerts
- Include unsubscribe link

### 4. SMS
- **Provider Options**:
  - Twilio (~$0.0075 per SMS)
  - AWS SNS (~$0.0075 per SMS)
  - Vonage (~$0.0068 per SMS)
- Most expensive but highest engagement
- Reserve for high-confidence alerts only

---

## User Preferences Model

```go
type NotificationPreferences struct {
    UserID          string

    // Channels
    EnablePush      bool
    EnableEmail     bool
    EnableSMS       bool

    // Contact info
    Email           string
    Phone           string  // E.164 format: +1234567890
    PushSubscription string // Web Push subscription JSON

    // Alert settings
    Threshold       float64 // Default: 2.0 units
    Sports          []string // ["nba", "nfl"] or empty for all

    // Quiet hours (no notifications)
    QuietStart      string  // "23:00"
    QuietEnd        string  // "08:00"
    Timezone        string  // "America/New_York"

    // Channel-specific thresholds
    SMSThreshold    float64 // Only SMS if diff >= this (default: 4.0)
    EmailThreshold  float64 // Only email if diff >= this (default: 3.0)
}
```

---

## Deduplication Strategy

Prevent notification spam for the same opportunity:

```go
type AlertKey struct {
    PlayerName   string
    PropCategory string
    Direction    string  // "over" or "under"
    GameID       string
}

type AlertHistory struct {
    Key           AlertKey
    LastAlerted   time.Time
    LineAtAlert   float64
    CooldownUntil time.Time
}

func ShouldAlert(key AlertKey, newLine float64) bool {
    history := GetAlertHistory(key)

    if history == nil {
        return true // Never alerted before
    }

    // Still in cooldown?
    if time.Now().Before(history.CooldownUntil) {
        // Only re-alert if line moved significantly (0.5+ units)
        if math.Abs(newLine - history.LineAtAlert) < 0.5 {
            return false
        }
    }

    return true
}

// Cooldown periods
const (
    CooldownLowConfidence    = 4 * time.Hour
    CooldownMediumConfidence = 2 * time.Hour
    CooldownHighConfidence   = 1 * time.Hour
)
```

---

## Rate Limiting

Protect against runaway notifications:

```go
type RateLimits struct {
    Push  RateLimit // 20 per hour
    Email RateLimit // 10 per hour
    SMS   RateLimit // 5 per hour
}

type RateLimit struct {
    MaxPerHour int
    MaxPerDay  int
    Current    int
    ResetAt    time.Time
}

func CanSend(channel string, userID string) bool {
    limits := GetUserRateLimits(userID, channel)

    if limits.Current >= limits.MaxPerHour {
        return false
    }

    return true
}
```

### Default Rate Limits

| Channel | Per Hour | Per Day |
|---------|----------|---------|
| Push    | 20       | 100     |
| Email   | 10       | 50      |
| SMS     | 5        | 20      |

---

## Notification Message Templates

### Push Notification
```
Title: 🎯 Value Alert: LeBron James OVER 25.5 pts
Body: Line at 25.5, avg 28.3 pts (+2.8). Best odds: -110 @ DraftKings
```

### Email
```
Subject: Value Alert: LeBron James Points OVER

Hi,

We detected a potential value opportunity:

Player: LeBron James (Lakers)
Game: Lakers @ Celtics - Tonight 7:30 PM

Prop: Points OVER 25.5
Average (L5): 28.3 points
Difference: +2.8 units below average

Best Odds: -110 at DraftKings

[View Details] [Manage Alerts]

---
You're receiving this because you enabled value alerts.
[Unsubscribe]
```

### SMS
```
LineFinder Alert: LeBron OVER 25.5 pts (avg 28.3). Best -110 @ DK. Lakers@Celtics 7:30PM
```

---

## Implementation Phases

### Phase 1: WebSocket Alerts (No external services)
1. Add value detection to polling service
2. Broadcast alerts via existing WebSocket
3. Show toast/badge in frontend
4. Store alert history for deduplication

### Phase 2: Web Push Notifications
1. Add service worker to frontend
2. Implement Web Push subscription
3. Store push subscriptions in backend
4. Send push via web-push library

### Phase 3: Email Notifications
1. Choose email provider (recommend: Resend or SendGrid)
2. Create email templates
3. Add email preferences to user settings
4. Implement email sending service

### Phase 4: SMS Notifications
1. Set up Twilio account
2. Add phone number collection
3. Implement SMS sending service
4. Add SMS-specific rate limiting (cost control)

---

## File Structure

```
internal/
├── alerts/
│   ├── detector.go      # Value detection logic
│   ├── models.go        # Alert types
│   └── history.go       # Deduplication/history
├── notifications/
│   ├── service.go       # Dispatcher orchestration
│   ├── push.go          # Web Push implementation
│   ├── email.go         # Email sending
│   ├── sms.go           # SMS sending
│   ├── templates/       # Message templates
│   │   ├── email.html
│   │   └── email.txt
│   └── ratelimit.go     # Rate limiting
├── preferences/
│   ├── store.go         # User preferences storage
│   └── models.go        # Preference types
```

---

## Configuration

```env
# Alert settings
ALERT_THRESHOLD=2.0           # Minimum units difference
ALERT_ENABLED=true            # Master switch

# Push notifications
VAPID_PUBLIC_KEY=xxx          # For Web Push
VAPID_PRIVATE_KEY=xxx
VAPID_SUBJECT=mailto:you@example.com

# Email (Resend)
RESEND_API_KEY=xxx
EMAIL_FROM=alerts@linefinder.app

# SMS (Twilio)
TWILIO_ACCOUNT_SID=xxx
TWILIO_AUTH_TOKEN=xxx
TWILIO_FROM_NUMBER=+1234567890

# Rate limits
RATE_LIMIT_PUSH_HOURLY=20
RATE_LIMIT_EMAIL_HOURLY=10
RATE_LIMIT_SMS_HOURLY=5
```

---

## Bottlenecks & Failure Points

### 1. Stale Alerts
**Problem**: Line moves before user sees alert
**Solution**: Include timestamp, show "X min ago", auto-dismiss stale alerts

### 2. SMS Cost Explosion
**Problem**: High-volume days could cost $$
**Solution**:
- Higher threshold for SMS (4+ units only)
- Daily SMS cap per user
- Require explicit SMS opt-in

### 3. Email Deliverability
**Problem**: Alerts go to spam
**Solution**:
- Use reputable provider
- SPF/DKIM/DMARC setup
- Send from proper domain
- Include unsubscribe link

### 4. False Positives
**Problem**: Average not representative (injury return, matchup, etc.)
**Solution**:
- Show context (games played, trend)
- Let users filter by confidence level
- Add "snooze player" option

### 5. Notification Fatigue
**Problem**: Too many alerts = user ignores all
**Solution**:
- Smart batching (digest mode option)
- Progressive cooldowns
- Let users set max alerts/day

### 6. Service Outages
**Problem**: Twilio/SendGrid down
**Solution**:
- Queue with retry logic
- Fallback channels
- Alert admin on failures

---

## Metrics to Monitor

| Metric | What it tells you |
|--------|-------------------|
| Alerts generated/hour | Volume sanity check |
| Alerts sent by channel | Channel usage distribution |
| Alert open rate (email) | Engagement quality |
| Alert → click rate | Are alerts actionable? |
| Unsubscribe rate | Alert fatigue indicator |
| False positive reports | Detection accuracy |
| SMS cost/day | Cost control |
| Delivery failures | Service health |

---

## Questions Before Implementation

1. **Start with which channels?**
   - Recommend: WebSocket → Push → Email → SMS

2. **User accounts or anonymous?**
   - Need user accounts for preferences/history
   - Or simpler: just browser-local settings + push

3. **Single user or multi-user?**
   - Single: simpler, store in env/config
   - Multi: need user auth, database

4. **Email/SMS provider preference?**
   - Email: Resend (modern API, generous free tier)
   - SMS: Twilio (reliable, good docs)

5. **Alert threshold configurable per-user or global?**
   - Per-user more flexible
   - Global simpler to start
