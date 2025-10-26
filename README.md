# Fleet Sustainability Dashboard

Simulates a vehicle fleet, ingests telemetry, and provides real-time insights (fuel/emissions, electrification planning, costs, maintenance, alerts). Backend in Go, frontend in React+TS, MongoDB for persistence. Real-time via SSE/WebSockets, optional MQTT.

## Project Structure
- `cmd/` — backend entrypoints (HTTP API, simulator)
- `internal/` — backend domain, handlers, db
- `frontend/` — React app
- `scripts/` — dev ops script (`scripts/fleet_sustainability.sh`)
- `configs/` — config files (e.g., Mosquitto)
- `docker-compose.yml` — dev stack (backend, mongo, mongo-express, mosquitto)
- `docker-compose.prod.yml` — prod stack (backend, frontend nginx, mongo, mosquitto)

## Architecture


```

## Features
- Telemetry ingest (HTTP POST, MQTT), storage (Mongo), queries (filters, metrics)
- Real-time updates: SSE + WebSockets; MQTT broker included (Mosquitto)
- Multi-tenant support (`tenant_id` in JWT, middleware, queries)
- Trip/Maintenance/Cost CRUD, deletes, and tenant scoping
- Electrification planning, driver leaderboard, CSV/PDF exports
- Mobile/responsive UI

## Prerequisites
- Docker + Docker Compose (recommended)
- Node 18+ and Go (if running locally outside containers)

## Quick start (dev)
```bash
./scripts/fleet_sustainability.sh start     # start backend+deps and frontend
./scripts/fleet_sustainability.sh sim-start # start simulator (MQTT by default)
```

URLs: Frontend http://localhost:3000, API http://localhost:8081, Mongo Express http://localhost:8082, MQTT tcp://localhost:1883

## Environment
Backend:
- MONGO_URI (default mongo service), MONGO_DB (fleet), JWT_SECRET, TELEMETRY_TTL_DAYS, WEBSOCKETS_ENABLED
- MQTT_BROKER_URL (docker: tcp://mosquitto:1883, host: tcp://localhost:1883), MQTT_TELEMETRY_TOPIC

Frontend (build-time):
- REACT_APP_API_URL (default http://localhost:8081)
- REACT_APP_SSE_URL, REACT_APP_WS_URL

Simulator:
- SIM_USE_MQTT=1, FLEET_SIZE, SIM_TICK_SECONDS

## Production
```bash
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```
Services: `app` (Go API 8080 internal), `frontend` (nginx 3000), `mongo`, `mosquitto`.

## API Authentication
JWT Bearer tokens required for protected endpoints. Obtain via `/api/auth/login` (script creates admin/admin123).

## Screenshots
Place images in `docs/screenshots/` using these filenames, then uncomment the sample links below.

Expected files:

```
docs/
  screenshots/
    dashboard.png
    live-view.png
    electrification.png
    vehicles.png
```

<!--
![Dashboard](docs/screenshots/dashboard.png)
![Live View](docs/screenshots/live-view.png)
![Electrification](docs/screenshots/electrification.png)
![Vehicles](docs/screenshots/vehicles.png)
-->

## License
MIT