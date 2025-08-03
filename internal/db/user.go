package db

import (
	"context"
	"time"

	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserCollection defines the interface for user database operations
type UserCollection interface {
	InsertUser(ctx context.Context, user models.User) error
	FindUserByID(ctx context.Context, id string) (*models.User, error)
	FindUserByUsername(ctx context.Context, username string) (*models.User, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindUsers(ctx context.Context, filter bson.M) (*mongo.Cursor, error)
	UpdateUser(ctx context.Context, id string, user models.User) error
	DeleteUser(ctx context.Context, id string) error
	UpdateLastLogin(ctx context.Context, id string) error
}

// MongoUserCollection implements UserCollection for MongoDB
type MongoUserCollection struct {
	Collection *mongo.Collection
}

// InsertUser inserts a new user into the database
func (c *MongoUserCollection) InsertUser(ctx context.Context, user models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true
	
	_, err := c.Collection.InsertOne(ctx, user)
	return err
}

// FindUserByID finds a user by their ID
func (c *MongoUserCollection) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	
	var user models.User
	err = c.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// FindUserByUsername finds a user by their username
func (c *MongoUserCollection) FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := c.Collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// FindUserByEmail finds a user by their email
func (c *MongoUserCollection) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := c.Collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// FindUsers finds users with optional filtering
func (c *MongoUserCollection) FindUsers(ctx context.Context, filter bson.M) (*mongo.Cursor, error) {
	return c.Collection.Find(ctx, filter)
}

// UpdateUser updates a user in the database
func (c *MongoUserCollection) UpdateUser(ctx context.Context, id string, user models.User) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	
	user.UpdatedAt = time.Now()
	user.ID = objectID
	
	_, err = c.Collection.ReplaceOne(ctx, bson.M{"_id": objectID}, user)
	return err
}

// DeleteUser deletes a user from the database
func (c *MongoUserCollection) DeleteUser(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	
	_, err = c.Collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// UpdateLastLogin updates the last login time for a user
func (c *MongoUserCollection) UpdateLastLogin(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	
	now := time.Now()
	_, err = c.Collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"last_login": now, "updated_at": now}},
	)
	return err
} 