package models

import (
	"encoding/json"
	"testing"
	"time"
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