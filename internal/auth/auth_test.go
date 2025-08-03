package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewService(t *testing.T) {
	service, err := NewService()
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.NotEmpty(t, service.jwtSecret)
	assert.Equal(t, 24*time.Hour, service.tokenExp)
}

func TestService_HashPassword(t *testing.T) {
	service, _ := NewService()
	
	password := "testpassword123"
	hash, err := service.HashPassword(password)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestService_CheckPassword(t *testing.T) {
	service, _ := NewService()
	
	password := "testpassword123"
	hash, _ := service.HashPassword(password)
	
	// Test correct password
	assert.True(t, service.CheckPassword(password, hash))
	
	// Test incorrect password
	assert.False(t, service.CheckPassword("wrongpassword", hash))
}

func TestService_GenerateToken(t *testing.T) {
	service, _ := NewService()
	
	user := &models.User{
		ID:       primitive.NewObjectID(),
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	
	token, err := service.GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestService_ValidateToken(t *testing.T) {
	service, _ := NewService()
	
	user := &models.User{
		ID:       primitive.NewObjectID(),
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	
	token, _ := service.GenerateToken(user)
	
	// Test valid token
	claims, err := service.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID.Hex(), claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Role, claims.Role)
	
	// Test invalid token
	_, err = service.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	
	// Test token with Bearer prefix
	_, err = service.ValidateToken("Bearer " + token)
	assert.NoError(t, err)
}

func TestService_ExtractTokenFromHeader(t *testing.T) {
	service, _ := NewService()
	
	// Test valid header
	token := "valid-token"
	header := "Bearer " + token
	extracted, err := service.ExtractTokenFromHeader(header)
	assert.NoError(t, err)
	assert.Equal(t, token, extracted)
	
	// Test empty header
	_, err = service.ExtractTokenFromHeader("")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	
	// Test invalid format
	_, err = service.ExtractTokenFromHeader("InvalidFormat")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	
	// Test missing token
	_, err = service.ExtractTokenFromHeader("Bearer ")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestService_ValidatePassword(t *testing.T) {
	service, _ := NewService()
	
	// Test valid password
	err := service.ValidatePassword("validpassword123")
	assert.NoError(t, err)
	
	// Test too short password
	err = service.ValidatePassword("short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 8 characters")
}

func TestService_ValidateEmail(t *testing.T) {
	service, _ := NewService()
	
	// Test valid email
	err := service.ValidateEmail("test@example.com")
	assert.NoError(t, err)
	
	// Test invalid email - no @
	err = service.ValidateEmail("testexample.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")
	
	// Test invalid email - no domain
	err = service.ValidateEmail("test@")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")
	
	// Test invalid email - no @ and no domain
	err = service.ValidateEmail("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")
}

func TestService_ValidateUsername(t *testing.T) {
	service, _ := NewService()
	
	// Test valid username
	err := service.ValidateUsername("testuser")
	assert.NoError(t, err)
	
	// Test too short username
	err = service.ValidateUsername("ab")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 3 characters")
	
	// Test too long username
	longUsername := ""
	for i := 0; i < 51; i++ {
		longUsername += "a"
	}
	err = service.ValidateUsername(longUsername)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "less than 50 characters")
}

func TestService_GenerateRefreshToken(t *testing.T) {
	service, _ := NewService()
	
	token, err := service.GenerateRefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Len(t, token, 44) // base64 encoded 32 bytes (32 * 4/3 = 42.67, rounded up to 44)
}

func TestService_TokenExpiration(t *testing.T) {
	service, _ := NewService()
	
	user := &models.User{
		ID:       primitive.NewObjectID(),
		Username: "testuser",
		Role:     models.RoleAdmin,
	}
	
	token, _ := service.GenerateToken(user)
	
	// Token should be valid immediately
	claims, err := service.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	
	// Check expiration time
	now := time.Now().Unix()
	assert.Greater(t, claims.Exp, now)
	assert.LessOrEqual(t, claims.Exp, now+int64(service.tokenExp.Seconds())+1)
} 