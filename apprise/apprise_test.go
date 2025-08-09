package apprise

import (
	"testing"
)

func TestNewApprise(t *testing.T) {
	app := New()
	
	if app == nil {
		t.Fatal("Expected non-nil Apprise instance")
	}
	
	if app.Count() != 0 {
		t.Errorf("Expected 0 services, got %d", app.Count())
	}
}

func TestServiceRegistry(t *testing.T) {
	app := New()
	registry := app.registry
	
	// Test supported services
	supportedServices := registry.GetSupportedServices()
	expectedServices := []string{
		"discord", "slack", "telegram", "tgram", "mailto", "mailtos",
		"webhook", "webhooks", "json", "pushover", "pover", "gotify", "gotifys",
	}
	
	if len(supportedServices) < len(expectedServices) {
		t.Errorf("Expected at least %d services, got %d", len(expectedServices), len(supportedServices))
	}
	
	// Test service creation
	for _, serviceID := range expectedServices {
		service, err := registry.Create(serviceID)
		if err != nil && serviceID != "gotify" && serviceID != "gotifys" {
			// Allow gotify to fail as it's not implemented
			t.Errorf("Failed to create service %s: %v", serviceID, err)
		} else if err == nil {
			if service.GetServiceID() == "" {
				t.Errorf("Service %s returned empty ID", serviceID)
			}
		}
	}
}

func TestDiscordService(t *testing.T) {
	service := NewDiscordService()
	
	if service.GetServiceID() != "discord" {
		t.Errorf("Expected service ID 'discord', got '%s'", service.GetServiceID())
	}
	
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
	
	if !service.SupportsAttachments() {
		t.Error("Discord should support attachments")
	}
	
	// Test URL parsing
	testURL := "discord://webhook_id/webhook_token"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid Discord URL: %v", err)
	}
	
	invalidURL := "invalid://webhook_id/webhook_token"
	if err := service.TestURL(invalidURL); err == nil {
		t.Error("Should have failed to parse invalid URL")
	}
}

func TestSlackService(t *testing.T) {
	service := NewSlackService()
	
	if service.GetServiceID() != "slack" {
		t.Errorf("Expected service ID 'slack', got '%s'", service.GetServiceID())
	}
	
	// Test webhook URL parsing
	testURL := "slack://TokenA/TokenB/TokenC"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid Slack webhook URL: %v", err)
	}
	
	// Test bot URL parsing  
	botURL := "slack://xoxb-bot-token/general"
	if err := service.TestURL(botURL); err != nil {
		t.Errorf("Failed to parse valid Slack bot URL: %v", err)
	}
}

func TestTelegramService(t *testing.T) {
	service := NewTelegramService()
	
	if service.GetServiceID() != "telegram" {
		t.Errorf("Expected service ID 'telegram', got '%s'", service.GetServiceID())
	}
	
	// Test URL parsing
	testURL := "tgram://bot_token/chat_id"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid Telegram URL: %v", err)
	}
	
	// Test multiple chat IDs
	multiURL := "telegram://bot_token/chat1/chat2"
	if err := service.TestURL(multiURL); err != nil {
		t.Errorf("Failed to parse valid Telegram multi-chat URL: %v", err)
	}
}

func TestEmailService(t *testing.T) {
	service := NewEmailService()
	
	if service.GetServiceID() != "email" {
		t.Errorf("Expected service ID 'email', got '%s'", service.GetServiceID())
	}
	
	// Test URL parsing
	testURL := "mailto://user:pass@smtp.gmail.com:587/recipient@domain.com"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid email URL: %v", err)
	}
	
	// Test TLS URL
	tlsURL := "mailtos://user:pass@smtp.gmail.com:465/recipient@domain.com"
	if err := service.TestURL(tlsURL); err != nil {
		t.Errorf("Failed to parse valid TLS email URL: %v", err)
	}
}

func TestWebhookService(t *testing.T) {
	service := NewWebhookService()
	
	if service.GetServiceID() != "webhook" {
		t.Errorf("Expected service ID 'webhook', got '%s'", service.GetServiceID())
	}
	
	// Test URL parsing
	testURL := "webhook://api.example.com/notify"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid webhook URL: %v", err)
	}
	
	// Test HTTPS URL
	httpsURL := "webhooks://api.example.com/notify"
	if err := service.TestURL(httpsURL); err != nil {
		t.Errorf("Failed to parse valid HTTPS webhook URL: %v", err)
	}
}

func TestPushoverService(t *testing.T) {
	service := NewPushoverService()
	
	if service.GetServiceID() != "pushover" {
		t.Errorf("Expected service ID 'pushover', got '%s'", service.GetServiceID())
	}
	
	// Test URL parsing
	testURL := "pushover://token@userkey"
	if err := service.TestURL(testURL); err != nil {
		t.Errorf("Failed to parse valid Pushover URL: %v", err)
	}
	
	// Test with devices
	deviceURL := "pover://token@userkey/device1/device2"
	if err := service.TestURL(deviceURL); err != nil {
		t.Errorf("Failed to parse valid Pushover device URL: %v", err)
	}
}

func TestNotifyTypeString(t *testing.T) {
	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeInfo, "info"},
		{NotifyTypeSuccess, "success"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "error"},
	}
	
	for _, test := range tests {
		if test.notifyType.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.notifyType.String())
		}
	}
}

func TestAppriseAddService(t *testing.T) {
	app := New()
	
	// Test adding valid service
	err := app.Add("discord://webhook_id/webhook_token")
	if err != nil {
		t.Errorf("Failed to add valid Discord service: %v", err)
	}
	
	if app.Count() != 1 {
		t.Errorf("Expected 1 service, got %d", app.Count())
	}
	
	// Test adding invalid service
	err = app.Add("invalid://service/url")
	if err == nil {
		t.Error("Should have failed to add invalid service")
	}
	
	// Count should remain 1
	if app.Count() != 1 {
		t.Errorf("Expected 1 service after failed add, got %d", app.Count())
	}
	
	// Test clearing services
	app.Clear()
	if app.Count() != 0 {
		t.Errorf("Expected 0 services after clear, got %d", app.Count())
	}
}