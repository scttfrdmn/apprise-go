package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token     string    `json:"token"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles,omitempty"`
}

// APIKeyCreateRequest represents an API key creation request
type APIKeyCreateRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents an API key response
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Key         string     `json:"key,omitempty"` // Only returned on creation
	ExpiresAt   *time.Time `json:"expires_at"`
	Created     time.Time  `json:"created"`
	LastUsed    *time.Time `json:"last_used"`
}

// handleLogin processes user login requests
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Username == "" || req.Password == "" {
		s.sendError(w, http.StatusBadRequest, "Username and password are required", nil)
		return
	}

	// Authenticate user (mock implementation)
	user, err := s.authenticateUser(req.Username, req.Password)
	if err != nil {
		s.sendError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	// Create JWT token
	token, err := s.CreateToken(user)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create token", err)
		return
	}

	// Calculate expiration time
	tokenDuration := time.Duration(s.config.TokenDuration) * time.Hour
	if tokenDuration == 0 {
		tokenDuration = 24 * time.Hour
	}
	expiresAt := time.Now().Add(tokenDuration)

	// Update user's last seen time
	now := time.Now()
	user.LastSeen = &now

	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}

	s.sendSuccess(w, "Login successful", response)
}

// handleRegister processes user registration requests
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		s.sendError(w, http.StatusBadRequest, "Username, email, and password are required", nil)
		return
	}

	// Check if user already exists (mock implementation)
	if s.userExists(req.Username) {
		s.sendError(w, http.StatusConflict, "Username already exists", nil)
		return
	}

	// Create user (mock implementation)
	user := &User{
		ID:       generateUserID(),
		Username: req.Username,
		Email:    req.Email,
		Roles:    req.Roles,
		Enabled:  true,
		Created:  time.Now(),
	}

	// In a real implementation, hash the password and store user in database
	if err := s.createUser(user, req.Password); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	// Remove sensitive information before sending response
	user.APIKey = ""
	
	s.sendSuccess(w, "User registered successfully", user)
}

// handleWhoAmI returns current user information
func (s *Server) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}

	s.sendSuccess(w, "Current user information", user)
}

// handleRefreshToken refreshes an existing JWT token
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}

	// Create new JWT token
	token, err := s.CreateToken(user)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create token", err)
		return
	}

	// Calculate expiration time
	tokenDuration := time.Duration(s.config.TokenDuration) * time.Hour
	if tokenDuration == 0 {
		tokenDuration = 24 * time.Hour
	}
	expiresAt := time.Now().Add(tokenDuration)

	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}

	s.sendSuccess(w, "Token refreshed successfully", response)
}

// handleCreateAPIKey creates a new API key for the current user
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}

	var req APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Name == "" {
		s.sendError(w, http.StatusBadRequest, "API key name is required", nil)
		return
	}

	// Generate API key
	apiKey := GenerateAPIKey()
	if apiKey == "" {
		s.sendError(w, http.StatusInternalServerError, "Failed to generate API key", nil)
		return
	}

	// Create API key record (mock implementation)
	keyRecord := &APIKeyResponse{
		ID:          generateKeyID(),
		Name:        req.Name,
		Description: req.Description,
		Key:         apiKey,
		ExpiresAt:   req.ExpiresAt,
		Created:     time.Now(),
	}

	// Store API key (mock implementation)
	if err := s.storeAPIKey(user.ID, keyRecord); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to store API key", err)
		return
	}

	s.sendSuccess(w, "API key created successfully", keyRecord)
}

// handleListAPIKeys lists all API keys for the current user
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}

	// Get API keys for user (mock implementation)
	keys, err := s.getUserAPIKeys(user.ID)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to retrieve API keys", err)
		return
	}

	// Remove sensitive key data from response
	for i := range keys {
		keys[i].Key = ""
	}

	s.sendSuccess(w, "API keys retrieved successfully", keys)
}

// handleDeleteAPIKey deletes an API key
func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r.Context())
	if !ok {
		s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
		return
	}

	vars := mux.Vars(r)
	keyID := vars["key_id"]
	if keyID == "" {
		s.sendError(w, http.StatusBadRequest, "API key ID is required", nil)
		return
	}

	// Delete API key (mock implementation)
	if err := s.deleteAPIKey(user.ID, keyID); err != nil {
		s.sendError(w, http.StatusNotFound, "API key not found", err)
		return
	}

	s.sendSuccess(w, "API key deleted successfully", nil)
}

// handleGetRateLimitStatus returns rate limit status for current user
func (s *Server) handleGetRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	if s.rateLimiter == nil {
		s.sendSuccess(w, "Rate limiting disabled", map[string]interface{}{"enabled": false})
		return
	}

	clientID := s.getClientID(r)
	status := s.rateLimiter.GetRateLimitStatus(clientID)
	
	s.sendSuccess(w, "Rate limit status", status)
}

// Mock authentication functions (replace with real database operations)

func (s *Server) authenticateUser(username, password string) (*User, error) {
	// Mock implementation - in production, verify against database with hashed passwords
	if username == "admin" && password == "admin" {
		return &User{
			ID:       "admin",
			Username: "admin",
			Email:    "admin@localhost",
			Roles:    []string{"admin", "user"},
			Enabled:  true,
			Created:  time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("invalid credentials")
}

func (s *Server) userExists(username string) bool {
	// Mock implementation
	return username == "admin"
}

func (s *Server) createUser(user *User, password string) error {
	// Mock implementation - in production, hash password and store in database
	return nil
}

func (s *Server) storeAPIKey(userID string, key *APIKeyResponse) error {
	// Mock implementation
	return nil
}

func (s *Server) getUserAPIKeys(userID string) ([]*APIKeyResponse, error) {
	// Mock implementation
	return []*APIKeyResponse{}, nil
}

func (s *Server) deleteAPIKey(userID, keyID string) error {
	// Mock implementation
	return nil
}

func generateUserID() string {
	return "user_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func generateKeyID() string {
	return "key_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}