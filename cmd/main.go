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

// corsMiddleware adds CORS headers to allow frontend requests
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Allow requests from the frontend
        w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

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

// VehicleHandler handles vehicle management API requests.
type VehicleHandler struct {
	Collection db.VehicleCollection
}

// ServeHTTP processes HTTP requests for vehicle management.
func (h *VehicleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all vehicles
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		cursor, err := h.Collection.FindVehicles(ctx, bson.M{})
		if err != nil {
			http.Error(w, "Failed to query vehicles", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)
		
		var vehicles []models.Vehicle
		if err := cursor.All(ctx, &vehicles); err != nil {
			http.Error(w, "Failed to decode vehicles", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vehicles)
		
	case http.MethodPost:
		// Create new vehicle
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var vehicleInput struct {
			Type            string           `json:"type"`
			Make            string           `json:"make"`
			Model           string           `json:"model"`
			Year            int              `json:"year"`
			CurrentLocation models.Location  `json:"current_location,omitempty"`
			Status          string           `json:"status"`
		}
		
		if err := json.Unmarshal(body, &vehicleInput); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if vehicleInput.Type == "" {
			http.Error(w, "type is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Type != "ICE" && vehicleInput.Type != "EV" {
			http.Error(w, "type must be 'ICE' or 'EV'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Status == "" {
			http.Error(w, "status is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Status != "active" && vehicleInput.Status != "inactive" {
			http.Error(w, "status must be 'active' or 'inactive'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Make == "" {
			http.Error(w, "make is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Model == "" {
			http.Error(w, "model is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Year < 1900 || vehicleInput.Year > 2030 {
			http.Error(w, "year must be between 1900 and 2030", http.StatusBadRequest)
			return
		}
		
		vehicle := models.Vehicle{
			ID:              primitive.NewObjectID(),
			Type:            vehicleInput.Type,
			Make:            vehicleInput.Make,
			Model:           vehicleInput.Model,
			Year:            vehicleInput.Year,
			CurrentLocation: vehicleInput.CurrentLocation,
			Status:          vehicleInput.Status,
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err = h.Collection.InsertVehicle(ctx, vehicle)
		if err != nil {
			http.Error(w, "Failed to store vehicle", http.StatusInternalServerError)
			return
		}
		
		log.WithFields(log.Fields{"vehicle_id": vehicle.ID}).Info("Created vehicle")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": vehicle.ID.Hex(),
			"message": "Vehicle created successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// vehicleHandler handles individual vehicle operations (PUT, DELETE).
func vehicleHandler(w http.ResponseWriter, r *http.Request) {
	// Extract vehicle ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid vehicle ID", http.StatusBadRequest)
		return
	}
	vehicleID := pathParts[len(pathParts)-1]
	
	// Validate vehicle ID format
	if len(vehicleID) != 24 {
		http.Error(w, "Invalid vehicle ID format", http.StatusBadRequest)
		return
	}
	
	switch r.Method {
	case http.MethodPut:
		// Update vehicle
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var vehicleInput struct {
			Type            string           `json:"type"`
			Make            string           `json:"make"`
			Model           string           `json:"model"`
			Year            int              `json:"year"`
			CurrentLocation models.Location  `json:"current_location,omitempty"`
			Status          string           `json:"status"`
		}
		
		if err := json.Unmarshal(body, &vehicleInput); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if vehicleInput.Type != "" && vehicleInput.Type != "ICE" && vehicleInput.Type != "EV" {
			http.Error(w, "type must be 'ICE' or 'EV'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Status != "" && vehicleInput.Status != "active" && vehicleInput.Status != "inactive" {
			http.Error(w, "status must be 'active' or 'inactive'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Year != 0 && (vehicleInput.Year < 1900 || vehicleInput.Year > 2030) {
			http.Error(w, "year must be between 1900 and 2030", http.StatusBadRequest)
			return
		}
		
		// For now, just return success (in real implementation, would update in DB)
		// In a real implementation, we would check if the vehicle exists first
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": vehicleID,
			"message": "Vehicle updated successfully",
		})
		
	case http.MethodDelete:
		// Delete vehicle
		// For now, just return success (in real implementation, would delete from DB)
		// In a real implementation, we would check if the vehicle exists first
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": vehicleID,
			"message": "Vehicle deleted successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// vehicleRouter handles both collection and individual vehicle operations.
func vehicleRouter(w http.ResponseWriter, r *http.Request) {
	// Check if this is an individual vehicle operation
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) > 3 && pathParts[3] != "" {
		// Individual vehicle operation
		vehicleHandler(w, r)
		return
	}
	
	// Collection operation (GET, POST) - use the handler from main
	vehicleCollectionHandler.ServeHTTP(w, r)
}

// VehicleCollectionHandler handles vehicle collection operations (GET, POST).
type VehicleCollectionHandler struct {
	Collection db.VehicleCollection
}

// ServeHTTP processes HTTP requests for vehicle collection operations.
func (h *VehicleCollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all vehicles
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		cursor, err := h.Collection.FindVehicles(ctx, bson.M{})
		if err != nil {
			http.Error(w, "Failed to query vehicles", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)
		
		var results []models.Vehicle
		if err := cursor.All(ctx, &results); err != nil {
			http.Error(w, "Failed to decode vehicles", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		
	case http.MethodPost:
		// Create new vehicle
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var vehicleInput struct {
			Type            string           `json:"type"`
			Make            string           `json:"make"`
			Model           string           `json:"model"`
			Year            int              `json:"year"`
			CurrentLocation models.Location  `json:"current_location,omitempty"`
			Status          string           `json:"status"`
		}
		
		if err := json.Unmarshal(body, &vehicleInput); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if vehicleInput.Type == "" {
			http.Error(w, "type is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Type != "ICE" && vehicleInput.Type != "EV" {
			http.Error(w, "type must be 'ICE' or 'EV'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Status == "" {
			http.Error(w, "status is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Status != "active" && vehicleInput.Status != "inactive" {
			http.Error(w, "status must be 'active' or 'inactive'", http.StatusBadRequest)
			return
		}
		if vehicleInput.Make == "" {
			http.Error(w, "make is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Model == "" {
			http.Error(w, "model is required", http.StatusBadRequest)
			return
		}
		if vehicleInput.Year < 1900 || vehicleInput.Year > 2030 {
			http.Error(w, "year must be between 1900 and 2030", http.StatusBadRequest)
			return
		}
		
		// Create vehicle model
		vehicle := models.Vehicle{
			Type:            vehicleInput.Type,
			Make:            vehicleInput.Make,
			Model:           vehicleInput.Model,
			Year:            vehicleInput.Year,
			CurrentLocation: vehicleInput.CurrentLocation,
			Status:          vehicleInput.Status,
		}
		
		// Store vehicle in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := h.Collection.InsertVehicle(ctx, vehicle); err != nil {
			log.WithError(err).Error("Failed to insert vehicle")
			http.Error(w, "Failed to create vehicle", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": primitive.NewObjectID().Hex(),
			"message": "Vehicle created successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// TripHandler handles trip management API requests.
type TripHandler struct {
	Collection db.TripCollection
}

// ServeHTTP processes HTTP requests for trip management.
func (h *TripHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all trips
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		cursor, err := h.Collection.FindTrips(ctx, bson.M{})
		if err != nil {
			http.Error(w, "Failed to query trips", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)
		
		var results []models.Trip
		if err := cursor.All(ctx, &results); err != nil {
			http.Error(w, "Failed to decode trips", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		
	case http.MethodPost:
		// Create new trip
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var trip models.Trip
		if err := json.Unmarshal(body, &trip); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if trip.VehicleID == "" {
			http.Error(w, "vehicle_id is required", http.StatusBadRequest)
			return
		}
		if trip.StartTime.IsZero() {
			http.Error(w, "start_time is required", http.StatusBadRequest)
			return
		}
		if trip.Status == "" {
			trip.Status = "planned"
		}
		
		// Store trip in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := h.Collection.InsertTrip(ctx, trip); err != nil {
			log.WithError(err).Error("Failed to insert trip")
			http.Error(w, "Failed to create trip", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": primitive.NewObjectID().Hex(),
			"message": "Trip created successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// MaintenanceHandler handles maintenance management API requests.
type MaintenanceHandler struct {
	Collection db.MaintenanceCollection
}

// ServeHTTP processes HTTP requests for maintenance management.
func (h *MaintenanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all maintenance records
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		cursor, err := h.Collection.FindMaintenance(ctx, bson.M{})
		if err != nil {
			http.Error(w, "Failed to query maintenance", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)
		
		var results []models.Maintenance
		if err := cursor.All(ctx, &results); err != nil {
			http.Error(w, "Failed to decode maintenance", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		
	case http.MethodPost:
		// Create new maintenance record
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var maintenance models.Maintenance
		if err := json.Unmarshal(body, &maintenance); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if maintenance.VehicleID == "" {
			http.Error(w, "vehicle_id is required", http.StatusBadRequest)
			return
		}
		if maintenance.ServiceType == "" {
			http.Error(w, "service_type is required", http.StatusBadRequest)
			return
		}
		if maintenance.Status == "" {
			maintenance.Status = "scheduled"
		}
		
		// Store maintenance in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := h.Collection.InsertMaintenance(ctx, maintenance); err != nil {
			log.WithError(err).Error("Failed to insert maintenance")
			http.Error(w, "Failed to create maintenance", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": primitive.NewObjectID().Hex(),
			"message": "Maintenance created successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CostHandler handles cost management API requests.
type CostHandler struct {
	Collection db.CostCollection
}

// ServeHTTP processes HTTP requests for cost management.
func (h *CostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get all cost records
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		cursor, err := h.Collection.FindCosts(ctx, bson.M{})
		if err != nil {
			http.Error(w, "Failed to query costs", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)
		
		var results []models.Cost
		if err := cursor.All(ctx, &results); err != nil {
			http.Error(w, "Failed to decode costs", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		
	case http.MethodPost:
		// Create new cost record
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		var cost models.Cost
		if err := json.Unmarshal(body, &cost); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Input validation
		if cost.VehicleID == "" {
			http.Error(w, "vehicle_id is required", http.StatusBadRequest)
			return
		}
		if cost.Category == "" {
			http.Error(w, "category is required", http.StatusBadRequest)
			return
		}
		if cost.Amount <= 0 {
			http.Error(w, "amount must be positive", http.StatusBadRequest)
			return
		}
		if cost.Status == "" {
			cost.Status = "pending"
		}
		
		// Store cost in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := h.Collection.InsertCost(ctx, cost); err != nil {
			log.WithError(err).Error("Failed to insert cost")
			http.Error(w, "Failed to create cost", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": primitive.NewObjectID().Hex(),
			"message": "Cost created successfully",
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var vehicleCollectionHandler *VehicleCollectionHandler

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
    vehicleCollection := &db.MongoCollection{Collection: client.Database(mongoDBName).Collection("vehicles")}
    tripCollection := &db.MongoCollection{Collection: client.Database(mongoDBName).Collection("trips")}
    maintenanceCollection := &db.MongoCollection{Collection: client.Database(mongoDBName).Collection("maintenance")}
    costCollection := &db.MongoCollection{Collection: client.Database(mongoDBName).Collection("costs")}
    
    telemetryHandler := &TelemetryHandler{Collection: telemetryCollection}
    vehicleCollectionHandler = &VehicleCollectionHandler{Collection: vehicleCollection}
    tripHandler := &TripHandler{Collection: tripCollection}
    maintenanceHandler := &MaintenanceHandler{Collection: maintenanceCollection}
    costHandler := &CostHandler{Collection: costCollection}
    telemetryMetricsHandler := TelemetryMetricsHandler{Collection: telemetryCollection}

    http.Handle("/api/telemetry", corsMiddleware(telemetryHandler))
    http.Handle("/api/vehicles", corsMiddleware(http.HandlerFunc(vehicleRouter)))
    http.Handle("/api/trips", corsMiddleware(tripHandler))
    http.Handle("/api/maintenance", corsMiddleware(maintenanceHandler))
    http.Handle("/api/costs", corsMiddleware(costHandler))
    http.Handle("/api/telemetry/metrics", corsMiddleware(telemetryMetricsHandler))
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