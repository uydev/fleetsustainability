package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Location struct {
	Lat float64 `bson:"lat" json:"lat"`
	Lon float64 `bson:"lon" json:"lon"`
}

type Vehicle struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type            string             `bson:"type" json:"type"` // "ICE" or "EV"
	Make            string             `bson:"make" json:"make"`
	Model           string             `bson:"model" json:"model"`
	Year            int                `bson:"year" json:"year"`
	CurrentLocation Location           `bson:"current_location" json:"current_location"`
	Status          string             `bson:"status" json:"status"` // "active" or "inactive"
} 