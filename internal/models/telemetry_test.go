package models

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestTelemetryMarshalUnmarshal(t *testing.T) {
	tele := Telemetry{
		Timestamp: time.Now(),
		Emissions: 10.0,
	}
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out Telemetry
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
}

func TestTelemetry_CompleteData(t *testing.T) {
	now := time.Now()
	fuelLevel := 75.0
	batteryLevel := 80.0
	
	tele := Telemetry{
		ID:           primitive.NewObjectID(),
		VehicleID:    primitive.NewObjectID(),
		Timestamp:    now,
		Location:     Location{Lat: 51.0, Lon: 0.0},
		Speed:        50.0,
		FuelLevel:    &fuelLevel,
		BatteryLevel: &batteryLevel,
		Emissions:    25.0,
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	
	// Test JSON unmarshaling
	var unmarshaled Telemetry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	
	// Verify fields
	if tele.ID != unmarshaled.ID {
		t.Errorf("ID mismatch: expected %v, got %v", tele.ID, unmarshaled.ID)
	}
	if tele.VehicleID != unmarshaled.VehicleID {
		t.Errorf("VehicleID mismatch: expected %v, got %v", tele.VehicleID, unmarshaled.VehicleID)
	}
	if tele.Speed != unmarshaled.Speed {
		t.Errorf("Speed mismatch: expected %f, got %f", tele.Speed, unmarshaled.Speed)
	}
	if tele.Emissions != unmarshaled.Emissions {
		t.Errorf("Emissions mismatch: expected %f, got %f", tele.Emissions, unmarshaled.Emissions)
	}
}

func TestTelemetry_ICEVehicle(t *testing.T) {
	fuelLevel := 75.0
	tele := Telemetry{
		ID:        primitive.NewObjectID(),
		VehicleID: primitive.NewObjectID(),
		Timestamp: time.Now(),
		Location:  Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		FuelLevel: &fuelLevel,
		Emissions: 25.0,
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	
	var unmarshaled Telemetry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	
	// ICE vehicle should have fuel level but no battery level
	if unmarshaled.FuelLevel == nil {
		t.Error("ICE vehicle should have fuel level")
	}
	if unmarshaled.BatteryLevel != nil {
		t.Error("ICE vehicle should not have battery level")
	}
}

func TestTelemetry_EVVehicle(t *testing.T) {
	batteryLevel := 80.0
	tele := Telemetry{
		ID:           primitive.NewObjectID(),
		VehicleID:    primitive.NewObjectID(),
		Timestamp:    time.Now(),
		Location:     Location{Lat: 51.0, Lon: 0.0},
		Speed:        50.0,
		BatteryLevel: &batteryLevel,
		Emissions:    10.0,
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	
	var unmarshaled Telemetry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	
	// EV vehicle should have battery level but no fuel level
	if unmarshaled.BatteryLevel == nil {
		t.Error("EV vehicle should have battery level")
	}
	if unmarshaled.FuelLevel != nil {
		t.Error("EV vehicle should not have fuel level")
	}
}

func TestTelemetry_ZeroValues(t *testing.T) {
	tele := Telemetry{
		Timestamp: time.Now(),
		Location:  Location{Lat: 0.0, Lon: 0.0},
		Speed:     0.0,
		Emissions: 0.0,
	}
	
	// Test JSON marshaling with zero values
	data, err := json.Marshal(tele)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	
	var unmarshaled Telemetry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	
	// Verify zero values are preserved
	if unmarshaled.Speed != 0.0 {
		t.Errorf("Speed should be 0.0, got %f", unmarshaled.Speed)
	}
	if unmarshaled.Emissions != 0.0 {
		t.Errorf("Emissions should be 0.0, got %f", unmarshaled.Emissions)
	}
}

func TestTelemetry_InvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": "json", "missing": "required fields"}`
	
	var tele Telemetry
	err := json.Unmarshal([]byte(invalidJSON), &tele)
	// This should not panic, even with invalid JSON
	if err == nil {
		t.Log("JSON unmarshaling succeeded with invalid data (this might be expected)")
	}
}

func TestLocation_MarshalUnmarshal(t *testing.T) {
	loc := Location{Lat: 51.0, Lon: 0.0}
	
	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	
	var unmarshaled Location
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	
	if unmarshaled.Lat != loc.Lat {
		t.Errorf("Lat mismatch: expected %f, got %f", loc.Lat, unmarshaled.Lat)
	}
	if unmarshaled.Lon != loc.Lon {
		t.Errorf("Lon mismatch: expected %f, got %f", loc.Lon, unmarshaled.Lon)
	}
}

func TestLocation_EdgeCases(t *testing.T) {
	testCases := []Location{
		{Lat: 90.0, Lon: 180.0},   // Max values
		{Lat: -90.0, Lon: -180.0}, // Min values
		{Lat: 0.0, Lon: 0.0},      // Zero values
		{Lat: 51.5074, Lon: -0.1278}, // London coordinates
	}
	
	for i, tc := range testCases {
		data, err := json.Marshal(tc)
		if err != nil {
			t.Fatalf("marshal failed for case %d: %v", i, err)
		}
		
		var unmarshaled Location
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("unmarshal failed for case %d: %v", i, err)
		}
		
		if unmarshaled.Lat != tc.Lat {
			t.Errorf("case %d: Lat mismatch: expected %f, got %f", i, tc.Lat, unmarshaled.Lat)
		}
		if unmarshaled.Lon != tc.Lon {
			t.Errorf("case %d: Lon mismatch: expected %f, got %f", i, tc.Lon, unmarshaled.Lon)
		}
	}
}

// Benchmark for Telemetry JSON marshaling
func BenchmarkTelemetryMarshal(b *testing.B) {
	tele := Telemetry{
		Timestamp: time.Now(),
		Emissions: 10.0,
	}
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(tele)
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
	}
}

func BenchmarkTelemetryUnmarshal(b *testing.B) {
	tele := Telemetry{
		Timestamp: time.Now(),
		Emissions: 10.0,
	}
	data, err := json.Marshal(tele)
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var unmarshaled Telemetry
		err := json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}