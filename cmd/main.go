package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/ukydev/fleet-sustainability/internal/auth"
	"github.com/ukydev/fleet-sustainability/internal/db"
	"github.com/ukydev/fleet-sustainability/internal/handlers"
	"github.com/ukydev/fleet-sustainability/internal/middleware"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// corsMiddleware adds CORS headers to allow frontend requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin (for development)
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
			VehicleID    string          `json:"vehicle_id"`
			Timestamp    string          `json:"timestamp"`
			Location     models.Location `json:"location"`
			Speed        float64         `json:"speed"`
			FuelLevel    *float64        `json:"fuel_level,omitempty"`
			BatteryLevel *float64        `json:"battery_level,omitempty"`
			Emissions    float64         `json:"emissions"`
			Type         string          `json:"type"`
			Status       string          `json:"status"`
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
		// Enforce EV emissions to zero server-side to prevent bad client data
		if teleIn.Type == "EV" {
			teleIn.Emissions = 0
		}
		// Optionally, add more checks for FuelLevel, BatteryLevel, Location, etc.
		timestamp, err := time.Parse(time.RFC3339, teleIn.Timestamp)
		if err != nil {
			http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
			return
		}
		// Preserve explicit zeros by using pointers from input
		var fuelPtr, batteryPtr *float64
		if teleIn.FuelLevel != nil {
			fuelPtr = teleIn.FuelLevel
		}
		if teleIn.BatteryLevel != nil {
			batteryPtr = teleIn.BatteryLevel
		}
		// Map provided vehicle_id string to ObjectID if valid; otherwise generate a new one
		var vehicleObjectID primitive.ObjectID
		if teleIn.VehicleID != "" && len(teleIn.VehicleID) == 24 {
			if oid, err := primitive.ObjectIDFromHex(teleIn.VehicleID); err == nil {
				vehicleObjectID = oid
			} else {
				vehicleObjectID = primitive.NewObjectID()
			}
		} else {
			vehicleObjectID = primitive.NewObjectID()
		}

        tele := models.Telemetry{
			VehicleID:    vehicleObjectID,
			Timestamp:    timestamp,
			Location:     teleIn.Location,
			Speed:        teleIn.Speed,
			FuelLevel:    fuelPtr,
			BatteryLevel: batteryPtr,
			Emissions:    teleIn.Emissions,
			Type:         teleIn.Type,
			Status:       teleIn.Status,
		}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            tele.TenantID = claims.TenantID
        }
		err = h.Collection.InsertTelemetry(r.Context(), tele)
		if err != nil {
			http.Error(w, "Failed to store telemetry", http.StatusInternalServerError)
			return
		}
		log.WithFields(log.Fields{"vehicle_id": tele.VehicleID}).Info("Stored telemetry")

		// Broadcast to SSE subscribers (use original string vehicle_id for clients)
        if telemetrySSEHub != nil {
			eventPayload := map[string]interface{}{
				"vehicle_id":    teleIn.VehicleID,
				"timestamp":     teleIn.Timestamp,
				"location":      teleIn.Location,
				"speed":         teleIn.Speed,
				"fuel_level":    teleIn.FuelLevel,
				"battery_level": teleIn.BatteryLevel,
				"emissions":     teleIn.Emissions,
				"type":          teleIn.Type,
				"status":        teleIn.Status,
			}
			if data, err := json.Marshal(eventPayload); err == nil {
                if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
                    // Send to tenant-scoped listeners and also to global listeners (unauth SSE clients in dev)
                    telemetrySSEHub.BroadcastToTenant(claims.TenantID, data)
                    telemetrySSEHub.Broadcast(data)
                } else {
                    telemetrySSEHub.Broadcast(data)
                }
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	case http.MethodGet:
		// Support time range and vehicle_id filtering
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		veh := r.URL.Query().Get("vehicle_id")
        var filter bson.M = bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }
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
		if veh != "" {
			if len(veh) == 24 {
				if oid, err := primitive.ObjectIDFromHex(veh); err == nil {
					filter["vehicle_id"] = oid
				}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Allow client to control sort and limit; when a time range is provided, do not cap by default
		sortOrder := int32(-1)
		if strings.ToLower(r.URL.Query().Get("sort")) == "asc" {
			sortOrder = 1
		}
		var limit int64 = 100
		if fromStr != "" || toStr != "" {
			limit = 0 // no limit when a time window is specified
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if l == "0" {
				limit = 0
			} else if n, err := strconv.ParseInt(l, 10, 64); err == nil && n >= 0 {
				limit = n
			}
		}
		opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: sortOrder}})
		if limit > 0 {
			opts = opts.SetLimit(limit)
		}
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
		// Ensure we return an empty array [] instead of null when there are no results
		if results == nil {
			results = make([]models.Telemetry, 0)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	case http.MethodDelete:
		// Method not allowed for collection-level delete in tests
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SSEHub is a simple in-memory hub for broadcasting telemetry updates over SSE.
type SSEHub struct {
    mu      sync.RWMutex
    clients map[chan []byte]string // chan -> tenant_id
}

// NewSSEHub creates and returns a new SSEHub.
func NewSSEHub() *SSEHub {
    return &SSEHub{clients: make(map[chan []byte]string)}
}

// Broadcast sends the given data to all connected SSE clients.
func (h *SSEHub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
			// Drop if client is slow
		}
	}
}

// BroadcastToTenant sends data only to clients of a specific tenant.
func (h *SSEHub) BroadcastToTenant(tenantID string, data []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for ch, t := range h.clients {
        if t != tenantID {
            continue
        }
        select {
        case ch <- data:
        default:
        }
    }
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

    clientCh := make(chan []byte, 16)
    tenantID := ""
    if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
        tenantID = claims.TenantID
    }
    h.mu.Lock()
    h.clients[clientCh] = tenantID
	h.mu.Unlock()

	defer func() {
        h.mu.Lock()
        delete(h.clients, clientCh)
		h.mu.Unlock()
		close(clientCh)
	}()

	// Initial comment to open stream
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// Keepalive comment
			_, _ = w.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		case msg := <-clientCh:
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

var telemetrySSEHub *SSEHub

// --- WebSocket support ---
var wsUpgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
}

func wsTelemetryHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Upgrade failed", http.StatusBadRequest)
        return
    }
    defer conn.Close()

    // Each WS client gets a channel and optional tenant binding
    clientCh := make(chan []byte, 16)
    tenantID := ""
    if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
        tenantID = claims.TenantID
    }
    telemetrySSEHub.mu.Lock()
    telemetrySSEHub.clients[clientCh] = tenantID
    telemetrySSEHub.mu.Unlock()
    defer func() {
        telemetrySSEHub.mu.Lock()
        delete(telemetrySSEHub.clients, clientCh)
        telemetrySSEHub.mu.Unlock()
        close(clientCh)
    }()

    // Pump: we only write server->client; read to drain pings/close
    go func() {
        for {
            if _, _, err := conn.NextReader(); err != nil {
                _ = conn.Close()
                return
            }
        }
    }()

    for msg := range clientCh {
        conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
        if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
            return
        }
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

        vf := bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            vf["tenant_id"] = claims.TenantID
        }
        cursor, err := h.Collection.FindVehicles(ctx, vf)
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
			Type            string          `json:"type"`
			Make            string          `json:"make"`
			Model           string          `json:"model"`
			Year            int             `json:"year"`
			CurrentLocation models.Location `json:"current_location,omitempty"`
			Status          string          `json:"status"`
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
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            vehicle.TenantID = claims.TenantID
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
			"id":      vehicle.ID.Hex(),
			"message": "Vehicle created successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// vehicleHandler handles individual vehicle operations (PUT, DELETE).
// vehicleHandler handles individual vehicle operations (GET, PUT, DELETE).
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
	case http.MethodGet:
		// Get individual vehicle
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		vehicle, err := vehicleCollectionHandler.Collection.FindVehicleByID(ctx, vehicleID)
		if err != nil {
			http.Error(w, "Vehicle not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vehicle)

	case http.MethodPut:
		// Update vehicle
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		var vehicleInput struct {
			Type            string          `json:"type"`
			Make            string          `json:"make"`
			Model           string          `json:"model"`
			Year            int             `json:"year"`
			CurrentLocation models.Location `json:"current_location,omitempty"`
			Status          string          `json:"status"`
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

		// Get existing vehicle to merge with updates
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		existingVehicle, err := vehicleCollectionHandler.Collection.FindVehicleByID(ctx, vehicleID)
		if err != nil {
			http.Error(w, "Vehicle not found", http.StatusNotFound)
			return
		}

		// Update fields if provided
		if vehicleInput.Type != "" {
			existingVehicle.Type = vehicleInput.Type
		}
		if vehicleInput.Make != "" {
			existingVehicle.Make = vehicleInput.Make
		}
		if vehicleInput.Model != "" {
			existingVehicle.Model = vehicleInput.Model
		}
		if vehicleInput.Year != 0 {
			existingVehicle.Year = vehicleInput.Year
		}
		if vehicleInput.Status != "" {
			existingVehicle.Status = vehicleInput.Status
		}
		if vehicleInput.CurrentLocation.Lat != 0 || vehicleInput.CurrentLocation.Lon != 0 {
			existingVehicle.CurrentLocation = vehicleInput.CurrentLocation
		}

		// Update in database
		if err := vehicleCollectionHandler.Collection.UpdateVehicle(ctx, vehicleID, *existingVehicle); err != nil {
			log.WithError(err).Error("Failed to update vehicle")
			http.Error(w, "Failed to update vehicle", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      vehicleID,
			"message": "Vehicle updated successfully",
		})

	case http.MethodDelete:
		// Delete vehicle
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Check if vehicle exists
		_, err := vehicleCollectionHandler.Collection.FindVehicleByID(ctx, vehicleID)
		if err != nil {
			http.Error(w, "Vehicle not found", http.StatusNotFound)
			return
		}

		// Delete from database
		if err := vehicleCollectionHandler.Collection.DeleteVehicle(ctx, vehicleID); err != nil {
			log.WithError(err).Error("Failed to delete vehicle")
			http.Error(w, "Failed to delete vehicle", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      vehicleID,
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
	log.Infof("VehicleCollectionHandler: %s %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		// Get all vehicles with optional time filtering
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

        // Parse time range parameters; enforce tenant scoping
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
        var filter bson.M = bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }

		// Use ObjectID timestamp for filtering existing vehicles
		if fromStr != "" || toStr != "" {
			filter["_id"] = bson.M{}
			if fromStr != "" {
				from, err := time.Parse(time.RFC3339, fromStr)
				if err != nil {
					http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
					return
				}
				// Convert time to ObjectID timestamp
				fromObjectID := primitive.NewObjectIDFromTimestamp(from)
				filter["_id"].(bson.M)["$gte"] = fromObjectID
			}
			if toStr != "" {
				to, err := time.Parse(time.RFC3339, toStr)
				if err != nil {
					http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
					return
				}
				// Convert time to ObjectID timestamp
				toObjectID := primitive.NewObjectIDFromTimestamp(to)
				filter["_id"].(bson.M)["$lte"] = toObjectID
			}
		}

		cursor, err := h.Collection.FindVehicles(ctx, filter)
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
			Type            string          `json:"type"`
			Make            string          `json:"make"`
			Model           string          `json:"model"`
			Year            int             `json:"year"`
			CurrentLocation models.Location `json:"current_location,omitempty"`
			Status          string          `json:"status"`
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

        // Create vehicle model with pre-assigned ID so we can return it
		vehicle := models.Vehicle{
			ID:              primitive.NewObjectID(),
            TenantID:        "",
			Type:            vehicleInput.Type,
			Make:            vehicleInput.Make,
			Model:           vehicleInput.Model,
			Year:            vehicleInput.Year,
			CurrentLocation: vehicleInput.CurrentLocation,
			Status:          vehicleInput.Status,
			CreatedAt:       time.Now(),
		}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            vehicle.TenantID = claims.TenantID
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
			"id":      vehicle.ID.Hex(),
			"message": "Vehicle created successfully",
		})

	case http.MethodDelete:
		// Method not allowed for collection-level delete in tests
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return

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
        // Get all trips with optional time filtering and tenant scoping
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Parse time range parameters
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
        var filter bson.M = bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }

		if fromStr != "" || toStr != "" {
			filter["start_time"] = bson.M{}
			if fromStr != "" {
				from, err := time.Parse(time.RFC3339, fromStr)
				if err != nil {
					http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
					return
				}
				filter["start_time"].(bson.M)["$gte"] = from
			}
			if toStr != "" {
				to, err := time.Parse(time.RFC3339, toStr)
				if err != nil {
					http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
					return
				}
				filter["start_time"].(bson.M)["$lte"] = to
			}
		}

		cursor, err := h.Collection.FindTrips(ctx, filter)
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

        // Stamp tenant and store trip in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            trip.TenantID = claims.TenantID
        }

		if err := h.Collection.InsertTrip(ctx, trip); err != nil {
			log.WithError(err).Error("Failed to insert trip")
			http.Error(w, "Failed to create trip", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      primitive.NewObjectID().Hex(),
			"message": "Trip created successfully",
		})

    case http.MethodDelete:
        // Delete all trip records for current tenant (bulk)
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        // Note: current db interface doesn't support tenant filter in bulk delete; retain as-is or extend if needed.
        if err := h.Collection.DeleteAll(ctx); err != nil {
            log.WithError(err).Error("Failed to delete trip records")
            http.Error(w, "Failed to delete trip records", http.StatusInternalServerError)
            return
        }

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "All trip records deleted successfully",
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
	log.Infof("MaintenanceHandler: %s %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
        // Get all maintenance records with optional time filtering and tenant scoping
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Parse time range parameters
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
        var filter bson.M = bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }

		if fromStr != "" || toStr != "" {
			filter["service_date"] = bson.M{}
			if fromStr != "" {
				from, err := time.Parse(time.RFC3339, fromStr)
				if err != nil {
					http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
					return
				}
				filter["service_date"].(bson.M)["$gte"] = from
			}
			if toStr != "" {
				to, err := time.Parse(time.RFC3339, toStr)
				if err != nil {
					http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
					return
				}
				filter["service_date"].(bson.M)["$lte"] = to
			}
		}

		cursor, err := h.Collection.FindMaintenance(ctx, filter)
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

        // Stamp tenant and store maintenance in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            maintenance.TenantID = claims.TenantID
        }

		if err := h.Collection.InsertMaintenance(ctx, maintenance); err != nil {
			log.WithError(err).Error("Failed to insert maintenance")
			http.Error(w, "Failed to create maintenance", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      primitive.NewObjectID().Hex(),
			"message": "Maintenance created successfully",
		})

	case http.MethodDelete:
		log.Infof("MaintenanceHandler DELETE hit")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.Collection.DeleteAll(ctx); err != nil {
			log.WithError(err).Error("Failed to delete maintenance records")
			http.Error(w, "Failed to delete maintenance records", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "All maintenance records deleted successfully",
		})
		return

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
	log.Infof("CostHandler: %s %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
        // Get all cost records with optional time filtering and tenant scoping
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Parse time range parameters
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
        var filter bson.M = bson.M{}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }

		if fromStr != "" || toStr != "" {
			filter["date"] = bson.M{}
			if fromStr != "" {
				from, err := time.Parse(time.RFC3339, fromStr)
				if err != nil {
					http.Error(w, "Invalid 'from' time format", http.StatusBadRequest)
					return
				}
				filter["date"].(bson.M)["$gte"] = from
			}
			if toStr != "" {
				to, err := time.Parse(time.RFC3339, toStr)
				if err != nil {
					http.Error(w, "Invalid 'to' time format", http.StatusBadRequest)
					return
				}
				filter["date"].(bson.M)["$lte"] = to
			}
		}

		cursor, err := h.Collection.FindCosts(ctx, filter)
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

        // Stamp tenant and store cost in database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
            cost.TenantID = claims.TenantID
        }

		if err := h.Collection.InsertCost(ctx, cost); err != nil {
			log.WithError(err).Error("Failed to insert cost")
			http.Error(w, "Failed to create cost", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      primitive.NewObjectID().Hex(),
			"message": "Cost created successfully",
		})

	case http.MethodDelete:
		log.Infof("CostHandler DELETE hit")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.Collection.DeleteAll(ctx); err != nil {
			log.WithError(err).Error("Failed to delete cost records")
			http.Error(w, "Failed to delete cost records", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "All cost records deleted successfully",
		})
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

var vehicleCollectionHandler *VehicleCollectionHandler

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
	userCollection := &db.MongoUserCollection{Collection: client.Database(mongoDBName).Collection("users")}

    // Ensure TTL index on telemetry to prevent unbounded growth and tenant indexes
	ttlDays := 30
	if v := os.Getenv("TELEMETRY_TTL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttlDays = n
    }
    // Secondary indexes per collection on tenant_id for scoping
    ensureIndex := func(coll *mongo.Collection, key string) {
        _, _ = coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
            Keys: bson.D{{Key: key, Value: 1}},
        })
    }
    ensureIndex(telemetryCollection.Collection, "tenant_id")
    ensureIndex(vehicleCollection.Collection, "tenant_id")
    ensureIndex(tripCollection.Collection, "tenant_id")
    ensureIndex(maintenanceCollection.Collection, "tenant_id")
    ensureIndex(costCollection.Collection, "tenant_id")
	}
	expireSeconds := int32(ttlDays * 24 * 60 * 60)
	{
		idxModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(expireSeconds).SetName("ttl_timestamp_seconds"),
		}
		if _, err := telemetryCollection.Collection.Indexes().CreateOne(context.Background(), idxModel); err != nil {
			log.WithError(err).Warn("Failed to ensure TTL index on telemetry")
		} else {
			log.WithFields(log.Fields{"days": ttlDays}).Info("TTL index ensured on telemetry.timestamp")
		}
	}

	// Initialize authentication services
	authService, err := auth.NewService()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize auth service")
	}

    // Initialize handlers
	telemetryHandler := &TelemetryHandler{Collection: telemetryCollection}
	vehicleCollectionHandler = &VehicleCollectionHandler{Collection: vehicleCollection}
	tripHandler := &TripHandler{Collection: tripCollection}
	maintenanceHandler := &MaintenanceHandler{Collection: maintenanceCollection}
	costHandler := &CostHandler{Collection: costCollection}
	telemetryMetricsHandler := TelemetryMetricsHandler{Collection: telemetryCollection}
    alertsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Threshold-based alerts from telemetry with optional time filtering
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		var ts bson.M
		if fromStr != "" || toStr != "" {
			ts = bson.M{}
			if fromStr != "" { if from, err := time.Parse(time.RFC3339, fromStr); err == nil { ts["$gte"] = from } }
			if toStr != "" { if to, err := time.Parse(time.RFC3339, toStr); err == nil { ts["$lte"] = to } }
		} else {
			ts = bson.M{"$gte": time.Now().Add(-1 * time.Hour)}
		}
        filter := bson.M{"timestamp": ts}
        if claims, ok := middleware.GetUserFromContext(r.Context()); ok && claims.TenantID != "" {
            filter["tenant_id"] = claims.TenantID
        }
		cursor, err := telemetryCollection.Find(ctx, filter)
		if err != nil { http.Error(w, "Failed to query telemetry", http.StatusInternalServerError); return }
		defer cursor.Close(ctx)
		var rows []models.Telemetry
		if err := cursor.All(ctx, &rows); err != nil { http.Error(w, "Failed to decode telemetry", http.StatusInternalServerError); return }
		alerts := []map[string]interface{}{}
		for _, t := range rows {
			if t.FuelLevel != nil && *t.FuelLevel <= 10 { alerts = append(alerts, map[string]interface{}{"type":"low_fuel","vehicle_id":t.VehicleID.Hex(),"value":*t.FuelLevel,"ts":t.Timestamp}) }
			if t.BatteryLevel != nil && *t.BatteryLevel <= 10 { alerts = append(alerts, map[string]interface{}{"type":"low_battery","vehicle_id":t.VehicleID.Hex(),"value":*t.BatteryLevel,"ts":t.Timestamp}) }
			if t.Emissions >= 50 { alerts = append(alerts, map[string]interface{}{"type":"high_emissions","vehicle_id":t.VehicleID.Hex(),"value":t.Emissions,"ts":t.Timestamp}) }
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts)
	})

	advancedMetricsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Naive aggregates: fuel used (delta), cost estimate, daily emissions trend
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		fromStr := r.URL.Query().Get("from")
		var filter bson.M = bson.M{}
		if fromStr != "" {
			if from, err := time.Parse(time.RFC3339, fromStr); err == nil { filter["timestamp"] = bson.M{"$gte": from} }
		}
		cursor, err := telemetryCollection.Find(ctx, filter)
		if err != nil { http.Error(w, "Failed to query telemetry", http.StatusInternalServerError); return }
		defer cursor.Close(ctx)
		var rows []models.Telemetry
		if err := cursor.All(ctx, &rows); err != nil { http.Error(w, "Failed to decode telemetry", http.StatusInternalServerError); return }
		// group by vehicle
		type agg struct{ first, last *models.Telemetry }
		m := map[string]*agg{}
		for i := range rows {
			v := rows[i]
			id := v.VehicleID.Hex()
			a := m[id]
			if a == nil { a = &agg{}; m[id] = a }
			if a.first == nil || v.Timestamp.Before(a.first.Timestamp) { a.first = &v }
			if a.last == nil || v.Timestamp.After(a.last.Timestamp) { a.last = &v }
		}
		fuelUsed := 0.0; energyUsed := 0.0; emissions := 0.0
		for _, a := range m {
			if a.first != nil && a.last != nil {
				emissions += a.last.Emissions // simplistic sum of last; could be integral
				if a.first.FuelLevel != nil && a.last.FuelLevel != nil {
					d := *a.first.FuelLevel - *a.last.FuelLevel
					if d > 0 { fuelUsed += d }
				}
				if a.first.BatteryLevel != nil && a.last.BatteryLevel != nil {
					d := *a.first.BatteryLevel - *a.last.BatteryLevel
					if d > 0 { energyUsed += d }
				}
			}
		}
		// cost estimate
		fuelCostPerPct := 0.02; energyCostPerPct := 0.005 // placeholder unit costs
		cost := fuelUsed*fuelCostPerPct + energyUsed*energyCostPerPct
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fuel_used_pct": fuelUsed,
			"energy_used_pct": energyUsed,
			"cost_estimate": cost,
			"emissions": emissions,
		})
	})
	authHandler := handlers.NewAuthHandler(authService, userCollection)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	// rateLimitMiddleware := middleware.NewRateLimitMiddleware() // Temporarily disabled for development

	// Authentication routes (no auth required)
	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(http.HandlerFunc(authHandler.Login)).ServeHTTP(w, r)
	})
	http.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(http.HandlerFunc(authHandler.Register)).ServeHTTP(w, r)
	})

	// Protected routes (require authentication)
	// Temporarily disable rate limiting for development
	http.Handle("/api/telemetry", corsMiddleware(authMiddleware.Authenticate(telemetryHandler)))
	// SSE endpoint (unauth for now; can wrap with authMiddleware if desired)
	telemetrySSEHub = NewSSEHub()
	http.Handle("/api/telemetry/stream", corsMiddleware(telemetrySSEHub))
	// WebSocket endpoint (auth optional; mirror SSE data)
	wsEnabled := os.Getenv("WEBSOCKETS_ENABLED")
	if wsEnabled == "" || strings.ToLower(wsEnabled) == "true" {
		http.Handle("/api/telemetry/ws", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(wsTelemetryHandler))))
	}
	http.Handle("/api/vehicles", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(vehicleRouter))))
	http.Handle("/api/vehicles/", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(vehicleRouter))))
	http.Handle("/api/trips", corsMiddleware(authMiddleware.Authenticate(tripHandler)))
    // Add item-level routes for tenant-scoped deletes/updates
    http.Handle("/api/trips/", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := strings.TrimPrefix(r.URL.Path, "/api/trips/")
        if len(id) != 24 { http.Error(w, "Invalid trip ID", http.StatusBadRequest); return }
        switch r.Method {
        case http.MethodDelete:
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            // Load trip and verify tenant
            trip, err := tripCollection.FindTripByID(ctx, id)
            if err != nil { http.Error(w, "Trip not found", http.StatusNotFound); return }
            if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
                if trip.TenantID != "" && trip.TenantID != claims.TenantID {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
            }
            if err := tripCollection.DeleteTrip(ctx, id); err != nil { http.Error(w, "Failed to delete trip", http.StatusInternalServerError); return }
            w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"id": id, "message": "Trip deleted"})
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    }))))
    http.Handle("/api/maintenance", corsMiddleware(authMiddleware.Authenticate(maintenanceHandler)))
    http.Handle("/api/maintenance/", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := strings.TrimPrefix(r.URL.Path, "/api/maintenance/")
        if len(id) != 24 { http.Error(w, "Invalid maintenance ID", http.StatusBadRequest); return }
        switch r.Method {
        case http.MethodDelete:
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            rec, err := maintenanceCollection.FindMaintenanceByID(ctx, id)
            if err != nil { http.Error(w, "Maintenance not found", http.StatusNotFound); return }
            if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
                if rec.TenantID != "" && rec.TenantID != claims.TenantID {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
            }
            if err := maintenanceCollection.DeleteMaintenance(ctx, id); err != nil { http.Error(w, "Failed to delete maintenance", http.StatusInternalServerError); return }
            w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"id": id, "message": "Maintenance deleted"})
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    }))))
    http.Handle("/api/costs", corsMiddleware(authMiddleware.Authenticate(costHandler)))
    http.Handle("/api/costs/", corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := strings.TrimPrefix(r.URL.Path, "/api/costs/")
        if len(id) != 24 { http.Error(w, "Invalid cost ID", http.StatusBadRequest); return }
        switch r.Method {
        case http.MethodDelete:
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer cancel()
            rec, err := costCollection.FindCostByID(ctx, id)
            if err != nil { http.Error(w, "Cost record not found", http.StatusNotFound); return }
            if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
                if rec.TenantID != "" && rec.TenantID != claims.TenantID {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
            }
            if err := costCollection.DeleteCost(ctx, id); err != nil { http.Error(w, "Failed to delete cost record", http.StatusInternalServerError); return }
            w.Header().Set("Content-Type", "application/json"); w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"id": id, "message": "Cost deleted"})
        default:
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        }
    }))))
	http.Handle("/api/telemetry/metrics", corsMiddleware(authMiddleware.Authenticate(telemetryMetricsHandler)))
	http.Handle("/api/telemetry/metrics/advanced", corsMiddleware(authMiddleware.Authenticate(advancedMetricsHandler)))
	http.Handle("/api/alerts", corsMiddleware(authMiddleware.Authenticate(alertsHandler)))

	// --- MQTT Subscriber (optional) ---
	mqttURL := os.Getenv("MQTT_BROKER_URL")
	mqttTopic := os.Getenv("MQTT_TELEMETRY_TOPIC")
	if mqttTopic == "" { mqttTopic = "fleet/telemetry" }
	if mqttURL != "" {
		opts := mqtt.NewClientOptions().AddBroker(mqttURL)
		if u := os.Getenv("MQTT_USERNAME"); u != "" { opts.SetUsername(u) }
		if p := os.Getenv("MQTT_PASSWORD"); p != "" { opts.SetPassword(p) }
		opts.SetClientID("fleet-backend-" + strconv.FormatInt(time.Now().UnixNano(), 10))
		opts.SetAutoReconnect(true)
		opts.SetConnectionLostHandler(func(c mqtt.Client, err error) { log.WithError(err).Warn("MQTT connection lost") })
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.WithError(token.Error()).Error("MQTT connect failed")
		} else {
			log.WithField("broker", mqttURL).Info("MQTT connected")
			// Subscribe to telemetry topic; payload should mirror POST /api/telemetry body
			cb := func(_ mqtt.Client, msg mqtt.Message) {
				var teleIn struct {
					VehicleID    string          `json:"vehicle_id"`
					Timestamp    string          `json:"timestamp"`
					Location     models.Location `json:"location"`
					Speed        float64         `json:"speed"`
					FuelLevel    *float64        `json:"fuel_level,omitempty"`
					BatteryLevel *float64        `json:"battery_level,omitempty"`
					Emissions    float64         `json:"emissions"`
					Type         string          `json:"type"`
					Status       string          `json:"status"`
					TenantID     string          `json:"tenant_id,omitempty"`
				}
				if err := json.Unmarshal(msg.Payload(), &teleIn); err != nil {
					log.WithError(err).Warn("Invalid MQTT telemetry JSON")
					return
				}
				// Normalize and store
				timestamp, err := time.Parse(time.RFC3339, teleIn.Timestamp)
				if err != nil { return }
				var vehicleObjectID primitive.ObjectID
				if len(teleIn.VehicleID) == 24 {
					if oid, err := primitive.ObjectIDFromHex(teleIn.VehicleID); err == nil { vehicleObjectID = oid } else { vehicleObjectID = primitive.NewObjectID() }
				} else { vehicleObjectID = primitive.NewObjectID() }
				// Ensure EV emissions are zero
				emissions := teleIn.Emissions
				if teleIn.Type == "EV" {
					emissions = 0
				}
				tele := models.Telemetry{
					VehicleID:    vehicleObjectID,
					Timestamp:    timestamp,
					Location:     teleIn.Location,
					Speed:        teleIn.Speed,
					FuelLevel:    teleIn.FuelLevel,
					BatteryLevel: teleIn.BatteryLevel,
					Emissions:    emissions,
					Type:         teleIn.Type,
					Status:       teleIn.Status,
					TenantID:     teleIn.TenantID,
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := telemetryCollection.InsertTelemetry(ctx, tele); err != nil {
					log.WithError(err).Error("Failed to store MQTT telemetry")
					return
				}
				// Broadcast via SSE (tenant-aware if provided)
				eventPayload := map[string]interface{}{
					"vehicle_id":    teleIn.VehicleID,
					"timestamp":     teleIn.Timestamp,
					"location":      teleIn.Location,
					"speed":         teleIn.Speed,
					"fuel_level":    teleIn.FuelLevel,
					"battery_level": teleIn.BatteryLevel,
					"emissions":     teleIn.Emissions,
					"type":          teleIn.Type,
					"status":        teleIn.Status,
				}
                if data, err := json.Marshal(eventPayload); err == nil {
                    if telemetrySSEHub != nil && tele.TenantID != "" {
                        // Send to tenant listeners and also to global listeners (unauth SSE clients in dev)
                        telemetrySSEHub.BroadcastToTenant(tele.TenantID, data)
                        telemetrySSEHub.Broadcast(data)
                    } else if telemetrySSEHub != nil {
                        telemetrySSEHub.Broadcast(data)
                    }
                }
			}
			if token := client.Subscribe(mqttTopic, 1, cb); token.Wait() && token.Error() != nil {
				log.WithError(token.Error()).Error("MQTT subscribe failed")
			} else {
				log.WithFields(log.Fields{"topic": mqttTopic}).Info("MQTT subscribed")
			}
		}
	}

	// User profile routes (require authentication)
	http.HandleFunc("/api/auth/profile", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(authHandler.GetProfile))).ServeHTTP(w, r)
	})
	http.HandleFunc("/api/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(authMiddleware.Authenticate(http.HandlerFunc(authHandler.ChangePassword))).ServeHTTP(w, r)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	useHTTPS := os.Getenv("USE_HTTPS")
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	// Create server with graceful shutdown
	server := &http.Server{
		Addr:    ":" + port,
		Handler: nil, // Use default ServeMux
	}

	// Channel to listen for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if useHTTPS == "true" && certFile != "" && keyFile != "" {
			log.WithField("port", port).Info("HTTPS server listening")
			if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.WithError(err).Fatal("Server failed to start")
			}
		} else {
			log.WithField("port", port).Info("HTTP server listening")
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.WithError(err).Fatal("Server failed to start")
			}
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Info("Shutting down server gracefully...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
	} else {
		log.Info("Server exited gracefully")
	}
}
