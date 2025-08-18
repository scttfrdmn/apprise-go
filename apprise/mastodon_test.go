package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMastodonService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedToken      string
		expectedInstance   string
		expectedVisibility string
	}{
		{
			name:               "Valid basic URL",
			url:                "mastodon://access_token@mastodon.social",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "https://mastodon.social",
			expectedVisibility: "public",
		},
		{
			name:               "Valid URL with visibility",
			url:                "mastodon://access_token@mastodon.social?visibility=unlisted",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "https://mastodon.social",
			expectedVisibility: "unlisted",
		},
		{
			name:               "Valid URL with private visibility",
			url:                "mastodon://access_token@mastodon.social?visibility=private",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "https://mastodon.social",
			expectedVisibility: "private",
		},
		{
			name:               "Valid URL with direct visibility",
			url:                "mastodon://access_token@mastodon.social?visibility=direct",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "https://mastodon.social",
			expectedVisibility: "direct",
		},
		{
			name:               "Valid URL with custom port",
			url:                "mastodon://access_token@example.com:8080",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "https://example.com:8080",
			expectedVisibility: "public",
		},
		{
			name:               "Valid HTTP URL",
			url:                "mastodon+http://access_token@localhost:3000",
			expectError:        false,
			expectedToken:      "access_token",
			expectedInstance:   "http://localhost:3000",
			expectedVisibility: "public",
		},
		{
			name:        "Missing access token",
			url:         "mastodon://@mastodon.social",
			expectError: true,
		},
		{
			name:        "Missing instance host",
			url:         "mastodon://access_token@",
			expectError: true,
		},
		{
			name:        "Invalid visibility",
			url:         "mastodon://access_token@mastodon.social?visibility=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMastodonService().(*MastodonService)
			
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

			if service.instanceURL != tt.expectedInstance {
				t.Errorf("Expected instance URL to be %s, got %s", tt.expectedInstance, service.instanceURL)
			}

			if service.visibility != tt.expectedVisibility {
				t.Errorf("Expected visibility to be %s, got %s", tt.expectedVisibility, service.visibility)
			}
		})
	}
}

func TestMastodonService_GetServiceID(t *testing.T) {
	service := NewMastodonService()
	if service.GetServiceID() != "mastodon" {
		t.Errorf("Expected service ID 'mastodon', got %s", service.GetServiceID())
	}
}

func TestMastodonService_GetDefaultPort(t *testing.T) {
	service := NewMastodonService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestMastodonService_SupportsAttachments(t *testing.T) {
	service := NewMastodonService()
	if !service.SupportsAttachments() {
		t.Error("Mastodon should support attachments")
	}
}

func TestMastodonService_GetMaxBodyLength(t *testing.T) {
	service := NewMastodonService()
	if service.GetMaxBodyLength() != 500 {
		t.Errorf("Expected max body length 500, got %d", service.GetMaxBodyLength())
	}
}

func TestMastodonService_Send(t *testing.T) {
	service := NewMastodonService().(*MastodonService)
	service.accessToken = "test_token"
	service.instanceURL = "https://mastodon.social"
	service.visibility = "public"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Toot",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid credentials/unreachable API
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Mastodon") {
		t.Errorf("Expected error to mention Mastodon, got: %v", err)
	}
}

func TestMastodonService_TestURL(t *testing.T) {
	service := NewMastodonService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid URL",
			url:         "mastodon://access_token@mastodon.social",
			expectError: false,
		},
		{
			name:        "Valid URL with visibility",
			url:         "mastodon://access_token@mastodon.social?visibility=unlisted",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "mastodon://@mastodon.social",
			expectError: true,
		},
		{
			name:        "Invalid visibility",
			url:         "mastodon://token@mastodon.social?visibility=bad",
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