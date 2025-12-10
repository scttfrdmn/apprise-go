package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// LarkService implements Lark (Feishu) custom bot notifications
// Lark is ByteDance's enterprise collaboration platform (also known as Feishu in China)
// Supports webhook-based notifications with text messages
type LarkService struct {
	token       string // 32-character webhook token
	webhookURL  string // Full webhook URL
	client      *http.Client
}

// LarkMessagePayload represents the Lark webhook message format
// Lark supports multiple message types, but text is the most common
type LarkMessagePayload struct {
	MsgType string          `json:"msg_type"`
	Content LarkTextContent `json:"content"`
}

// LarkTextContent represents text message content
type LarkTextContent struct {
	Text string `json:"text"`
}

// LarkRichTextPayload represents rich text message format (alternative)
type LarkRichTextPayload struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Post map[string]interface{} `json:"post"`
	} `json:"content"`
}

// NewLarkService creates a new Lark service instance
func NewLarkService() Service {
	return &LarkService{
		client: GetWebhookHTTPClient("lark"),
	}
}

// GetServiceID returns the service identifier
func (l *LarkService) GetServiceID() string {
	return "lark"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (l *LarkService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Lark webhook URL
// Format: lark://token
// Format: lark://open.larksuite.com/open-apis/bot/v2/hook/token
// Format: feishu://token (alias for China market)
//
// The token is a 32-character webhook integration key
//
// Examples:
//   lark://1234567890abcdef1234567890abcdef
//   feishu://1234567890abcdef1234567890abcdef
//   lark://open.larksuite.com/open-apis/bot/v2/hook/1234567890abcdef1234567890abcdef
func (l *LarkService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "lark" && scheme != "feishu" {
		return fmt.Errorf("invalid scheme: expected 'lark' or 'feishu', got '%s'", scheme)
	}

	// Check if it's a full webhook URL format
	if serviceURL.Host == "open.larksuite.com" || serviceURL.Host == "open.feishu.cn" {
		// Full URL format: lark://open.larksuite.com/open-apis/bot/v2/hook/token
		l.webhookURL = fmt.Sprintf("https://%s%s", serviceURL.Host, serviceURL.Path)

		// Extract token from path
		parts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		if len(parts) > 0 {
			l.token = parts[len(parts)-1]
		}
	} else {
		// Simplified format: lark://token or lark://token@host
		l.token = serviceURL.Host
		if l.token == "" && serviceURL.Path != "" {
			l.token = strings.Trim(serviceURL.Path, "/")
		}

		// Build full webhook URL
		// Default to international Lark instance
		domain := "open.larksuite.com"
		if scheme == "feishu" {
			domain = "open.feishu.cn" // China instance
		}

		l.webhookURL = fmt.Sprintf("https://%s/open-apis/bot/v2/hook/%s", domain, l.token)
	}

	// Validate token
	if l.token == "" {
		return fmt.Errorf("missing webhook token")
	}

	// Token should be 32 characters (typical format)
	if len(l.token) < 20 {
		return fmt.Errorf("invalid token format: too short (expected ~32 characters)")
	}

	return nil
}

// Send sends a notification via Lark webhook
func (l *LarkService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message text
	messageText := l.buildMessageText(req)

	// Create payload
	payload := LarkMessagePayload{
		MsgType: "text",
		Content: LarkTextContent{
			Text: messageText,
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Lark payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", l.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Send request
	resp, err := l.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error messages
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	// Lark returns JSON response with status code
	var larkResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(body, &larkResp); err == nil {
		if larkResp.Code != 0 {
			return fmt.Errorf("Lark API error (code %d): %s", larkResp.Code, larkResp.Msg)
		}
	}

	return nil
}

// buildMessageText constructs the notification message text
func (l *LarkService) buildMessageText(req NotificationRequest) string {
	var builder strings.Builder

	// Add emoji indicator based on notification type
	emoji := l.getNotificationEmoji(req.NotifyType)
	if emoji != "" {
		builder.WriteString(emoji)
		builder.WriteString(" ")
	}

	// Add title if present
	if req.Title != "" {
		builder.WriteString(req.Title)
		if req.Body != "" {
			builder.WriteString("\n\n")
		}
	}

	// Add body
	if req.Body != "" {
		builder.WriteString(req.Body)
	}

	// Add tags if present
	if len(req.Tags) > 0 {
		builder.WriteString("\n\nTags: ")
		builder.WriteString(strings.Join(req.Tags, ", "))
	}

	message := builder.String()

	// Lark has a character limit of ~20,000
	if len(message) > 20000 {
		message = message[:19997] + "..."
	}

	return message
}

// getNotificationEmoji returns an emoji based on notification type
func (l *LarkService) getNotificationEmoji(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeInfo:
		return "ℹ️"
	case NotifyTypeSuccess:
		return "✅"
	case NotifyTypeWarning:
		return "⚠️"
	case NotifyTypeError:
		return "🚨"
	default:
		return ""
	}
}

// TestURL validates the Lark webhook URL format
func (l *LarkService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return l.ParseURL(parsedURL)
}

// SupportsAttachments returns false as Lark text webhooks don't support attachments
// Note: Lark does support rich content through interactive cards, but not via simple webhooks
func (l *LarkService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns the maximum message length (20,000 characters)
func (l *LarkService) GetMaxBodyLength() int {
	return 20000
}
