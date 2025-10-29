package apprise

import (
	"net/url"
	"testing"
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
	service := NewSlackService().(*SlackService)
	// Note: The path includes the leading slash, so tokenC is the 3rd part, channel is the 4th
	parsedURL, err := url.Parse("slack://host/tokenA/tokenB/tokenC/channel")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if service.mode != "webhook" {
		t.Errorf("Expected webhook mode, got '%s'", service.mode)
	}

	if service.channel != "channel" {
		t.Errorf("Expected channel 'channel', got '%s'", service.channel)
	}
}

func TestSlackService_ParseURL_Errors(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"Invalid scheme", "http://tokenA/tokenB/tokenC", true},
		{"Missing tokens", "slack://", true},
		{"Valid bot URL", "slack://bot-token/channel", false},
		{"Valid webhook URL", "slack://tokenA/tokenB/tokenC", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSlackService().(*SlackService)
			parsedURL, _ := url.Parse(tt.url)
			err := service.ParseURL(parsedURL)

			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestSlackService_TestURL(t *testing.T) {
	service := NewSlackService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid webhook URL",
			url:         "slack://tokenA/tokenB/tokenC/general",
			expectError: false,
		},
		{
			name:        "Valid bot URL",
			url:         "slack://bot-token/channel",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://tokenA/tokenB/tokenC",
			expectError: true,
		},
		{
			name:        "Missing tokens",
			url:         "slack://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestSlackService_Properties(t *testing.T) {
	service := NewSlackService()

	if !service.SupportsAttachments() {
		t.Error("Slack should support attachments")
	}

	expectedMaxLength := 4000
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d", expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestSlackService_getColorForNotifyType(t *testing.T) {
	service := &SlackService{}

	tests := []struct {
		notifyType    NotifyType
		expectedColor string
	}{
		{NotifyTypeSuccess, "good"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "danger"},
		{NotifyTypeInfo, "#36a64f"},
	}

	for _, tt := range tests {
		color := service.getColorForNotifyType(tt.notifyType)
		if color != tt.expectedColor {
			t.Errorf("For notify type %v, expected color '%s', got '%s'", tt.notifyType, tt.expectedColor, color)
		}
	}
}

func TestSlackService_Send_InvalidConfig(t *testing.T) {
	service := &SlackService{
		mode: "webhook",
		// Missing webhookURL
	}

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// This test just ensures Send doesn't panic with invalid config
	// Actual send will fail but shouldn't crash
	_ = service.Send(nil, req)
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

func TestSlackService_TimestampParameter(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectedTimestamp bool
	}{
		{
			name:              "Default (no parameter) - should include timestamp",
			url:               "slack://tokenA/tokenB/tokenC/channel",
			expectedTimestamp: true,
		},
		{
			name:              "Explicit timestamp=yes",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=yes",
			expectedTimestamp: true,
		},
		{
			name:              "Explicit timestamp=true",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=true",
			expectedTimestamp: true,
		},
		{
			name:              "Explicit timestamp=no",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=no",
			expectedTimestamp: false,
		},
		{
			name:              "Explicit timestamp=false",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=false",
			expectedTimestamp: false,
		},
		{
			name:              "Case insensitive - YES",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=YES",
			expectedTimestamp: true,
		},
		{
			name:              "Case insensitive - NO",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=NO",
			expectedTimestamp: false,
		},
		{
			name:              "Invalid value - should keep default (true)",
			url:               "slack://tokenA/tokenB/tokenC/channel?timestamp=invalid",
			expectedTimestamp: true,
		},
		{
			name:              "Bot mode with timestamp=no",
			url:               "slack://bot-token/channel?timestamp=no",
			expectedTimestamp: false,
		},
		{
			name:              "Combined with other parameters",
			url:               "slack://tokenA/tokenB/tokenC/channel?username=Bot&timestamp=no&icon_emoji=:ghost:",
			expectedTimestamp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSlackService().(*SlackService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			if service.includeTimestamp != tt.expectedTimestamp {
				t.Errorf("Expected includeTimestamp to be %v, got %v", tt.expectedTimestamp, service.includeTimestamp)
			}
		})
	}
}
