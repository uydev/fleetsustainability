package main

import (
    "context"
    "encoding/json"
    "io"
    log "github.com/sirupsen/logrus"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/ukydev/fleet-sustainability/internal/db"
    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/joho/godotenv"
)

// TelemetryHandler handles telemetry API requests with injected collection.
type TelemetryHandler struct {
    Collection db.TelemetryCollection
}

// ServeHTTP processes HTTP requests for telemetry data.
func (h *TelemetryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodPost:
        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "Failed to read body", http.StatusBadRequest)
            return
        }
        var teleIn struct {
            VehicleID    string           `json:"vehicle_id"`
            Timestamp    string           `json:"timestamp"`
            Location     models.Location  `json:"location"`
            Speed        float64          `json:"speed"`
            FuelLevel    float64          `json:"fuel_level,omitempty"`
            BatteryLevel float64          `json:"battery_level,omitempty"`
            Emissions    float64          `json:"emissions"`
            Type         string           `json:"type"`
            Status       string           `json:"status"`
        }
        if err := json.Unmarshal(body, &teleIn); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
        // Input validation and sanitization
        if teleIn.VehicleID == "" {
            http.Error(w, "vehicle_id is required", http.StatusBadRequest)
            return
        }
        if teleIn.Timestamp == "" {
            http.Error(w, "timestamp is required", http.StatusBadRequest)
            return
        }
        if teleIn.Type != "ICE" && teleIn.Type != "EV" {
            http.Error(w, "type must be 'ICE' or 'EV'", http.StatusBadRequest)
            return
        }
        if teleIn.Status != "active" && teleIn.Status != "inactive" {
            http.Error(w, "status must be 'active' or 'inactive'", http.StatusBadRequest)
            return
        }
        if teleIn.Speed < 0 || teleIn.Speed > 300 {
            http.Error(w, "speed out of range", http.StatusBadRequest)
            return
        }
        if teleIn.Emissions < 0 {
            http.Error(w, "emissions must be non-negative", http.StatusBadRequest)
            return
        }
        // Optionally, add more checks for FuelLevel, BatteryLevel, Location, etc.
        vehicleObjID := primitive.NewObjectID()
        timestamp, _ := time.Parse(time.RFC3339, teleIn.Timestamp)
        var fuelPtr, batteryPtr *float64
        if teleIn.FuelLevel != 0 {
            fuelPtr = &teleIn.FuelLevel
        }
        if teleIn.BatteryLevel != 0 {
            batteryPtr = &teleIn.BatteryLevel
        }
        tele := models.Telemetry{
            VehicleID:    vehicleObjID, // In a real app, map string to ObjectID
            Timestamp:    timestamp,
            Location:     teleIn.Location,
            Speed:        teleIn.Speed,
            FuelLevel:    fuelPtr,
            BatteryLevel: batteryPtr,
            Emissions:    teleIn.Emissions,
        }
        err = h.Collection.InsertTelemetry(r.Context(), tele)
        if err != nil {
            http.Error(w, "Failed to store telemetry", http.StatusInternalServerError)
            return
        }
        log.WithFields(log.Fields{"vehicle_id": tele.VehicleID}).Info("Stored telemetry")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    case http.MethodGet:
        // Support time range filtering
        fromStr := r.URL.Query().Get("from")
        toStr := r.URL.Query().Get("to")
        var filter bson.M = bson.M{}
        if fromStr != "" || toStr != "" {
            filter["timestamp"] = bson.M{}
            if fromStr != "" {
                from, err := time.Parse(time.RFC3339, fromStr)
                if err != nil {
                    http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
                    return
                }
                filter["timestamp"].(bson.M)["$gte"] = from
            }
            if toStr != "" {
                to, err := time.Parse(time.RFC3339, toStr)
                if err != nil {
                    http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
                    return
                }
                filter["timestamp"].(bson.M)["$lte"] = to
            }
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        opts := options.Find().SetSort(bson.D{{"timestamp", -1}}).SetLimit(100)
        cursor, err := h.Collection.Find(ctx, filter, opts)
        if err != nil {
            http.Error(w, "Failed to query telemetry", http.StatusInternalServerError)
            return
        }
        defer cursor.Close(ctx)
        var results []models.Telemetry
        if err := cursor.All(ctx, &results); err != nil {
            http.Error(w, "Failed to decode telemetry", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(results)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// TelemetryMetricsHandler handles metrics API requests for telemetry data.
type TelemetryMetricsHandler struct {
    Collection db.TelemetryCollection
}

// ServeHTTP processes HTTP requests for telemetry metrics.
func (h TelemetryMetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    // Optionally support time range filtering
    fromStr := r.URL.Query().Get("from")
    toStr := r.URL.Query().Get("to")
    var filter bson.M = bson.M{}
    if fromStr != "" || toStr != "" {
        filter["timestamp"] = bson.M{}
        if fromStr != "" {
            from, err := time.Parse(time.RFC3339, fromStr)
            if err != nil {
                http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
                return
            }
            filter["timestamp"].(bson.M)["$gte"] = from
        }
        if toStr != "" {
            to, err := time.Parse(time.RFC3339, toStr)
            if err != nil {
                http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
                return
            }
            filter["timestamp"].(bson.M)["$lte"] = to
        }
    }
    cursor, err := h.Collection.Find(ctx, filter)
    if err != nil {
        http.Error(w, "Failed to query telemetry", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)
    var results []models.Telemetry
    if err := cursor.All(ctx, &results); err != nil {
        http.Error(w, "Failed to decode telemetry", http.StatusInternalServerError)
        return
    }
    var totalEmissions float64
    var evCount, iceCount int
    for _, t := range results {
        totalEmissions += t.Emissions
        if t.BatteryLevel != nil {
            evCount++
        } else if t.FuelLevel != nil {
            iceCount++
        }
    }
    total := evCount + iceCount
    evPercent := 0.0
    if total > 0 {
        evPercent = float64(evCount) * 100.0 / float64(total)
    }
    resp := map[string]interface{}{
        "total_emissions": totalEmissions,
        "ev_percent":      evPercent,
        "total_records":   total,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func vehiclesHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // For now, return an empty array (stub)
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte("[]"))
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// jwtAuthMiddleware is a middleware that enforces JWT authentication for protected endpoints.
func jwtAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
            return
        }
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })
        if err != nil || !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// main is the entry point for the Fleet Sustainability backend service.
func main() {
    // Load .env file for local development
    if err := godotenv.Load(); err != nil {
        log.WithError(err).Warn("No .env file found (this is fine in production)")
    }
    // Connect to MongoDB
    client, err := db.ConnectMongo()
    if err != nil {
        log.WithError(err).Fatal("Failed to connect to MongoDB")
    }
    log.Info("Connected to MongoDB successfully!")
    mongoDBName := os.Getenv("MONGO_DB")
    if mongoDBName == "" {
        mongoDBName = "fleet"
    }
    telemetryCollection := &db.MongoCollection{Collection: client.Database(mongoDBName).Collection("telemetry")}
    telemetryHandler := &TelemetryHandler{Collection: telemetryCollection}
    telemetryMetricsHandler := TelemetryMetricsHandler{Collection: telemetryCollection}

    http.Handle("/api/telemetry", jwtAuthMiddleware(telemetryHandler))
    http.HandleFunc("/api/vehicles", vehiclesHandler)
    http.Handle("/api/telemetry/metrics", jwtAuthMiddleware(telemetryMetricsHandler))
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    useHTTPS := os.Getenv("USE_HTTPS")
    certFile := os.Getenv("TLS_CERT_FILE")
    keyFile := os.Getenv("TLS_KEY_FILE")

    if useHTTPS == "true" && certFile != "" && keyFile != "" {
        log.WithField("port", port).Info("HTTPS server listening")
        log.Fatal(http.ListenAndServeTLS(":"+port, certFile, keyFile, nil))
    } else {
        log.WithField("port", port).Info("HTTP server listening")
        log.Fatal(http.ListenAndServe(":"+port, nil))
    }
}