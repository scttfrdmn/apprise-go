package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSendGridService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAPIKey     string
		expectedFromEmail  string
		expectedFromName   string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "sendgrid://api_key@host/to@example.com?from=sender@example.com",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedFromEmail:  "sender@example.com",
			expectedRecipients: []string{"to@example.com"},
		},
		{
			name:               "Valid with sender name",
			url:                "sendgrid://api_key@host/to@example.com?from=sender@example.com&name=John+Doe",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedFromEmail:  "sender@example.com",
			expectedFromName:   "John Doe",
			expectedRecipients: []string{"to@example.com"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "sendgrid://api_key@host/to1@example.com/to2@example.com?from=sender@example.com",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedFromEmail:  "sender@example.com",
			expectedRecipients: []string{"to1@example.com", "to2@example.com"},
		},
		{
			name:        "Missing API key",
			url:         "sendgrid://@host/to@example.com?from=sender@example.com",
			expectError: true,
		},
		{
			name:        "Missing from email",
			url:         "sendgrid://api_key@host/to@example.com",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "sendgrid://api_key@host?from=sender@example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSendGridService().(*SendGridService)
			
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

			if service.fromEmail != tt.expectedFromEmail {
				t.Errorf("Expected from email to be %s, got %s", tt.expectedFromEmail, service.fromEmail)
			}

			if service.fromName != tt.expectedFromName {
				t.Errorf("Expected from name to be %s, got %s", tt.expectedFromName, service.fromName)
			}

			if len(service.to) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.to))
			}

			for i, expected := range tt.expectedRecipients {
				if i < len(service.to) && service.to[i] != expected {
					t.Errorf("Expected recipient %d to be %s, got %s", i, expected, service.to[i])
				}
			}
		})
	}
}

func TestSendGridService_GetServiceID(t *testing.T) {
	service := NewSendGridService()
	if service.GetServiceID() != "sendgrid" {
		t.Errorf("Expected service ID 'sendgrid', got %s", service.GetServiceID())
	}
}

func TestSendGridService_GetDefaultPort(t *testing.T) {
	service := NewSendGridService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestSendGridService_SupportsAttachments(t *testing.T) {
	service := NewSendGridService()
	if !service.SupportsAttachments() {
		t.Error("SendGrid should support attachments")
	}
}

func TestSendGridService_GetMaxBodyLength(t *testing.T) {
	service := NewSendGridService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestSendGridService_Send(t *testing.T) {
	service := NewSendGridService().(*SendGridService)
	service.apiKey = "test_api_key"
	service.fromEmail = "from@example.com"
	service.fromName = "Test Sender"
	service.to = []string{"to@example.com"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Subject",
		Body:  "Test email body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid credentials/unreachable API
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "SendGrid") {
		t.Errorf("Expected error to mention SendGrid, got: %v", err)
	}
}