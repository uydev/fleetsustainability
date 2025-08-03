package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/ukydev/fleet-sustainability/internal/auth"
	"github.com/ukydev/fleet-sustainability/internal/db"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockUserCollection is a mock implementation of UserCollection
type MockUserCollection struct {
	mock.Mock
}

func (m *MockUserCollection) InsertUser(ctx context.Context, user models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserCollection) FindUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserCollection) FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserCollection) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserCollection) FindUsers(ctx context.Context, filter bson.M) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockUserCollection) UpdateUser(ctx context.Context, id string, user models.User) error {
	args := m.Called(ctx, id, user)
	return args.Error(0)
}

func (m *MockUserCollection) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserCollection) UpdateLastLogin(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAuthHandler_Login(t *testing.T) {
	authService, _ := auth.NewService()

	t.Run("successful login", func(t *testing.T) {
		mockUserCollection := new(MockUserCollection)
		handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

		// Create a real password hash
		passwordHash, _ := authService.HashPassword("password123")
		user := &models.User{
			ID:           primitive.NewObjectID(),
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: passwordHash,
			Role:         models.RoleAdmin,
			IsActive:     true,
		}

		mockUserCollection.On("FindUserByUsername", mock.Anything, "testuser").Return(user, nil)
		mockUserCollection.On("UpdateLastLogin", mock.Anything, user.ID.Hex()).Return(nil)

		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.LoginResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEmpty(t, response.Token)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, user.Username, response.User.Username)

		mockUserCollection.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockUserCollection := new(MockUserCollection)
		handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

		mockUserCollection.On("FindUserByUsername", mock.Anything, "testuser").Return(nil, assert.AnError)

		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "wrongpassword",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockUserCollection.AssertExpectations(t)
	})

	t.Run("inactive user", func(t *testing.T) {
		mockUserCollection := new(MockUserCollection)
		handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

		// Create a real password hash
		passwordHash, _ := authService.HashPassword("password123")
		user := &models.User{
			ID:           primitive.NewObjectID(),
			Username:     "testuser",
			PasswordHash: passwordHash,
			IsActive:     false,
		}

		mockUserCollection.On("FindUserByUsername", mock.Anything, "testuser").Return(user, nil)

		loginReq := models.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}

		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockUserCollection.AssertExpectations(t)
	})
}

func TestAuthHandler_Register(t *testing.T) {
	authService, _ := auth.NewService()
	mockUserCollection := new(MockUserCollection)
	handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

	t.Run("successful registration", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Username:  "newuser",
			Email:     "newuser@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
			Role:      models.RoleViewer,
		}

		// Mock that user doesn't exist
		mockUserCollection.On("FindUserByUsername", mock.Anything, "newuser").Return(nil, assert.AnError)
		mockUserCollection.On("FindUserByEmail", mock.Anything, "newuser@example.com").Return(nil, assert.AnError)
		mockUserCollection.On("InsertUser", mock.Anything, mock.AnythingOfType("models.User")).Return(nil)

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.LoginResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.NotEmpty(t, response.Token)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, registerReq.Username, response.User.Username)

		mockUserCollection.AssertExpectations(t)
	})

	t.Run("username already exists", func(t *testing.T) {
		existingUser := &models.User{Username: "existinguser"}
		registerReq := models.RegisterRequest{
			Username:  "existinguser",
			Email:     "newuser@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
			Role:      models.RoleViewer,
		}

		mockUserCollection.On("FindUserByUsername", mock.Anything, "existinguser").Return(existingUser, nil)

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockUserCollection.AssertExpectations(t)
	})

	t.Run("invalid role", func(t *testing.T) {
		registerReq := models.RegisterRequest{
			Username:  "newuser",
			Email:     "newuser@example.com",
			Password:  "password123",
			FirstName: "New",
			LastName:  "User",
			Role:      "invalid_role",
		}

		body, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_GetProfile(t *testing.T) {
	authService, _ := auth.NewService()
	mockUserCollection := new(MockUserCollection)
	handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

	t.Run("successful profile retrieval", func(t *testing.T) {
		userID := primitive.NewObjectID()
		user := &models.User{
			ID:        userID,
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			Role:      models.RoleAdmin,
		}

		claims := &models.Claims{
			UserID:   userID.Hex(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}

		mockUserCollection.On("FindUserByID", mock.Anything, userID.Hex()).Return(user, nil)

		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		ctx := context.WithValue(req.Context(), "user", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.User
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, user.Username, response.Username)
		assert.Equal(t, user.Email, response.Email)

		mockUserCollection.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := primitive.NewObjectID()
		claims := &models.Claims{
			UserID:   userID.Hex(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}

		mockUserCollection.On("FindUserByID", mock.Anything, userID.Hex()).Return(nil, assert.AnError)

		req := httptest.NewRequest("GET", "/api/auth/profile", nil)
		ctx := context.WithValue(req.Context(), "user", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetProfile(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockUserCollection.AssertExpectations(t)
	})
}

func TestAuthHandler_UpdateProfile(t *testing.T) {
	authService, _ := auth.NewService()
	mockUserCollection := new(MockUserCollection)
	handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

	t.Run("successful profile update", func(t *testing.T) {
		userID := primitive.NewObjectID()
		user := &models.User{
			ID:        userID,
			Username:  "testuser",
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			Role:      models.RoleAdmin,
		}

		claims := &models.Claims{
			UserID:   userID.Hex(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}

		updateReq := map[string]string{
			"first_name": "Updated",
			"last_name":  "Name",
		}

		mockUserCollection.On("FindUserByID", mock.Anything, userID.Hex()).Return(user, nil)
		mockUserCollection.On("UpdateUser", mock.Anything, userID.Hex(), mock.AnythingOfType("models.User")).Return(nil)

		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.UpdateProfile(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUserCollection.AssertExpectations(t)
	})
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	authService, _ := auth.NewService()
	mockUserCollection := new(MockUserCollection)
	handler := NewAuthHandler(authService, db.UserCollection(mockUserCollection))

	t.Run("successful password change", func(t *testing.T) {
		userID := primitive.NewObjectID()
		// Create a real password hash
		passwordHash, _ := authService.HashPassword("oldpassword")
		user := &models.User{
			ID:           userID,
			Username:     "testuser",
			PasswordHash: passwordHash,
		}

		claims := &models.Claims{
			UserID:   userID.Hex(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}

		passwordReq := map[string]string{
			"current_password": "oldpassword",
			"new_password":     "newpassword123",
		}

		mockUserCollection.On("FindUserByID", mock.Anything, userID.Hex()).Return(user, nil)
		mockUserCollection.On("UpdateUser", mock.Anything, userID.Hex(), mock.AnythingOfType("models.User")).Return(nil)

		body, _ := json.Marshal(passwordReq)
		req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUserCollection.AssertExpectations(t)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		userID := primitive.NewObjectID()
		// Create a real password hash
		passwordHash, _ := authService.HashPassword("oldpassword")
		user := &models.User{
			ID:           userID,
			Username:     "testuser",
			PasswordHash: passwordHash,
		}

		claims := &models.Claims{
			UserID:   userID.Hex(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}

		passwordReq := map[string]string{
			"current_password": "wrongpassword",
			"new_password":     "newpassword123",
		}

		mockUserCollection.On("FindUserByID", mock.Anything, userID.Hex()).Return(user, nil)

		body, _ := json.Marshal(passwordReq)
		req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockUserCollection.AssertExpectations(t)
	})
} 