package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMailgunService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAPIKey     string
		expectedDomain     string
		expectedFromEmail  string
		expectedFromName   string
		expectedRegion     string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "mailgun://api_key@example.com/to@example.com",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedDomain:     "example.com",
			expectedFromEmail:  "noreply@example.com",
			expectedRegion:     "us",
			expectedRecipients: []string{"to@example.com"},
		},
		{
			name:               "Valid with from email and name",
			url:                "mailgun://api_key@example.com/to@example.com?from=sender@example.com&name=John+Doe",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedDomain:     "example.com",
			expectedFromEmail:  "sender@example.com",
			expectedFromName:   "John Doe",
			expectedRegion:     "us",
			expectedRecipients: []string{"to@example.com"},
		},
		{
			name:               "Valid EU region",
			url:                "mailgun://api_key@example.com/to@example.com?region=eu",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedDomain:     "example.com",
			expectedFromEmail:  "noreply@example.com",
			expectedRegion:     "eu",
			expectedRecipients: []string{"to@example.com"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "mailgun://api_key@example.com/to1@example.com/to2@example.com",
			expectError:        false,
			expectedAPIKey:     "api_key",
			expectedDomain:     "example.com",
			expectedFromEmail:  "noreply@example.com",
			expectedRegion:     "us",
			expectedRecipients: []string{"to1@example.com", "to2@example.com"},
		},
		{
			name:        "Missing API key",
			url:         "mailgun://@example.com/to@example.com",
			expectError: true,
		},
		{
			name:        "Missing domain",
			url:         "mailgun://api_key@/to@example.com",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "mailgun://api_key@example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMailgunService().(*MailgunService)
			
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

			if service.domain != tt.expectedDomain {
				t.Errorf("Expected domain to be %s, got %s", tt.expectedDomain, service.domain)
			}

			if service.fromEmail != tt.expectedFromEmail {
				t.Errorf("Expected from email to be %s, got %s", tt.expectedFromEmail, service.fromEmail)
			}

			if service.fromName != tt.expectedFromName {
				t.Errorf("Expected from name to be %s, got %s", tt.expectedFromName, service.fromName)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region to be %s, got %s", tt.expectedRegion, service.region)
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

func TestMailgunService_GetServiceID(t *testing.T) {
	service := NewMailgunService()
	if service.GetServiceID() != "mailgun" {
		t.Errorf("Expected service ID 'mailgun', got %s", service.GetServiceID())
	}
}

func TestMailgunService_GetDefaultPort(t *testing.T) {
	service := NewMailgunService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestMailgunService_SupportsAttachments(t *testing.T) {
	service := NewMailgunService()
	if !service.SupportsAttachments() {
		t.Error("Mailgun should support attachments")
	}
}

func TestMailgunService_GetMaxBodyLength(t *testing.T) {
	service := NewMailgunService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestMailgunService_Send(t *testing.T) {
	service := NewMailgunService().(*MailgunService)
	service.apiKey = "test_api_key"
	service.domain = "example.com"
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

	if err != nil && !strings.Contains(err.Error(), "Mailgun") {
		t.Errorf("Expected error to mention Mailgun, got: %v", err)
	}
}