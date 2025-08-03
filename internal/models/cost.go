package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Cost represents a fleet cost record.
type Cost struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VehicleID   string            `json:"vehicle_id" bson:"vehicle_id"`
	Category    string            `json:"category" bson:"category"` // "fuel", "maintenance", "insurance", "registration", "tolls", "parking", "other"
	Description string            `json:"description" bson:"description"`
	Amount      float64           `json:"amount" bson:"amount"` // in USD
	Date        time.Time         `json:"date" bson:"date"`
	InvoiceNumber string          `json:"invoice_number" bson:"invoice_number"`
	Vendor      string            `json:"vendor" bson:"vendor"`
	Location    string            `json:"location" bson:"location"`
	PaymentMethod string          `json:"payment_method" bson:"payment_method"` // "credit_card", "cash", "check", "electronic"
	Status      string            `json:"status" bson:"status"` // "pending", "paid", "disputed", "cancelled"
	ReceiptURL  string            `json:"receipt_url" bson:"receipt_url"`
	Notes       string            `json:"notes" bson:"notes"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
} 