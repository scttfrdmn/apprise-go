package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPlivoService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAuthID     string
		expectedAuthToken  string
		expectedFrom       string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "plivo://auth_id:auth_token@host/+1234567890",
			expectError:        false,
			expectedAuthID:     "auth_id",
			expectedAuthToken:  "auth_token",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid with from parameter",
			url:                "plivo://auth_id:auth_token@host/+1234567890?from=Company",
			expectError:        false,
			expectedAuthID:     "auth_id",
			expectedAuthToken:  "auth_token",
			expectedFrom:       "Company",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "plivo://auth_id:auth_token@host/+1234567890/+0987654321",
			expectError:        false,
			expectedAuthID:     "auth_id",
			expectedAuthToken:  "auth_token",
			expectedRecipients: []string{"+1234567890", "+0987654321"},
		},
		{
			name:        "Missing Auth ID",
			url:         "plivo://@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing Auth Token",
			url:         "plivo://auth_id@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "plivo://auth_id:auth_token@host",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPlivoService().(*PlivoService)
			
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

			if service.authID != tt.expectedAuthID {
				t.Errorf("Expected Auth ID to be %s, got %s", tt.expectedAuthID, service.authID)
			}

			if service.authToken != tt.expectedAuthToken {
				t.Errorf("Expected Auth Token to be %s, got %s", tt.expectedAuthToken, service.authToken)
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

func TestPlivoService_GetServiceID(t *testing.T) {
	service := NewPlivoService()
	if service.GetServiceID() != "plivo" {
		t.Errorf("Expected service ID 'plivo', got %s", service.GetServiceID())
	}
}

func TestPlivoService_GetDefaultPort(t *testing.T) {
	service := NewPlivoService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestPlivoService_SupportsAttachments(t *testing.T) {
	service := NewPlivoService()
	if service.SupportsAttachments() {
		t.Error("Plivo should not support attachments")
	}
}

func TestPlivoService_GetMaxBodyLength(t *testing.T) {
	service := NewPlivoService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestPlivoService_Send(t *testing.T) {
	service := NewPlivoService().(*PlivoService)
	service.authID = "test_id"
	service.authToken = "test_token"
	service.from = "TestSender"
	service.to = []string{"+1234567890"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid credentials/unreachable API
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Plivo") {
		t.Errorf("Expected error to mention Plivo, got: %v", err)
	}
}