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
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
    if err != nil {
        return nil, fmt.Errorf("mongo.NewClient error: %w", err)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
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

// DeleteAll deletes all telemetry records from the collection.
func (c *MongoCollection) DeleteAll(ctx context.Context) error {
    if c.Collection == nil {
        return fmt.Errorf("mongo collection is nil")
    }
    _, err := c.Collection.DeleteMany(ctx, bson.M{})
    return err
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



// InsertTrip inserts a trip record into the collection.
func (c *MongoCollection) InsertTrip(ctx context.Context, trip models.Trip) error {
	trip.CreatedAt = time.Now()
	trip.UpdatedAt = time.Now()
	_, err := c.Collection.InsertOne(ctx, trip)
	return err
}

// FindTrips queries trip records from the collection.
func (c *MongoCollection) FindTrips(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (TripCursor, error) {
	cursor, err := c.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return &mongoTripCursor{cursor: cursor}, nil
}

// FindTripByID finds a trip by its ID.
func (c *MongoCollection) FindTripByID(ctx context.Context, id string) (*models.Trip, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var trip models.Trip
	err = c.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&trip)
	if err != nil {
		return nil, err
	}
	return &trip, nil
}

// UpdateTrip updates a trip by its ID.
func (c *MongoCollection) UpdateTrip(ctx context.Context, id string, trip models.Trip) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	trip.UpdatedAt = time.Now()
	_, err = c.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": trip})
	return err
}

// DeleteTrip deletes a trip by its ID.
func (c *MongoCollection) DeleteTrip(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = c.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// InsertMaintenance inserts a maintenance record into the collection.
func (c *MongoCollection) InsertMaintenance(ctx context.Context, maintenance models.Maintenance) error {
	maintenance.CreatedAt = time.Now()
	maintenance.UpdatedAt = time.Now()
	_, err := c.Collection.InsertOne(ctx, maintenance)
	return err
}

// FindMaintenance queries maintenance records from the collection.
func (c *MongoCollection) FindMaintenance(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (MaintenanceCursor, error) {
	cursor, err := c.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return &mongoMaintenanceCursor{cursor: cursor}, nil
}

// FindMaintenanceByID finds a maintenance record by its ID.
func (c *MongoCollection) FindMaintenanceByID(ctx context.Context, id string) (*models.Maintenance, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var maintenance models.Maintenance
	err = c.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&maintenance)
	if err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// UpdateMaintenance updates a maintenance record by its ID.
func (c *MongoCollection) UpdateMaintenance(ctx context.Context, id string, maintenance models.Maintenance) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	maintenance.UpdatedAt = time.Now()
	_, err = c.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": maintenance})
	return err
}

// DeleteMaintenance deletes a maintenance record by its ID.
func (c *MongoCollection) DeleteMaintenance(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = c.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// InsertCost inserts a cost record into the collection.
func (c *MongoCollection) InsertCost(ctx context.Context, cost models.Cost) error {
	cost.CreatedAt = time.Now()
	cost.UpdatedAt = time.Now()
	_, err := c.Collection.InsertOne(ctx, cost)
	return err
}

// FindCosts queries cost records from the collection.
func (c *MongoCollection) FindCosts(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (CostCursor, error) {
	cursor, err := c.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return &mongoCostCursor{cursor: cursor}, nil
}

// FindCostByID finds a cost record by its ID.
func (c *MongoCollection) FindCostByID(ctx context.Context, id string) (*models.Cost, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var cost models.Cost
	err = c.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&cost)
	if err != nil {
		return nil, err
	}
	return &cost, nil
}

// UpdateCost updates a cost record by its ID.
func (c *MongoCollection) UpdateCost(ctx context.Context, id string, cost models.Cost) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	cost.UpdatedAt = time.Now()
	_, err = c.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": cost})
	return err
}

// DeleteCost deletes a cost record by its ID.
func (c *MongoCollection) DeleteCost(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = c.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// Cursor implementations
type mongoTripCursor struct {
	cursor *mongo.Cursor
}

func (c *mongoTripCursor) All(ctx context.Context, out interface{}) error {
	return c.cursor.All(ctx, out)
}

func (c *mongoTripCursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}

type mongoMaintenanceCursor struct {
	cursor *mongo.Cursor
}

func (c *mongoMaintenanceCursor) All(ctx context.Context, out interface{}) error {
	return c.cursor.All(ctx, out)
}

func (c *mongoMaintenanceCursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}

type mongoCostCursor struct {
	cursor *mongo.Cursor
}

func (c *mongoCostCursor) All(ctx context.Context, out interface{}) error {
	return c.cursor.All(ctx, out)
}

func (c *mongoCostCursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}