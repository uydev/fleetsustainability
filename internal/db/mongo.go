package db

import (
    "context"
    "fmt"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/ukydev/fleet-sustainability/internal/models"
)

func ConnectMongo() (*mongo.Client, error) {
    uri := os.Getenv("MONGO_URI")
    if uri == "" {
        uri = "mongodb://root:example@mongo:27017"
    }
    client, err := mongo.NewClient(options.Client().ApplyURI(uri))
    if err != nil {
        return nil, fmt.Errorf("mongo.NewClient error: %w", err)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := client.Connect(ctx); err != nil {
        return nil, fmt.Errorf("mongo.Connect error: %w", err)
    }
    // Ping to verify connection
    if err := client.Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("mongo.Ping error: %w", err)
    }
    return client, nil
}

func InsertTelemetry(client *mongo.Client, telemetry models.Telemetry) error {
    if client == nil {
        return fmt.Errorf("mongo client is nil")
    }
    collection := client.Database("fleet").Collection("telemetry")
    _, err := collection.InsertOne(context.Background(), telemetry)
    return err
}