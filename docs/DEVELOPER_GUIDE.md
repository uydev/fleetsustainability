# Developer Guide — Fleet Sustainability Dashboard

This document explains the system architecture, technologies, data models, security, real-time, simulation, and how to develop, test, and deploy the project.

## Overview
- Backend: Go (net/http), MongoDB, JWT auth, SSE/WS real-time, optional MQTT consumer
- Frontend: React + TypeScript (MUI, Recharts), SSE/WS client
- Simulator: Go program generating telemetry over HTTP or MQTT
- Infrastructure: Docker Compose (dev), Mosquitto (MQTT), Mongo + Mongo Express

## Architecture and Deployment

### High-level architecture

- Frontend: React SPA communicates with API over HTTPS, listens to realtime via SSE or WebSockets
- Backend API: Stateless Go service (REST) + in-memory broadcast hub used by both SSE and WS
- Ingestion: HTTP `POST /api/telemetry` and MQTT (Mosquitto) → backend subscriber
- Database: MongoDB with `tenant_id` indexes, optional TTL on telemetry
- Routing assist: OSRM (public or local) used by the simulator for realistic movement/road snapping

```mermaid
flowchart TD
  client[React Frontend (SPA)]
  be[Go Backend API]
  mq[(MQTT Broker)]
  db[(MongoDB)]
  osrm[OSRM Routing]
  sim[Simulator]

  client -->|HTTP REST| be
  client -->|SSE| be
  client -->|WebSocket| be
  sim -->|HTTP POST| be
  sim -->|MQTT publish| mq
  mq -->|MQTT subscribe| be
  be --> db
  db --> be
  be -.->|optional route snap| osrm
```

### Deployment profiles

- Development (Docker Compose)
  - Services: backend, MongoDB (+ Mongo Express optional), Mosquitto; frontend runs with Vite/CRA dev server
  - Simulator runs locally; toggle HTTP vs MQTT via env vars
- Production (recommended)
  - Build the React app and serve via Nginx
  - Run the Go API behind a reverse proxy with TLS and proper timeouts (SSE/WS)
  - Keep MQTT broker internal when possible; enable TLS/accounts if devices connect externally
  - Configure healthchecks, restart policies, resource limits; consider moving broadcast fan-out to a shared pub/sub (e.g., Redis) if horizontally scaling API replicas

## Repository Structure
- `cmd/main.go`: backend HTTP server, routes, SSE hub, WS endpoint, MQTT subscriber
- `cmd/simulator/main.go`: simulator producing telemetry and creating vehicles
- `internal/`
  - `auth`: JWT auth service (hash/verify, token issue/validate)
  - `middleware`: JWT middleware injecting claims into context
  - `handlers`: auth handlers (login/register/profile)
  - `db`: MongoDB collections and CRUD helpers (telemetry, vehicles, trips, maintenance, costs)
  - `models`: Go structs for all entities (with `tenant_id`)
- `frontend/`: React app (components, services/api.ts auth + API client)
- `scripts/fleet_sustainability.sh`: dev workflow (compose up/down, frontend dev server, simulator, OSRM)
- `configs/`: mosquitto config and env examples
- `docs/`: developer guide, screenshots

## Backend
### Tech Stack
- Go 1.24.x, stdlib `net/http`
- MongoDB official driver
- JWT: `github.com/golang-jwt/jwt/v5`
- MQTT: `github.com/eclipse/paho.mqtt.golang`

### HTTP Endpoints (high-level)
- Auth: `POST /api/auth/login`, `POST /api/auth/register`, `GET /api/auth/profile`
- Telemetry: `POST /api/telemetry`, `GET /api/telemetry?from&to&vehicle_id&limit&sort`
- Telemetry metrics: `GET /api/telemetry/metrics`, `GET /api/telemetry/metrics/advanced`
- Vehicles: `GET/POST /api/vehicles`, `GET/PUT/DELETE /api/vehicles/:id`
- Trips/Maintenance/Costs: `GET/POST /api/trips|maintenance|costs`, `DELETE /api/trips|maintenance|costs/:id`
- Alerts: `GET /api/alerts`
- Real-time: `GET /api/telemetry/stream` (SSE), `GET /api/telemetry/ws` (WebSocket)

### Real-time transports in depth (SSE vs WebSocket vs MQTT)

- Server-Sent Events (SSE)
  - One-way: server → browser over HTTP.
  - Simple (EventSource API in browsers), works through most proxies/CDNs.
  - Great for dashboards and push updates; no client → server messages.
  - In this project: live telemetry updates to the frontend.

- WebSocket (WS)
  - Full-duplex: server ⇄ browser, persistent TCP (ws:// / wss://).
  - Good for interactive UIs, commands, backpressure control.
  - In this project: optional equivalent stream of telemetry, future-ready for commands.

- MQTT
  - Pub/Sub broker protocol; clients publish/subscribe to topics (e.g., `fleet/telemetry`).
  - Designed for IoT; decouples producers (simulator/devices) and consumers (backend).
  - In this project: simulator publishes telemetry to Mosquitto; backend subscribes and ingests/broadcasts.

When to use which
- Pure web dashboard: SSE is simplest and reliable.
- Interactive controls (send commands back): WebSocket.
- IoT device integration, offline queues, multi-producer: MQTT.

### How the broadcast hub works (SSEHub)

The backend maintains a simple in-memory hub to fan out messages to connected clients. SSE and WS clients both register a buffered channel in this hub.

```go
// Simplified: cmd/main.go
type SSEHub struct {
    mu      sync.RWMutex
    clients map[chan []byte]string // chan -> tenant_id
}

func NewSSEHub() *SSEHub { return &SSEHub{clients: make(map[chan []byte]string)} }

func (h *SSEHub) Broadcast(data []byte) {
    h.mu.RLock(); defer h.mu.RUnlock()
    for ch := range h.clients {
        select { case ch <- data: default: /* drop if slow */ }
    }
}

func (h *SSEHub) BroadcastToTenant(tenantID string, data []byte) {
    h.mu.RLock(); defer h.mu.RUnlock()
    for ch, t := range h.clients {
        if t != tenantID { continue }
        select { case ch <- data: default: }
    }
}
```

- Each client gets a dedicated channel; tenant_id is recorded for scoping.
- Broadcast pushes to all channels; BroadcastToTenant filters by tenant.

### SSE endpoint (server → client)
```go
// cmd/main.go (ServeHTTP)
w.Header().Set("Content-Type", "text/event-stream")
// Each client gets a channel and (optional) tenant binding
clientCh := make(chan []byte, 16)
h.mu.Lock(); h.clients[clientCh] = tenantID; h.mu.Unlock()
// Write messages as: "data: <json>\n\n"
```
- EventSource on the frontend listens and updates the UI.

### WebSocket endpoint (server ⇄ client)
```go
var wsUpgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }
func wsTelemetryHandler(w http.ResponseWriter, r *http.Request) {
    conn, _ := wsUpgrader.Upgrade(w, r, nil)
    clientCh := make(chan []byte, 16)
    telemetrySSEHub.mu.Lock(); telemetrySSEHub.clients[clientCh] = tenantID; telemetrySSEHub.mu.Unlock()
    for msg := range clientCh {
        conn.WriteMessage(websocket.TextMessage, msg)
    }
}
```
- Uses the same hub; writes text frames for each telemetry JSON event.

### MQTT subscriber (broker → backend)
```go
opts := mqtt.NewClientOptions().AddBroker(os.Getenv("MQTT_BROKER_URL"))
client := mqtt.NewClient(opts)
client.Connect()
client.Subscribe("fleet/telemetry", 1, func(_ mqtt.Client, msg mqtt.Message) {
    // 1) json.Unmarshal payload to teleIn (vehicle_id, timestamp, location, speed, ...)
    // 2) Convert types, enforce EV emissions=0, set tenant if provided
    // 3) Insert into MongoDB
    // 4) telemetrySSEHub.Broadcast or BroadcastToTenant
})
```
- The backend acts as a consumer; this decouples producers (simulator/devices) from the HTTP ingest path.

## Multi-tenancy
- `tenant_id` is included in JWT claims.
- Middleware reads claims and injects tenant_id into request context.
- Handlers filter queries by tenant when present; item-level deletes validate ownership.
- Indexes on `tenant_id` improve query performance.

## Authentication & Security
- Bcrypt for password hashing; JWT HS256 for tokens.
- `JWT_SECRET` configured via env; `JWT_EXPIRY` controls token lifetime.
- CORS middleware is permissive in dev; tighten for prod.
- Optional HTTPS via `USE_HTTPS`, `TLS_CERT_FILE`, `TLS_KEY_FILE`.

Example token generation:
```go
// internal/auth/auth.go
claims := jwt.MapClaims{
  "user_id": user.ID.Hex(),
  "username": user.Username,
  "role": string(user.Role),
  "tenant_id": user.TenantID,
  "exp": time.Now().Add(s.tokenExp).Unix(),
}
```

## Data Model (MongoDB)
- Telemetry stores current and historical readings; TTL prevents unbounded growth.
- Vehicles, Trips, Maintenance, Cost models include timestamps and tenant.

## Frontend
- `src/services/api.ts` centralizes API calls; attaches JWT automatically.
- Real-time: the Live View subscribes to SSE (or WS) and updates markers.

Example consuming SSE in the browser:
```ts
const es = new EventSource(`${apiBase}/api/telemetry/stream`);
es.onmessage = (e) => {
  const data = JSON.parse(e.data);
  // update state with new telemetry
};
```

## Simulator (movement + energy model)
- Plans a route via OSRM or uses jitter; advances by km per tick based on speed.
- Random dwell/stop periods; refuel/charge while stopped.
- Emits telemetry via HTTP POST or MQTT publish.

Key movement loop:
```go
tick := time.NewTicker(interval)
for range tick.C {
  // adjust speed towards target with accel cap
  stepAlongRoute(state, interval.Seconds())
  // consume fuel/battery and refill while stopped
  sendTelemetry(apiURL, telemetryFromState(state))
}
```

## Reporting & Exports
- CSV export (Blob + anchor download) from `VehicleDetail`.
- PDF export via jsPDF for simple reports.
- Metrics endpoints compute aggregates (emissions, EV%).

## Configuration
- Backend: `MONGO_URI`, `MONGO_DB`, `JWT_SECRET`, `TELEMETRY_TTL_DAYS`, `WEBSOCKETS_ENABLED`, `MQTT_*`
- Frontend build: `REACT_APP_API_URL`, (optional) SSE/WS URLs
- Simulator: `FLEET_SIZE`, `SIM_TICK_SECONDS`, `SIM_GLOBAL`, `SIM_USE_MQTT`, `OSRM_BASE_URL`

## Development Script (scripts/fleet_sustainability.sh)
- `start`: brings up Docker services (backend, Mongo, Mongo Express, Mosquitto), ensures admin user, launches frontend dev server (localhost:3000).
- `stop`: stops frontend process, `docker-compose down`, stops local OSRM if running, cleans logs.
- `status`: prints running containers, ports, API responsiveness, OSRM status.
- `troubleshoot`: menu to free ports, kill Docker processes, view logs, auto-fix (reset DB + seed).
- `sim-start [local|global]`: builds/starts simulator with envs (JWT token, API_BASE_URL, SIM_TICK_SECONDS, FLEET_SIZE, SIM_USE_MQTT, OSRM_BASE_URL).
- `sim-stop`: stops the simulator process; `sim-status` prints its PID.
- `osrm-start|stop|status`: manage local OSRM (Monaco dataset) on port 5000.

Notes
- The script normalizes directories, checks for Docker availability, and adds resiliency (PID files, retries, log tails).
- MQTT ports freed in troubleshooting (1883 TCP, 9001 WS). Simulator defaults to MQTT publish if `SIM_USE_MQTT=1`.

## Development Workflow
- Start stack + frontend: `./scripts/fleet_sustainability.sh start`
- Start simulator: `./scripts/fleet_sustainability.sh sim-start`
- Troubleshoot: `./scripts/fleet_sustainability.sh troubleshoot`

### Testing & Linting
- Go: `go vet`, `staticcheck`; `go build ./...`
- Frontend: `npm test`, ESLint during build

## Deployment
- Dev: `docker-compose.yml` (backend 8081, mongo-express 8082, mosquitto 1883/9001)
- Prod: `docker-compose.prod.yml` (backend, nginx-served frontend, mongo, mosquitto)
- Dockerfile: Debian multi-stage; vendored modules; CA certs included
- Set `JWT_SECRET` and externalize `MONGO_URI` in production

## Security Hardening
- Rotate `JWT_SECRET`, reduce `JWT_EXPIRY`, enforce HTTPS in prod
- Tighten CORS, add rate limiting, size limits
- Introduce RBAC (`role` claim checks) where needed
- Validate payloads strictly (schemas)
- MQTT with TLS and ACLs when external

## Extensibility
- Add telematics providers (Samsara, Geotab) via mappers to our schema; map devices → tenant_id
- Bidirectional WS for remote control
- Cloud IoT brokers (AWS IoT, Azure IoT) with feature flags

## Troubleshooting
- Backend logs: `docker compose logs app`
- Frontend logs: `frontend/frontend.log`
- Simulator logs: `simulator.out`
- DB UI: http://localhost:8082

## Frontend Map Implementation
- Technology: a lightweight SVG-based world map in `frontend/src/components/FleetMap.tsx`. We chose this for zero external dependencies and predictable rendering in dev demos. A production alternative is Leaflet or Mapbox GL; the API surface is similar (markers, popups, viewport control).
- Core idea: convert lat/lon deltas from a current center into relative positions and render markers absolutely.

Key snippet (positioning and markers):
```ts
// Convert lat/lon to relative position on the SVG canvas
const getVehiclePosition = (lat: number, lon: number) => {
  const latDiff = lat - centerLat;
  const lonDiff = lon - centerLon;
  const scale = 10 * zoom;            // control zoom sensitivity
  const x = 50 + (lonDiff * scale);    // % from left
  const y = 50 - (latDiff * scale);    // % from top
  return { x, y };
};

// Render a marker with tooltip and click → reverse geocode
<Box sx={{ position: 'absolute', left: `${pos.x}%`, top: `${pos.y}%`, transform: 'translate(-50%, -50%)' }}
     onClick={() => getLocationInfo(vehicle.location.lat, vehicle.location.lon)}>
  <Box sx={{ width: 40, height: 40, borderRadius: '50%', background: isEV ? evGrad : iceGrad }}>
    {isEV ? '⚡' : '🚗'}
  </Box>
</Box>
```
Notes:
- The map SVG draws simplified continents for context. For real cartography, swap the SVG for Leaflet/Mapbox and plot markers at lat/lon (they handle projections for you).
- `getLocationInfo` uses OpenStreetMap Nominatim reverse geocoding for human-readable addresses.

## Vehicle Movement and Staying on Roads
Movement is simulated in `cmd/simulator/main.go`:
- We plan a polyline route between two snapped points (start/end) using OSRM (public or local) and traverse segments at each tick.
- Periodically, we re-snap current position to the nearest road to correct drift.

Key snippets:
```go
// Snap a point to nearest road using OSRM Nearest API
func snapToRoad(p Location) Location {
  url := fmt.Sprintf("%s/nearest/v1/driving/%.6f,%.6f?number=1", osrmBaseURL, p.Lon, p.Lat)
  resp, _ := osrmHTTPClient.Get(url)
  // parse JSON → return snapped coordinates when available
}

// Get a drivable route polyline between two snapped locations
func fetchOSRMRoute(start, end Location) ([]Location, error) {
  url := fmt.Sprintf("%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=full&geometries=geojson", osrmBaseURL, start.Lon, start.Lat, end.Lon, end.Lat)
  // decode GeoJSON → []Location
}

// Plan a new route with retries and fallback jitter when OSRM is unavailable
func planNewRoute(s *VehicleState) { /* chooses endpoints and sets s.Route = &VehicleRoute{Points: pts} */ }

// Walk along the route in segment space based on speed and tick duration
func stepAlongRoute(s *VehicleState, tickSec float64) {
  remKm := s.SpeedKmh * (tickSec / 3600.0)
  for remKm > 0 && s.Route.SegIndex < len(s.Route.Points)-1 {
    // advance across segments; update s.Position via linear interpolation (lerp)
  }
  if s.Route.SegIndex >= len(s.Route.Points)-1 { planNewRoute(s) }
}
```
Energy and dwell model:
```go
// While stopped, refuel/charge at a rate; otherwise consume percent per km
y := s.SpeedKmh * (interval.Seconds() / 3600.0)
if s.Type == "ICE" { s.FuelPct -= y * s.ConsumePctPerKm } else { s.BatteryPct -= y * s.ConsumePctPerKm }
if stopped && lowEnergy { s.RefuelActive = true; level += s.RefillPctPerSec * dt }
```

## Reporting and Exports (CSV/PDF)
Exports are implemented in `frontend/src/components/VehicleDetail.tsx`.

CSV export (core logic):
```ts
const headers = ['timestamp','speed','fuel_level','battery_level','emissions']
const rows = data.map(d => [d.timestamp, d.speed, d.fuel_level ?? '', d.battery_level ?? '', isEV ? 0 : d.emissions])
const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n')
const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
const url = URL.createObjectURL(blob)
const a = document.createElement('a'); a.href = url; a.download = `vehicle_${vehicleId}_telemetry.csv`; a.click()
URL.revokeObjectURL(url)
```

PDF export (excerpt using jsPDF):
```ts
const doc = new jsPDF();
doc.setFontSize(20); doc.text(`Vehicle ${vehicleId} Telemetry Report`, pageWidth/2, 20, { align: 'center' })
// summary stats
doc.text(`Average Speed: ${avgSpeed.toFixed(2)} km/h`, 20, y)
// tabular data (limited rows)
doc.text(new Date(record.timestamp).toLocaleString(), x, y)
// ...
doc.save(`vehicle_${vehicleId}_telemetry_report.pdf`)
```

## Security Model & GDPR Considerations
- Password storage: bcrypt hashes (one-way) via `golang.org/x/crypto/bcrypt`. Hashes are salted and computationally expensive to reverse → not reversible encryption by design.
- Authentication: JWT HS256; claims include `user_id`, `username`, `role`, `tenant_id`, `exp`. Tokens are validated on each request by middleware; handlers read claims from context.
- Authorization: tenant scoping enforced in queries; item-level deletes verify tenant ownership. Role-based checks can be extended using the `role` claim.
- Transport security: enable HTTPS in production (`USE_HTTPS=true` with cert/key). For MQTT, prefer TLS and broker ACLs.
- Data access without login: Main REST endpoints are protected by the auth middleware. Note: the SSE stream is currently unauthenticated in dev for convenience; in production, wrap it with the auth middleware just like other endpoints.
- Mongo Express: exposed on port 8082 with admin credentials in compose. In production, disable this or restrict with network policies and strong credentials.
- GDPR:
  - Data minimization: telemetry contains vehicle data, not personal PII by default. If user data is added, update privacy notices and retention.
  - Retention: TTL index on `telemetry.timestamp` enforces automatic deletion (`TELEMETRY_TTL_DAYS`).
  - Rights: implement delete/export endpoints per user/tenant if personal data is introduced.
  - Security: protect in transit (HTTPS/MQTT TLS) and at rest (consider encrypted volumes/managed DB with at-rest encryption).
  - Consent/Lawful basis: ensure appropriate consents/contracts if tracking drivers.

## License
MIT
