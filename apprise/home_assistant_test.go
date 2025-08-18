package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHomeAssistantService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedToken      string
		expectedBaseURL    string
		expectedService    string
	}{
		{
			name:            "Valid Home Assistant URL",
			url:             "homeassistant://access_token@localhost:8123",
			expectError:     false,
			expectedToken:   "access_token",
			expectedBaseURL: "http://localhost:8123",
			expectedService: "persistent_notification.create",
		},
		{
			name:            "Valid HTTPS Home Assistant URL",
			url:             "homeassistant+https://access_token@ha.example.com",
			expectError:     false,
			expectedToken:   "access_token",
			expectedBaseURL: "https://ha.example.com:8123",
			expectedService: "persistent_notification.create",
		},
		{
			name:            "Valid with custom service",
			url:             "homeassistant://access_token@localhost:8123/notify/mobile_app",
			expectError:     false,
			expectedToken:   "access_token",
			expectedBaseURL: "http://localhost:8123",
			expectedService: "notify.mobile_app",
		},
		{
			name:        "Missing access token",
			url:         "homeassistant://@localhost:8123",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "homeassistant://access_token@",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewHomeAssistantService().(*HomeAssistantService)
			
			parsedURL, parseErr := url.Parse(tt.url)
			if parseErr != nil && !tt.expectError {
				t.Fatalf("URL parsing failed: %v", parseErr)
			}

			if parseErr != nil {
				return
			}

			err := service.ParseURL(parsedURL)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if service.accessToken != tt.expectedToken {
				t.Errorf("Expected access token to be %s, got %s", tt.expectedToken, service.accessToken)
			}

			if service.baseURL != tt.expectedBaseURL {
				t.Errorf("Expected base URL to be %s, got %s", tt.expectedBaseURL, service.baseURL)
			}

			if service.service != tt.expectedService {
				t.Errorf("Expected service to be %s, got %s", tt.expectedService, service.service)
			}
		})
	}
}

func TestHomeAssistantService_GetServiceID(t *testing.T) {
	service := NewHomeAssistantService()
	if service.GetServiceID() != "homeassistant" {
		t.Errorf("Expected service ID 'homeassistant', got %s", service.GetServiceID())
	}
}

func TestHomeAssistantService_GetDefaultPort(t *testing.T) {
	service := NewHomeAssistantService()
	if service.GetDefaultPort() != 8123 {
		t.Errorf("Expected default port 8123, got %d", service.GetDefaultPort())
	}
}

func TestHomeAssistantService_SupportsAttachments(t *testing.T) {
	service := NewHomeAssistantService()
	if service.SupportsAttachments() {
		t.Error("Home Assistant should not support attachments")
	}
}

func TestHomeAssistantService_GetMaxBodyLength(t *testing.T) {
	service := NewHomeAssistantService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestHomeAssistantService_Send(t *testing.T) {
	service := NewHomeAssistantService().(*HomeAssistantService)
	service.accessToken = "test_token"
	service.baseURL = "http://localhost:8123"
	service.service = "persistent_notification.create"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Notification",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to unreachable Home Assistant instance
	if err == nil {
		t.Error("Expected error due to unreachable Home Assistant instance, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Home Assistant") {
		t.Errorf("Expected error to mention Home Assistant, got: %v", err)
	}
}