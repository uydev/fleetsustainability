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

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

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
		Lat: 51.0 + rand.Float64(),
		Lon: -0.1 + rand.Float64(),
	}
}

func randomTelemetry(vehicleID, vtype string) Telemetry {
	tele := Telemetry{
		VehicleID: vehicleID,
		Timestamp: time.Now(),
		Location:  randomLocation(),
		Speed:     rand.Float64()*100 + 10,
		Emissions: rand.Float64() * 50,
		Type:      vtype,
		Status:    []string{"active", "inactive"}[rand.Intn(2)],
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
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
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
	rand.Seed(time.Now().UnixNano())
	fleetSize := 10
	if val := os.Getenv("FLEET_SIZE"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			fleetSize = n
		}
	}
	apiURL := os.Getenv("TELEMETRY_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080/api/telemetry"
	}
	interval := 5 * time.Second
	for i := 0; i < fleetSize; i++ {
		vehicleID := fmt.Sprintf("vehicle-%d", i+1)
		vtype := []string{"ICE", "EV"}[rand.Intn(2)]
		go simulateVehicle(apiURL, vehicleID, vtype, interval)
	}
	select {} // Block forever
}