# Fleet Sustainability Dashboard

Simulates a vehicle fleet, ingests telemetry, and provides real-time insights (fuel/emissions, electrification planning, costs, maintenance, alerts). Backend in Go, frontend in React+TS, MongoDB for persistence. Real-time via SSE/WebSockets, optional MQTT.

## Structure
- `cmd/` — backend entrypoints (HTTP API, simulator)
- `internal/` — backend domain, handlers, db
- `frontend/` — React app
- `scripts/` — dev ops script (`scripts/fleet_sustainability.sh`)
- `configs/` — config files (e.g., Mosquitto, env example)
- `docker-compose.yml` — dev stack (backend, mongo, mongo-express, mosquitto)
- `docker-compose.prod.yml` — prod stack (backend, frontend nginx, mongo, mosquitto)

## Features
- Telemetry ingest (HTTP POST, MQTT), storage (Mongo), queries (filters, metrics)
- Real-time updates: SSE + WebSockets; MQTT broker included (Mosquitto)
- Multi-tenant support (`tenant_id` in JWT, middleware, queries)
- Trip/Maintenance/Cost CRUD, deletes, and tenant scoping
- Electrification planning, driver leaderboard, CSV/PDF exports
- Mobile/responsive UI

## Prereqs
- Docker + Docker Compose (recommended for dev/prod)
- Node 18+ and Go (if running locally outside containers)

## Quick start (dev)
```bash
# Start entire stack (backend, mongo, mongo-express, mosquitto) and frontend
./scripts/fleet_sustainability.sh start

# Start simulator (publishes telemetry via MQTT by default)
./scripts/fleet_sustainability.sh sim-start

# Stop everything
./scripts/fleet_sustainability.sh stop
```

URLs:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8081
- Mongo Express: http://localhost:8082
- MQTT: tcp://localhost:1883 (raw), ws://localhost:9001 (websocket)

## Environment configuration
- Dev template: `configs/env.example`
- Common back-end variables:
  - `MONGO_URI` (default: mongodb://root:example@mongo:27017)
  - `MONGO_DB` (default: fleet)
  - `JWT_SECRET` (required)
  - `TELEMETRY_TTL_DAYS` (default: 30)
  - `WEBSOCKETS_ENABLED` (default: true)
  - `MQTT_BROKER_URL` (docker: tcp://mosquitto:1883, host: tcp://localhost:1883)
  - `MQTT_TELEMETRY_TOPIC` (default: fleet/telemetry)

Frontend (build-time):
- `REACT_APP_API_URL` (default: http://localhost:8081)
- `REACT_APP_SSE_URL` (default: http://localhost:8081/api/telemetry/stream)
- `REACT_APP_WS_URL` (default: ws://localhost:8081/api/telemetry/ws)

Simulator (optional):
- `SIM_USE_MQTT=1`, `FLEET_SIZE`, `SIM_TICK_SECONDS`

## Production (compose)
```bash
# copy envs and set secrets
cp configs/env.example .env
export JWT_SECRET=change-me

# build images and start
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d

# Frontend (nginx): http://localhost:3000
# Backend API: published per your setup (e.g., via reverse proxy)
```

`docker-compose.prod.yml` runs:
- `app`: Go backend (port 8080 inside)
- `frontend`: nginx serving React build on port 3000
- `mongo`: MongoDB
- `mosquitto`: MQTT broker (1883)

## API auth
JWT Bearer tokens required for protected endpoints. Obtain via `/api/auth/login` with seeded users (script ensures `admin/admin123`).

## Real-time options
- SSE: `/api/telemetry/stream`
- WebSockets: `/api/telemetry/ws` (toggle with `WEBSOCKETS_ENABLED=false`)
- MQTT: subscribe to `fleet/telemetry`

## Build locally
```bash
go build ./...
(cd frontend && npm ci && npm run build)
```

## CI
GitHub Actions runs lint/vet/staticcheck. Extend to build/test/publish images as needed.

## License
MIT