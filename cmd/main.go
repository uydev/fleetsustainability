package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"

    "github.com/ukydev/fleet-sustainability/internal/db"
)

type Location struct {
    Lat float64 `json:"lat"`
    Lon float64 `json:"lon"`
}

type Telemetry struct {
    VehicleID    string    `json:"vehicle_id"`
    Timestamp    string    `json:"timestamp"`
    Location     Location  `json:"location"`
    Speed        float64   `json:"speed"`
    FuelLevel    float64   `json:"fuel_level,omitempty"`
    BatteryLevel float64   `json:"battery_level,omitempty"`
    Emissions    float64   `json:"emissions"`
    Type         string    `json:"type"`
    Status       string    `json:"status"`
}

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
    var tele Telemetry
    if err := json.Unmarshal(body, &tele); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    fmt.Printf("Received telemetry: %+v\n", tele)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
}

func main() {
    // Connect to MongoDB (for future use)
    client, err := db.ConnectMongo()
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    fmt.Println("Connected to MongoDB successfully!")
    _ = client

    http.HandleFunc("/api/telemetry", telemetryHandler)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    fmt.Printf("HTTP server listening on :%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}