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
    "math"
    "fmt"

    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/ukydev/fleet-sustainability/internal/db"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type mockTelemetryCollection struct {
    insertErr error
    findErr   error
    results   []models.Telemetry
}

func (m *mockTelemetryCollection) InsertOne(ctx context.Context, doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
    return nil, m.insertErr
}

// Mock cursor for testing
type mockCursor struct {
    results []models.Telemetry
}

func (m *mockCursor) All(ctx context.Context, out interface{}) error {
    ptr, ok := out.(*[]models.Telemetry)
    if !ok {
        return errors.New("wrong type for out")
    }
    *ptr = m.results
    return nil
}
func (m *mockCursor) Close(ctx context.Context) error { return nil }

// Update mockTelemetryCollection.Find to return (db.TelemetryCursor, error) to match the interface. Return nil as the error in the mock implementation.
func (m *mockTelemetryCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (db.TelemetryCursor, error) {
    if m.findErr != nil {
        return nil, m.findErr
    }
    // Debug: print the filter
    fmt.Printf("DEBUG: filter = %#v\n", filter)
    results := m.results
    if f, ok := filter.(primitive.M); ok {
        fmt.Printf("DEBUG: filter keys = %v\n", f)
        if ts, ok := f["timestamp"].(primitive.M); ok {
            fmt.Printf("DEBUG: timestamp filter = %v\n", ts)
            var filtered []models.Telemetry
            for _, t := range results {
                inRange := true
                if gte, ok := ts["$gte"].(time.Time); ok {
                    fmt.Printf("DEBUG: $gte = %v\n", gte)
                    if t.Timestamp.Before(gte) {
                        inRange = false
                    }
                }
                if lte, ok := ts["$lte"].(time.Time); ok {
                    fmt.Printf("DEBUG: $lte = %v\n", lte)
                    if t.Timestamp.After(lte) {
                        inRange = false
                    }
                }
                if inRange {
                    filtered = append(filtered, t)
                }
            }
            results = filtered
        }
    }
    return &mockCursor{results: results}, nil
}

func (m *mockTelemetryCollection) InsertTelemetry(ctx context.Context, telemetry models.Telemetry) error {
    return m.insertErr // simulate DB error if set
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
        "type":       "EV",
        "status":     "active",
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
        "type":       "EV",
        "status":     "active",
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

func TestTelemetryHandler_GET_TimeRange(t *testing.T) {
    now := time.Now().UTC()
    inRange := models.Telemetry{Timestamp: now}
    outOfRange := models.Telemetry{Timestamp: now.Add(-48 * time.Hour)}
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{results: []models.Telemetry{inRange, outOfRange}}}
    from := now.Add(-1 * time.Hour).Format(time.RFC3339)
    to := now.Add(1 * time.Hour).Format(time.RFC3339)
    req := httptest.NewRequest(http.MethodGet, "/api/telemetry?from="+from+"&to="+to, nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    var res []models.Telemetry
    json.NewDecoder(w.Body).Decode(&res)
    if len(res) != 1 {
        t.Errorf("expected 1 record in range, got %d", len(res))
    }
}

func TestTelemetryHandler_GET_TimeRange_InvalidParams(t *testing.T) {
    handler := &TelemetryHandler{Collection: &mockTelemetryCollection{results: []models.Telemetry{}}}
    req := httptest.NewRequest(http.MethodGet, "/api/telemetry?from=badtime", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400 for invalid time param, got %d", w.Code)
    }
}

func TestTelemetryMetricsHandler_Basic(t *testing.T) {
    // Simulate 2 EV, 1 ICE, with various metrics
    now := time.Now()
    ev1 := models.Telemetry{Timestamp: now, BatteryLevel: floatPtr(80), Emissions: 10}
    ev2 := models.Telemetry{Timestamp: now, BatteryLevel: floatPtr(60), Emissions: 12}
    ice := models.Telemetry{Timestamp: now, FuelLevel: floatPtr(50), Emissions: 30}
    mock := &mockTelemetryCollection{results: []models.Telemetry{ev1, ev2, ice}}
    req := httptest.NewRequest(http.MethodGet, "/api/telemetry/metrics", nil)
    w := httptest.NewRecorder()
    metricsHandler := TelemetryMetricsHandler{Collection: mock}
    metricsHandler.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    var res map[string]interface{}
    json.NewDecoder(w.Body).Decode(&res)
    if math.Abs(res["ev_percent"].(float64)-66.66666666666667) > 0.01 {
        t.Errorf("expected ev_percent ~66.66, got %v", res["ev_percent"])
    }
    if res["total_emissions"] != 52.0 {
        t.Errorf("expected total_emissions 52, got %v", res["total_emissions"])
    }
}

func floatPtr(f float64) *float64 { return &f }

// Add more tests as needed for edge cases