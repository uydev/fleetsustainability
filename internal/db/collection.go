package db

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/ukydev/fleet-sustainability/internal/models"
)

type TelemetryCursor interface {
    All(ctx context.Context, out interface{}) error
    Close(ctx context.Context) error
}

type TelemetryCollection interface {
    InsertTelemetry(ctx context.Context, telemetry models.Telemetry) error
    Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (TelemetryCursor, error)
}