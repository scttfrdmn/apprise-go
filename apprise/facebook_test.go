package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestFacebookService_ParseURL(t *testing.T) {

	tests := []struct {
		name           string
		url            string
		expectError    bool
		expectedToken  string
		expectedPageID string
	}{
		{
			name:           "Valid Facebook URL",
			url:            "facebook://access_token@123456789",
			expectError:    false,
			expectedToken:  "access_token",
			expectedPageID: "123456789",
		},
		{
			name:        "Missing access token",
			url:         "facebook://@123456789",
			expectError: true,
		},
		{
			name:        "Missing page ID",
			url:         "facebook://access_token@",
			expectError: true,
		},
		{
			name:        "Invalid URL format",
			url:         "facebook://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewFacebookService().(*FacebookService)
			
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

			if service.accessToken != tt.expectedToken {
				t.Errorf("Expected access token to be %s, got %s", tt.expectedToken, service.accessToken)
			}

			if service.pageID != tt.expectedPageID {
				t.Errorf("Expected page ID to be %s, got %s", tt.expectedPageID, service.pageID)
			}
		})
	}
}

func TestFacebookService_GetServiceID(t *testing.T) {
	service := NewFacebookService()
	if service.GetServiceID() != "facebook" {
		t.Errorf("Expected service ID 'facebook', got %s", service.GetServiceID())
	}
}

func TestFacebookService_GetDefaultPort(t *testing.T) {
	service := NewFacebookService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestFacebookService_SupportsAttachments(t *testing.T) {
	service := NewFacebookService()
	if !service.SupportsAttachments() {
		t.Error("Facebook should support attachments")
	}
}

func TestFacebookService_GetMaxBodyLength(t *testing.T) {
	service := NewFacebookService()
	if service.GetMaxBodyLength() != 63206 {
		t.Errorf("Expected max body length 63206, got %d", service.GetMaxBodyLength())
	}
}

func TestFacebookService_Send(t *testing.T) {
	service := NewFacebookService().(*FacebookService)
	service.accessToken = "test_token"
	service.pageID = "123456789"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Post",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid credentials/unreachable API
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Facebook") {
		t.Errorf("Expected error to mention Facebook, got: %v", err)
	}
}