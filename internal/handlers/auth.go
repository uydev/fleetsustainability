package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ukydev/fleet-sustainability/internal/auth"
	"github.com/ukydev/fleet-sustainability/internal/db"
	"github.com/ukydev/fleet-sustainability/internal/middleware"
	"github.com/ukydev/fleet-sustainability/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService    *auth.Service
	userCollection db.UserCollection
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.Service, userCollection db.UserCollection) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		userCollection: userCollection,
	}
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var loginReq models.LoginRequest
	if err := json.Unmarshal(body, &loginReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if loginReq.Username == "" || loginReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Find user by username
	user, err := h.userCollection.FindUserByUsername(r.Context(), loginReq.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check if user is active
	if !user.IsActive {
		http.Error(w, "Account is deactivated", http.StatusUnauthorized)
		return
	}

	// Verify password
	if !h.authService.CheckPassword(loginReq.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate tokens
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Update last login
	err = h.userCollection.UpdateLastLogin(r.Context(), user.ID.Hex())
	if err != nil {
		// Log error but don't fail the login
		// log.WithError(err).Error("Failed to update last login")
	}

	// Create response
	response := models.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var registerReq models.RegisterRequest
	if err := json.Unmarshal(body, &registerReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if err := h.authService.ValidateUsername(registerReq.Username); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.authService.ValidateEmail(registerReq.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.authService.ValidatePassword(registerReq.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !models.IsValidRole(registerReq.Role) {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	_, err = h.userCollection.FindUserByUsername(r.Context(), registerReq.Username)
	if err == nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Check if email already exists
	_, err = h.userCollection.FindUserByEmail(r.Context(), registerReq.Email)
	if err == nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(registerReq.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create user
	user := models.User{
		ID:           primitive.NewObjectID(),
		Username:     registerReq.Username,
		Email:        registerReq.Email,
		PasswordHash: passwordHash,
		Role:         registerReq.Role,
		FirstName:    registerReq.FirstName,
		LastName:     registerReq.LastName,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save user to database
	err = h.userCollection.InsertUser(r.Context(), user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate tokens
	token, err := h.authService.GenerateToken(&user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := h.authService.GenerateRefreshToken()
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Create response
	response := models.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User context not found", http.StatusUnauthorized)
		return
	}

	user, err := h.userCollection.FindUserByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User context not found", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var updateReq struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}

	if err := json.Unmarshal(body, &updateReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get current user
	user, err := h.userCollection.FindUserByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if updateReq.FirstName != "" {
		user.FirstName = updateReq.FirstName
	}
	if updateReq.LastName != "" {
		user.LastName = updateReq.LastName
	}
	if updateReq.Email != "" {
		// Validate email
		if err := h.authService.ValidateEmail(updateReq.Email); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Check if email is already taken by another user
		existingUser, err := h.userCollection.FindUserByEmail(r.Context(), updateReq.Email)
		if err == nil && existingUser.ID.Hex() != claims.UserID {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		user.Email = updateReq.Email
	}

	// Update user
	err = h.userCollection.UpdateUser(r.Context(), claims.UserID, *user)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
}

// ChangePassword changes the current user's password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User context not found", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var passwordReq struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.Unmarshal(body, &passwordReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if passwordReq.CurrentPassword == "" || passwordReq.NewPassword == "" {
		http.Error(w, "Current password and new password are required", http.StatusBadRequest)
		return
	}

	// Validate new password
	if err := h.authService.ValidatePassword(passwordReq.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current user
	user, err := h.userCollection.FindUserByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	if !h.authService.CheckPassword(passwordReq.CurrentPassword, user.PasswordHash) {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash new password
	newPasswordHash, err := h.authService.HashPassword(passwordReq.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update password
	user.PasswordHash = newPasswordHash
	err = h.userCollection.UpdateUser(r.Context(), claims.UserID, *user)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}
