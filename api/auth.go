package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims structure
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// User represents an API user
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	APIKey   string   `json:"api_key,omitempty"`
	Enabled  bool     `json:"enabled"`
	Created  time.Time `json:"created"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled       bool   `json:"enabled"`
	JWTSecret     string `json:"jwt_secret"`
	TokenDuration int    `json:"token_duration"` // hours
	RequireAuth   bool   `json:"require_auth"`
}

// contextKey for storing user in request context
type contextKey string

const userContextKey contextKey = "user"

// GenerateAPIKey generates a secure API key
func GenerateAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return "ak_" + hex.EncodeToString(bytes)
}

// CreateToken creates a JWT token for a user
func (s *Server) CreateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(time.Duration(s.config.TokenDuration) * time.Hour)
	if s.config.TokenDuration == 0 {
		expirationTime = time.Now().Add(24 * time.Hour) // default 24 hours
	}

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   user.Username,
			Issuer:    "apprise-go-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateToken validates and parses a JWT token
func (s *Server) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AuthMiddleware provides JWT-based authentication
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for public endpoints
		if s.isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Skip if auth is disabled
		if !s.config.RequireAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header or API key
		token := s.extractToken(r)
		if token == "" {
			s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
			return
		}

		// Validate JWT token
		if strings.HasPrefix(token, "ak_") {
			// API Key authentication
			user, err := s.validateAPIKey(token)
			if err != nil {
				s.sendError(w, http.StatusUnauthorized, "Invalid API key", err)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// JWT token authentication
		claims, err := s.ValidateToken(token)
		if err != nil {
			s.sendError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		// Create user from claims
		user := &User{
			ID:       claims.UserID,
			Username: claims.Username,
			Roles:    claims.Roles,
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicEndpoint checks if an endpoint should be publicly accessible
func (s *Server) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/version",
		"/docs",
		"/dashboard",
		"/dashboard/",
		"/dashboard.html",
		"/dashboard.js",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint || strings.HasPrefix(path, "/dashboard/") {
			return true
		}
	}
	return false
}

// extractToken extracts token from Authorization header or API key header
func (s *Server) extractToken(r *http.Request) string {
	// Check Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Check X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Check query parameter (for testing purposes)
	return r.URL.Query().Get("api_key")
}

// validateAPIKey validates an API key and returns the associated user
func (s *Server) validateAPIKey(apiKey string) (*User, error) {
	// In a real implementation, this would query a database
	// For now, return a mock admin user
	if apiKey == "ak_development_admin_key_do_not_use_in_production" {
		return &User{
			ID:       "admin",
			Username: "admin",
			Email:    "admin@localhost",
			Roles:    []string{"admin", "user"},
			Enabled:  true,
			Created:  time.Now(),
		}, nil
	}
	
	return nil, fmt.Errorf("invalid API key")
}

// GetUserFromContext extracts the user from the request context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

// RequireRole middleware to check if user has required role
func (s *Server) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				s.sendError(w, http.StatusUnauthorized, "Authentication required", nil)
				return
			}

			hasRole := false
			for _, userRole := range user.Roles {
				if userRole == role || userRole == "admin" {
					hasRole = true
					break
				}
			}

			if !hasRole {
				s.sendError(w, http.StatusForbidden, "Insufficient permissions", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}