package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ukydev/fleet-sustainability/internal/auth"
	"github.com/ukydev/fleet-sustainability/internal/models"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	UserContextKey contextKey = "user"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authService *auth.Service
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *auth.Service) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate validates JWT tokens and adds user context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain endpoints
		if shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := m.authService.ValidateToken(authHeader)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user context to request
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware checks if the user has the required role
func (m *AuthMiddleware) RequireRole(requiredRole models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*models.Claims)
			if !ok {
				http.Error(w, "User context not found", http.StatusUnauthorized)
				return
			}

			if claims.Role != requiredRole && claims.Role != models.RoleAdmin {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission middleware checks if the user has the required permission
func (m *AuthMiddleware) RequirePermission(requiredAction string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*models.Claims)
			if !ok {
				http.Error(w, "User context not found", http.StatusUnauthorized)
				return
			}

			// Create a temporary user object to check permissions
			user := &models.User{
				Role: claims.Role,
			}

			if !user.HasPermission(requiredAction) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts user claims from request context
func GetUserFromContext(ctx context.Context) (*models.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*models.Claims)
	return claims, ok
}

// shouldSkipAuth determines if authentication should be skipped for a given path
func shouldSkipAuth(path string) bool {
	// Skip auth for login and register endpoints
	skipPaths := []string{
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/refresh",
		"/health",
		"/metrics",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// RateLimitMiddleware provides basic rate limiting
type RateLimitMiddleware struct {
	requests map[string][]int64 // IP -> timestamps
	mu       sync.RWMutex       // Mutex for thread-safe access
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware() *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requests: make(map[string][]int64),
	}
}

// RateLimit applies rate limiting based on IP address
func (m *RateLimitMiddleware) RateLimit(maxRequests int, windowSeconds int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := getClientIP(r)

			// Clean old requests outside the window
			now := time.Now().Unix()
			windowStart := now - int64(windowSeconds)

			// Use write lock for map operations
			m.mu.Lock()

			if timestamps, exists := m.requests[clientIP]; exists {
				var validTimestamps []int64
				for _, ts := range timestamps {
					if ts >= windowStart {
						validTimestamps = append(validTimestamps, ts)
					}
				}
				m.requests[clientIP] = validTimestamps
			}

			// Check if rate limit exceeded
			if len(m.requests[clientIP]) >= maxRequests {
				m.mu.Unlock()
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Add current request
			m.requests[clientIP] = append(m.requests[clientIP], now)

			// Release lock
			m.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}
