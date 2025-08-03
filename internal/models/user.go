package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Role represents user roles in the system
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// User represents a user in the system
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username     string            `bson:"username" json:"username"`
	Email        string            `bson:"email" json:"email"`
	PasswordHash string            `bson:"password_hash" json:"-"`
	Role         Role              `bson:"role" json:"role"`
	FirstName    string            `bson:"first_name" json:"first_name"`
	LastName     string            `bson:"last_name" json:"last_name"`
	IsActive     bool              `bson:"is_active" json:"is_active"`
	LastLogin    *time.Time        `bson:"last_login,omitempty" json:"last_login,omitempty"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at" json:"updated_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      Role   `json:"role"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	Exp      int64  `json:"exp"`
}

// IsValidRole checks if a role is valid
func IsValidRole(role Role) bool {
	switch role {
	case RoleAdmin, RoleManager, RoleOperator, RoleViewer:
		return true
	default:
		return false
	}
}

// HasPermission checks if a user has permission for a specific action
func (u *User) HasPermission(action string) bool {
	switch u.Role {
	case RoleAdmin:
		return true
	case RoleManager:
		return action != "delete_user" && action != "manage_users"
	case RoleOperator:
		return action == "view_telemetry" || action == "view_vehicles" ||
			action == "create_trip" || action == "update_trip" ||
			action == "create_maintenance" || action == "update_maintenance"
	case RoleViewer:
		return action == "view_telemetry" || action == "view_vehicles" ||
			action == "view_trips" || action == "view_maintenance" ||
			action == "view_costs"
	default:
		return false
	}
} 