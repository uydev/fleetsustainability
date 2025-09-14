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
	DeleteAll(ctx context.Context) error
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
	DeleteAll(ctx context.Context) error
}

// VehicleCursor defines the interface for vehicle cursor operations.
type VehicleCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}

// TripCollection defines the interface for trip data operations.
type TripCollection interface {
	InsertTrip(ctx context.Context, trip models.Trip) error
	FindTrips(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (TripCursor, error)
	FindTripByID(ctx context.Context, id string) (*models.Trip, error)
	UpdateTrip(ctx context.Context, id string, trip models.Trip) error
	DeleteTrip(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error
}

// TripCursor defines the interface for trip cursor operations.
type TripCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}

// MaintenanceCollection defines the interface for maintenance data operations.
type MaintenanceCollection interface {
	InsertMaintenance(ctx context.Context, maintenance models.Maintenance) error
	FindMaintenance(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (MaintenanceCursor, error)
	FindMaintenanceByID(ctx context.Context, id string) (*models.Maintenance, error)
	UpdateMaintenance(ctx context.Context, id string, maintenance models.Maintenance) error
	DeleteMaintenance(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error
}

// MaintenanceCursor defines the interface for maintenance cursor operations.
type MaintenanceCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}

// CostCollection defines the interface for cost data operations.
type CostCollection interface {
	InsertCost(ctx context.Context, cost models.Cost) error
	FindCosts(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CostCursor, error)
	FindCostByID(ctx context.Context, id string) (*models.Cost, error)
	UpdateCost(ctx context.Context, id string, cost models.Cost) error
	DeleteCost(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) error
}

// CostCursor defines the interface for cost cursor operations.
type CostCursor interface {
	All(ctx context.Context, out interface{}) error
	Close(ctx context.Context) error
}
