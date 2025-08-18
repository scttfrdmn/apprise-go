package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestZapierService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedWebhookURL string
	}{
		{
			name:               "Valid Zapier URL",
			url:                "zapier://hooks.zapier.com/hooks/catch/123456/abcdef",
			expectError:        false,
			expectedWebhookURL: "https://hooks.zapier.com/hooks/catch/123456/abcdef",
		},
		{
			name:               "Valid Zapier HTTP URL",
			url:                "zapier+http://hooks.zapier.com/hooks/catch/123456/abcdef",
			expectError:        false,
			expectedWebhookURL: "http://hooks.zapier.com/hooks/catch/123456/abcdef",
		},
		{
			name:        "Missing host",
			url:         "zapier:///hooks/catch/123456/abcdef",
			expectError: true,
		},
		{
			name:               "Invalid URL format",
			url:                "zapier://invalid",
			expectError:        false, // This will still parse but might not work
			expectedWebhookURL: "https://invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewZapierService().(*ZapierService)
			
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

			if service.webhookURL != tt.expectedWebhookURL {
				t.Errorf("Expected webhook URL to be %s, got %s", tt.expectedWebhookURL, service.webhookURL)
			}
		})
	}
}

func TestZapierService_GetServiceID(t *testing.T) {
	service := NewZapierService()
	if service.GetServiceID() != "zapier" {
		t.Errorf("Expected service ID 'zapier', got %s", service.GetServiceID())
	}
}

func TestZapierService_GetDefaultPort(t *testing.T) {
	service := NewZapierService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestZapierService_SupportsAttachments(t *testing.T) {
	service := NewZapierService()
	if service.SupportsAttachments() {
		t.Error("Zapier should not support attachments")
	}
}

func TestZapierService_GetMaxBodyLength(t *testing.T) {
	service := NewZapierService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestZapierService_Send(t *testing.T) {
	service := NewZapierService().(*ZapierService)
	service.webhookURL = "https://hooks.zapier.com/hooks/catch/123456/abcdef"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Zap",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid webhook URL
	if err == nil {
		t.Error("Expected error due to invalid webhook URL, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Zapier") {
		t.Errorf("Expected error to mention Zapier, got: %v", err)
	}
}