package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestTelegramServiceURLParsing(t *testing.T) {

	testCases := []struct {
		name              string
		url               string
		shouldError       bool
		expectedToken     string
		expectedChatIDs   []string
		expectedSilent    bool
		expectedPreview   bool
		expectedParseMode string
	}{
		{
			name:              "Basic bot URL",
			url:               "tgram://bot_token/chat_id",
			shouldError:       false,
			expectedToken:     "bot_token",
			expectedChatIDs:   []string{"chat_id"},
			expectedPreview:   true,
			expectedParseMode: "Markdown",
		},
		{
			name:            "Multiple chat IDs",
			url:             "telegram://bot_token/chat1/chat2/chat3",
			shouldError:     false,
			expectedToken:   "bot_token",
			expectedChatIDs: []string{"chat1", "chat2", "chat3"},
			expectedPreview: true,
		},
		{
			name:            "With username chat ID",
			url:             "tgram://bot_token/@username",
			shouldError:     false,
			expectedToken:   "bot_token",
			expectedChatIDs: []string{"@username"},
			expectedPreview: true,
		},
		{
			name:              "With query parameters",
			url:               "tgram://bot_token/chat_id?silent=yes&preview=no&format=html",
			shouldError:       false,
			expectedToken:     "bot_token",
			expectedChatIDs:   []string{"chat_id"},
			expectedSilent:    true,
			expectedPreview:   false,
			expectedParseMode: "HTML",
		},
		{
			name:              "MarkdownV2 format",
			url:               "telegram://bot_token/chat_id?format=markdownv2",
			shouldError:       false,
			expectedToken:     "bot_token",
			expectedChatIDs:   []string{"chat_id"},
			expectedPreview:   true,
			expectedParseMode: "MarkdownV2",
		},
		{
			name:        "Invalid scheme",
			url:         "discord://bot_token/chat_id",
			shouldError: true,
		},
		{
			name:        "Missing bot token",
			url:         "tgram:///chat_id",
			shouldError: true,
		},
		{
			name:        "Missing chat ID",
			url:         "tgram://bot_token/",
			shouldError: true,
		},
		{
			name:        "Empty bot token",
			url:         "tgram://",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh service instance for each test case
			service := NewTelegramService()

			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			telegramService := service.(*TelegramService)
			err = telegramService.ParseURL(parsedURL)

			if tc.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if telegramService.botToken != tc.expectedToken {
				t.Errorf("Expected bot token %q, got %q", tc.expectedToken, telegramService.botToken)
			}

			if !stringSlicesEqual(telegramService.chatIDs, tc.expectedChatIDs) {
				t.Errorf("Expected chat IDs %v, got %v", tc.expectedChatIDs, telegramService.chatIDs)
			}

			if telegramService.silent != tc.expectedSilent {
				t.Errorf("Expected silent %v, got %v", tc.expectedSilent, telegramService.silent)
			}

			if telegramService.preview != tc.expectedPreview {
				t.Errorf("Expected preview %v, got %v", tc.expectedPreview, telegramService.preview)
			}

			if tc.expectedParseMode != "" && telegramService.parseMode != tc.expectedParseMode {
				t.Errorf("Expected parse mode %q, got %q", tc.expectedParseMode, telegramService.parseMode)
			}
		})
	}
}

func TestTelegramEmojiMapping(t *testing.T) {
	service := NewTelegramService().(*TelegramService)

	testCases := []struct {
		notifyType    NotifyType
		expectedEmoji string
	}{
		{NotifyTypeInfo, "ℹ️"},
		{NotifyTypeSuccess, "✅"},
		{NotifyTypeWarning, "⚠️"},
		{NotifyTypeError, "❌"},
	}

	for _, tc := range testCases {
		emoji := service.getEmojiForNotifyType(tc.notifyType)
		if emoji != tc.expectedEmoji {
			t.Errorf("For notify type %s, expected emoji %q, got %q",
				tc.notifyType.String(), tc.expectedEmoji, emoji)
		}
	}
}

func TestTelegramMessageFormatting(t *testing.T) {
	service := NewTelegramService().(*TelegramService)

	testCases := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
		parseMode  string
		contains   []string
	}{
		{
			name:       "Plain text with title",
			title:      "Test Title",
			body:       "Test body",
			notifyType: NotifyTypeInfo,
			parseMode:  "",
			contains:   []string{"ℹ️", "Test Title", "Test body"},
		},
		{
			name:       "Markdown formatting",
			title:      "Bold Title",
			body:       "Normal body",
			notifyType: NotifyTypeSuccess,
			parseMode:  "Markdown",
			contains:   []string{"✅", "*Bold Title*", "Normal body"},
		},
		{
			name:       "HTML formatting",
			title:      "HTML Title",
			body:       "HTML body",
			notifyType: NotifyTypeError,
			parseMode:  "HTML",
			contains:   []string{"❌", "<b>HTML Title</b>", "HTML body"},
		},
		{
			name:       "No title",
			title:      "",
			body:       "Just body",
			notifyType: NotifyTypeWarning,
			parseMode:  "",
			contains:   []string{"⚠️", "Just body"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service.parseMode = tc.parseMode
			message := service.formatMessage(tc.title, tc.body, tc.notifyType)

			for _, expected := range tc.contains {
				if !strings.Contains(message, expected) {
					t.Errorf("Expected message to contain %q, got: %q", expected, message)
				}
			}
		})
	}
}

func TestTelegramMarkdownV2Escaping(t *testing.T) {
	service := NewTelegramService().(*TelegramService)

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Hello_world",
			expected: "Hello\\_world",
		},
		{
			input:    "Test*bold*text",
			expected: "Test\\*bold\\*text",
		},
		{
			input:    "URL: https://example.com",
			expected: "URL: https://example\\.com",
		},
		{
			input:    "Special chars: []()~`>#+-=|{}!",
			expected: "Special chars: \\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\!",
		},
	}

	for _, tc := range testCases {
		result := service.escapeMarkdownV2(tc.input)
		if result != tc.expected {
			t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestTelegramServiceCapabilities(t *testing.T) {
	service := NewTelegramService()

	if service.GetServiceID() != "telegram" {
		t.Errorf("Expected service ID 'telegram', got %q", service.GetServiceID())
	}

	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}

	if !service.SupportsAttachments() {
		t.Error("Telegram should support attachments")
	}

	expectedMaxLength := 4096
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestTelegramChatIDValidation(t *testing.T) {
	service := NewTelegramService().(*TelegramService)

	validChatIDs := []string{
		"123456789",
		"-123456789",
		"@username",
		"@channel_name",
	}

	for _, chatID := range validChatIDs {
		if !service.validateChatID(chatID) {
			t.Errorf("Chat ID %q should be valid", chatID)
		}
	}

	invalidChatIDs := []string{
		"@",
		"",
		"not_a_number_or_username",
	}

	for _, chatID := range invalidChatIDs {
		if service.validateChatID(chatID) {
			t.Errorf("Chat ID %q should be invalid", chatID)
		}
	}
}

func TestTelegramSendMethod(t *testing.T) {
	service := NewTelegramService()
	parsedURL, _ := url.Parse("tgram://test_bot_token/test_chat_id")
	_ = service.(*TelegramService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting the real Telegram API
	if err == nil {
		t.Error("Expected network error for invalid bot token, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "telegram") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "Not Found") {
		t.Errorf("Error should be network-related or API error, got: %v", err)
	}
}
