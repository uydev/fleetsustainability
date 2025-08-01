package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Telemetry struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VehicleID   primitive.ObjectID `bson:"vehicle_id" json:"vehicle_id"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Location    Location           `bson:"location" json:"location"`
	Speed       float64            `bson:"speed" json:"speed"`
	FuelLevel   *float64           `bson:"fuel_level,omitempty" json:"fuel_level,omitempty"`
	BatteryLevel *float64          `bson:"battery_level,omitempty" json:"battery_level,omitempty"`
	Emissions   float64            `bson:"emissions" json:"emissions"`
} 