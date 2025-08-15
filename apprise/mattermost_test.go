package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestMattermostService_GetServiceID(t *testing.T) {
	service := NewMattermostService()
	if service.GetServiceID() != "mattermost" {
		t.Errorf("Expected service ID 'mattermost', got %q", service.GetServiceID())
	}
}

func TestMattermostService_GetDefaultPort(t *testing.T) {
	service := NewMattermostService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestMattermostService_ParseURL(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedServerURL  string
		expectedToken      string
		expectedUsername   string
		expectedPassword   string
		expectedChannels   []string
		expectedBotName    string
		expectedIconURL    string
		expectedIconEmoji  string
	}{
		{
			name:              "Basic HTTP with username/password",
			url:               "mattermost://user:pass@mattermost.example.com/general",
			expectError:       false,
			expectedServerURL: "http://mattermost.example.com:8065",
			expectedUsername:  "user",
			expectedPassword:  "pass",
			expectedChannels:  []string{"general"},
		},
		{
			name:              "HTTPS with token",
			url:               "mmosts://token123@mattermost.example.com/alerts",
			expectError:       false,
			expectedServerURL: "https://mattermost.example.com:443",
			expectedToken:     "token123",
			expectedChannels:  []string{"alerts"},
		},
		{
			name:              "Custom port",
			url:               "mattermost://user:pass@mattermost.company.com:8080/general",
			expectError:       false,
			expectedServerURL: "http://mattermost.company.com:8080",
			expectedUsername:  "user",
			expectedPassword:  "pass",
			expectedChannels:  []string{"general"},
		},
		{
			name:              "Multiple channels",
			url:               "mmosts://token@mm.example.com/general/alerts/dev-team",
			expectError:       false,
			expectedServerURL: "https://mm.example.com:443",
			expectedToken:     "token",
			expectedChannels:  []string{"general", "alerts", "dev-team"},
		},
		{
			name:              "Channel with # prefix",
			url:               "mattermost://token@mm.example.com/#general",
			expectError:       false,
			expectedServerURL: "http://mm.example.com:8065",
			expectedToken:     "token",
			expectedChannels:  []string{"general"},
		},
		{
			name:              "Channel with @ prefix",
			url:               "mattermost://token@mm.example.com/@user",
			expectError:       false,
			expectedServerURL: "http://mm.example.com:8065",
			expectedToken:     "token",
			expectedChannels:  []string{"user"},
		},
		{
			name:              "With bot name and icon URL",
			url:               "mattermost://token@mm.example.com/general?bot=AlertBot&icon_url=https://example.com/icon.png",
			expectError:       false,
			expectedServerURL: "http://mm.example.com:8065",
			expectedToken:     "token",
			expectedChannels:  []string{"general"},
			expectedBotName:   "AlertBot",
			expectedIconURL:   "https://example.com/icon.png",
		},
		{
			name:              "With icon emoji",
			url:               "mmosts://token@mm.example.com/alerts?icon_emoji=:warning:",
			expectError:       false,
			expectedServerURL: "https://mm.example.com:443",
			expectedToken:     "token",
			expectedChannels:  []string{"alerts"},
			expectedIconEmoji: ":warning:",
		},
		{
			name:              "Token in query parameter",
			url:               "mattermost://user@mm.example.com/general?token=abc123",
			expectError:       false,
			expectedServerURL: "http://mm.example.com:8065",
			expectedToken:     "abc123",
			expectedUsername:  "user",
			expectedChannels:  []string{"general"},
		},
		{
			name:        "Invalid scheme",
			url:         "http://token@mm.example.com/general",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "mattermost://token@/general",
			expectError: true,
		},
		{
			name:        "Missing authentication",
			url:         "mattermost://@mm.example.com/general",
			expectError: true,
		},
		{
			name:        "Missing channels",
			url:         "mattermost://token@mm.example.com",
			expectError: true,
		},
		{
			name:        "Empty channel",
			url:         "mattermost://token@mm.example.com/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMattermostService().(*MattermostService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				return
			}

			if service.serverURL != tt.expectedServerURL {
				t.Errorf("Expected serverURL %q, got %q", tt.expectedServerURL, service.serverURL)
			}

			if tt.expectedToken != "" && service.token != tt.expectedToken {
				t.Errorf("Expected token %q, got %q", tt.expectedToken, service.token)
			}

			if tt.expectedUsername != "" && service.username != tt.expectedUsername {
				t.Errorf("Expected username %q, got %q", tt.expectedUsername, service.username)
			}

			if tt.expectedPassword != "" && service.password != tt.expectedPassword {
				t.Errorf("Expected password %q, got %q", tt.expectedPassword, service.password)
			}

			if tt.expectedChannels != nil && !stringSlicesEqual(service.channels, tt.expectedChannels) {
				t.Errorf("Expected channels %v, got %v", tt.expectedChannels, service.channels)
			}

			if tt.expectedBotName != "" && service.botName != tt.expectedBotName {
				t.Errorf("Expected botName %q, got %q", tt.expectedBotName, service.botName)
			}

			if tt.expectedIconURL != "" && service.iconURL != tt.expectedIconURL {
				t.Errorf("Expected iconURL %q, got %q", tt.expectedIconURL, service.iconURL)
			}

			if tt.expectedIconEmoji != "" && service.iconEmoji != tt.expectedIconEmoji {
				t.Errorf("Expected iconEmoji %q, got %q", tt.expectedIconEmoji, service.iconEmoji)
			}
		})
	}
}

func TestMattermostService_NormalizeChannelName(t *testing.T) {
	service := &MattermostService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"general", "general"},
		{"#general", "general"},
		{"@user", "user"},
		{"dev-team", "dev-team"},
		{"#alerts", "alerts"},
		{"@admin", "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.normalizeChannelName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeChannelName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMattermostService_TestURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid mattermost://user:pass@server/channel",
			url:         "mattermost://user:pass@mattermost.example.com/general",
			expectError: false,
		},
		{
			name:        "Valid mmosts://token@server/channel",
			url:         "mmosts://token123@mattermost.example.com/alerts",
			expectError: false,
		},
		{
			name:        "Valid with custom port",
			url:         "mattermost://user:pass@mm.company.com:9000/general",
			expectError: false,
		},
		{
			name:        "Valid with query parameters",
			url:         "mmosts://token@mm.example.com/general?bot=AlertBot&icon_emoji=:warning:",
			expectError: false,
		},
		{
			name:        "Invalid http://server/channel",
			url:         "http://token@mattermost.example.com/general",
			expectError: true,
		},
		{
			name:        "Invalid mattermost://@server/channel (no auth)",
			url:         "mattermost://@mattermost.example.com/general",
			expectError: true,
		},
		{
			name:        "Invalid mattermost://token@server (no channel)",
			url:         "mattermost://token@mattermost.example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMattermostService()
			err := service.TestURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				}
			}
		})
	}
}

func TestMattermostService_Properties(t *testing.T) {
	service := NewMattermostService()

	if !service.SupportsAttachments() {
		t.Error("Mattermost should support attachments")
	}

	expectedMaxLength := 4000
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestMattermostService_GetEmojiForNotifyType(t *testing.T) {
	service := &MattermostService{}

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, ":white_check_mark:"},
		{NotifyTypeWarning, ":warning:"},
		{NotifyTypeError, ":x:"},
		{NotifyTypeInfo, ":information_source:"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			result := service.getEmojiForNotifyType(tt.notifyType)
			if result != tt.expected {
				t.Errorf("Expected emoji %q for %v, got %q", tt.expected, tt.notifyType, result)
			}
		})
	}
}

func TestMattermostService_FormatMessage(t *testing.T) {
	service := &MattermostService{}

	tests := []struct {
		name           string
		title          string
		body           string
		notifyType     NotifyType
		expectedResult string
	}{
		{
			name:           "Title and body",
			title:          "Alert",
			body:           "System error occurred",
			notifyType:     NotifyTypeError,
			expectedResult: ":x: **Alert**\n\nSystem error occurred",
		},
		{
			name:           "Title only",
			title:          "Success",
			body:           "",
			notifyType:     NotifyTypeSuccess,
			expectedResult: ":white_check_mark: **Success**",
		},
		{
			name:           "Body only",
			title:          "",
			body:           "System is running normally",
			notifyType:     NotifyTypeInfo,
			expectedResult: "System is running normally",
		},
		{
			name:           "Empty title and body",
			title:          "",
			body:           "",
			notifyType:     NotifyTypeWarning,
			expectedResult: ":warning: Notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.formatMessage(tt.title, tt.body, tt.notifyType)
			if result != tt.expectedResult {
				t.Errorf("Expected formatted message %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestMattermostService_Send_InvalidConfig(t *testing.T) {
	service := NewMattermostService()
	parsedURL, _ := url.Parse("mattermost://test_token@mattermost.example.com/general")
	_ = service.(*MattermostService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real Mattermost server
	if err == nil {
		t.Error("Expected network error for invalid Mattermost configuration, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "mattermost") &&
		!strings.Contains(err.Error(), "Mattermost") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should be network-related, got: %v", err)
	}
}

func TestMattermostService_PayloadGeneration(t *testing.T) {
	service := NewMattermostService()
	parsedURL, _ := url.Parse("mmosts://test_token@mattermost.example.com:443/general?bot=AlertBot&icon_emoji=:warning:")
	_ = service.(*MattermostService).ParseURL(parsedURL)

	mattermostService := service.(*MattermostService)

	// Test that configuration is parsed correctly
	if mattermostService.token != "test_token" {
		t.Error("Expected token to be set from URL")
	}

	if mattermostService.serverURL != "https://mattermost.example.com:443" {
		t.Errorf("Expected serverURL to be parsed correctly, got: %s", mattermostService.serverURL)
	}

	expectedChannels := []string{"general"}
	if !stringSlicesEqual(mattermostService.channels, expectedChannels) {
		t.Errorf("Expected channels %v, got: %v", expectedChannels, mattermostService.channels)
	}

	if mattermostService.botName != "AlertBot" {
		t.Errorf("Expected botName to be 'AlertBot', got: %s", mattermostService.botName)
	}

	if mattermostService.iconEmoji != ":warning:" {
		t.Errorf("Expected iconEmoji to be ':warning:', got: %s", mattermostService.iconEmoji)
	}
}

func TestMattermostService_AuthenticationMethods(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectToken      bool
		expectCredentials bool
	}{
		{
			name:              "Token authentication",
			url:               "mattermost://token123@mm.example.com/general",
			expectToken:       true,
			expectCredentials: false,
		},
		{
			name:              "Username/password authentication",
			url:               "mattermost://user:pass@mm.example.com/general",
			expectToken:       false,
			expectCredentials: true,
		},
		{
			name:              "Token in query parameter",
			url:               "mattermost://user@mm.example.com/general?token=abc123",
			expectToken:       true,
			expectCredentials: false, // Token overrides username-only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMattermostService().(*MattermostService)
			parsedURL, _ := url.Parse(tt.url)
			_ = service.ParseURL(parsedURL)

			hasToken := service.token != ""
			hasCredentials := service.username != "" && service.password != ""

			if hasToken != tt.expectToken {
				t.Errorf("Expected token present: %v, got: %v", tt.expectToken, hasToken)
			}

			if hasCredentials != tt.expectCredentials {
				t.Errorf("Expected credentials present: %v, got: %v", tt.expectCredentials, hasCredentials)
			}
		})
	}
}