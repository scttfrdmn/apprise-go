package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMessageBirdService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedRecipients []string
		expectedOriginator string
	}{
		{
			name:               "Valid single recipient",
			url:                "messagebird://api_key@+1234567890",
			expectError:        false,
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid with originator parameter",
			url:                "messagebird://api_key@+1234567890?from=Company",
			expectError:        false,
			expectedRecipients: []string{"+1234567890"},
			expectedOriginator: "Company",
		},
		{
			name:               "Multiple recipients",
			url:                "messagebird://api_key@+1234567890/+0987654321",
			expectError:        false,
			expectedRecipients: []string{"+1234567890", "+0987654321"},
			expectedOriginator: "", // No originator parameter in this test
		},
		{
			name:        "Missing API key",
			url:         "messagebird://@+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "messagebird://api_key@",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMessageBirdService().(*MessageBirdService)
			
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

			if len(service.recipients) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.recipients))
			}

			if service.originator != tt.expectedOriginator {
				t.Errorf("Expected originator to be %s, got %s", tt.expectedOriginator, service.originator)
			}
		})
	}
}

func TestMessageBirdService_GetServiceID(t *testing.T) {
	service := NewMessageBirdService()
	if service.GetServiceID() != "messagebird" {
		t.Errorf("Expected service ID 'messagebird', got %s", service.GetServiceID())
	}
}

func TestMessageBirdService_SupportsAttachments(t *testing.T) {
	service := NewMessageBirdService()
	if service.SupportsAttachments() {
		t.Error("MessageBird should not support attachments")
	}
}

func TestMessageBirdService_GetMaxBodyLength(t *testing.T) {
	service := NewMessageBirdService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestMessageBirdService_Send(t *testing.T) {
	service := NewMessageBirdService().(*MessageBirdService)
	service.apiKey = "test-api-key"
	service.originator = "TestSender"
	service.recipients = []string{"+1234567890"}

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

	if err != nil && !strings.Contains(err.Error(), "MessageBird") {
		t.Errorf("Expected error to mention MessageBird, got: %v", err)
	}
}