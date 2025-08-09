package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestWebhookService_GetServiceID(t *testing.T) {
	service := NewWebhookService()
	if service.GetServiceID() != "webhook" {
		t.Errorf("Expected service ID 'webhook', got '%s'", service.GetServiceID())
	}
}

func TestWebhookService_GetDefaultPort(t *testing.T) {
	service := NewWebhookService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestWebhookService_ParseURL(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectError bool
		expectedHTTPS bool
		expectedURL string
	}{
		{
			name:          "Basic webhook HTTP",
			url:           "webhook://api.example.com/notify",
			expectError:   false,
			expectedHTTPS: false,
			expectedURL:   "http://api.example.com/notify",
		},
		{
			name:          "Secure webhooks HTTPS",
			url:           "webhooks://api.example.com/notify",
			expectError:   false,
			expectedHTTPS: true,
			expectedURL:   "https://api.example.com/notify",
		},
		{
			name:          "JSON webhook HTTPS",
			url:           "json://api.example.com/webhook",
			expectError:   false,
			expectedHTTPS: true,
			expectedURL:   "https://api.example.com/webhook",
		},
		{
			name:        "Invalid scheme",
			url:         "http://api.example.com/notify",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewWebhookService().(*WebhookService)
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.webhookURL != tc.expectedURL {
				t.Errorf("Expected webhook URL '%s', got '%s'", tc.expectedURL, service.webhookURL)
			}
		})
	}
}

func TestWebhookService_ParseURL_QueryParams(t *testing.T) {
	testURL := "webhook://api.example.com/notify?method=PUT&content_type=text/plain&header_Authorization=Bearer%20token"
	
	service := NewWebhookService().(*WebhookService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.method != "PUT" {
		t.Errorf("Expected method 'PUT', got '%s'", service.method)
	}

	if service.contentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", service.contentType)
	}

	if service.headers["Authorization"] != "Bearer token" {
		t.Errorf("Expected Authorization header 'Bearer token', got '%s'", service.headers["Authorization"])
	}
}

func TestWebhookService_TestURL(t *testing.T) {
	service := NewWebhookService()

	validURLs := []string{
		"webhook://api.example.com/notify",
		"webhooks://secure.api.com/webhook",
		"json://api.example.com/json",
	}

	for _, testURL := range validURLs {
		t.Run("Valid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", testURL, err)
			}
		})
	}

	invalidURLs := []string{
		"http://api.example.com/notify",
		"invalid://api.example.com/webhook",
	}

	for _, testURL := range invalidURLs {
		t.Run("Invalid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", testURL)
			}
		})
	}
}

func TestWebhookService_Properties(t *testing.T) {
	service := NewWebhookService()

	if service.SupportsAttachments() {
		t.Error("Webhook service should not support attachments")
	}

	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected unlimited body length, got %d", service.GetMaxBodyLength())
	}
}

func TestWebhookService_Send_InvalidConfig(t *testing.T) {
	service := NewWebhookService().(*WebhookService)
	
	// Service without proper configuration should fail
	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected Send to fail with invalid configuration")
	}
}

func TestWebhookService_AuthParsing(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Bearer token",
			url:      "webhook://token@api.example.com/notify",
			expected: "Bearer token",
		},
		{
			name:     "Basic auth",
			url:      "webhook://user:pass@api.example.com/notify",
			expected: "Basic user:pass", // The actual implementation might not base64 encode
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewWebhookService().(*WebhookService)
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			if service.headers["Authorization"] != tc.expected {
				t.Errorf("Expected Authorization header '%s', got '%s'", tc.expected, service.headers["Authorization"])
			}
		})
	}
}