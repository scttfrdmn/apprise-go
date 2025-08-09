package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestSlackService_GetServiceID(t *testing.T) {
	service := NewSlackService()
	if service.GetServiceID() != "slack" {
		t.Errorf("Expected service ID 'slack', got '%s'", service.GetServiceID())
	}
}

func TestSlackService_GetDefaultPort(t *testing.T) {
	service := NewSlackService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestSlackService_ParseURL_Basic(t *testing.T) {
	// Test simple bot token
	service := NewSlackService().(*SlackService)
	parsedURL, err := url.Parse("slack://bot-token/general")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.mode != "bot" {
		t.Errorf("Expected mode 'bot', got '%s'", service.mode)
	}

	if service.botToken != "general" { // Path becomes the bot token in this URL structure
		t.Errorf("Expected bot token 'general', got '%s'", service.botToken)
	}
}

func TestSlackService_ParseURL_Errors(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"Invalid scheme", "http://token"},
		{"Empty URL", "slack://"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewSlackService().(*SlackService)
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				// Some URLs may fail to parse entirely
				return
			}

			err = service.ParseURL(parsedURL)
			if err == nil {
				t.Errorf("Expected error for URL %s but got none", tc.url)
			}
		})
	}
}


func TestSlackService_TestURL(t *testing.T) {
	service := NewSlackService()

	validURLs := []string{
		"slack://TokenA/TokenB/TokenC",
		"slack://TokenA/TokenB/TokenC/general",
		"slack://xoxb-bot-token/general",
		"slack://xoxb-bot-token/@username",
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
		"http://TokenA/TokenB/TokenC",
		"slack://",
		"slack://token", // Not enough tokens
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

func TestSlackService_Properties(t *testing.T) {
	service := NewSlackService()

	if !service.SupportsAttachments() {
		t.Error("Slack service should support attachments")
	}

	if service.GetMaxBodyLength() != 4000 {
		t.Errorf("Expected max body length 4000, got %d", service.GetMaxBodyLength())
	}
}

func TestSlackService_getColorForNotifyType(t *testing.T) {
	service := NewSlackService().(*SlackService)

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, "good"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "danger"},
		{NotifyTypeInfo, "#36a64f"},
	}

	for _, test := range tests {
		result := service.getColorForNotifyType(test.notifyType)
		if result != test.expected {
			t.Errorf("Expected color '%s' for %v, got '%s'", test.expected, test.notifyType, result)
		}
	}
}

func TestSlackService_Send_InvalidConfig(t *testing.T) {
	service := NewSlackService().(*SlackService)
	
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

func TestSlackService_QueryParams(t *testing.T) {
	service := NewSlackService().(*SlackService)
	parsedURL, err := url.Parse("slack://bot-token/channel?username=TestBot&icon_emoji=:ghost:")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.username != "TestBot" {
		t.Errorf("Expected username 'TestBot', got '%s'", service.username)
	}

	if service.iconEmoji != ":ghost:" {
		t.Errorf("Expected icon emoji ':ghost:', got '%s'", service.iconEmoji)
	}
}