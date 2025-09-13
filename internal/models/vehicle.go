package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Vehicle represents a fleet vehicle.
type Vehicle struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type            string             `bson:"type" json:"type"` // "ICE" or "EV"
	Make            string             `bson:"make" json:"make"`
	Model           string             `bson:"model" json:"model"`
	Year            int                `bson:"year" json:"year"`
	CurrentLocation Location           `bson:"current_location" json:"current_location"`
	Status          string             `bson:"status" json:"status"` // "active" or "inactive"
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}
