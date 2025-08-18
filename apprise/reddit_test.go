package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRedditService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedClientID   string
		expectedSecret     string
		expectedUsername   string
		expectedPassword   string
		expectedSubreddit  string
		expectedRecipient  string
	}{
		{
			name:              "Valid subreddit post",
			url:               "reddit://client123:secret456@host/testsubreddit?username=user&password=pass",
			expectError:       false,
			expectedClientID:  "client123",
			expectedSecret:    "secret456",
			expectedUsername:  "user",
			expectedPassword:  "pass",
			expectedSubreddit: "testsubreddit",
		},
		{
			name:              "Valid direct message",
			url:               "reddit://client123:secret456@host/targetuser?username=user&password=pass&mode=message",
			expectError:       false,
			expectedClientID:  "client123",
			expectedSecret:    "secret456",
			expectedUsername:  "user",
			expectedPassword:  "pass",
			expectedRecipient: "targetuser",
		},
		{
			name:              "Valid direct message with dm mode",
			url:               "reddit://client123:secret456@host/targetuser?username=user&password=pass&mode=dm",
			expectError:       false,
			expectedClientID:  "client123",
			expectedSecret:    "secret456",
			expectedUsername:  "user",
			expectedPassword:  "pass",
			expectedRecipient: "targetuser",
		},
		{
			name:        "Missing client credentials",
			url:         "reddit://host/testsubreddit?username=user&password=pass",
			expectError: true,
		},
		{
			name:        "Missing user credentials",
			url:         "reddit://client123:secret456@host/testsubreddit",
			expectError: true,
		},
		{
			name:        "Missing target",
			url:         "reddit://client123:secret456@host?username=user&password=pass",
			expectError: true,
		},
		{
			name:        "Invalid format",
			url:         "reddit://invalid_format",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRedditService().(*RedditService)
			
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

			if service.clientID != tt.expectedClientID {
				t.Errorf("Expected client ID to be %s, got %s", tt.expectedClientID, service.clientID)
			}

			if service.clientSecret != tt.expectedSecret {
				t.Errorf("Expected client secret to be %s, got %s", tt.expectedSecret, service.clientSecret)
			}

			if service.username != tt.expectedUsername {
				t.Errorf("Expected username to be %s, got %s", tt.expectedUsername, service.username)
			}

			if service.password != tt.expectedPassword {
				t.Errorf("Expected password to be %s, got %s", tt.expectedPassword, service.password)
			}

			if service.subreddit != tt.expectedSubreddit {
				t.Errorf("Expected subreddit to be %s, got %s", tt.expectedSubreddit, service.subreddit)
			}

			if service.recipient != tt.expectedRecipient {
				t.Errorf("Expected recipient to be %s, got %s", tt.expectedRecipient, service.recipient)
			}
		})
	}
}

func TestRedditService_GetServiceID(t *testing.T) {
	service := NewRedditService()
	if service.GetServiceID() != "reddit" {
		t.Errorf("Expected service ID 'reddit', got %s", service.GetServiceID())
	}
}

func TestRedditService_GetDefaultPort(t *testing.T) {
	service := NewRedditService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestRedditService_SupportsAttachments(t *testing.T) {
	service := NewRedditService()
	if service.SupportsAttachments() {
		t.Error("Reddit should not support attachments")
	}
}

func TestRedditService_GetMaxBodyLength(t *testing.T) {
	service := NewRedditService()
	if service.GetMaxBodyLength() != 40000 {
		t.Errorf("Expected max body length 40000, got %d", service.GetMaxBodyLength())
	}
}

func TestRedditService_Send(t *testing.T) {
	service := NewRedditService().(*RedditService)
	service.clientID = "test_client"
	service.clientSecret = "test_secret"
	service.username = "test_user"
	service.password = "test_pass"
	service.subreddit = "test"

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

	if err != nil && !strings.Contains(err.Error(), "Reddit") {
		t.Errorf("Expected error to mention Reddit, got: %v", err)
	}
}

func TestRedditService_TestURL(t *testing.T) {
	service := NewRedditService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid subreddit URL",
			url:         "reddit://client123:secret456@host/testsubreddit?username=user&password=pass",
			expectError: false,
		},
		{
			name:        "Valid message URL",
			url:         "reddit://client123:secret456@host/targetuser?username=user&password=pass&mode=message",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			url:         "invalid-url",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "reddit://host/testsubreddit",
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