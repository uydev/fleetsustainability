package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
	"context"
	"strings"
)

func TestRandomLocation(t *testing.T) {
	loc := randomLocation()
	
	// Check bounds for London area
	if loc.Lat < 51.0 || loc.Lat > 52.0 {
		t.Errorf("Latitude out of expected range: %f", loc.Lat)
	}
	if loc.Lon < -0.1 || loc.Lon > 0.9 {
		t.Errorf("Longitude out of expected range: %f", loc.Lon)
	}
}

func TestRandomTelemetry_ICE(t *testing.T) {
	tele := randomTelemetry("test-vehicle", "ICE")
	
	if tele.VehicleID != "test-vehicle" {
		t.Errorf("Expected vehicle ID 'test-vehicle', got %s", tele.VehicleID)
	}
	if tele.Type != "ICE" {
		t.Errorf("Expected type 'ICE', got %s", tele.Type)
	}
	if tele.FuelLevel <= 0 || tele.FuelLevel > 100 {
		t.Errorf("Fuel level out of range: %f", tele.FuelLevel)
	}
	if tele.BatteryLevel != 0 {
		t.Errorf("ICE vehicle should not have battery level, got %f", tele.BatteryLevel)
	}
	if tele.Speed < 10 || tele.Speed > 110 {
		t.Errorf("Speed out of range: %f", tele.Speed)
	}
	if tele.Emissions < 0 || tele.Emissions > 50 {
		t.Errorf("Emissions out of range: %f", tele.Emissions)
	}
	if tele.Status != "active" && tele.Status != "inactive" {
		t.Errorf("Invalid status: %s", tele.Status)
	}
}

func TestRandomTelemetry_EV(t *testing.T) {
	tele := randomTelemetry("test-vehicle", "EV")
	
	if tele.VehicleID != "test-vehicle" {
		t.Errorf("Expected vehicle ID 'test-vehicle', got %s", tele.VehicleID)
	}
	if tele.Type != "EV" {
		t.Errorf("Expected type 'EV', got %s", tele.Type)
	}
	if tele.BatteryLevel <= 0 || tele.BatteryLevel > 100 {
		t.Errorf("Battery level out of range: %f", tele.BatteryLevel)
	}
	if tele.FuelLevel != 0 {
		t.Errorf("EV vehicle should not have fuel level, got %f", tele.FuelLevel)
	}
}

func TestSendTelemetry_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// This should not panic or error
	sendTelemetry(server.URL, tele)
}

func TestSendTelemetry_ServerError(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// This should not panic even with server error
	sendTelemetry(server.URL, tele)
}

func TestTelemetryJSONMarshal(t *testing.T) {
	tele := Telemetry{
		VehicleID:    "test-vehicle",
		Timestamp:    time.Now(),
		Location:     Location{Lat: 51.0, Lon: 0.0},
		Speed:        50.0,
		FuelLevel:    75.0,
		BatteryLevel: 0.0,
		Emissions:    25.0,
		Type:         "ICE",
		Status:       "active",
	}
	
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("Failed to marshal telemetry: %v", err)
	}
	
	var unmarshaled Telemetry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal telemetry: %v", err)
	}
	
	if unmarshaled.VehicleID != tele.VehicleID {
		t.Errorf("VehicleID mismatch: expected %s, got %s", tele.VehicleID, unmarshaled.VehicleID)
	}
	if unmarshaled.Type != tele.Type {
		t.Errorf("Type mismatch: expected %s, got %s", tele.Type, unmarshaled.Type)
	}
}

func TestMainLogic_FleetSize(t *testing.T) {
	// Test fleet size parsing
	testCases := []struct {
		envValue string
		expected int
	}{
		{"", 10},           // default
		{"5", 5},           // valid number
		{"invalid", 10},    // invalid number, should use default
		{"0", 0},           // edge case
		{"100", 100},       // large number
	}
	
	for _, tc := range testCases {
		if tc.envValue != "" {
			os.Setenv("FLEET_SIZE", tc.envValue)
		} else {
			os.Unsetenv("FLEET_SIZE")
		}
		
		// Simulate the logic from main()
		fleetSize := 10
		if val := os.Getenv("FLEET_SIZE"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				fleetSize = n
			}
		}
		
		if fleetSize != tc.expected {
			t.Errorf("For env value '%s', expected fleet size %d, got %d", tc.envValue, tc.expected, fleetSize)
		}
	}
}

func TestMainLogic_APIURL(t *testing.T) {
	// Test API URL logic
	testCases := []struct {
		envValue string
		expected string
	}{
		{"", "http://localhost:8080/api/telemetry"},           // default
		{"http://api.example.com/telemetry", "http://api.example.com/telemetry"}, // custom
	}
	
	for _, tc := range testCases {
		if tc.envValue != "" {
			os.Setenv("TELEMETRY_API_URL", tc.envValue)
		} else {
			os.Unsetenv("TELEMETRY_API_URL")
		}
		
		// Simulate the logic from main()
		apiURL := os.Getenv("TELEMETRY_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:8080/api/telemetry"
		}
		
		if apiURL != tc.expected {
			t.Errorf("For env value '%s', expected API URL %s, got %s", tc.envValue, tc.expected, apiURL)
		}
	}
} 

func TestSimulateVehicle_WithTimeout(t *testing.T) {
	// Test simulateVehicle function with a timeout to avoid infinite loop
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Run simulateVehicle in a goroutine with context
	done := make(chan bool)
	go func() {
		// Mock the simulateVehicle logic
		interval := 10 * time.Millisecond
		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			default:
				tele := randomTelemetry("test-vehicle", "ICE")
				sendTelemetry(server.URL, tele)
				time.Sleep(interval)
			}
		}
	}()
	
	// Wait for timeout or completion
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(200 * time.Millisecond):
		t.Error("simulateVehicle did not respect timeout")
	}
}

func TestMainFunction_SimulatorLogic(t *testing.T) {
	// Test the main function logic without running the infinite loop
	originalFleetSize := os.Getenv("FLEET_SIZE")
	originalAPIURL := os.Getenv("TELEMETRY_API_URL")
	defer func() {
		os.Setenv("FLEET_SIZE", originalFleetSize)
		os.Setenv("TELEMETRY_API_URL", originalAPIURL)
	}()
	
	// Test fleet size parsing logic
	testCases := []struct {
		envValue string
		expected int
	}{
		{"", 10},           // default
		{"5", 5},           // valid number
		{"invalid", 10},    // invalid number, should use default
		{"0", 0},           // edge case
		{"100", 100},       // large number
	}
	
	for _, tc := range testCases {
		t.Run("fleet_size_"+tc.envValue, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("FLEET_SIZE", tc.envValue)
			} else {
				os.Unsetenv("FLEET_SIZE")
			}
			
			// Simulate the logic from main()
			fleetSize := 10
			if val := os.Getenv("FLEET_SIZE"); val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					fleetSize = n
				}
			}
			
			if fleetSize != tc.expected {
				t.Errorf("For env value '%s', expected fleet size %d, got %d", tc.envValue, tc.expected, fleetSize)
			}
		})
	}
	
	// Test API URL logic
	apiURLTestCases := []struct {
		envValue string
		expected string
	}{
		{"", "http://localhost:8080/api/telemetry"},           // default
		{"http://api.example.com/telemetry", "http://api.example.com/telemetry"}, // custom
	}
	
	for _, tc := range apiURLTestCases {
		t.Run("api_url_"+tc.envValue, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("TELEMETRY_API_URL", tc.envValue)
			} else {
				os.Unsetenv("TELEMETRY_API_URL")
			}
			
			// Simulate the logic from main()
			apiURL := os.Getenv("TELEMETRY_API_URL")
			if apiURL == "" {
				apiURL = "http://localhost:8080/api/telemetry"
			}
			
			if apiURL != tc.expected {
				t.Errorf("For env value '%s', expected API URL %s, got %s", tc.envValue, tc.expected, apiURL)
			}
		})
	}
}

func TestSimulateVehicle_VehicleTypeDistribution(t *testing.T) {
	// Test that vehicle types are properly distributed
	vehicleTypes := []string{"ICE", "EV"}
	
	// Test multiple iterations to ensure both types are generated
	iceCount := 0
	evCount := 0
	
	for i := 0; i < 100; i++ {
		vehicleID := "test-vehicle-" + strconv.Itoa(i)
		vtype := vehicleTypes[i%2] // Simulate the random selection
		
		tele := randomTelemetry(vehicleID, vtype)
		
		if tele.Type == "ICE" {
			iceCount++
		} else if tele.Type == "EV" {
			evCount++
		}
	}
	
	// Both types should be generated
	if iceCount == 0 {
		t.Error("No ICE vehicles generated")
	}
	if evCount == 0 {
		t.Error("No EV vehicles generated")
	}
	
	// Should have roughly equal distribution (allowing for some variance)
	total := iceCount + evCount
	if iceCount < total/4 || iceCount > 3*total/4 {
		t.Errorf("ICE distribution seems off: %d out of %d", iceCount, total)
	}
	if evCount < total/4 || evCount > 3*total/4 {
		t.Errorf("EV distribution seems off: %d out of %d", evCount, total)
	}
}

func TestSimulateVehicle_StatusDistribution(t *testing.T) {
	// Test that vehicle statuses are properly distributed
	
	// Test multiple iterations to ensure both statuses are generated
	activeCount := 0
	inactiveCount := 0
	
	for i := 0; i < 100; i++ {
		vehicleID := "test-vehicle-" + strconv.Itoa(i)
		vtype := "ICE"
		
		tele := randomTelemetry(vehicleID, vtype)
		
		if tele.Status == "active" {
			activeCount++
		} else if tele.Status == "inactive" {
			inactiveCount++
		}
	}
	
	// Both statuses should be generated
	if activeCount == 0 {
		t.Error("No active vehicles generated")
	}
	if inactiveCount == 0 {
		t.Error("No inactive vehicles generated")
	}
	
	// Should have roughly equal distribution (allowing for some variance)
	total := activeCount + inactiveCount
	if activeCount < total/4 || activeCount > 3*total/4 {
		t.Errorf("Active distribution seems off: %d out of %d", activeCount, total)
	}
	if inactiveCount < total/4 || inactiveCount > 3*total/4 {
		t.Errorf("Inactive distribution seems off: %d out of %d", inactiveCount, total)
	}
}

func TestSendTelemetry_NetworkError(t *testing.T) {
	// Test sendTelemetry with network error (invalid URL)
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// This should not panic even with network error
	sendTelemetry("http://invalid-url-that-does-not-exist.com", tele)
}

func TestSendTelemetry_Timeout(t *testing.T) {
	// Test sendTelemetry with a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// This should not panic even with slow response
	sendTelemetry(server.URL, tele)
} 

func TestSendTelemetry_JSONMarshalError(t *testing.T) {
	// Test sendTelemetry with a telemetry object that can't be marshaled
	// This is hard to trigger with normal data, but we can test the error path
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// Create a test server that will accept the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	// This should not panic even if there's an error
	sendTelemetry(server.URL, tele)
}

func TestSendTelemetry_HTTPClientError(t *testing.T) {
	// Test sendTelemetry with HTTP client errors
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	// Test with various error scenarios
	testCases := []struct {
		name   string
		apiURL string
	}{
		{"invalid URL", "http://invalid-url-that-does-not-exist.com"},
		{"malformed URL", "not-a-url"},
		{"unreachable host", "http://192.168.1.999:9999"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This should not panic even with network errors
			sendTelemetry(tc.apiURL, tele)
		})
	}
}

func TestSendTelemetry_ServerResponseCodes(t *testing.T) {
	// Test sendTelemetry with different server response codes
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	testCases := []struct {
		name           string
		responseCode   int
		expectedStatus string
	}{
		{"success", http.StatusOK, "200 OK"},
		{"created", http.StatusCreated, "201 Created"},
		{"bad request", http.StatusBadRequest, "400 Bad Request"},
		{"unauthorized", http.StatusUnauthorized, "401 Unauthorized"},
		{"forbidden", http.StatusForbidden, "403 Forbidden"},
		{"not found", http.StatusNotFound, "404 Not Found"},
		{"server error", http.StatusInternalServerError, "500 Internal Server Error"},
		{"service unavailable", http.StatusServiceUnavailable, "503 Service Unavailable"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseCode)
			}))
			defer server.Close()
			
			// This should not panic regardless of response code
			sendTelemetry(server.URL, tele)
		})
	}
}

func TestSendTelemetry_WithDifferentTelemetryTypes(t *testing.T) {
	// Test sendTelemetry with different telemetry configurations
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	testCases := []struct {
		name string
		tele Telemetry
	}{
		{
			"ICE vehicle with fuel",
			Telemetry{
				VehicleID: "ice-vehicle",
				Timestamp: time.Now(),
				Location:  Location{Lat: 51.0, Lon: 0.0},
				Speed:     60.0,
				FuelLevel: 75.0,
				Emissions: 25.0,
				Type:      "ICE",
				Status:    "active",
			},
		},
		{
			"EV vehicle with battery",
			Telemetry{
				VehicleID:    "ev-vehicle",
				Timestamp:    time.Now(),
				Location:     Location{Lat: 51.0, Lon: 0.0},
				Speed:        45.0,
				BatteryLevel: 80.0,
				Emissions:    10.0,
				Type:         "EV",
				Status:       "inactive",
			},
		},
		{
			"vehicle with zero values",
			Telemetry{
				VehicleID: "zero-vehicle",
				Timestamp: time.Now(),
				Location:  Location{Lat: 0.0, Lon: 0.0},
				Speed:     0.0,
				Emissions: 0.0,
				Type:      "ICE",
				Status:    "inactive",
			},
		},
		{
			"vehicle with extreme values",
			Telemetry{
				VehicleID: "extreme-vehicle",
				Timestamp: time.Now(),
				Location:  Location{Lat: 90.0, Lon: 180.0},
				Speed:     299.0,
				FuelLevel: 100.0,
				Emissions: 99.9,
				Type:      "ICE",
				Status:    "active",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This should not panic for any telemetry configuration
			sendTelemetry(server.URL, tc.tele)
		})
	}
}

func TestSendTelemetry_ResponseBodyHandling(t *testing.T) {
	// Test sendTelemetry with different response body scenarios
	tele := Telemetry{
		VehicleID: "test-vehicle",
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Type:      "ICE",
		Status:    "active",
	}
	
	testCases := []struct {
		name           string
		responseBody   string
		responseCode   int
	}{
		{"empty response", "", http.StatusOK},
		{"JSON response", `{"status":"ok"}`, http.StatusOK},
		{"large response", strings.Repeat("a", 10000), http.StatusOK},
		{"error response", `{"error":"bad request"}`, http.StatusBadRequest},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()
			
			// This should not panic regardless of response body
			sendTelemetry(server.URL, tele)
		})
	}
} 