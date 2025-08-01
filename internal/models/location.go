package models

type Location struct {
    Lat float64 `bson:"lat" json:"lat"`
    Lon float64 `bson:"lon" json:"lon"`
}