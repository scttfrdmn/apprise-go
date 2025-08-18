package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestClickSendService_ParseURL(t *testing.T) {

	tests := []struct {
		name         string
		url          string
		expectError  bool
		expectedTo   []string
		expectedFrom string
	}{
		{
			name:        "Valid single recipient",
			url:         "clicksend://user:apikey@+1234567890",
			expectError: false,
			expectedTo:  []string{"+1234567890"},
		},
		{
			name:         "Valid with from parameter",
			url:          "clicksend://user:apikey@+1234567890?from=Company",
			expectError:  false,
			expectedTo:   []string{"+1234567890"},
			expectedFrom: "Company",
		},
		{
			name:        "Multiple recipients",
			url:         "clicksend://user:apikey@+1234567890/+0987654321",
			expectError: false,
			expectedTo:  []string{"+1234567890", "+0987654321"},
			expectedFrom: "", // No from parameter in this test
		},
		{
			name:        "Missing credentials",
			url:         "clicksend://+1234567890",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "clicksend://user@+1234567890",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewClickSendService().(*ClickSendService)
			
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

			if len(service.to) != len(tt.expectedTo) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedTo), len(service.to))
			}

			if service.from != tt.expectedFrom {
				t.Errorf("Expected from to be %s, got %s", tt.expectedFrom, service.from)
			}
		})
	}
}

func TestClickSendService_GetServiceID(t *testing.T) {
	service := NewClickSendService()
	if service.GetServiceID() != "clicksend" {
		t.Errorf("Expected service ID 'clicksend', got %s", service.GetServiceID())
	}
}

func TestClickSendService_SupportsAttachments(t *testing.T) {
	service := NewClickSendService()
	if service.SupportsAttachments() {
		t.Error("ClickSend should not support attachments")
	}
}

func TestClickSendService_GetMaxBodyLength(t *testing.T) {
	service := NewClickSendService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestClickSendService_Send(t *testing.T) {
	service := NewClickSendService().(*ClickSendService)
	service.username = "testuser"
	service.apiKey = "testapikey"
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

	if err != nil && !strings.Contains(err.Error(), "ClickSend") {
		t.Errorf("Expected error to mention ClickSend, got: %v", err)
	}
}