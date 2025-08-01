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