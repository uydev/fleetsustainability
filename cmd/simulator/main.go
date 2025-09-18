package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Location represents a geographical location with latitude and longitude coordinates.
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Vehicle represents a fleet vehicle with its specifications and current status.
type Vehicle struct {
	Type            string   `json:"type"`
	Make            string   `json:"make"`
	Model           string   `json:"model"`
	Year            int      `json:"year"`
	CurrentLocation Location `json:"current_location,omitempty"`
	Status          string   `json:"status"`
}

// Telemetry represents real-time vehicle telemetry data including location, speed, and status.
type Telemetry struct {
	VehicleID    string    `json:"vehicle_id"`
	Timestamp    time.Time `json:"timestamp"`
	Location     Location  `json:"location"`
	Speed        float64   `json:"speed"`
	FuelLevel    float64   `json:"fuel_level,omitempty"`
	BatteryLevel float64   `json:"battery_level,omitempty"`
	Emissions    float64   `json:"emissions"`
	Type         string    `json:"type"`   // "ICE" or "EV"
	Status       string    `json:"status"` // "active" or "inactive"
}

// Cities for realistic routes
var cities = []Location{
	{Lat: 51.5074, Lon: -0.1278},  // London
	{Lat: 40.7128, Lon: -74.0060}, // New York
	{Lat: 40.4168, Lon: -3.7038},  // Madrid
	{Lat: 35.1856, Lon: 33.3823},  // Nicosia
	{Lat: 4.7110, Lon: -74.0721},  // Bogotá
	{Lat: 48.8566, Lon: 2.3522},   // Paris
	{Lat: 41.0082, Lon: 28.9784},  // Istanbul
	{Lat: 51.4816, Lon: -3.1791},  // Cardiff
	// Added more cities for wider global spread
	{Lat: 34.0522, Lon: -118.2437}, // Los Angeles
	{Lat: 37.7749, Lon: -122.4194}, // San Francisco
	{Lat: 52.5200, Lon: 13.4050},   // Berlin
	{Lat: 35.6762, Lon: 139.6503},  // Tokyo
	{Lat: -33.8688, Lon: 151.2093}, // Sydney
	{Lat: 1.3521, Lon: 103.8198},   // Singapore
	{Lat: -23.5505, Lon: -46.6333}, // São Paulo
	{Lat: 43.6532, Lon: -79.3832},  // Toronto
	{Lat: 25.2048, Lon: 55.2708},   // Dubai
	{Lat: 19.0760, Lon: 72.8777},   // Mumbai
	{Lat: -26.2041, Lon: 28.0473},  // Johannesburg
	{Lat: -37.8136, Lon: 144.9631}, // Melbourne
}

// loadExtraCities augments the built-in city list from environment.
// SIM_CITIES_FILE: path to JSON array of objects {"lat": number, "lon": number}
// SIM_EXTRA_CITIES: semicolon-separated list of "lat,lon" pairs (e.g., "35.68,139.65;28.61,77.20")
func loadExtraCities() {
    added := 0
    if path := os.Getenv("SIM_CITIES_FILE"); path != "" {
        if data, err := os.ReadFile(path); err == nil {
            var arr []struct{ Lat float64 `json:"lat"`; Lon float64 `json:"lon"` }
            if err := json.Unmarshal(data, &arr); err == nil {
                for _, c := range arr {
                    if c.Lat != 0 || c.Lon != 0 {
                        cities = append(cities, Location{Lat: c.Lat, Lon: c.Lon})
                        added++
                    }
                }
            } else {
                log.WithError(err).Warn("Failed to parse SIM_CITIES_FILE JSON")
            }
        } else {
            log.WithError(err).Warn("Failed to read SIM_CITIES_FILE")
        }
    }
    if extra := os.Getenv("SIM_EXTRA_CITIES"); extra != "" {
        for _, part := range strings.Split(extra, ";") {
            fields := strings.Split(strings.TrimSpace(part), ",")
            if len(fields) != 2 { continue }
            lat, err1 := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
            lon, err2 := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
            if err1 == nil && err2 == nil {
                cities = append(cities, Location{Lat: lat, Lon: lon})
                added++
            }
        }
    }
    if added > 0 {
        log.WithField("added_cities", added).Info("Loaded extra cities for simulator")
    }
}

func jitterLocation(base Location, meters float64) Location {
	latMetersPerDeg := 111320.0
	lonMetersPerDeg := 111320.0 * math.Cos(base.Lat*math.Pi/180)
	dLat := (rand.Float64()*2 - 1) * (meters / latMetersPerDeg)
	dLon := (rand.Float64()*2 - 1) * (meters / lonMetersPerDeg)
	return Location{Lat: base.Lat + dLat, Lon: base.Lon + dLon}
}

var osrmBaseURL = func() string {
	if v := os.Getenv("OSRM_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://router.project-osrm.org"
}()

var osrmHTTPClient = &http.Client{ Timeout: 4 * time.Second }

func snapToRoad(p Location) Location {
	url := fmt.Sprintf("%s/nearest/v1/driving/%.6f,%.6f?number=1", osrmBaseURL, p.Lon, p.Lat)
	resp, err := osrmHTTPClient.Get(url)
	if err != nil {
		return p
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return p
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return p
	}
	var obj struct {
		Waypoints []struct {
			Location []float64 `json:"location"`
		} `json:"waypoints"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return p
	}
	if len(obj.Waypoints) > 0 && len(obj.Waypoints[0].Location) >= 2 {
		return Location{Lat: obj.Waypoints[0].Location[1], Lon: obj.Waypoints[0].Location[0]}
	}
	return p
}

func randomLocation() Location {
    if os.Getenv("SIM_GLOBAL") == "1" {
        lat := -60 + rand.Float64()*135
        lon := -180 + rand.Float64()*360
        p := Location{Lat: lat, Lon: lon}
        if os.Getenv("SIM_SNAP_TO_ROAD") == "1" {
            return snapToRoad(p)
        }
        return p
    }
	base := cities[rand.Intn(len(cities))]
	j := jitterLocation(base, 500) // start close to roads
	if os.Getenv("SIM_SNAP_TO_ROAD") == "1" {
		return snapToRoad(j)
	}
	return j
}

var authToken string

func authorizedPost(url string, contentType string, body *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func createVehicle(apiURL string, initialVehicleID string, vtype string) (string, error) {
	makes := map[string][]string{
		"ICE": {"Ford", "Chevrolet", "Toyota", "Honda", "BMW"},
		"EV":  {"Tesla", "Nissan", "Chevrolet", "Ford", "Audi"},
	}
	models := map[string][]string{
		"ICE": {"F-150", "Silverado", "Camry", "Civic", "X5"},
		"EV":  {"Model 3", "Leaf", "Bolt", "Mach-E", "e-tron"},
	}

	make := makes[vtype][rand.Intn(len(makes[vtype]))]
	model := models[vtype][rand.Intn(len(models[vtype]))]
	year := 2020 + rand.Intn(5) // 2020-2024

	vehicle := Vehicle{
		Type:            vtype,
		Make:            make,
		Model:           model,
		Year:            year,
		CurrentLocation: randomLocation(),
		Status:          "active",
	}

	data, err := json.Marshal(vehicle)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vehicle: %w", err)
	}

	resp, err := authorizedPost(apiURL+"/vehicles", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create vehicle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("vehicle creation failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	createdVehicleID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid vehicle ID in response")
	}

	log.WithFields(log.Fields{
		"vehicle_id": createdVehicleID,
		"type":       vtype,
		"make":       make,
		"model":      model,
	}).Info("Created vehicle")

	return createdVehicleID, nil
}

// --- Routing & movement ---

// VehicleRoute describes a polyline route and current traversal state.
type VehicleRoute struct {
	Points    []Location
	SegIndex  int
	SegOffset float64 // km along current segment
}

// VehicleState holds per-vehicle dynamic state for the simulator.
type VehicleState struct {
	VehicleID      string
	Type           string
	Position       Location
	SpeedKmh       float64
	TargetSpeedKmh float64
	FuelPct        float64
	BatteryPct     float64
	Route          *VehicleRoute
	StopUntil      time.Time
	RefuelActive   bool
	// consumption model parameters (percent per km)
	ConsumePctPerKm float64
	// charging/refuel rate while stopped (percent per second)
	RefillPctPerSec float64
}

// Simulation tuning (can be overridden via env vars)
var (
    simMaxSpeedKmh    = 60.0 // max cruising speed
    simAccelKmhPerSec = 4.0  // acceleration cap (km/h per second)
)

// minimal shape of vehicles returned by backend
type existingVehicle struct {
    ID              string   `json:"id"`
    Type            string   `json:"type"`
    CurrentLocation Location `json:"current_location"`
    Status          string   `json:"status"`
}

func fetchExistingVehicles(apiURL string) ([]existingVehicle, error) {
    req, err := http.NewRequest(http.MethodGet, apiURL+"/vehicles", nil)
    if err != nil { return nil, err }
    if authToken != "" {
        req.Header.Set("Authorization", "Bearer "+authToken)
    }
    client := &http.Client{ Timeout: 10 * time.Second }
    resp, err := client.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("vehicles GET status %d", resp.StatusCode)
    }
    var rows []existingVehicle
    dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&rows); err != nil { return nil, err }
    return rows, nil
}

func haversineKm(a, b Location) float64 {
	R := 6371.0
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	s := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(s), math.Sqrt(1-s))
	return R * c
}

func lerp(a, b Location, t float64) Location {
	return Location{Lat: a.Lat + (b.Lat-a.Lat)*t, Lon: a.Lon + (b.Lon-a.Lon)*t}
}

func fetchOSRMRoute(start, end Location) ([]Location, error) {
	url := fmt.Sprintf("%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=full&geometries=geojson", osrmBaseURL, start.Lon, start.Lat, end.Lon, end.Lat)
	resp, err := osrmHTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("osrm status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var obj struct {
		Routes []struct {
			Geometry struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	if len(obj.Routes) == 0 || len(obj.Routes[0].Geometry.Coordinates) < 2 {
		return nil, fmt.Errorf("no route")
	}
	coords := obj.Routes[0].Geometry.Coordinates
	pts := make([]Location, 0, len(coords))
	for _, c := range coords {
		if len(c) < 2 {
			continue
		}
		pts = append(pts, Location{Lat: c[1], Lon: c[0]})
	}
	return pts, nil
}

func planNewRoute(s *VehicleState) {
	start := snapToRoad(s.Position)
	// Try multiple candidate endpoints, preferring nearby cities (10-60km)
	for attempt := 0; attempt < 12; attempt++ {
		var end Location
		if attempt < 8 {
			// Prefer a city within 10..60 km
			cand := cities[rand.Intn(len(cities))]
			d := haversineKm(start, cand)
			if d < 10 || d > 60 {
				continue
			}
			end = snapToRoad(jitterLocation(cand, 400))
		} else {
			// Jitter around start 5..25 km
			radius := 5000 + rand.Float64()*20000
			end = snapToRoad(jitterLocation(start, radius))
		}
		pts, err := fetchOSRMRoute(start, end)
		if err == nil && len(pts) >= 2 {
			s.Route = &VehicleRoute{Points: pts, SegIndex: 0, SegOffset: 0}
			return
		}
	}
	// As a last resort, small snapped jitter loop
	s.Route = &VehicleRoute{Points: []Location{start, snapToRoad(jitterLocation(start, 2000))}, SegIndex: 0, SegOffset: 0}
}

func stepAlongRoute(s *VehicleState, tickSec float64) {
	if s.Route == nil || len(s.Route.Points) < 2 {
		planNewRoute(s)
	}
	remKm := s.SpeedKmh * (tickSec / 3600.0)
	for remKm > 0 && s.Route.SegIndex < len(s.Route.Points)-1 {
		a := s.Route.Points[s.Route.SegIndex]
		b := s.Route.Points[s.Route.SegIndex+1]
		segLen := haversineKm(a, b)
		leftOnSeg := segLen - s.Route.SegOffset
		if remKm >= leftOnSeg {
			// advance to next segment
			s.Position = b
			s.Route.SegIndex++
			s.Route.SegOffset = 0
			remKm -= leftOnSeg
			continue
		}
		// stay on current segment
		t := (s.Route.SegOffset + remKm) / segLen
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		s.Position = lerp(a, b, t)
		s.Route.SegOffset += remKm
		remKm = 0
	}
	// if reached end, plan new
	if s.Route.SegIndex >= len(s.Route.Points)-1 {
		planNewRoute(s)
	}
}

func telemetryFromState(s *VehicleState) Telemetry {
	em := 0.0
	if s.Type == "ICE" {
		if s.SpeedKmh < 1 {
			em = 0
		} else {
			em = 120 + 0.3*s.SpeedKmh
		}
	}
	t := Telemetry{
		VehicleID: s.VehicleID,
		Timestamp: time.Now(),
		Location:  s.Position,
		Speed:     s.SpeedKmh,
		Emissions: em,
		Type:      s.Type,
		Status:    "active",
	}
	if s.Type == "ICE" {
		t.FuelLevel = s.FuelPct
	} else {
		t.BatteryLevel = s.BatteryPct
	}
	return t
}

func sendTelemetry(apiURL string, tele Telemetry) {
	data, err := json.Marshal(tele)
	if err != nil {
		log.WithError(err).Error("Failed to marshal telemetry")
		return
	}
	if broker := os.Getenv("MQTT_BROKER_URL"); broker != "" && os.Getenv("SIM_USE_MQTT") == "1" {
		// Publish to MQTT
		opts := mqtt.NewClientOptions().AddBroker(broker)
		if u := os.Getenv("MQTT_USERNAME"); u != "" { opts.SetUsername(u) }
		if p := os.Getenv("MQTT_PASSWORD"); p != "" { opts.SetPassword(p) }
		opts.SetClientID("fleet-sim-" + tele.VehicleID)
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.WithError(token.Error()).Error("MQTT connect failed (simulator)")
			return
		}
		topic := os.Getenv("MQTT_TELEMETRY_TOPIC")
		if topic == "" { topic = "fleet/telemetry" }
		if token := client.Publish(topic, 1, false, data); token.Wait() && token.Error() != nil {
			log.WithError(token.Error()).Error("MQTT publish failed")
		}
		client.Disconnect(250)
		log.WithFields(log.Fields{"vehicle_id": tele.VehicleID, "topic": topic}).Info("Published telemetry via MQTT")
		return
	}
	// Default: send via HTTP
	resp, err := authorizedPost(apiURL+"/telemetry", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.WithError(err).Error("Failed to send telemetry")
		return
	}
	defer resp.Body.Close()
	log.WithFields(log.Fields{"vehicle_id": tele.VehicleID, "status": resp.Status}).Info("Sent telemetry")
}

func simulateVehicle(apiURL string, s *VehicleState, interval time.Duration) {
	if s.Route == nil {
		planNewRoute(s)
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		// Random stops and dwell logic: guarantee explicit 0-speed samples while dwelling
		if time.Now().After(s.StopUntil) && rand.Float64() < 0.02 {
			s.StopUntil = time.Now().Add(time.Duration(10+rand.Intn(35)) * time.Second)
			s.TargetSpeedKmh = 0
		}
		if time.Now().Before(s.StopUntil) {
			s.TargetSpeedKmh = 0
			s.SpeedKmh = 0
		} else {
			if s.TargetSpeedKmh == 0 {
				s.TargetSpeedKmh = 20 + rand.Float64()*30
			}
			s.TargetSpeedKmh += (rand.Float64()*2 - 1) * 1.0
			if s.TargetSpeedKmh < 0 { s.TargetSpeedKmh = 0 }
			if s.TargetSpeedKmh > simMaxSpeedKmh { s.TargetSpeedKmh = simMaxSpeedKmh }

			// Accelerate/decelerate towards target with rate limit
			maxDelta := simAccelKmhPerSec * interval.Seconds()
			delta := s.TargetSpeedKmh - s.SpeedKmh
			if delta > maxDelta {
				s.SpeedKmh += maxDelta
			} else if delta < -maxDelta {
				s.SpeedKmh -= maxDelta
			} else {
				s.SpeedKmh = s.TargetSpeedKmh
			}
			if s.SpeedKmh < 0 { s.SpeedKmh = 0 }
		}

		stepAlongRoute(s, interval.Seconds())

		// Periodically re-snap to nearest road to keep alignment (lightweight correction)
		if rand.Float64() < 0.1 {
			s.Position = snapToRoad(s.Position)
		}

		// correlated consumption/refill
		km := s.SpeedKmh * (interval.Seconds() / 3600.0)
		if s.Type == "ICE" {
			// consume by distance traveled
			s.FuelPct -= km * s.ConsumePctPerKm
			// refuel only when stopped and low, randomly
			if s.SpeedKmh < 0.5 {
				if s.RefuelActive || (rand.Float64() < 0.03 && s.FuelPct < 30) {
					s.RefuelActive = true
					s.FuelPct += s.RefillPctPerSec * interval.Seconds()
					if s.FuelPct >= 70+rand.Float64()*25 { // stop 70-95
						s.RefuelActive = false
					}
				}
			} else {
				s.RefuelActive = false
			}
			if s.FuelPct < 0 { s.FuelPct = 0 }
			if s.FuelPct > 100 { s.FuelPct = 100 }
		} else {
			s.BatteryPct -= km * s.ConsumePctPerKm
			if s.SpeedKmh < 0.5 {
				if s.RefuelActive || (rand.Float64() < 0.04 && s.BatteryPct < 25) {
					s.RefuelActive = true
					s.BatteryPct += s.RefillPctPerSec * interval.Seconds()
					if s.BatteryPct >= 80+rand.Float64()*20 { // stop 80-100
						s.RefuelActive = false
					}
				}
			} else {
				s.RefuelActive = false
			}
			if s.BatteryPct < 0 { s.BatteryPct = 0 }
			if s.BatteryPct > 100 { s.BatteryPct = 100 }
		}

		sendTelemetry(apiURL, telemetryFromState(s))
	}
}

func main() {
	// Optional JWT for protected API
	authToken = os.Getenv("SIM_AUTH_TOKEN")

	fleetSize := 50
	if val := os.Getenv("FLEET_SIZE"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			fleetSize = n
		}
	}

	apiURL := os.Getenv("API_BASE_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8081/api"
	}

	interval := 2 * time.Second
	if v := os.Getenv("SIM_TICK_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			interval = time.Duration(n) * time.Second
		}
	}

	// Optional tuning via env vars
	if v := os.Getenv("SIM_MAX_SPEED_KMH"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 5 {
			simMaxSpeedKmh = n
		}
	}
	if v := os.Getenv("SIM_ACCEL_KMH_PER_S"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0.1 {
			simAccelKmhPerSec = n
		}
	}

	loadExtraCities()

	log.WithFields(log.Fields{
		"fleet_size": fleetSize,
		"api_url":    apiURL,
		"interval":   interval,
		"osrm":       osrmBaseURL,
		"max_kmh":    simMaxSpeedKmh,
		"accel_kmhps": simAccelKmhPerSec,
	}).Info("Starting fleet simulation")

	// Create vehicles and states
	states := make([]*VehicleState, 0, fleetSize)
	useExisting := os.Getenv("SIM_USE_EXISTING") == "1"
	if useExisting {
		// Try to animate existing vehicles from backend
		rows, err := fetchExistingVehicles(apiURL)
		if err != nil {
			log.WithError(err).Warn("Failed to fetch existing vehicles; falling back to creating new ones")
		} else if len(rows) > 0 {
			log.WithField("existing_count", len(rows)).Info("Using existing vehicles for simulation")
			for _, v := range rows {
				vtype := v.Type
				if vtype != "ICE" && vtype != "EV" {
					if rand.Intn(2) == 0 { vtype = "ICE" } else { vtype = "EV" }
				}
				pos := v.CurrentLocation
				if pos.Lat == 0 && pos.Lon == 0 {
					pos = randomLocation()
				}
				state := &VehicleState{
					VehicleID:        v.ID,
					Type:             vtype,
					Position:         pos,
					SpeedKmh:         0,
					TargetSpeedKmh:   30 + rand.Float64()*30,
					FuelPct:          50 + rand.Float64()*50,
					BatteryPct:       50 + rand.Float64()*50,
					ConsumePctPerKm:  func() float64 { if vtype=="ICE" { return 0.08 + rand.Float64()*0.05 } ; return 0.12 + rand.Float64()*0.06 }(),
					RefillPctPerSec:  func() float64 { if vtype=="ICE" { return 0.25 + rand.Float64()*0.20 } ; return 0.40 + rand.Float64()*0.30 }(),
				}
				states = append(states, state)
			}
		}
	}
	// If not using existing or none found, create new vehicles as before
	if len(states) == 0 {
		for i := 0; i < fleetSize; i++ {
			vtype := []string{"ICE", "EV"}[rand.Intn(2)]
			vehicleID, err := createVehicle(apiURL, fmt.Sprintf("vehicle-%d", i+1), vtype)
			if err != nil {
				log.WithError(err).Error("Failed to create vehicle")
				continue
			}
			start := randomLocation()
			state := &VehicleState{
				VehicleID:        vehicleID,
				Type:             vtype,
				Position:         start,
				SpeedKmh:         0,
				TargetSpeedKmh:   30 + rand.Float64()*30,
				FuelPct:          50 + rand.Float64()*50,
				BatteryPct:       50 + rand.Float64()*50,
				ConsumePctPerKm:  func() float64 { if vtype=="ICE" { return 0.08 + rand.Float64()*0.05 } ; return 0.12 + rand.Float64()*0.06 }(),
				RefillPctPerSec:  func() float64 { if vtype=="ICE" { return 0.25 + rand.Float64()*0.20 } ; return 0.40 + rand.Float64()*0.30 }(),
			}
			states = append(states, state)
		}
	}

	log.WithField("created_vehicles", len(states)).Info("Vehicle creation completed")
	if len(states) == 0 {
		log.Error("No vehicles created. Ensure SIM_AUTH_TOKEN is valid and API is reachable. Exiting.")
		time.Sleep(2 * time.Second)
		return
	}

	for _, s := range states {
		// Ensure starting point is snapped to road
		s.Position = snapToRoad(s.Position)
		go simulateVehicle(apiURL, s, interval)
	}

	log.Info("Telemetry simulation started")
	select {} // Block forever
}
