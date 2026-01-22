# LineFinder

Sports betting odds comparison tool with real-time updates and value alert notifications.

## Features

- **Odds Comparison**: Compare NFL/NBA odds across DraftKings, FanDuel, and BetMGM
- **Player Props**: View player prop lines with L5 averages and injury status
- **Real-time Updates**: WebSocket-based live odds updates with polling service
- **Value Alerts**: Automatic detection when lines differ significantly from player averages
- **Push Notifications**: Web Push alerts for value opportunities with batching and quiet hours

## Architecture

```
linefinder/
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── api/             # HTTP handlers and routing
│   ├── alerts/          # Value detection logic
│   ├── database/        # SQLite persistence
│   ├── metrics/         # System health tracking
│   ├── models/          # Data structures
│   ├── notifications/   # Push notification service
│   ├── oddsapi/         # The Odds API client
│   ├── polling/         # Background polling service
│   ├── service/         # Business logic
│   ├── sportsdata/      # SportsDataIO client
│   ├── store/           # In-memory data store
│   └── websocket/       # WebSocket hub and clients
├── web/                 # React frontend
│   ├── public/sw.js     # Service worker for push
│   └── src/
│       ├── components/  # React components
│       ├── hooks/       # Custom hooks (WebSocket)
│       └── utils/       # Push notification utilities
└── docs/                # Architecture documentation
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- [The Odds API](https://the-odds-api.com/) key

### Backend

```bash
# Copy environment config
cp .env.example .env

# Add your API key to .env
ODDS_API_KEY=your_key_here

# Run the server
./start.sh
# or: go run cmd/server/main.go
```

### Frontend

```bash
cd web
npm install
npm run dev
```

Open http://localhost:5173

## API Endpoints

### Core

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check with metrics |
| GET | `/api/games/{sport}` | List games (nfl/nba) |
| GET | `/api/odds/{sport}` | Get odds data |
| POST | `/api/refresh/{sport}` | Fetch fresh data from API |

### Player Data

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/props/{sport}/{gameId}` | Player props with value alerts |
| GET | `/api/injuries/{sport}/{gameId}` | Injury report |
| GET | `/api/averages/{sport}/{gameId}` | Player L5 averages |

### Real-time

| Method | Endpoint | Description |
|--------|----------|-------------|
| WS | `/api/ws` | WebSocket for live updates |
| GET | `/api/metrics` | System metrics |
| POST | `/api/polling/toggle` | Toggle polling on/off |
| POST | `/api/polling/enable` | Enable polling |
| POST | `/api/polling/disable` | Disable polling |

### Notifications

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/alerts/check` | Check for value alerts |
| GET | `/api/preferences` | Get notification preferences |
| PUT | `/api/preferences` | Update preferences |
| POST | `/api/subscribe` | Subscribe to push notifications |
| POST | `/api/unsubscribe` | Unsubscribe from all |
| GET | `/api/vapid-public-key` | Get VAPID public key |

## Configuration

Environment variables (`.env`):

```bash
# Required
ODDS_API_KEY=your_api_key_here

# Optional: SportsDataIO for real injury/stats data
SPORTSDATA_API_KEY=your_sportsdata_key

# Server
PORT=8080

# Database
DATABASE_PATH=~/.linefinder/linefinder.db

# API quota (default: 500 for free tier)
API_QUOTA_LIMIT=500

# Polling (disabled by default)
POLL_ENABLED=false
POLL_INTERVAL_SECONDS=60
POLL_SPORTS=nba,nfl

# WebSocket
WS_MAX_CONNECTIONS=1000

# Push notifications (generate with: go run cmd/vapid/main.go)
VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_SUBJECT=mailto:your-email@example.com

# Notification batching
NOTIFICATION_BATCH_SECONDS=60
```

## Value Alert Thresholds

Alerts trigger when line differs from player average by:

| Prop Type | Default Threshold |
|-----------|-------------------|
| Points | 2.0 |
| Rebounds | 1.5 |
| Assists | 1.0 |
| Three Pointers | 0.5 |
| Other | 2.0 |

Configure thresholds in the Settings UI or via `/api/preferences`.

## Push Notifications Setup

1. Generate VAPID keys:
   ```bash
   go run cmd/vapid/main.go
   ```

2. Add keys to `.env`:
   ```bash
   VAPID_PUBLIC_KEY=<generated_public_key>
   VAPID_PRIVATE_KEY=<generated_private_key>
   ```

3. Restart server and enable push in Settings.

## WebSocket Messages

Subscribe to sport-specific updates:
```json
{"type": "subscribe", "sport": "nba"}
```

Receive updates:
```json
{
  "type": "odds_update",
  "sport": "nba",
  "data": [...games],
  "timestamp": "2024-01-17T19:00:00Z"
}
```

Value alerts:
```json
{
  "type": "value_alert",
  "data": {
    "player_name": "LeBron James",
    "prop_category": "points",
    "line": 25.5,
    "average": 28.2,
    "direction": "under",
    "confidence": "high"
  }
}
```

## License

MIT
