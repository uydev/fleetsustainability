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

// ConnectMongo connects to MongoDB using the MONGO_URI environment variable.
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

// MongoCollection wraps a MongoDB collection for telemetry operations.
type MongoCollection struct {
    Collection *mongo.Collection
}

// InsertTelemetry inserts a telemetry record into the collection.
func (c *MongoCollection) InsertTelemetry(ctx context.Context, telemetry models.Telemetry) error {
    if c.Collection == nil {
        return fmt.Errorf("mongo collection is nil")
    }
    _, err := c.Collection.InsertOne(ctx, telemetry)
    return err
}

// mongoTelemetryCursor wraps a MongoDB cursor for telemetry queries.
type mongoTelemetryCursor struct {
    cursor *mongo.Cursor
}

// All retrieves all results from the cursor.
func (m *mongoTelemetryCursor) All(ctx context.Context, out interface{}) error {
    return m.cursor.All(ctx, out)
}
func (m *mongoTelemetryCursor) Close(ctx context.Context) error {
    return m.cursor.Close(ctx)
}

// Find queries telemetry records from the collection.
func (c *MongoCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (TelemetryCursor, error) {
    cursor, err := c.Collection.Find(ctx, filter, opts...)
    if err != nil {
        return nil, err
    }
    return &mongoTelemetryCursor{cursor: cursor}, nil
}