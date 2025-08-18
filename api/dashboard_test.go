package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scttfrdmn/apprise-go/apprise"
)

func TestDashboard_ServeHTML(t *testing.T) {
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
	server, err := NewServer(config, appriseInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test dashboard endpoint
	req := httptest.NewRequest("GET", "/dashboard", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected HTML content type, got %s", contentType)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected HTML content, got empty body")
	}

	// Check for essential HTML elements
	if !contains(body, "<html") {
		t.Error("Expected HTML document")
	}

	if !contains(body, "Apprise-Go Dashboard") {
		t.Error("Expected dashboard title")
	}
}

func TestDashboard_ServeJS(t *testing.T) {
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
	server, err := NewServer(config, appriseInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test JavaScript file endpoint
	req := httptest.NewRequest("GET", "/dashboard.js", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/javascript; charset=utf-8" {
		t.Errorf("Expected JavaScript content type, got %s", contentType)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected JavaScript content, got empty body")
	}

	// Check for essential JS content
	if !contains(body, "dashboard") {
		t.Error("Expected dashboard JavaScript content")
	}
}

func TestDashboard_PathTraversalProtection(t *testing.T) {
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
	server, err := NewServer(config, appriseInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test path traversal attempt
	req := httptest.NewRequest("GET", "/dashboard/../../../etc/passwd", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// The Gorilla Mux router will handle this and return a redirect to /dashboard/
	// So we expect either a 400 (from our handler) or a redirect
	if w.Code != http.StatusBadRequest && w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status 400 or 301 for path traversal, got %d", w.Code)
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/test.html", "text/html; charset=utf-8"},
		{"/script.js", "application/javascript; charset=utf-8"},
		{"/style.css", "text/css; charset=utf-8"},
		{"/data.json", "application/json; charset=utf-8"},
		{"/image.png", "image/png"},
		{"/image.jpg", "image/jpeg"},
		{"/unknown.txt", "text/plain; charset=utf-8"},
	}

	for _, test := range tests {
		result := getContentType(test.path)
		if result != test.expected {
			t.Errorf("getContentType(%s) = %s, expected %s", test.path, result, test.expected)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) && 
		   func() bool {
			   for i := 0; i <= len(s)-len(substr); i++ {
				   if s[i:i+len(substr)] == substr {
					   return true
				   }
			   }
			   return false
		   }()
}