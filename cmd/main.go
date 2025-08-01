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
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

var telemetryCollectionName = "telemetry"
var mongoClient *mongo.Client
var mongoDBName string

func telemetryHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
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
    // Convert to models.Telemetry for MongoDB
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
    collection := mongoClient.Database(mongoDBName).Collection(telemetryCollectionName)
    _, err = collection.InsertOne(ctx, tele)
    if err != nil {
        http.Error(w, "Failed to store telemetry", http.StatusInternalServerError)
        return
    }
    fmt.Printf("Stored telemetry: %+v\n", tele)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}

func main() {
    // Connect to MongoDB
    client, err := db.ConnectMongo()
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    fmt.Println("Connected to MongoDB successfully!")
    mongoClient = client
    mongoDBName = os.Getenv("MONGO_DB")
    if mongoDBName == "" {
        mongoDBName = "fleet"
    }

    http.HandleFunc("/api/telemetry", telemetryHandler)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Printf("HTTP server listening on :%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}