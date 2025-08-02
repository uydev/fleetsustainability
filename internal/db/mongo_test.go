package db

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func TestConnectMongo_BadURI(t *testing.T) {
    os.Setenv("MONGO_URI", "mongodb://bad:uri")
    client, err := ConnectMongo()
    if err == nil {
        t.Error("expected error for bad URI, got nil")
    }
    if client != nil {
        t.Error("expected nil client on error")
    }
}

func TestInsertTelemetry_NilClient(t *testing.T) {
    tele := models.Telemetry{}
    coll := &MongoCollection{Collection: nil}
    err := coll.InsertTelemetry(context.Background(), tele)
    if err == nil {
        t.Error("expected error when collection is nil")
    }
}

// Integration test (requires running MongoDB)
func TestInsertTelemetry_Integration(t *testing.T) {
    uri := os.Getenv("MONGO_URI")
    if uri == "" || uri == "uri" {
        t.Skip("MONGO_URI not set or invalid, skipping integration test")
        return
    }
    client, err := mongo.NewClient(options.Client().ApplyURI(uri))
    if err != nil {
        t.Skipf("failed to create client: %v, skipping integration test", err)
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := client.Connect(ctx); err != nil {
        t.Skipf("failed to connect: %v, skipping integration test", err)
        return
    }
    dbName := os.Getenv("MONGO_DB")
    if dbName == "" {
        dbName = "fleet"
    }
    coll := &MongoCollection{Collection: client.Database(dbName).Collection("telemetry")}
    tele := models.Telemetry{}
    err = coll.InsertTelemetry(context.Background(), tele)
    if err != nil {
        t.Errorf("expected insert to succeed, got error: %v", err)
    }
}