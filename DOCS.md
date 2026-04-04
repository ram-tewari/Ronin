# Ronin - Desktop Sports Companion

Ronin is an Electron desktop application featuring a pixel-art chibi character that tracks sports teams and displays real-time alerts via a stylized dialogue bubble. The app consists of a React/Electron frontend and a Go HTTP backend.

---

## Architecture Overview

```
Electron (main.js)
  ├── Main Window (280x360, transparent, frameless, always-on-top)
  │   └── React App: ChibiEngine + DialogueBubble (App.jsx)
  ├── Settings Window (400x600, standard frame)
  │   └── React App: Settings UI (Settings.jsx)
  └── IPC Bridge (preload.js)
         ↕ save-settings / get-settings / settings-saved
Go Backend (localhost:8080)
  ├── GET  /status     → Oracle report (mood + filtered alerts)
  ├── GET  /discovery  → Available sports & teams from ESPN
  ├── GET  /config     → Current backend config
  └── POST /config     → Update selected teams
```

### Data Flow

1. **Discovery**: Settings UI fetches `GET /discovery` → Go backend calls ESPN API → returns all available teams grouped by sport.
2. **Selection**: User checks team checkboxes → clicks Save → config saved to Electron (`config.json`) AND posted to Go backend (`POST /config`).
3. **Polling**: App.jsx polls `GET /status` every 15 seconds → Go backend generates alerts only for selected teams → returns pre-filtered Oracle report.
4. **Display**: App.jsx shows the chibi character with a mood-styled dialogue bubble. Bubble appears for 8 seconds when new alerts arrive.

---

## Go Backend

### Files

| File | Purpose |
|------|---------|
| `backend/main.go` | HTTP server, endpoints, config management |
| `backend/scraper.go` | ESPN API integration, team discovery |
| `backend/go.mod` | Go module definition |
| `backend/ronin_config.json` | Persisted selected team IDs (auto-created) |

### Endpoints

#### `GET /status` — Oracle Report

Returns the current mood and alerts for tracked teams only.

**Response:**
```json
{
  "mood": "alert",
  "message": "Tracking 2 team(s).",
  "alerts": [
    {
      "teamId": "2509",
      "team": "Purdue Boilermakers",
      "event": "Tracking Purdue Boilermakers",
      "link": "https://www.espn.com/mens-college-basketball/team/_/id/2509",
      "priority": "high"
    }
  ]
}
```

**Mood Values:**
- `idle` — No teams tracked or no active events
- `alert` — Active tracking, events upcoming
- `hyped` — High-priority event imminent (future use)
- `exhausted` — Post-event cooldown (future use)

**Behavior:**
- Returns `idle` mood with empty alerts if no teams are selected.
- Alerts are generated only for teams in the user's selected list.
- First alert in the list gets `high` priority; others get `medium`.

#### `GET /discovery` — Team Discovery

Fetches all available teams from ESPN's public API, grouped by sport.

**Response:**
```json
{
  "sports": [
    {
      "id": "ncaa_mbb",
      "name": "NCAA Men's Basketball",
      "teams": [
        { "id": "2509", "name": "Purdue Boilermakers" },
        { "id": "150", "name": "Duke Blue Devils" }
      ]
    }
  ]
}
```

**Notes:**
- Teams are sorted alphabetically by name.
- Handles ESPN API pagination automatically (the API returns ~25 teams per page; the scraper fetches all pages).
- Updates an in-memory team name cache used by `/status` to resolve team IDs to display names.
- Currently supports NCAA Men's Basketball. Additional sports can be added to the `DiscoveryResponse` in `discoveryHandler`.

#### `GET /config` — Read Config

Returns the current backend configuration.

```json
{
  "selectedTeams": ["2509", "150"]
}
```

#### `POST /config` — Update Config

Receives updated selected team IDs. Persists to `ronin_config.json`.

**Request body:**
```json
{
  "selectedTeams": ["2509", "150", "2633"]
}
```

**Response:**
```json
{
  "status": "ok"
}
```

### Scraper (`scraper.go`)

The `FetchNCAAMBBTeams()` function:
1. Hits `https://site.api.espn.com/apis/site/v2/sports/basketball/mens-college-basketball/teams?limit=500&page=N`
2. Parses the nested ESPN JSON structure (`sports[].leagues[].teams[].team`)
3. Extracts `id` and `displayName` for each team
4. Handles pagination (iterates pages until `pageIndex >= pageCount`)
5. Returns a sorted `[]DiscoveryTeam` slice

### Running the Backend

```bash
cd backend
go run .
```

The server starts on `http://localhost:8080`. Config is persisted to `ronin_config.json` in the working directory.

---

## Electron Layer

### Files

| File | Purpose |
|------|---------|
| `main.js` | Electron main process, window management, IPC, config persistence |
| `preload.js` | Secure IPC bridge via `contextBridge` |

### Configuration

**File location:** `{app.getPath('userData')}/config.json`

**Schema:**
```json
{
  "data": {
    "selectedTeams": ["2509", "150"]
  },
  "aesthetics": {
    "themeColor": "#1a5fa8",
    "bubbleDarkMode": false
  }
}
```

**Migration:** On startup, if the config has the old boolean-flag format (`{ purdue: false, miami: false, cricket: false }`), it is automatically migrated to `{ selectedTeams: [] }`.

### IPC Channels

| Channel | Type | Direction | Payload | Purpose |
|---------|------|-----------|---------|---------|
| `open-external` | send | Renderer → Main | URL string | Opens URL in default browser (validates http/https) |
| `show-context-menu` | send | Renderer → Main | — | Shows right-click context menu (Settings / Quit) |
| `save-settings` | send | Renderer → Main | Config object | Saves config to disk, broadcasts to main window, syncs to Go backend |
| `get-settings` | invoke | Renderer → Main | — | Returns current config (async) |
| `settings-saved` | receive | Main → Renderer | Config object | Broadcast after settings are saved |

### preload.js API (`window.ronin`)

```typescript
interface RoninBridge {
  platform: string;
  openExternal(url: string): void;
  showContextMenu(): void;
  saveSettings(config: Config): void;
  onSettingsSaved(callback: (config: Config) => void): () => void;  // returns unsubscribe
  getSettings(): Promise<Config>;
}
```

### Window Specifications

| Window | Size | Properties |
|--------|------|------------|
| Main (Chibi) | 280x360 | Transparent, frameless, always-on-top, skip taskbar, no shadow |
| Settings | 400x600 | Standard frame, menu bar hidden |

---

## React Frontend

### Files

| File | Purpose |
|------|---------|
| `src/main.jsx` | React entry point with hash router (`/chibi`, `/settings`) |
| `src/App.jsx` | Main view: chibi character + dialogue bubble + status polling |
| `src/Settings.jsx` | Settings UI: dynamic team discovery + aesthetics |
| `src/ChibiEngine.jsx` | Pixel-art character renderer with mood animations |
| `src/index.css` | Tailwind CSS + custom animations |

### App.jsx — Main View

**State:**
- `mood` — Current mood string (`idle`, `alert`, `hyped`, `exhausted`)
- `bubbleVisible` — Whether the dialogue bubble is showing
- `message` — Text displayed in the bubble
- `alerts` — Array of current alerts from `/status`
- `config` — User config from Electron IPC

**Status Polling:**
- Fetches `GET http://localhost:8080/status` every 15 seconds
- Alerts arrive pre-filtered by the Go backend (only selected teams)
- If alerts exist: shows dialogue bubble for 8 seconds, sets mood
- If no alerts: hides bubble, sets mood to `idle`

**Dialogue Bubble:**
- Pixel-art retro styling with scanline overlay effect
- Color scheme changes based on mood (border, background, text, glow)
- Config overrides: `themeColor` replaces border color, `bubbleDarkMode` forces black background
- Clickable: opens the first alert's link in the default browser
- Arrow pointer at bottom pointing to the chibi character

**Mood Color Schemes:**

| Mood | Border | Background | Text |
|------|--------|------------|------|
| idle | `#1a5fa8` (blue) | `#080f20` | `#90beff` |
| alert | `#cc1800` (red) | `#1a0500` | `#ff9070` |
| hyped | `#ffaa00` (orange) | `#221100` | `#ffdd88` |
| exhausted | `#3a5a3a` (green) | `#06100a` | `#80b880` |

### Settings.jsx — Configuration UI

**Two tabs:** Data Tracking and Aesthetics.

#### Data Tracking Tab

- **Discovery fetch**: On mount, fetches `GET /discovery` from the Go backend
- **Dynamic rendering**: Maps through the response to generate checkboxes grouped by sport
- **Search**: Text input filters the team list in real-time
- **Collapsible sections**: Each sport group can be collapsed/expanded
- **Bulk actions**: "All" / "None" buttons per sport to select/deselect all teams
- **Selection counter**: Shows total selected count in header and per-sport counts
- **Loading state**: Animated "Discovering available teams..." message
- **Error state**: Red banner with troubleshooting hint if discovery fails

**On Save:**
1. Sends full config to Electron via `window.ronin.saveSettings(config)`
2. POSTs `{ selectedTeams: [...] }` to `http://localhost:8080/config`

#### Aesthetics Tab

- **Theme Color**: Dropdown with 5 preset colors (Blue, Red, Green, Orange, Crimson)
- **Force Bubble Dark Mode**: Checkbox to override bubble background to pure black

### ChibiEngine.jsx — Character Renderer

**Rendering:**
- 2D string grid converted to SVG rectangles
- 26-color palette, 3px per grid cell, 200x200 viewBox
- Pixel-art character (Gaara-style chibi)

**Mood Animations:**
- `idle` — Gentle float + breathing (`animate-float`)
- `hyped` — Energetic bounce
- `exhausted` — Slow pulse + 30% grayscale + reduced opacity

**Blinking:** Overlays closed-eye frame every ~3 seconds for 150ms.

---

## Development

### Prerequisites

- Node.js (v18+)
- Go (1.26+)
- npm

### Running in Development

**Terminal 1 — Go Backend:**
```bash
cd backend
go run .
```

**Terminal 2 — Electron + Vite:**
```bash
npm run dev
```

This starts Vite on port 51234 and launches Electron, which loads from the dev server.

### Building for Production

```bash
npm run build   # Build React/Vite to dist/
npm run dist    # Package with electron-builder (NSIS/DMG/AppImage)
```

### Project Dependencies

**Node (package.json):**
- `electron` — Desktop runtime
- `react`, `react-dom`, `react-router-dom` — UI framework
- `vite` — Build tool and dev server
- `tailwindcss`, `postcss`, `autoprefixer` — Styling
- `electron-builder` — Packaging

**Go (go.mod):**
- Standard library only (`net/http`, `encoding/json`, `os`, `sync`)

---

## Adding a New Sport

To add a new sport (e.g., NFL, Premier League):

1. **Scraper**: Add a new fetch function in `backend/scraper.go` (e.g., `FetchNFLTeams()`) that hits the appropriate ESPN API endpoint.
2. **Discovery endpoint**: In `backend/main.go`, call the new fetch function in `discoveryHandler` and append the result as another entry in the `Sports` slice.
3. **Status endpoint**: Update `statusHandler` to generate alerts for the new sport's teams (currently all selected teams are treated as NCAA MBB).
4. **Frontend**: No changes needed — Settings.jsx dynamically renders whatever sports/teams the `/discovery` endpoint returns.

### ESPN API URL Pattern

```
https://site.api.espn.com/apis/site/v2/sports/{sport}/{league}/teams
```

Examples:
- NCAA MBB: `.../basketball/mens-college-basketball/teams`
- NFL: `.../football/nfl/teams`
- NBA: `.../basketball/nba/teams`
- MLB: `.../baseball/mlb/teams`
- NHL: `.../hockey/nhl/teams`
- MLS: `.../soccer/usa.1/teams`
- Premier League: `.../soccer/eng.1/teams`
