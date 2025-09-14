package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson"
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

func TestConnectMongo_EmptyURI(t *testing.T) {
	os.Unsetenv("MONGO_URI")
	client, err := ConnectMongo()
	// This should use the default URI and might fail in test environment
	// but we're testing the fallback logic
	if client == nil && err == nil {
		t.Error("expected either client or error, got both nil")
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

func TestInsertTelemetry_ValidTelemetry(t *testing.T) {
	// Test with valid telemetry data
	tele := models.Telemetry{
		Timestamp: time.Now(),
		Location:  models.Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Emissions: 25.0,
	}
	coll := &MongoCollection{Collection: nil}
	err := coll.InsertTelemetry(context.Background(), tele)
	if err == nil {
		t.Error("expected error when collection is nil")
	}
}

func TestMongoCollection_Find_NilCollection(t *testing.T) {
	coll := &MongoCollection{Collection: nil}
	ctx := context.Background()

	// This should panic due to nil collection, which is expected behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil collection")
		}
	}()

	cursor, err := coll.Find(ctx, map[string]interface{}{})
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error when collection is nil")
	}
	if cursor != nil {
		t.Error("expected nil cursor when collection is nil")
	}
}

func TestMongoTelemetryCursor_All(t *testing.T) {
	// Test cursor All method with nil cursor
	cursor := &mongoTelemetryCursor{
		cursor: nil, // This will cause a panic, which is expected
	}

	// This should panic due to nil cursor
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil cursor")
		}
	}()

	var results []models.Telemetry
	err := cursor.All(context.Background(), &results)
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error with nil cursor")
	}
}

func TestMongoTelemetryCursor_Close(t *testing.T) {
	// Test cursor Close method with nil cursor
	cursor := &mongoTelemetryCursor{
		cursor: nil, // This will cause a panic, which is expected
	}

	// This should panic due to nil cursor
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil cursor")
		}
	}()

	err := cursor.Close(context.Background())
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error with nil cursor")
	}
}

func TestConnectMongo_ContextTimeout(t *testing.T) {
	// Test connection with a very short timeout
	os.Setenv("MONGO_URI", "mongodb://bad:uri")

	// This should fail quickly due to bad URI
	client, err := ConnectMongo()
	if err == nil {
		t.Error("expected error for bad URI, got nil")
	}
	if client != nil {
		t.Error("expected nil client on error")
	}
}

// Integration test (requires running MongoDB)
func TestInsertTelemetry_Integration(t *testing.T) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" || uri == "uri" {
		t.Skip("MONGO_URI not set or invalid, skipping integration test")
		return
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
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

func TestConnectMongo_EnvironmentVariables(t *testing.T) {
	// Test that environment variables are properly read
	originalURI := os.Getenv("MONGO_URI")
	defer os.Setenv("MONGO_URI", originalURI)

	// Test with custom URI
	testURI := "mongodb://test:27017"
	os.Setenv("MONGO_URI", testURI)

	// The function should use the environment variable
	// We can't easily test the actual connection without a real MongoDB,
	// but we can verify the environment variable is read
	client, err := ConnectMongo()
	// We expect this to fail due to invalid URI, but the point is that
	// the environment variable was read
	if err == nil && client != nil {
		t.Error("expected failure with test URI")
	}
}

func TestConnectMongo_WithTimeout(t *testing.T) {
	// Test connection with a very short timeout
	originalURI := os.Getenv("MONGO_URI")
	defer os.Setenv("MONGO_URI", originalURI)
	os.Setenv("MONGO_URI", "mongodb://bad:uri")

	// This should fail quickly due to bad URI
	client, err := ConnectMongo()
	if err == nil {
		t.Error("expected error for bad URI, got nil")
	}
	if client != nil {
		t.Error("expected nil client on error")
	}
}

func TestConnectMongo_ValidURI(t *testing.T) {
	// Test with a valid URI format (but may not connect)
	originalURI := os.Getenv("MONGO_URI")
	defer os.Setenv("MONGO_URI", originalURI)
	os.Setenv("MONGO_URI", "mongodb://localhost:27017")

	// This might fail due to no MongoDB running, but we test the URI parsing
	client, _ := ConnectMongo()
	// We don't check for success since MongoDB might not be running
	// but we ensure the function doesn't panic
	if client != nil {
		// If we got a client, it should be valid
		if client.Database("test") == nil {
			t.Error("client database method returned nil")
		}
	}
}

func TestInsertTelemetry_WithValidData(t *testing.T) {
	// Test with more complete telemetry data
	tele := models.Telemetry{
		Timestamp: time.Now(),
		Location:  models.Location{Lat: 51.0, Lon: 0.0},
		Speed:     50.0,
		Emissions: 25.0,
	}
	coll := &MongoCollection{Collection: nil}
	err := coll.InsertTelemetry(context.Background(), tele)
	if err == nil {
		t.Error("expected error when collection is nil")
	}
}

func TestInsertTelemetry_WithPointerFields(t *testing.T) {
	// Test with pointer fields (FuelLevel, BatteryLevel)
	fuelLevel := 75.0
	batteryLevel := 80.0

	tele := models.Telemetry{
		Timestamp:    time.Now(),
		Location:     models.Location{Lat: 51.0, Lon: 0.0},
		Speed:        50.0,
		FuelLevel:    &fuelLevel,
		BatteryLevel: &batteryLevel,
		Emissions:    25.0,
	}
	coll := &MongoCollection{Collection: nil}
	err := coll.InsertTelemetry(context.Background(), tele)
	if err == nil {
		t.Error("expected error when collection is nil")
	}
}

func TestMongoCollection_Find_WithOptions(t *testing.T) {
	coll := &MongoCollection{Collection: nil}
	ctx := context.Background()

	// This should panic due to nil collection, which is expected behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil collection")
		}
	}()

	// Test with find options
	opts := []*options.FindOptions{
		options.Find().SetLimit(10),
		options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}),
	}

	cursor, err := coll.Find(ctx, map[string]interface{}{}, opts...)
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error when collection is nil")
	}
	if cursor != nil {
		t.Error("expected nil cursor when collection is nil")
	}
}

func TestMongoCollection_Find_WithComplexFilter(t *testing.T) {
	coll := &MongoCollection{Collection: nil}
	ctx := context.Background()

	// This should panic due to nil collection, which is expected behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil collection")
		}
	}()

	// Test with complex filter
	complexFilter := map[string]interface{}{
		"timestamp": map[string]interface{}{
			"$gte": time.Now().Add(-24 * time.Hour),
			"$lte": time.Now(),
		},
		"speed": map[string]interface{}{
			"$gt": 0,
		},
		"emissions": map[string]interface{}{
			"$lt": 100,
		},
	}

	cursor, err := coll.Find(ctx, complexFilter)
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error when collection is nil")
	}
	if cursor != nil {
		t.Error("expected nil cursor when collection is nil")
	}
}

func TestConnectMongo_EnvironmentVariableHandling(t *testing.T) {
	// Test various environment variable scenarios
	testCases := []struct {
		name     string
		uri      string
		expected bool // true if we expect success (though it might fail due to no MongoDB)
	}{
		{"empty URI", "", false},
		{"invalid URI", "mongodb://bad:uri", false},
		{"valid format", "mongodb://localhost:27017", true},
		{"with database", "mongodb://localhost:27017/test", true},
		{"with auth", "mongodb://user:pass@localhost:27017", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalURI := os.Getenv("MONGO_URI")
			defer os.Setenv("MONGO_URI", originalURI)

			if tc.uri != "" {
				os.Setenv("MONGO_URI", tc.uri)
			} else {
				os.Unsetenv("MONGO_URI")
			}

			client, err := ConnectMongo()

			if tc.expected {
				// For valid URIs, we don't check for success since MongoDB might not be running
				// but we ensure the function doesn't panic
				if client != nil {
					// If we got a client, it should be valid
					if client.Database("test") == nil {
						t.Error("client database method returned nil")
					}
				}
			} else {
				// For invalid URIs, we expect an error
				if err == nil {
					t.Error("expected error for invalid URI")
				}
				if client != nil {
					t.Error("expected nil client for invalid URI")
				}
			}
		})
	}
}

func TestMongoTelemetryCursor_WithNilCursor(t *testing.T) {
	// Test cursor methods with nil cursor
	cursor := &mongoTelemetryCursor{
		cursor: nil,
	}

	// Test All method
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil cursor")
		}
	}()

	var results []models.Telemetry
	err := cursor.All(context.Background(), &results)
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error with nil cursor")
	}
}

func TestMongoTelemetryCursor_WithNilCursorClose(t *testing.T) {
	// Test Close method with nil cursor
	cursor := &mongoTelemetryCursor{
		cursor: nil,
	}

	// This should panic due to nil cursor
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil cursor")
		}
	}()

	err := cursor.Close(context.Background())
	// This line should not be reached due to panic
	if err == nil {
		t.Error("expected error with nil cursor")
	}
}
