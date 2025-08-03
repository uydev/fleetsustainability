package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ukydev/fleet-sustainability/internal/auth"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAuthMiddleware_Authenticate(t *testing.T) {
	authService, _ := auth.NewService()
	middleware := NewAuthMiddleware(authService)

	// Test successful authentication
	t.Run("valid token", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "testuser",
			Role:     models.RoleAdmin,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/telemetry", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			claims, ok := GetUserFromContext(r.Context())
			assert.True(t, ok)
			assert.Equal(t, user.Username, claims.Username)
			assert.Equal(t, user.Role, claims.Role)
		})

		middleware.Authenticate(handler).ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test missing authorization header
	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/telemetry", nil)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware.Authenticate(handler).ServeHTTP(w, req)
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test invalid token
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/telemetry", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware.Authenticate(handler).ServeHTTP(w, req)
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test skip auth paths
	t.Run("skip auth path", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		middleware.Authenticate(handler).ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthMiddleware_RequireRole(t *testing.T) {
	authService, _ := auth.NewService()
	middleware := NewAuthMiddleware(authService)

	// Test admin can access manager endpoint
	t.Run("admin accessing manager endpoint", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "admin",
			Role:     models.RoleAdmin,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/managers", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		// Add authentication first
		authHandler := middleware.Authenticate(middleware.RequireRole(models.RoleManager)(handler))
		authHandler.ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test manager cannot access admin endpoint
	t.Run("manager accessing admin endpoint", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "manager",
			Role:     models.RoleManager,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		authHandler := middleware.Authenticate(middleware.RequireRole(models.RoleAdmin)(handler))
		authHandler.ServeHTTP(w, req)
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestAuthMiddleware_RequirePermission(t *testing.T) {
	authService, _ := auth.NewService()
	middleware := NewAuthMiddleware(authService)

	// Test admin can access any permission
	t.Run("admin accessing any permission", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "admin",
			Role:     models.RoleAdmin,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		authHandler := middleware.Authenticate(middleware.RequirePermission("delete_user")(handler))
		authHandler.ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test viewer cannot access admin permission
	t.Run("viewer accessing admin permission", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "viewer",
			Role:     models.RoleViewer,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		authHandler := middleware.Authenticate(middleware.RequirePermission("delete_user")(handler))
		authHandler.ServeHTTP(w, req)
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Test viewer can access view permission
	t.Run("viewer accessing view permission", func(t *testing.T) {
		user := &models.User{
			ID:       primitive.NewObjectID(),
			Username: "viewer",
			Role:     models.RoleViewer,
		}
		token, _ := authService.GenerateToken(user)

		req := httptest.NewRequest("GET", "/api/telemetry", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		authHandler := middleware.Authenticate(middleware.RequirePermission("view_telemetry")(handler))
		authHandler.ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	middleware := NewRateLimitMiddleware()

	t.Run("rate limit not exceeded", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rateLimitHandler := middleware.RateLimit(5, 60)(handler)
		rateLimitHandler.ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		})

		rateLimitHandler := middleware.RateLimit(1, 60)(handler)
		
		// First request should succeed
		rateLimitHandler.ServeHTTP(w, req)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)

		// Second request should be rate limited
		w = httptest.NewRecorder()
		handlerCalled = false
		rateLimitHandler.ServeHTTP(w, req)
		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}

func TestGetUserFromContext(t *testing.T) {
	claims := &models.Claims{
		UserID:   "test-id",
		Username: "testuser",
		Role:     models.RoleAdmin,
	}

	ctx := context.WithValue(context.Background(), "user", claims)
	
	retrievedClaims, ok := GetUserFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, claims.UserID, retrievedClaims.UserID)
	assert.Equal(t, claims.Username, retrievedClaims.Username)
	assert.Equal(t, claims.Role, retrievedClaims.Role)

	// Test with no user in context
	emptyCtx := context.Background()
	_, ok = GetUserFromContext(emptyCtx)
	assert.False(t, ok)
} 