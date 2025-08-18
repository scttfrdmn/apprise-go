package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBulkSMSService_ParseURL(t *testing.T) {

	tests := []struct {
		name        string
		url         string
		expectError bool
		expectedTo  []string
		expectedFrom string
	}{
		{
			name:        "Valid single recipient",
			url:         "bulksms://user:pass@+1234567890",
			expectError: false,
			expectedTo:  []string{"+1234567890"},
		},
		{
			name:         "Valid with from parameter",
			url:          "bulksms://user:pass@+1234567890?from=Company",
			expectError:  false,
			expectedTo:   []string{"+1234567890"},
			expectedFrom: "Company",
		},
		{
			name:        "Multiple recipients",
			url:         "bulksms://user:pass@+1234567890/+0987654321",
			expectError: false,
			expectedTo:  []string{"+1234567890", "+0987654321"},
			expectedFrom: "", // No from parameter in this test
		},
		{
			name:        "Missing credentials",
			url:         "bulksms://+1234567890",
			expectError: true,
		},
		{
			name:        "Missing password",
			url:         "bulksms://user@+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "bulksms://user:pass@",
			expectError: true,
		},
		{
			name:        "Invalid URL",
			url:         "invalid-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewBulkSMSService().(*BulkSMSService)
			
			parsedURL, parseErr := url.Parse(tt.url)
			if parseErr != nil && !tt.expectError {
				t.Fatalf("URL parsing failed: %v", parseErr)
			}

			if parseErr != nil {
				return // Skip if URL can't be parsed
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

			// Verify recipients
			if len(service.to) != len(tt.expectedTo) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedTo), len(service.to))
			}

			for i, expected := range tt.expectedTo {
				if i < len(service.to) && service.to[i] != expected {
					t.Errorf("Expected recipient %d to be %s, got %s", i, expected, service.to[i])
				}
			}

			// Verify from parameter
			if service.from != tt.expectedFrom {
				t.Errorf("Expected from to be %s, got %s", tt.expectedFrom, service.from)
			}
		})
	}
}

func TestBulkSMSService_GetServiceID(t *testing.T) {
	service := NewBulkSMSService()
	if service.GetServiceID() != "bulksms" {
		t.Errorf("Expected service ID 'bulksms', got %s", service.GetServiceID())
	}
}

func TestBulkSMSService_GetDefaultPort(t *testing.T) {
	service := NewBulkSMSService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestBulkSMSService_SupportsAttachments(t *testing.T) {
	service := NewBulkSMSService()
	if service.SupportsAttachments() {
		t.Error("BulkSMS should not support attachments")
	}
}

func TestBulkSMSService_GetMaxBodyLength(t *testing.T) {
	service := NewBulkSMSService()
	if service.GetMaxBodyLength() != 160 {
		t.Errorf("Expected max body length 160, got %d", service.GetMaxBodyLength())
	}
}

func TestBulkSMSService_TestURL(t *testing.T) {
	service := NewBulkSMSService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid URL",
			url:         "bulksms://user:pass@+1234567890",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "bulksms://+1234567890",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestBulkSMSService_Send(t *testing.T) {
	service := NewBulkSMSService().(*BulkSMSService)
	service.username = "testuser"
	service.password = "testpass"
	service.from = "TestSender"
	service.to = []string{"+1234567890"}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	// Note: This test will fail because it tries to reach the real BulkSMS API
	// In a real implementation, we would mock the HTTP client
	err := service.Send(ctx, notification)

	// For now, we expect this to fail since we can't reach the API without valid credentials
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	// Verify the error contains expected information
	if err != nil && !strings.Contains(err.Error(), "BulkSMS") {
		t.Errorf("Expected error to mention BulkSMS, got: %v", err)
	}
}

func TestBulkSMSService_MultipleRecipients(t *testing.T) {
	service := NewBulkSMSService()

	err := service.TestURL("bulksms://user:pass@+1111111111/+2222222222/+3333333333")
	if err != nil {
		t.Fatalf("Failed to parse service URL: %v", err)
	}

	bulkService := service.(*BulkSMSService)
	expectedRecipients := []string{"+1111111111", "+2222222222", "+3333333333"}
	if len(bulkService.to) != len(expectedRecipients) {
		t.Errorf("Expected %d recipients, got %d", len(expectedRecipients), len(bulkService.to))
	}

	for i, expected := range expectedRecipients {
		if bulkService.to[i] != expected {
			t.Errorf("Expected recipient %d to be %s, got %s", i, expected, bulkService.to[i])
		}
	}
}