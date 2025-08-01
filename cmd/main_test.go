package main

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type mockTelemetryCollection struct {
    insertErr error
    findErr   error
    results   []models.Telemetry
}

func (m *mockTelemetryCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
    return nil, m.insertErr
}

func (m *mockTelemetryCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
    return nil, m.findErr
}

func TestTelemetryHandler_POST_InvalidJSON(t *testing.T) {
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{}}
    req := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewBuffer([]byte("{bad json")))
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}

func TestTelemetryHandler_POST_DBError(t *testing.T) {
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{insertErr: errors.New("db error")}}
    tele := map[string]interface{}{
        "vehicle_id": "507f1f77bcf86cd799439011",
        "timestamp":  time.Now().Format(time.RFC3339),
        "location":   map[string]float64{"lat": 51.0, "lon": 0.1},
        "speed":      42.0,
        "emissions":  10.0,
    }
    data, _ := json.Marshal(tele)
    req := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewBuffer(data))
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusInternalServerError {
        t.Errorf("expected 500, got %d", w.Code)
    }
}

func TestTelemetryHandler_POST_Valid(t *testing.T) {
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{}}
    tele := map[string]interface{}{
        "vehicle_id": "507f1f77bcf86cd799439011",
        "timestamp":  time.Now().Format(time.RFC3339),
        "location":   map[string]float64{"lat": 51.0, "lon": 0.1},
        "speed":      42.0,
        "emissions":  10.0,
    }
    data, _ := json.Marshal(tele)
    req := httptest.NewRequest(http.MethodPost, "/api/telemetry", bytes.NewBuffer(data))
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
}

func TestTelemetryHandler_GET_DBError(t *testing.T) {
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{findErr: errors.New("db error")}}
    req := httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusInternalServerError {
        t.Errorf("expected 500, got %d", w.Code)
    }
}

// Add more tests as needed for edge cases