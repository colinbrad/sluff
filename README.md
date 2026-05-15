# Sluff

A web-based multiplayer mapping game for backcountry ski route planning. Teams compete to draw the best route between two points in mountainous terrain.

## Tech Stack

- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS v4
- **Routing**: React Router v7
- **Map**: MapLibre GL JS with OpenTopoMap tiles
- **Drawing**: Terra Draw (freehand linestring + polygon modes)
- **State**: Zustand
- **Backend**: Go 1.25 + Chi v5 router
- **Auth**: JWT (golang-jwt/jwt)
- **WebSocket**: coder/websocket
- **Database**: SQLite (CGo) + Litestream replication to Cloudflare R2
- **Geospatial**: paulmach/orb (server) + Turf.js (client)
- **Hosting**: Render.com (Docker backend + static frontend)

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js 20+
- gcc (for SQLite CGo compilation)

### Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run both backend and frontend concurrently
make dev
```

Backend runs on `:8080`, frontend on `:5173` with proxy to backend.

### Environment Variables

The backend reads these from the environment (no `.env` file loading — set them in your shell or a local env file):

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_SECRET` | yes | — | Secret for signing guide auth tokens |
| `PORT` | no | `8080` | HTTP listen port |
| `DB_PATH` | no | `data/sluff.db` | SQLite database path |
| `CORS_ORIGINS` | no | `http://localhost:5173` | Comma-separated allowed origins |
| `LITESTREAM_ENDPOINT` | no | — | S3-compatible endpoint for DB replication |
| `LITESTREAM_BUCKET` | no | — | Bucket name |
| `LITESTREAM_ACCESS_KEY_ID` | no | — | R2/S3 access key |
| `LITESTREAM_SECRET_ACCESS_KEY` | no | — | R2/S3 secret key |

The frontend takes one build-time variable:

| Variable | Default | Description |
|---|---|---|
| `VITE_API_URL` | `` (relative) | Backend API base URL |

### Testing

```bash
# All tests
make test

# Backend only (uses real SQLite DBs in temp dirs)
cd backend && go test ./...

# Frontend only (Vitest)
cd frontend && npm test
```

### Build

```bash
make build
```

## How It Works

### Roles

- **Guide** — authenticated user who creates maps and hosts sessions. Registers/logs in at `/guide/login`.
- **Player** — joins a session via a 6-character code, no account required.

### Game Flow

1. **Guide** creates a map with one or more rounds at `/guide`. Each round defines start/end points, a corridor of correctness, and optional no-go zones.
2. **Guide** creates a game session from a map and shares the join code.
3. **Players** join via code and form teams (2–4 teams, 2–8 players total).
4. **Each round**: teams collaboratively draw a ski route on the topo map within a time limit.
5. **Scoring** (0–1000 per round):
   - Corridor adherence — 600 pts: % of route inside the correct corridor
   - Endpoint connection — 200 pts: route must reach within 50m of start/end points
   - Route efficiency — 100 pts: ratio of direct distance to drawn route length
   - Low deviation — 100 pts: penalizes max deviation from corridor boundary
   - No-go zone penalty: up to −300 pts for routes passing through restricted areas
6. **Winner**: team with highest cumulative score across all rounds.

## Deployment

The backend is deployed as a Docker container on Render. The frontend is a Render static site.

### Architecture

```
render.yaml
├── sluff-backend   (Docker, :8080)
│   └── Dockerfile  → Go binary + Litestream
│   └── entrypoint.sh
│       ├── if LITESTREAM_ENDPOINT set:
│       │   restore DB from R2 → run server under litestream replicate
│       └── else: run server directly
└── sluff-frontend  (static, npm run build)
```

### Setting Environment Variables on Render

Production secrets are managed in the Render dashboard (not committed to the repo). To update them from the local env file:

1. Go to the `sluff-backend` service in the Render dashboard
2. Navigate to **Environment**
3. Use **Upload .env File** and select `backend/sluff-backend.env`

The `backend/sluff-backend.env` file is gitignored.
