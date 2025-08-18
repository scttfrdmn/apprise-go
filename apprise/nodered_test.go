package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNodeREDService_ParseURL(t *testing.T) {

	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedBaseURL   string
		expectedPath      string
	}{
		{
			name:            "Valid Node-RED URL",
			url:             "nodered://localhost:1880/webhook",
			expectError:     false,
			expectedBaseURL: "http://localhost:1880",
			expectedPath:    "/webhook",
		},
		{
			name:            "Valid HTTPS Node-RED URL",
			url:             "nodered+https://nodered.example.com:443/api/webhook",
			expectError:     false,
			expectedBaseURL: "https://nodered.example.com:443",
			expectedPath:    "/api/webhook",
		},
		{
			name:            "Default port and path",
			url:             "nodered://localhost",
			expectError:     false,
			expectedBaseURL: "http://localhost:1880",
			expectedPath:    "/webhook",
		},
		{
			name:            "Custom port with default path",
			url:             "nodered://localhost:8080",
			expectError:     false,
			expectedBaseURL: "http://localhost:8080",
			expectedPath:    "/webhook",
		},
		{
			name:        "Missing host",
			url:         "nodered:///webhook",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNodeREDService().(*NodeREDService)
			
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

			if service.baseURL != tt.expectedBaseURL {
				t.Errorf("Expected base URL to be %s, got %s", tt.expectedBaseURL, service.baseURL)
			}

			if service.path != tt.expectedPath {
				t.Errorf("Expected path to be %s, got %s", tt.expectedPath, service.path)
			}
		})
	}
}

func TestNodeREDService_GetServiceID(t *testing.T) {
	service := NewNodeREDService()
	if service.GetServiceID() != "nodered" {
		t.Errorf("Expected service ID 'nodered', got %s", service.GetServiceID())
	}
}

func TestNodeREDService_GetDefaultPort(t *testing.T) {
	service := NewNodeREDService()
	if service.GetDefaultPort() != 1880 {
		t.Errorf("Expected default port 1880, got %d", service.GetDefaultPort())
	}
}

func TestNodeREDService_SupportsAttachments(t *testing.T) {
	service := NewNodeREDService()
	if service.SupportsAttachments() {
		t.Error("Node-RED should not support attachments")
	}
}

func TestNodeREDService_GetMaxBodyLength(t *testing.T) {
	service := NewNodeREDService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestNodeREDService_Send(t *testing.T) {
	service := NewNodeREDService().(*NodeREDService)
	service.baseURL = "http://localhost:1880"
	service.path = "/webhook"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Flow Trigger",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to unreachable Node-RED instance
	if err == nil {
		t.Error("Expected error due to unreachable Node-RED instance, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Node-RED") {
		t.Errorf("Expected error to mention Node-RED, got: %v", err)
	}
}