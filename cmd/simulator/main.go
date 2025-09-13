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
	"time"
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

func jitterLocation(base Location, meters float64) Location {
	latMetersPerDeg := 111320.0
	lonMetersPerDeg := 111320.0 * math.Cos(base.Lat*math.Pi/180)
	dLat := (rand.Float64()*2 - 1) * (meters / latMetersPerDeg)
	dLon := (rand.Float64()*2 - 1) * (meters / lonMetersPerDeg)
	return Location{Lat: base.Lat + dLat, Lon: base.Lon + dLon}
}

func randomLocation() Location {
	base := cities[rand.Intn(len(cities))]
	return jitterLocation(base, 500) // start close to roads
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

type VehicleRoute struct {
	Points    []Location
	SegIndex  int
	SegOffset float64 // km along current segment
}

type VehicleState struct {
	VehicleID  string
	Type       string
	Position   Location
	SpeedKmh   float64
	FuelPct    float64
	BatteryPct float64
	Route      *VehicleRoute
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
	url := fmt.Sprintf("https://router.project-osrm.org/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=full&geometries=geojson", start.Lon, start.Lat, end.Lon, end.Lat)
	resp, err := http.Get(url)
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
	start := s.Position
	// pick far city
	var end Location
	for i := 0; i < 10; i++ {
		cand := cities[rand.Intn(len(cities))]
		if haversineKm(start, cand) > 50 {
			end = jitterLocation(cand, 500)
			break
		}
	}
	pts, err := fetchOSRMRoute(start, end)
	if err != nil {
		// fallback small jitter loop
		s.Route = &VehicleRoute{Points: []Location{start, jitterLocation(start, 2000)}, SegIndex: 0, SegOffset: 0}
		return
	}
	s.Route = &VehicleRoute{Points: pts, SegIndex: 0, SegOffset: 0}
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
		em = 120 + 0.3*s.SpeedKmh
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
		// small speed noise
		s.SpeedKmh += (rand.Float64()*2 - 1) * 1.5
		if s.SpeedKmh < 15 {
			s.SpeedKmh = 15
		}
		if s.SpeedKmh > 90 {
			s.SpeedKmh = 90
		}

		stepAlongRoute(s, interval.Seconds())

		// consume energy
		km := s.SpeedKmh * (interval.Seconds() / 3600.0)
		if s.Type == "ICE" {
			s.FuelPct -= km * 0.4
			if s.FuelPct < 5 {
				s.FuelPct = 100
			}
		} else {
			s.BatteryPct -= km * 0.8
			if s.BatteryPct < 5 {
				s.BatteryPct = 100
			}
		}

		sendTelemetry(apiURL, telemetryFromState(s))
	}
}

func main() {
	// Optional JWT for protected API
	authToken = os.Getenv("SIM_AUTH_TOKEN")

	fleetSize := 10
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

	log.WithFields(log.Fields{
		"fleet_size": fleetSize,
		"api_url":    apiURL,
		"interval":   interval,
	}).Info("Starting fleet simulation")

	// Create vehicles and states
	states := make([]*VehicleState, 0, fleetSize)
	for i := 0; i < fleetSize; i++ {
		vtype := []string{"ICE", "EV"}[rand.Intn(2)]
		vehicleID, err := createVehicle(apiURL, fmt.Sprintf("vehicle-%d", i+1), vtype)
		if err != nil {
			log.WithError(err).Error("Failed to create vehicle")
			continue
		}
		start := randomLocation()
		state := &VehicleState{
			VehicleID:  vehicleID,
			Type:       vtype,
			Position:   start,
			SpeedKmh:   30 + rand.Float64()*30,
			FuelPct:    50 + rand.Float64()*50,
			BatteryPct: 50 + rand.Float64()*50,
		}
		states = append(states, state)
	}

	log.WithField("created_vehicles", len(states)).Info("Vehicle creation completed")
	if len(states) == 0 {
		log.Error("No vehicles created. Ensure SIM_AUTH_TOKEN is valid and API is reachable. Exiting.")
		time.Sleep(2 * time.Second)
		return
	}

	for _, s := range states {
		go simulateVehicle(apiURL, s, interval)
	}

	log.Info("Telemetry simulation started")
	select {} // Block forever
}
