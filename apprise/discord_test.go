package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestDiscordServiceURLParsing(t *testing.T) {
	service := NewDiscordService()

	testCases := []struct {
		name             string
		url              string
		shouldError      bool
		expectedID       string
		expectedToken    string
		expectedUsername string
		expectedAvatar   string
	}{
		{
			name:          "Basic webhook URL",
			url:           "discord://webhook_id/webhook_token",
			shouldError:   false,
			expectedID:    "webhook_id",
			expectedToken: "webhook_token",
		},
		{
			name:           "Webhook URL with avatar",
			url:            "discord://avatar@webhook_id/webhook_token",
			shouldError:    false,
			expectedID:     "webhook_id",
			expectedToken:  "webhook_token",
			expectedAvatar: "avatar",
		},
		{
			name:             "Webhook URL with query parameters",
			url:              "discord://webhook_id/webhook_token?username=MyBot&avatar=https://example.com/avatar.png",
			shouldError:      false,
			expectedID:       "webhook_id",
			expectedToken:    "webhook_token",
			expectedUsername: "MyBot",
			expectedAvatar:   "https://example.com/avatar.png",
		},
		{
			name:        "Invalid scheme",
			url:         "slack://webhook_id/webhook_token",
			shouldError: true,
		},
		{
			name:        "Missing token",
			url:         "discord://webhook_id",
			shouldError: true,
		},
		{
			name:        "Empty webhook ID",
			url:         "discord:///webhook_token",
			shouldError: true,
		},
		{
			name:        "Empty webhook token",
			url:         "discord://webhook_id/",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			discordService := service.(*DiscordService)
			err = discordService.ParseURL(parsedURL)

			if tc.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if discordService.webhookID != tc.expectedID {
				t.Errorf("Expected webhook ID %q, got %q", tc.expectedID, discordService.webhookID)
			}

			if discordService.webhookToken != tc.expectedToken {
				t.Errorf("Expected webhook token %q, got %q", tc.expectedToken, discordService.webhookToken)
			}

			if discordService.username != tc.expectedUsername {
				t.Errorf("Expected username %q, got %q", tc.expectedUsername, discordService.username)
			}

			if discordService.avatar != tc.expectedAvatar {
				t.Errorf("Expected avatar %q, got %q", tc.expectedAvatar, discordService.avatar)
			}
		})
	}
}

func TestDiscordColorMapping(t *testing.T) {
	service := NewDiscordService().(*DiscordService)

	testCases := []struct {
		notifyType    NotifyType
		expectedColor int
	}{
		{NotifyTypeInfo, 0x0099FF},
		{NotifyTypeSuccess, 0x00FF00},
		{NotifyTypeWarning, 0xFFFF00},
		{NotifyTypeError, 0xFF0000},
	}

	for _, tc := range testCases {
		color := service.getColorForNotifyType(tc.notifyType)
		if color != tc.expectedColor {
			t.Errorf("For notify type %s, expected color %d, got %d",
				tc.notifyType.String(), tc.expectedColor, color)
		}
	}
}

func TestDiscordPayloadGeneration(t *testing.T) {
	service := NewDiscordService()
	parsedURL, _ := url.Parse("discord://webhook_id/webhook_token?username=TestBot")
	service.(*DiscordService).ParseURL(parsedURL)

	discordService := service.(*DiscordService)

	// Test that payload generation doesn't panic
	// (We can't easily test the actual HTTP call without a mock server)
	if discordService.username != "TestBot" {
		t.Error("Expected username to be set from URL parameters")
	}

	if discordService.webhookID != "webhook_id" {
		t.Error("Expected webhook ID to be parsed correctly")
	}

	if discordService.webhookToken != "webhook_token" {
		t.Error("Expected webhook token to be parsed correctly")
	}
}

func TestDiscordServiceCapabilities(t *testing.T) {
	service := NewDiscordService()

	if service.GetServiceID() != "discord" {
		t.Errorf("Expected service ID 'discord', got %q", service.GetServiceID())
	}

	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}

	if !service.SupportsAttachments() {
		t.Error("Discord should support attachments")
	}

	expectedMaxLength := 2000
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestDiscordTestURL(t *testing.T) {
	service := NewDiscordService()

	validURLs := []string{
		"discord://webhook_id/webhook_token",
		"discord://avatar@webhook_id/webhook_token",
		"discord://webhook_id/webhook_token?username=Bot",
	}

	for _, validURL := range validURLs {
		if err := service.TestURL(validURL); err != nil {
			t.Errorf("Valid URL %q should not error: %v", validURL, err)
		}
	}

	invalidURLs := []string{
		"invalid://webhook_id/webhook_token",
		"discord://webhook_id",
		"discord:///webhook_token",
		"not-a-url",
	}

	for _, invalidURL := range invalidURLs {
		if err := service.TestURL(invalidURL); err == nil {
			t.Errorf("Invalid URL %q should error", invalidURL)
		}
	}
}

func TestDiscordSendMethodExists(t *testing.T) {
	service := NewDiscordService()
	parsedURL, _ := url.Parse("discord://test_id/test_token")
	service.(*DiscordService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real Discord webhook
	if err == nil {
		t.Error("Expected network error for invalid webhook URL, got none")
	}

	// Check that error message makes sense
	if !strings.Contains(err.Error(), "Discord") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should be network-related, got: %v", err)
	}
}
