package db

import (
    "context"
    "fmt"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/ukydev/fleet-sustainability/internal/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
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

// mongoVehicleCursor wraps a MongoDB cursor for vehicle queries.
type mongoVehicleCursor struct {
	cursor *mongo.Cursor
}

// All retrieves all results from the cursor.
func (m *mongoVehicleCursor) All(ctx context.Context, out interface{}) error {
	return m.cursor.All(ctx, out)
}

// Close closes the cursor.
func (m *mongoVehicleCursor) Close(ctx context.Context) error {
	return m.cursor.Close(ctx)
}

// InsertVehicle inserts a vehicle record into the collection.
func (c *MongoCollection) InsertVehicle(ctx context.Context, vehicle models.Vehicle) error {
	if c.Collection == nil {
		return fmt.Errorf("mongo collection is nil")
	}
	_, err := c.Collection.InsertOne(ctx, vehicle)
	return err
}

// FindVehicles queries vehicle records from the collection.
func (c *MongoCollection) FindVehicles(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (VehicleCursor, error) {
	if c.Collection == nil {
		return nil, fmt.Errorf("mongo collection is nil")
	}
	
	var findOptions *options.FindOptions
	if len(opts) > 0 {
		findOptions = opts[0]
	}
	
	cursor, err := c.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	
	return &mongoVehicleCursor{cursor: cursor}, nil
}

// FindVehicleByID finds a vehicle by its ID.
func (c *MongoCollection) FindVehicleByID(ctx context.Context, id string) (*models.Vehicle, error) {
	if c.Collection == nil {
		return nil, fmt.Errorf("mongo collection is nil")
	}
	
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid vehicle ID: %w", err)
	}
	
	var vehicle models.Vehicle
	err = c.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("vehicle not found")
		}
		return nil, err
	}
	
	return &vehicle, nil
}

// UpdateVehicle updates a vehicle by its ID.
func (c *MongoCollection) UpdateVehicle(ctx context.Context, id string, vehicle models.Vehicle) error {
	if c.Collection == nil {
		return fmt.Errorf("mongo collection is nil")
	}
	
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid vehicle ID: %w", err)
	}
	
	result, err := c.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": vehicle})
	if err != nil {
		return err
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("vehicle not found")
	}
	
	return nil
}

// DeleteVehicle deletes a vehicle by its ID.
func (c *MongoCollection) DeleteVehicle(ctx context.Context, id string) error {
	if c.Collection == nil {
		return fmt.Errorf("mongo collection is nil")
	}
	
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid vehicle ID: %w", err)
	}
	
	result, err := c.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("vehicle not found")
	}
	
	return nil
}