package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
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
	Type         string    `json:"type"` // "ICE" or "EV"
	Status       string    `json:"status"` // "active" or "inactive"
}

func randomLocation() Location {
	return Location{
		Lat: 40.7 + rand.Float64()*0.1, // NYC area
		Lon: -74.0 + rand.Float64()*0.1,
	}
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
	
	resp, err := http.Post(apiURL+"/vehicles", "application/json", bytes.NewBuffer(data))
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

func randomTelemetry(vehicleID, vtype string) Telemetry {
	tele := Telemetry{
		VehicleID: vehicleID,
		Timestamp: time.Now(),
		Location:  randomLocation(),
		Speed:     rand.Float64()*100 + 10,
		Emissions: rand.Float64() * 50,
		Type:      vtype,
		Status:    func() string { if rand.Float64() < 0.8 { return "active" } else { return "inactive" } }(), // 80% active, 20% inactive
	}
	if vtype == "ICE" {
		tele.FuelLevel = rand.Float64() * 100
	} else {
		tele.BatteryLevel = rand.Float64() * 100
	}
	return tele
}

func sendTelemetry(apiURL string, tele Telemetry) {
	data, err := json.Marshal(tele)
	if err != nil {
		log.WithError(err).Error("Failed to marshal telemetry")
		return
	}
	resp, err := http.Post(apiURL+"/telemetry", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.WithError(err).Error("Failed to send telemetry")
		return
	}
	defer resp.Body.Close()
	log.WithFields(log.Fields{"vehicle_id": tele.VehicleID, "status": resp.Status}).Info("Sent telemetry")
}

func simulateVehicle(apiURL, vehicleID, vtype string, interval time.Duration) {
	for {
		tele := randomTelemetry(vehicleID, vtype)
		sendTelemetry(apiURL, tele)
		time.Sleep(interval)
	}
}

func main() {
	
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
	
	interval := 5 * time.Second
	
	log.WithFields(log.Fields{
		"fleet_size": fleetSize,
		"api_url":    apiURL,
		"interval":   interval,
	}).Info("Starting fleet simulation")
	
	// Create vehicles first
	vehicleIDs := make([]string, 0, fleetSize)
	for i := 0; i < fleetSize; i++ {
		vtype := []string{"ICE", "EV"}[rand.Intn(2)]
		vehicleID, err := createVehicle(apiURL, fmt.Sprintf("vehicle-%d", i+1), vtype)
		if err != nil {
			log.WithError(err).Error("Failed to create vehicle")
			continue
		}
		vehicleIDs = append(vehicleIDs, vehicleID)
	}
	
	log.WithField("created_vehicles", len(vehicleIDs)).Info("Vehicle creation completed")
	
	// Start telemetry simulation for created vehicles
	for _, vehicleID := range vehicleIDs {
		vtype := []string{"ICE", "EV"}[rand.Intn(2)]
		go simulateVehicle(apiURL, vehicleID, vtype, interval)
	}
	
	log.Info("Telemetry simulation started")
	select {} // Block forever
}