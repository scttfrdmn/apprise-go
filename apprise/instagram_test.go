package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestInstagramService_ParseURL(t *testing.T) {

	tests := []struct {
		name           string
		url            string
		expectError    bool
		expectedToken  string
		expectedUserID string
	}{
		{
			name:           "Valid Instagram URL",
			url:            "instagram://access_token@123456789",
			expectError:    false,
			expectedToken:  "access_token",
			expectedUserID: "123456789",
		},
		{
			name:        "Missing access token",
			url:         "instagram://@123456789",
			expectError: true,
		},
		{
			name:        "Missing user ID",
			url:         "instagram://access_token@",
			expectError: true,
		},
		{
			name:        "Invalid URL format",
			url:         "instagram://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewInstagramService().(*InstagramService)
			
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

			if service.userID != tt.expectedUserID {
				t.Errorf("Expected user ID to be %s, got %s", tt.expectedUserID, service.userID)
			}
		})
	}
}

func TestInstagramService_GetServiceID(t *testing.T) {
	service := NewInstagramService()
	if service.GetServiceID() != "instagram" {
		t.Errorf("Expected service ID 'instagram', got %s", service.GetServiceID())
	}
}

func TestInstagramService_GetDefaultPort(t *testing.T) {
	service := NewInstagramService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestInstagramService_SupportsAttachments(t *testing.T) {
	service := NewInstagramService()
	if !service.SupportsAttachments() {
		t.Error("Instagram should support attachments")
	}
}

func TestInstagramService_GetMaxBodyLength(t *testing.T) {
	service := NewInstagramService()
	if service.GetMaxBodyLength() != 2200 {
		t.Errorf("Expected max body length 2200, got %d", service.GetMaxBodyLength())
	}
}

func TestInstagramService_Send(t *testing.T) {
	service := NewInstagramService().(*InstagramService)
	service.accessToken = "test_token"
	service.userID = "123456789"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Post",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to API limitations
	if err == nil {
		t.Error("Expected error due to API limitations, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Instagram") {
		t.Errorf("Expected error to mention Instagram, got: %v", err)
	}
}