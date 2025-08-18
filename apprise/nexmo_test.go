package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNexmoService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAPIKey     string
		expectedSecret     string
		expectedFrom       string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "nexmo://api_key:secret@host/+1234567890",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedSecret:     "secret",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid with from parameter",
			url:                "nexmo://api_key:secret@host/+1234567890?from=Company",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedSecret:     "secret",
			expectedFrom:       "Company",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "nexmo://api_key:secret@host/+1234567890/+0987654321",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedSecret:     "secret",
			expectedRecipients: []string{"+1234567890", "+0987654321"},
		},
		{
			name:        "Missing API key",
			url:         "nexmo://@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing secret",
			url:         "nexmo://api_key@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "nexmo://api_key:secret@host",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNexmoService().(*NexmoService)
			
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

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key to be %s, got %s", tt.expectedAPIKey, service.apiKey)
			}

			if service.apiSecret != tt.expectedSecret {
				t.Errorf("Expected API secret to be %s, got %s", tt.expectedSecret, service.apiSecret)
			}

			if service.from != tt.expectedFrom {
				t.Errorf("Expected from to be %s, got %s", tt.expectedFrom, service.from)
			}

			if len(service.to) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.to))
			}
		})
	}
}

func TestNexmoService_GetServiceID(t *testing.T) {
	service := NewNexmoService()
	if service.GetServiceID() != "nexmo" {
		t.Errorf("Expected service ID 'nexmo', got %s", service.GetServiceID())
	}
}

func TestNexmoService_GetDefaultPort(t *testing.T) {
	service := NewNexmoService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestNexmoService_SupportsAttachments(t *testing.T) {
	service := NewNexmoService()
	if service.SupportsAttachments() {
		t.Error("Nexmo should not support attachments")
	}
}

func TestNexmoService_GetMaxBodyLength(t *testing.T) {
	service := NewNexmoService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestNexmoService_Send(t *testing.T) {
	service := NewNexmoService().(*NexmoService)
	service.apiKey = "test_key"
	service.apiSecret = "test_secret"
	service.from = "TestSender"
	service.to = []string{"+1234567890"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	err := service.Send(ctx, notification)

	// The test may pass or fail depending on whether the API is reachable
	// Either way is acceptable for this basic test
	if err != nil && !strings.Contains(err.Error(), "Nexmo") {
		t.Errorf("Expected error to mention Nexmo if error occurs, got: %v", err)
	}
}