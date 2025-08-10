package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// TelegramService implements Telegram Bot API notifications
type TelegramService struct {
	botToken  string
	chatIDs   []string
	silent    bool
	preview   bool
	parseMode string
	threadID  string
	client    *http.Client
}

// NewTelegramService creates a new Telegram service instance
func NewTelegramService() Service {
	return &TelegramService{
		client:    &http.Client{},
		preview:   true,       // Enable web page preview by default
		parseMode: "Markdown", // Default to Markdown parsing
	}
}

// GetServiceID returns the service identifier
func (t *TelegramService) GetServiceID() string {
	return "telegram"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (t *TelegramService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Telegram service URL
// Format: tgram://bot_token/chat_id[/chat_id2/...]
// Format: telegram://bot_token/chat_id[/chat_id2/...]
func (t *TelegramService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "tgram" && serviceURL.Scheme != "telegram" {
		return fmt.Errorf("invalid scheme: expected 'tgram' or 'telegram', got '%s'", serviceURL.Scheme)
	}

	// Extract bot token from host
	t.botToken = serviceURL.Host
	if t.botToken == "" {
		// Try extracting from user info (alternative format)
		if serviceURL.User != nil {
			t.botToken = serviceURL.User.Username()
		}
	}

	if t.botToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}

	// Extract chat IDs from path
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	for _, part := range pathParts {
		if part != "" {
			t.chatIDs = append(t.chatIDs, part)
		}
	}

	if len(t.chatIDs) == 0 {
		return fmt.Errorf("at least one Telegram chat ID is required")
	}

	// Parse query parameters
	query := serviceURL.Query()

	if silent := query.Get("silent"); silent != "" {
		t.silent = strings.ToLower(silent) == "yes" || strings.ToLower(silent) == "true"
	}

	if preview := query.Get("preview"); preview != "" {
		t.preview = strings.ToLower(preview) == "yes" || strings.ToLower(preview) == "true"
	}

	if parseMode := query.Get("format"); parseMode != "" {
		switch strings.ToLower(parseMode) {
		case "html":
			t.parseMode = "HTML"
		case "markdown", "md":
			t.parseMode = "Markdown"
		case "markdownv2", "mdv2":
			t.parseMode = "MarkdownV2"
		default:
			t.parseMode = "" // No parsing
		}
	}

	if threadID := query.Get("thread"); threadID != "" {
		t.threadID = threadID
	}

	return nil
}

// TelegramMessage represents a Telegram API message
type TelegramMessage struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
	MessageThreadID       string `json:"message_thread_id,omitempty"`
}

// TelegramResponse represents the Telegram API response
type TelegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

// Send sends a notification to Telegram
func (t *TelegramService) Send(ctx context.Context, req NotificationRequest) error {
	// Combine title and body
	message := t.formatMessage(req.Title, req.Body, req.NotifyType)

	// Send to each chat ID
	var lastError error
	successCount := 0

	for _, chatID := range t.chatIDs {
		if err := t.sendToChat(ctx, chatID, message); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Return error only if all sends failed
	if successCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}

// sendToChat sends a message to a specific Telegram chat
func (t *TelegramService) sendToChat(ctx context.Context, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	payload := TelegramMessage{
		ChatID:                chatID,
		Text:                  message,
		ParseMode:             t.parseMode,
		DisableWebPagePreview: !t.preview,
		DisableNotification:   t.silent,
		MessageThreadID:       t.threadID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Telegram notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var result TelegramResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Telegram response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error (%d): %s", result.ErrorCode, result.Description)
	}

	return nil
}

// formatMessage formats the title and body into a single message
func (t *TelegramService) formatMessage(title, body string, notifyType NotifyType) string {
	var message strings.Builder

	// Add notification type emoji based on type
	emoji := t.getEmojiForNotifyType(notifyType)
	if emoji != "" {
		message.WriteString(emoji + " ")
	}

	// Add title if present
	if title != "" {
		switch t.parseMode {
		case "HTML":
			message.WriteString(fmt.Sprintf("<b>%s</b>\n", title))
		case "Markdown":
			message.WriteString(fmt.Sprintf("*%s*\n", title))
		case "MarkdownV2":
			// Escape special characters for MarkdownV2
			title = t.escapeMarkdownV2(title)
			message.WriteString(fmt.Sprintf("*%s*\n", title))
		default:
			message.WriteString(fmt.Sprintf("%s\n", title))
		}
	}

	// Add body
	if body != "" {
		if t.parseMode == "MarkdownV2" {
			body = t.escapeMarkdownV2(body)
		}
		message.WriteString(body)
	}

	return message.String()
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (t *TelegramService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "✅"
	case NotifyTypeWarning:
		return "⚠️"
	case NotifyTypeError:
		return "❌"
	case NotifyTypeInfo:
		return "ℹ️"
	default:
		return ""
	}
}

// escapeMarkdownV2 escapes special characters for Telegram's MarkdownV2
func (t *TelegramService) escapeMarkdownV2(text string) string {
	// Characters that need to be escaped in MarkdownV2
	specialChars := []string{
		"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!",
	}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	return result
}

// TestURL validates a Telegram service URL
func (t *TelegramService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return t.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Telegram supports file attachments
func (t *TelegramService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Telegram's message length limit
func (t *TelegramService) GetMaxBodyLength() int {
	return 4096 // Telegram's character limit for messages
}

// validateChatID validates that a chat ID is in the correct format
func (t *TelegramService) validateChatID(chatID string) bool {
	// Chat ID can be:
	// - Numeric (positive or negative)
	// - Username starting with @
	// - Channel username

	if strings.HasPrefix(chatID, "@") {
		return len(chatID) > 1
	}

	if _, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		return true
	}

	return false
}

// Example usage and URL formats:
// tgram://bot_token/chat_id
// tgram://bot_token/@username
// tgram://bot_token/chat_id1/chat_id2
// tgram://bot_token/chat_id?silent=yes&preview=no
// telegram://bot_token/chat_id?format=html&thread=123
