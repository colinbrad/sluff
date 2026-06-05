# Sluff

A multiplayer web game for drawing backcountry ski routes. Teams draw a line from A to B on a topo map and are scored on how well it stays inside a guide-authored "safe corridor" and avoids no-go zones.

Developed with AMGA ski guides for teaching route selection.

## Tech Stack

- React 19 + TypeScript + Vite + Tailwind v4
- React Router v7, Zustand
- MapLibre GL + OpenTopoMap, Terra Draw, Turf.js
- Go 1.25, Chi v5, JWT, coder/websocket, paulmach/orb
- SQLite + Litestream → Cloudflare R2
- Logging: `log/slog`
- Hosting: Render

## Getting Started

```bash
cd frontend && npm install && cd ..
make dev
```

Backend on `:8080`, frontend on `:5173`.

### Environment Variables

Backend:

| Variable | Default | Notes |
|---|---|---|
| `JWT_SECRET` | random ephemeral | Required in prod; fatal if unset when `HOST=0.0.0.0`. |
| `HOST` | — | Set to `0.0.0.0` in prod. Switches slog to JSON and enforces `JWT_SECRET`. |
| `PORT` | `8080` | |
| `DB_PATH` | `data/sluff.db` | |
| `CORS_ORIGINS` | `http://localhost:5173` | Comma-separated. |
| `DEFAULT_GUIDE_USERNAME` | — | Seeds a guide on startup if set with the password. |
| `DEFAULT_GUIDE_PASSWORD` | — | |
| `LITESTREAM_ENDPOINT` | — | R2/S3 endpoint. |
| `LITESTREAM_BUCKET` | — | |
| `LITESTREAM_ACCESS_KEY_ID` | — | |
| `LITESTREAM_SECRET_ACCESS_KEY` | — | |

Frontend: `VITE_API_URL` (default: relative).

### Test / Build

```bash
make test     # backend go test + frontend vitest
make build
```

## Roles

- **Guide** — authenticated, creates maps and hosts sessions. `/guide/login`.
- **Player** — joins via 6-char code, no account.
- **Demo** — `/demo`, no account, single round.

## Game Flow

1. Guide authors a map at `/guide` or imports GeoJSON/KML at `/guide/import` and labels features (start, end, corridor, no-go).
2. Guide creates a session from the map; players join by code (2–4 teams, 2–8 players). Solo play at `/solo`.
3. Each round: teams draw a route in a time limit.
4. Score (0–1000 per round):
   - 600 — % of route inside corridor
   - 200 — endpoints within 50 m of start/end
   - 100 — drawn length vs direct distance
   - 100 — max deviation penalty
   - up to −300 — no-go zone intersection
5. Round Review shows the corridor and the team's route side by side.

## Ops Notes

- **Auth**: guide endpoints check ownership of the resource; cross-guide access returns 403. Submissions verify the team belongs to the session and the round belongs to the session's map; duplicate `(round, team)` submissions are rejected.
- **Rate limit**: 5 req/s, burst 10, per IP, on login/register. Stale entries swept by a goroutine tied to the server context.
- **Shutdown**: `main` traps SIGINT/SIGTERM and cancels the server context.

## Deployment

Render: Docker backend + static frontend. `render.yaml` is the source of truth. Entrypoint restores from Litestream and runs the server under `litestream replicate` if `LITESTREAM_ENDPOINT` is set.

Production secrets live in the Render dashboard — upload `backend/sluff-backend.env` via **Environment → Upload .env File**. That file is gitignored.
