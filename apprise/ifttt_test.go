package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIFTTTService_ParseURL(t *testing.T) {

	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedWebhookKey string
		expectedEvent     string
	}{
		{
			name:               "Valid IFTTT URL",
			url:                "ifttt://webhook_key@test_event",
			expectError:        false,
			expectedWebhookKey: "webhook_key",
			expectedEvent:      "test_event",
		},
		{
			name:        "Missing webhook key",
			url:         "ifttt://@test_event",
			expectError: true,
		},
		{
			name:        "Missing event name",
			url:         "ifttt://webhook_key@",
			expectError: true,
		},
		{
			name:        "Invalid URL format",
			url:         "ifttt://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewIFTTTService().(*IFTTTService)
			
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

			if service.webhookKey != tt.expectedWebhookKey {
				t.Errorf("Expected webhook key to be %s, got %s", tt.expectedWebhookKey, service.webhookKey)
			}

			if service.event != tt.expectedEvent {
				t.Errorf("Expected event to be %s, got %s", tt.expectedEvent, service.event)
			}
		})
	}
}

func TestIFTTTService_GetServiceID(t *testing.T) {
	service := NewIFTTTService()
	if service.GetServiceID() != "ifttt" {
		t.Errorf("Expected service ID 'ifttt', got %s", service.GetServiceID())
	}
}

func TestIFTTTService_GetDefaultPort(t *testing.T) {
	service := NewIFTTTService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestIFTTTService_SupportsAttachments(t *testing.T) {
	service := NewIFTTTService()
	if service.SupportsAttachments() {
		t.Error("IFTTT should not support attachments")
	}
}

func TestIFTTTService_GetMaxBodyLength(t *testing.T) {
	service := NewIFTTTService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestIFTTTService_Send(t *testing.T) {
	service := NewIFTTTService().(*IFTTTService)
	service.webhookKey = "test_key"
	service.event = "test_event"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Event",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid webhook key
	if err == nil {
		t.Error("Expected error due to invalid webhook key, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "IFTTT") {
		t.Errorf("Expected error to mention IFTTT, got: %v", err)
	}
}