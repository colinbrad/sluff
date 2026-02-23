# Sluff

A web-based multiplayer mapping game for backcountry ski route planning. Teams compete to draw the best route between two points in mountainous terrain.

## Tech Stack

- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS
- **Map**: MapLibre GL JS with OpenTopoMap tiles
- **Drawing**: Terra Draw (polygon + linestring modes)
- **State**: Zustand
- **Backend**: Go + Chi router
- **WebSocket**: coder/websocket
- **Database**: SQLite
- **Geospatial**: paulmach/orb (server) + Turf.js (client)

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- gcc (for SQLite CGo compilation)

### Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run both backend and frontend
make dev
```

Backend runs on `:8080`, frontend on `:5173` with proxy to backend.

### Build

```bash
make build
```

## How It Works

1. **Admin** creates maps with rounds (start/end points + corridor of correctness) at `/admin`
2. **Host** creates a game session from a map, gets a 6-character join code
3. **Players** join via code, form teams (2-4 teams, 2-8 players total)
4. **Each round**: Teams collaboratively draw a ski route on the topo map
5. **Scoring**: Routes scored 0-1000 based on corridor adherence, endpoint connection, efficiency, and deviation
6. **Winner**: Highest cumulative score across all rounds
