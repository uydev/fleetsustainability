package models

import (
	"testing"
	"time"
)

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{"admin role", RoleAdmin, true},
		{"manager role", RoleManager, true},
		{"operator role", RoleOperator, true},
		{"viewer role", RoleViewer, true},
		{"invalid role", "invalid", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidRole(tt.role)
			if result != tt.expected {
				t.Errorf("IsValidRole(%s) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestUser_HasPermission(t *testing.T) {
	admin := &User{Role: RoleAdmin}
	manager := &User{Role: RoleManager}
	operator := &User{Role: RoleOperator}
	viewer := &User{Role: RoleViewer}

	tests := []struct {
		name     string
		user     *User
		action   string
		expected bool
	}{
		// Admin permissions - should have all permissions
		{"admin can delete user", admin, "delete_user", true},
		{"admin can manage users", admin, "manage_users", true},
		{"admin can view telemetry", admin, "view_telemetry", true},
		{"admin can create trip", admin, "create_trip", true},

		// Manager permissions - can do most things except user management
		{"manager cannot delete user", manager, "delete_user", false},
		{"manager cannot manage users", manager, "manage_users", false},
		{"manager can view telemetry", manager, "view_telemetry", true},
		{"manager can create trip", manager, "create_trip", true},

		// Operator permissions - limited to operational tasks
		{"operator can view telemetry", operator, "view_telemetry", true},
		{"operator can view vehicles", operator, "view_vehicles", true},
		{"operator can create trip", operator, "create_trip", true},
		{"operator can update trip", operator, "update_trip", true},
		{"operator can create maintenance", operator, "create_maintenance", true},
		{"operator can update maintenance", operator, "update_maintenance", true},
		{"operator cannot delete user", operator, "delete_user", false},
		{"operator cannot manage users", operator, "manage_users", false},

		// Viewer permissions - read-only access
		{"viewer can view telemetry", viewer, "view_telemetry", true},
		{"viewer can view vehicles", viewer, "view_vehicles", true},
		{"viewer can view trips", viewer, "view_trips", true},
		{"viewer can view maintenance", viewer, "view_maintenance", true},
		{"viewer can view costs", viewer, "view_costs", true},
		{"viewer cannot create trip", viewer, "create_trip", false},
		{"viewer cannot update trip", viewer, "update_trip", false},
		{"viewer cannot delete user", viewer, "delete_user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.HasPermission(tt.action)
			if result != tt.expected {
				t.Errorf("User with role %s HasPermission(%s) = %v, want %v", 
					tt.user.Role, tt.action, result, tt.expected)
			}
		})
	}
}

func TestUser_StructFields(t *testing.T) {
	now := time.Now()
	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         RoleAdmin,
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
		LastLogin:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Test that all fields are properly set
	if user.Username != "testuser" {
		t.Errorf("Expected Username to be 'testuser', got %s", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected Email to be 'test@example.com', got %s", user.Email)
	}
	if user.PasswordHash != "hashedpassword" {
		t.Errorf("Expected PasswordHash to be 'hashedpassword', got %s", user.PasswordHash)
	}
	if user.Role != RoleAdmin {
		t.Errorf("Expected Role to be RoleAdmin, got %s", user.Role)
	}
	if user.FirstName != "Test" {
		t.Errorf("Expected FirstName to be 'Test', got %s", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("Expected LastName to be 'User', got %s", user.LastName)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true, got %v", user.IsActive)
	}
	if user.LastLogin == nil {
		t.Errorf("Expected LastLogin to be set, got nil")
	}
	if user.CreatedAt != now {
		t.Errorf("Expected CreatedAt to be set, got %v", user.CreatedAt)
	}
	if user.UpdatedAt != now {
		t.Errorf("Expected UpdatedAt to be set, got %v", user.UpdatedAt)
	}
} 