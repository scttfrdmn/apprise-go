package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func TestAPIServer_HealthEndpoint(t *testing.T) {
	// Create test server
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: "",
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	server, err := NewServer(config, appriseInstance, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true in health response")
	}
}

func TestAPIServer_NotifyEndpoint(t *testing.T) {
	// Create test server
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: "",
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	server, err := NewServer(config, appriseInstance, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test notification endpoint with webhook (safe for testing)
	notifyReq := NotificationRequest{
		URLs:  []string{"webhook://httpbin.org/post"},
		Title: "Test Notification",
		Body:  "This is a test notification from API server",
		Type:  "info",
	}

	jsonData, err := json.Marshal(notifyReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/notify", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true in notification response")
	}
}

func TestAPIServer_ServicesEndpoint(t *testing.T) {
	// Create test server
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: "",
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	server, err := NewServer(config, appriseInstance, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test services listing endpoint
	req := httptest.NewRequest("GET", "/api/v1/services", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true in services response")
	}

	// Check that we have services in the response
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	total, exists := data["total"]
	if !exists {
		t.Error("Expected 'total' field in services response")
	}

	if totalFloat, ok := total.(float64); !ok || totalFloat <= 0 {
		t.Errorf("Expected total > 0, got %v", total)
	}
}

func TestAPIServer_WithScheduler(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_api_scheduler.db")

	// Create test server with scheduler
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: dbPath,
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	scheduler, err := apprise.NewNotificationScheduler(dbPath, appriseInstance)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer scheduler.Close()

	server, err := NewServer(config, appriseInstance, scheduler, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test scheduler jobs endpoint
	req := httptest.NewRequest("GET", "/api/v1/scheduler/jobs", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true in scheduler jobs response")
	}
}

func TestAPIServer_ConfigEndpoint(t *testing.T) {
	// Create test server
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: "",
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	server, err := NewServer(config, appriseInstance, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test config endpoint
	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true in config response")
	}

	// Verify config structure - response.Data is interface{}, need to cast differently
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if version, exists := dataMap["version"]; !exists || version == "" {
		t.Error("Expected version to be set in config response")
	}

	if services, exists := dataMap["supported_services"]; !exists {
		t.Error("Expected services to be listed in config response")
	} else if servicesList, ok := services.([]interface{}); !ok || len(servicesList) == 0 {
		t.Error("Expected services to be a non-empty list")
	}

	if features, exists := dataMap["features"]; !exists {
		t.Error("Expected features to be set in config response")
	} else if featuresMap, ok := features.(map[string]interface{}); !ok || len(featuresMap) == 0 {
		t.Error("Expected features to be a non-empty map")
	}
}

func TestAPIServer_ErrorHandling(t *testing.T) {
	// Create test server
	config := &ServerConfig{
		Host:         "localhost",
		Port:         "8080",
		DatabasePath: "",
		CORSOrigins:  []string{"*"},
		JWTSecret:    "test-secret",
		LogLevel:     "info",
	}

	appriseInstance := apprise.New()
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	server, err := NewServer(config, appriseInstance, nil, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test invalid notification request
	invalidReq := map[string]interface{}{
		"body": "", // Empty body should cause error
	}

	jsonData, _ := json.Marshal(invalidReq)
	req := httptest.NewRequest("POST", "/api/v1/notify", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("Expected error status for invalid request")
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false for invalid request")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	secret := GenerateJWTSecret()
	if len(secret) < 32 {
		t.Errorf("Expected JWT secret to be at least 32 characters, got %d", len(secret))
	}
}

// Helper function for timing tests
func timeTrack(start time.Time, name string, t *testing.T) {
	elapsed := time.Since(start)
	t.Logf("%s took %v", name, elapsed)
}