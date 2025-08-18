package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func TestAuthenticationFlow(t *testing.T) {
	// Create test server with authentication enabled
	config := &ServerConfig{
		Host:          "localhost",
		Port:          "8080",
		DatabasePath:  "",
		CORSOrigins:   []string{"*"},
		JWTSecret:     "test-secret-key",
		LogLevel:      "info",
		RequireAuth:   true,
		TokenDuration: 1, // 1 hour
		RateLimit: RateLimitConfig{
			Enabled:        false, // Disable for auth tests
			RequestsPerMin: 60,
			BurstSize:      15,
			WindowSize:     time.Minute,
		},
	}

	appriseInstance := apprise.New()
	server, err := NewServer(config, appriseInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("Login with valid credentials", func(t *testing.T) {
		loginData := LoginRequest{
			Username: "admin",
			Password: "admin",
		}

		body, _ := json.Marshal(loginData)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected successful login")
		}

		// Check if response contains token
		if response.Data == nil {
			t.Error("Expected login response data")
		}
	})

	t.Run("Login with invalid credentials", func(t *testing.T) {
		loginData := LoginRequest{
			Username: "admin",
			Password: "wrong",
		}

		body, _ := json.Marshal(loginData)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Protected endpoint without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Protected endpoint with valid token", func(t *testing.T) {
		// First login to get token
		loginData := LoginRequest{
			Username: "admin",
			Password: "admin",
		}

		body, _ := json.Marshal(loginData)
		loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		loginReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		server.router.ServeHTTP(loginW, loginReq)

		var loginResponse APIResponse
		json.NewDecoder(loginW.Body).Decode(&loginResponse)

		loginData2, ok := loginResponse.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Invalid login response format")
		}

		token, ok := loginData2["token"].(string)
		if !ok {
			t.Fatal("No token in login response")
		}

		// Now use token to access protected endpoint
		req := httptest.NewRequest("GET", "/api/v1/services", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with valid token, got %d", w.Code)
		}
	})

	t.Run("Public endpoints accessible without auth", func(t *testing.T) {
		publicEndpoints := []string{
			"/health",
			"/version",
			"/dashboard",
		}

		for _, endpoint := range publicEndpoints {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Errorf("Public endpoint %s should not require auth", endpoint)
			}
		}
	})
}

func TestJWTTokenValidation(t *testing.T) {
	config := &ServerConfig{
		JWTSecret:     "test-secret-key",
		TokenDuration: 1,
	}

	server := &Server{config: config}

	user := &User{
		ID:       "test-user",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	t.Run("Create and validate token", func(t *testing.T) {
		token, err := server.CreateToken(user)
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		claims, err := server.ValidateToken(token)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}

		if claims.UserID != user.ID {
			t.Errorf("Expected user ID %s, got %s", user.ID, claims.UserID)
		}

		if claims.Username != user.Username {
			t.Errorf("Expected username %s, got %s", user.Username, claims.Username)
		}
	})

	t.Run("Validate invalid token", func(t *testing.T) {
		invalidToken := "invalid.token.here"

		_, err := server.ValidateToken(invalidToken)
		if err == nil {
			t.Error("Expected error for invalid token")
		}
	})
}

func TestAPIKeyAuthentication(t *testing.T) {
	config := &ServerConfig{
		RequireAuth: true,
		JWTSecret:   "test-secret",
	}

	server := &Server{config: config}

	t.Run("Valid API key authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/services", nil)
		req.Header.Set("X-API-Key", "ak_development_admin_key_do_not_use_in_production")

		// Simulate middleware execution
		user, err := server.validateAPIKey("ak_development_admin_key_do_not_use_in_production")
		if err != nil {
			t.Fatalf("Expected valid API key to work: %v", err)
		}

		if user.Username != "admin" {
			t.Errorf("Expected admin user, got %s", user.Username)
		}
	})

	t.Run("Invalid API key authentication", func(t *testing.T) {
		_, err := server.validateAPIKey("invalid-api-key")
		if err == nil {
			t.Error("Expected error for invalid API key")
		}
	})
}

func TestUserContext(t *testing.T) {
	user := &User{
		ID:       "test-user",
		Username: "testuser",
	}

	ctx := context.WithValue(context.Background(), userContextKey, user)

	retrievedUser, ok := GetUserFromContext(ctx)
	if !ok {
		t.Error("Expected to retrieve user from context")
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key := GenerateAPIKey()

	if key == "" {
		t.Error("Expected non-empty API key")
	}

	if len(key) < 10 {
		t.Error("Expected API key to be at least 10 characters long")
	}

	if key[:3] != "ak_" {
		t.Error("Expected API key to start with 'ak_'")
	}

	// Generate another key to ensure they're different
	key2 := GenerateAPIKey()
	if key == key2 {
		t.Error("Expected different API keys on each generation")
	}
}