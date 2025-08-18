package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestYouTubeService_ParseURL(t *testing.T) {

	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedAPIKey    string
		expectedChannelID string
	}{
		{
			name:              "Valid YouTube URL",
			url:               "youtube://api_key@channel_id",
			expectError:       false,
			expectedAPIKey:    "api_key",
			expectedChannelID: "channel_id",
		},
		{
			name:        "Missing API key",
			url:         "youtube://@channel_id",
			expectError: true,
		},
		{
			name:        "Missing channel ID",
			url:         "youtube://api_key@",
			expectError: true,
		},
		{
			name:        "Invalid URL format",
			url:         "youtube://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewYouTubeService().(*YouTubeService)
			
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

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key to be %s, got %s", tt.expectedAPIKey, service.apiKey)
			}

			if service.channelID != tt.expectedChannelID {
				t.Errorf("Expected channel ID to be %s, got %s", tt.expectedChannelID, service.channelID)
			}
		})
	}
}

func TestYouTubeService_GetServiceID(t *testing.T) {
	service := NewYouTubeService()
	if service.GetServiceID() != "youtube" {
		t.Errorf("Expected service ID 'youtube', got %s", service.GetServiceID())
	}
}

func TestYouTubeService_GetDefaultPort(t *testing.T) {
	service := NewYouTubeService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestYouTubeService_SupportsAttachments(t *testing.T) {
	service := NewYouTubeService()
	if service.SupportsAttachments() {
		t.Error("YouTube should not support attachments for comments")
	}
}

func TestYouTubeService_GetMaxBodyLength(t *testing.T) {
	service := NewYouTubeService()
	if service.GetMaxBodyLength() != 10000 {
		t.Errorf("Expected max body length 10000, got %d", service.GetMaxBodyLength())
	}
}

func TestYouTubeService_Send(t *testing.T) {
	service := NewYouTubeService().(*YouTubeService)
	service.apiKey = "test_api_key"
	service.channelID = "test_channel"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Comment",
		Body:  "Test notification body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to API limitations
	if err == nil {
		t.Error("Expected error due to API limitations, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "YouTube") {
		t.Errorf("Expected error to mention YouTube, got: %v", err)
	}
}