package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMongoUserCollection_InsertUser(t *testing.T) {
	// Setup test database
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	
	// Clean up before test
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	assert.NoError(t, err)
	
	// Verify user was inserted
	var foundUser models.User
	err = collection.FindOne(context.Background(), bson.M{"username": "testuser"}).Decode(&foundUser)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Email, foundUser.Email)
	assert.Equal(t, user.Role, foundUser.Role)
	assert.True(t, foundUser.IsActive)
	assert.NotZero(t, foundUser.CreatedAt)
	assert.NotZero(t, foundUser.UpdatedAt)
}

func TestMongoUserCollection_FindUserByID(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	// Insert test user
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Get the inserted user's ID
	var insertedUser models.User
	err = collection.FindOne(context.Background(), bson.M{"username": "testuser"}).Decode(&insertedUser)
	require.NoError(t, err)
	
	// Find user by ID
	foundUser, err := userCollection.FindUserByID(context.Background(), insertedUser.ID.Hex())
	assert.NoError(t, err)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Email, foundUser.Email)
	
	// Test with invalid ID
	_, err = userCollection.FindUserByID(context.Background(), "invalid-id")
	assert.Error(t, err)
}

func TestMongoUserCollection_FindUserByUsername(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Find user by username
	foundUser, err := userCollection.FindUserByUsername(context.Background(), "testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Email, foundUser.Email)
	
	// Test with non-existent username
	_, err = userCollection.FindUserByUsername(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestMongoUserCollection_FindUserByEmail(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Find user by email
	foundUser, err := userCollection.FindUserByEmail(context.Background(), "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, foundUser.Username)
	assert.Equal(t, user.Email, foundUser.Email)
	
	// Test with non-existent email
	_, err = userCollection.FindUserByEmail(context.Background(), "nonexistent@example.com")
	assert.Error(t, err)
}

func TestMongoUserCollection_UpdateUser(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	// Insert test user
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Get the inserted user
	var insertedUser models.User
	err = collection.FindOne(context.Background(), bson.M{"username": "testuser"}).Decode(&insertedUser)
	require.NoError(t, err)
	
	// Update user
	updatedUser := insertedUser
	updatedUser.FirstName = "Updated"
	updatedUser.LastName = "Name"
	
	err = userCollection.UpdateUser(context.Background(), insertedUser.ID.Hex(), updatedUser)
	assert.NoError(t, err)
	
	// Verify update
	foundUser, err := userCollection.FindUserByID(context.Background(), insertedUser.ID.Hex())
	assert.NoError(t, err)
	assert.Equal(t, "Updated", foundUser.FirstName)
	assert.Equal(t, "Name", foundUser.LastName)
	assert.True(t, foundUser.UpdatedAt.After(insertedUser.UpdatedAt))
}

func TestMongoUserCollection_DeleteUser(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	// Insert test user
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Get the inserted user
	var insertedUser models.User
	err = collection.FindOne(context.Background(), bson.M{"username": "testuser"}).Decode(&insertedUser)
	require.NoError(t, err)
	
	// Delete user
	err = userCollection.DeleteUser(context.Background(), insertedUser.ID.Hex())
	assert.NoError(t, err)
	
	// Verify user was deleted
	_, err = userCollection.FindUserByID(context.Background(), insertedUser.ID.Hex())
	assert.Error(t, err)
}

func TestMongoUserCollection_UpdateLastLogin(t *testing.T) {
	client, err := ConnectMongo()
	if err != nil {
		t.Skipf("failed to create client: %v, skipping integration test", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database("test_fleet")
	collection := db.Collection("users")
	collection.Drop(context.Background())
	
	userCollection := &MongoUserCollection{Collection: collection}
	
	// Insert test user
	user := models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         models.RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
	}
	
	err = userCollection.InsertUser(context.Background(), user)
	require.NoError(t, err)
	
	// Get the inserted user
	var insertedUser models.User
	err = collection.FindOne(context.Background(), bson.M{"username": "testuser"}).Decode(&insertedUser)
	require.NoError(t, err)
	
	// Update last login
	err = userCollection.UpdateLastLogin(context.Background(), insertedUser.ID.Hex())
	assert.NoError(t, err)
	
	// Verify last login was updated
	updatedUser, err := userCollection.FindUserByID(context.Background(), insertedUser.ID.Hex())
	assert.NoError(t, err)
	assert.NotNil(t, updatedUser.LastLogin)
	assert.True(t, updatedUser.LastLogin.After(insertedUser.CreatedAt))
} 