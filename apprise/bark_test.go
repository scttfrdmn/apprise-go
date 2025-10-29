package apprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBarkService_GetServiceID(t *testing.T) {
	service := NewBarkService()
	if service.GetServiceID() != "bark" {
		t.Errorf("Expected service ID 'bark', got '%s'", service.GetServiceID())
	}
}

func TestBarkService_GetDefaultPort(t *testing.T) {
	tests := []struct {
		name         string
		secure       bool
		expectedPort int
	}{
		{"HTTP", false, 80},
		{"HTTPS", true, 443},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BarkService{secure: tt.secure}
			if service.GetDefaultPort() != tt.expectedPort {
				t.Errorf("Expected default port %d, got %d", tt.expectedPort, service.GetDefaultPort())
			}
		})
	}
}

func TestBarkService_SupportsAttachments(t *testing.T) {
	service := NewBarkService()
	if service.SupportsAttachments() {
		t.Error("Bark should not support attachments")
	}
}

func TestBarkService_GetMaxBodyLength(t *testing.T) {
	service := NewBarkService()
	expected := 4096
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestBarkService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedKey      string
		expectedHost     string
		expectedPort     int
		expectedSecure   bool
		expectedIcon     string
		expectedSound    string
		expectedBadge    int
		expectedURL      string
		expectedCategory string
		expectedGroup    string
	}{
		{
			name:           "Basic bark URL",
			url:            "bark://devicekey@api.day.app/",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedPort:   -1,
			expectedSecure: false,
		},
		{
			name:           "Secure bark URL",
			url:            "barks://devicekey@api.day.app/",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedPort:   -1,
			expectedSecure: true,
		},
		{
			name:           "With custom port",
			url:            "bark://devicekey@localhost:8080/",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "localhost",
			expectedPort:   8080,
			expectedSecure: false,
		},
		{
			name:           "With icon parameter",
			url:            "bark://devicekey@api.day.app/?icon=https://example.com/icon.png",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedIcon:   "https://example.com/icon.png",
			expectedSecure: false,
		},
		{
			name:           "With sound parameter",
			url:            "bark://devicekey@api.day.app/?sound=alarm",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedSound:  "alarm",
			expectedSecure: false,
		},
		{
			name:           "With badge parameter",
			url:            "bark://devicekey@api.day.app/?badge=5",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedBadge:  5,
			expectedSecure: false,
		},
		{
			name:           "With URL parameter",
			url:            "bark://devicekey@api.day.app/?url=https://example.com",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedURL:    "https://example.com",
			expectedSecure: false,
		},
		{
			name:             "With category and group",
			url:              "bark://devicekey@api.day.app/?category=news&group=alerts",
			expectError:      false,
			expectedKey:      "devicekey",
			expectedHost:     "api.day.app",
			expectedCategory: "news",
			expectedGroup:    "alerts",
			expectedSecure:   false,
		},
		{
			name:           "With all parameters",
			url:            "bark://devicekey@api.day.app/?icon=https://example.com/icon.png&sound=alarm&badge=3&url=https://example.com&category=news&group=alerts",
			expectError:    false,
			expectedKey:    "devicekey",
			expectedHost:   "api.day.app",
			expectedIcon:   "https://example.com/icon.png",
			expectedSound:  "alarm",
			expectedBadge:  3,
			expectedURL:    "https://example.com",
			expectedCategory: "news",
			expectedGroup:    "alerts",
			expectedSecure: false,
		},
		{
			name:        "Invalid scheme",
			url:         "http://devicekey@api.day.app/",
			expectError: true,
		},
		{
			name:        "Missing device key",
			url:         "bark://api.day.app/",
			expectError: true,
		},
		{
			name:        "Missing hostname",
			url:         "bark://devicekey@/",
			expectError: true,
		},
		{
			name:        "Invalid port",
			url:         "bark://devicekey@api.day.app:invalid/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewBarkService().(*BarkService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse URL: %v", err)
				}
				return
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.deviceKey != tt.expectedKey {
				t.Errorf("Expected device key '%s', got '%s'", tt.expectedKey, service.deviceKey)
			}

			if service.hostname != tt.expectedHost {
				t.Errorf("Expected hostname '%s', got '%s'", tt.expectedHost, service.hostname)
			}

			if tt.expectedPort != 0 && service.port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, service.port)
			}

			if service.secure != tt.expectedSecure {
				t.Errorf("Expected secure %v, got %v", tt.expectedSecure, service.secure)
			}

			if tt.expectedIcon != "" && service.icon != tt.expectedIcon {
				t.Errorf("Expected icon '%s', got '%s'", tt.expectedIcon, service.icon)
			}

			if tt.expectedSound != "" && service.sound != tt.expectedSound {
				t.Errorf("Expected sound '%s', got '%s'", tt.expectedSound, service.sound)
			}

			if tt.expectedBadge != 0 && service.badge != tt.expectedBadge {
				t.Errorf("Expected badge %d, got %d", tt.expectedBadge, service.badge)
			}

			if tt.expectedURL != "" && service.url != tt.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, service.url)
			}

			if tt.expectedCategory != "" && service.category != tt.expectedCategory {
				t.Errorf("Expected category '%s', got '%s'", tt.expectedCategory, service.category)
			}

			if tt.expectedGroup != "" && service.group != tt.expectedGroup {
				t.Errorf("Expected group '%s', got '%s'", tt.expectedGroup, service.group)
			}
		})
	}
}

func TestBarkService_TestURL(t *testing.T) {
	service := NewBarkService()

	validURLs := []string{
		"bark://devicekey@api.day.app/",
		"barks://devicekey@api.day.app/",
		"bark://devicekey@localhost:8080/",
		"bark://devicekey@api.day.app/?icon=https://example.com/icon.png",
	}

	for _, validURL := range validURLs {
		if err := service.TestURL(validURL); err != nil {
			t.Errorf("Valid URL %q should not error: %v", validURL, err)
		}
	}

	invalidURLs := []string{
		"http://devicekey@api.day.app/",
		"bark://api.day.app/",
		"not-a-url",
	}

	for _, invalidURL := range invalidURLs {
		if err := service.TestURL(invalidURL); err == nil {
			t.Errorf("Invalid URL %q should error", invalidURL)
		}
	}
}

func TestBarkService_Send(t *testing.T) {
	// Create mock Bark server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify User-Agent
		if !strings.Contains(r.Header.Get("User-Agent"), "Apprise-Go") {
			t.Errorf("Expected User-Agent to contain Apprise-Go, got %s", r.Header.Get("User-Agent"))
		}

		// Verify path
		if r.URL.Path != "/push" {
			t.Errorf("Expected path /push, got %s", r.URL.Path)
		}

		// Parse and verify request body
		var payload BarkPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload fields
		if payload.DeviceKey != "testkey" {
			t.Errorf("Expected device key 'testkey', got '%s'", payload.DeviceKey)
		}

		if payload.Title != "Test Alert" {
			t.Errorf("Expected title 'Test Alert', got '%s'", payload.Title)
		}

		if payload.Body != "This is a test notification" {
			t.Errorf("Expected body 'This is a test notification', got '%s'", payload.Body)
		}

		if payload.Icon != "https://example.com/icon.png" {
			t.Errorf("Expected icon 'https://example.com/icon.png', got '%s'", payload.Icon)
		}

		if payload.Sound != "alarm" {
			t.Errorf("Expected sound 'alarm', got '%s'", payload.Sound)
		}

		if payload.Badge != 5 {
			t.Errorf("Expected badge 5, got %d", payload.Badge)
		}

		// Send success response
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"code":    200,
			"message": "success",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	// Configure service
	service := NewBarkService().(*BarkService)
	service.deviceKey = "testkey"
	service.hostname = serverURL.Hostname()
	if portStr := serverURL.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil {
			service.port = port
		}
	}
	service.icon = "https://example.com/icon.png"
	service.sound = "alarm"
	service.badge = 5

	req := NotificationRequest{
		Title:      "Test Alert",
		Body:       "This is a test notification",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestBarkService_SendError(t *testing.T) {
	// Create mock Bark server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"code":    400,
			"message": "invalid device key",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Parse server URL
	serverURL, _ := url.Parse(server.URL)

	// Configure service
	service := NewBarkService().(*BarkService)
	service.deviceKey = "testkey"
	service.hostname = serverURL.Hostname()
	if portStr := serverURL.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil {
			service.port = port
		}
	}

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for Bark API error response")
	}

	if !strings.Contains(err.Error(), "bark API returned error code") {
		t.Errorf("Expected Bark API error message, got: %v", err)
	}
}
