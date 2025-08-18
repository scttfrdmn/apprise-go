package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTextMagicService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedUsername   string
		expectedAPIKey     string
		expectedFrom       string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "textmagic://username:api_key@host/+1234567890",
			expectError:        false,
			expectedUsername:   "username",
			expectedAPIKey:     "api_key",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid with from parameter",
			url:                "textmagic://username:api_key@host/+1234567890?from=Company",
			expectError:        false,
			expectedUsername:   "username",
			expectedAPIKey:     "api_key",
			expectedFrom:       "Company",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "textmagic://username:api_key@host/+1234567890/+0987654321",
			expectError:        false,
			expectedUsername:   "username",
			expectedAPIKey:     "api_key",
			expectedRecipients: []string{"+1234567890", "+0987654321"},
		},
		{
			name:        "Missing username",
			url:         "textmagic://@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "textmagic://username@host/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "textmagic://username:api_key@host",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTextMagicService().(*TextMagicService)
			
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

			if service.username != tt.expectedUsername {
				t.Errorf("Expected username to be %s, got %s", tt.expectedUsername, service.username)
			}

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key to be %s, got %s", tt.expectedAPIKey, service.apiKey)
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

func TestTextMagicService_GetServiceID(t *testing.T) {
	service := NewTextMagicService()
	if service.GetServiceID() != "textmagic" {
		t.Errorf("Expected service ID 'textmagic', got %s", service.GetServiceID())
	}
}

func TestTextMagicService_GetDefaultPort(t *testing.T) {
	service := NewTextMagicService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestTextMagicService_SupportsAttachments(t *testing.T) {
	service := NewTextMagicService()
	if service.SupportsAttachments() {
		t.Error("TextMagic should not support attachments")
	}
}

func TestTextMagicService_GetMaxBodyLength(t *testing.T) {
	service := NewTextMagicService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestTextMagicService_Send(t *testing.T) {
	service := NewTextMagicService().(*TextMagicService)
	service.username = "test_user"
	service.apiKey = "test_key"
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

	if err != nil && !strings.Contains(err.Error(), "TextMagic") {
		t.Errorf("Expected error to mention TextMagic, got: %v", err)
	}
}