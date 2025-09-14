package models

// Location represents a geographical location with latitude and longitude coordinates.
type Location struct {
	Lat float64 `bson:"lat" json:"lat"`
	Lon float64 `bson:"lon" json:"lon"`
}
