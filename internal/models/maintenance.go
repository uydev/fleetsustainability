package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Maintenance represents a vehicle maintenance record.
type Maintenance struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    TenantID        string             `json:"tenant_id" bson:"tenant_id"`
	VehicleID       string             `json:"vehicle_id" bson:"vehicle_id"`
	ServiceType     string             `json:"service_type" bson:"service_type"` // "oil_change", "tire_rotation", "brake_service", "battery_service", "inspection"
	Description     string             `json:"description" bson:"description"`
	ServiceDate     time.Time          `json:"service_date" bson:"service_date"`
	NextServiceDate time.Time          `json:"next_service_date" bson:"next_service_date"`
	Mileage         float64            `json:"mileage" bson:"mileage"` // in kilometers
	Cost            float64            `json:"cost" bson:"cost"`       // in USD
	LaborCost       float64            `json:"labor_cost" bson:"labor_cost"`
	PartsCost       float64            `json:"parts_cost" bson:"parts_cost"`
	Technician      string             `json:"technician" bson:"technician"`
	ServiceLocation string             `json:"service_location" bson:"service_location"`
	Status          string             `json:"status" bson:"status"`     // "scheduled", "in_progress", "completed", "cancelled"
	Priority        string             `json:"priority" bson:"priority"` // "low", "medium", "high", "critical"
	Notes           string             `json:"notes" bson:"notes"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}
