package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Trip represents a vehicle trip from start to end location.
type Trip struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    TenantID           string             `json:"tenant_id" bson:"tenant_id"`
	VehicleID          string             `json:"vehicle_id" bson:"vehicle_id"`
	DriverID           string             `json:"driver_id" bson:"driver_id"`
	StartLocation      Location           `json:"start_location" bson:"start_location"`
	EndLocation        Location           `json:"end_location" bson:"end_location"`
	StartTime          time.Time          `json:"start_time" bson:"start_time"`
	EndTime            time.Time          `json:"end_time" bson:"end_time"`
	Distance           float64            `json:"distance" bson:"distance"`                       // in kilometers
	Duration           float64            `json:"duration" bson:"duration"`                       // in hours
	FuelConsumption    float64            `json:"fuel_consumption" bson:"fuel_consumption"`       // in liters
	BatteryConsumption float64            `json:"battery_consumption" bson:"battery_consumption"` // in kWh
	Cost               float64            `json:"cost" bson:"cost"`                               // in USD
	Purpose            string             `json:"purpose" bson:"purpose"`                         // "business", "personal", "delivery"
	Status             string             `json:"status" bson:"status"`                           // "planned", "in_progress", "completed", "cancelled"
	Notes              string             `json:"notes" bson:"notes"`
	CreatedAt          time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" bson:"updated_at"`
}
