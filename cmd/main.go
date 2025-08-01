package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/ukydev/fleet-sustainability/internal/db"
    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// TelemetryHandler handles telemetry API requests with injected collection
type TelemetryHandler struct {
    Collection db.TelemetryCollection
}

func (h *TelemetryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodPost:
        body, err := ioutil.ReadAll(r.Body)
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
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _, err = h.Collection.InsertOne(ctx, tele)
        if err != nil {
            http.Error(w, "Failed to store telemetry", http.StatusInternalServerError)
            return
        }
        fmt.Printf("Stored telemetry: %+v\n", tele)
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    case http.MethodGet:
        // Return all telemetry, most recent first
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        opts := options.Find().SetSort(bson.D{{"timestamp", -1}}).SetLimit(100)
        cursor, err := h.Collection.Find(ctx, bson.D{}, opts)
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

func main() {
    // Connect to MongoDB
    client, err := db.ConnectMongo()
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    fmt.Println("Connected to MongoDB successfully!")
    mongoDBName := os.Getenv("MONGO_DB")
    if mongoDBName == "" {
        mongoDBName = "fleet"
    }
    telemetryCollection := client.Database(mongoDBName).Collection("telemetry")
    telemetryHandler := &TelemetryHandler{Collection: telemetryCollection}

    http.Handle("/api/telemetry", telemetryHandler)
    http.HandleFunc("/api/vehicles", vehiclesHandler)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Printf("HTTP server listening on :%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}