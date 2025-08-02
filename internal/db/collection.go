package db

import (
	"context"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TelemetryCollection defines the interface for telemetry data operations.
type TelemetryCollection interface {
	InsertTelemetry(ctx context.Context, telemetry models.Telemetry) error
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (TelemetryCursor, error)
}

// TelemetryCursor defines the interface for telemetry cursor operations.
type TelemetryCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}

// VehicleCollection defines the interface for vehicle data operations.
type VehicleCollection interface {
	InsertVehicle(ctx context.Context, vehicle models.Vehicle) error
	FindVehicles(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (VehicleCursor, error)
	FindVehicleByID(ctx context.Context, id string) (*models.Vehicle, error)
	UpdateVehicle(ctx context.Context, id string, vehicle models.Vehicle) error
	DeleteVehicle(ctx context.Context, id string) error
}

// VehicleCursor defines the interface for vehicle cursor operations.
type VehicleCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}