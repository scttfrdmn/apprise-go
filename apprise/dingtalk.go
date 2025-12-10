package apprise

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DingTalkService implements DingTalk (钉钉) webhook notification service
// DingTalk is Alibaba's enterprise collaboration platform with 500M+ users in China
// Dominant in Chinese enterprise market, similar to Lark/Feishu
type DingTalkService struct {
	webhookURL string // Webhook URL with access token
	secret     string // Secret key for signature verification (optional)
	atAll      bool   // @all users in the group
	atMobiles  []string // Mobile numbers to @mention
	client     *http.Client
}

// DingTalkRequest represents the webhook request payload
type DingTalkRequest struct {
	MsgType  string                 `json:"msgtype"` // "text", "markdown", "link"
	Text     *DingTalkText          `json:"text,omitempty"`
	Markdown *DingTalkMarkdown      `json:"markdown,omitempty"`
	At       *DingTalkAt            `json:"at,omitempty"`
}

// DingTalkText represents a text message
type DingTalkText struct {
	Content string `json:"content"`
}

// DingTalkMarkdown represents a markdown message
type DingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkAt represents @ mention configuration
type DingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll"`
}

// DingTalkResponse represents the webhook response
type DingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// NewDingTalkService creates a new DingTalk service instance
func NewDingTalkService() Service {
	return &DingTalkService{
		atAll: false,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (d *DingTalkService) GetServiceID() string {
	return "dingtalk"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (d *DingTalkService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a DingTalk service URL
// Format: dingtalk://access_token
// Format: dingtalk://access_token?secret=SEC123&atall=true
// Format: dingtalk://oapi.dingtalk.com/robot/send?access_token=TOKEN&secret=SEC
//
// Parameters:
//   - access_token: Webhook access token (required)
//
// Query Parameters:
//   - secret=key - Secret key for signature verification
//   - atall=true|false - @all users in group (default: false)
//   - atmobile=phone1,phone2 - Mobile numbers to @mention (comma-separated)
//
// Examples:
//   dingtalk://a1b2c3d4e5f6
//   dingtalk://token?secret=SEC123abc
//   dingtalk://token?atall=true
//   dingtalk://token?atmobile=13800138000,13900139000
//   dingtalk://oapi.dingtalk.com/robot/send?access_token=abc123&secret=SEC456
func (d *DingTalkService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "dingtalk" && scheme != "dingding" {
		return fmt.Errorf("invalid scheme: expected 'dingtalk' or 'dingding', got '%s'", scheme)
	}

	// Check if this is a full webhook URL or just a token
	host := serviceURL.Host
	if host == "oapi.dingtalk.com" {
		// Full webhook URL format: dingtalk://oapi.dingtalk.com/robot/send?access_token=xxx
		query := serviceURL.Query()
		accessToken := query.Get("access_token")
		if accessToken == "" {
			return fmt.Errorf("missing access_token in webhook URL")
		}
		d.webhookURL = fmt.Sprintf("https://oapi.dingtalk.com%s?access_token=%s", serviceURL.Path, accessToken)

		// Parse secret from query
		if secret := query.Get("secret"); secret != "" {
			d.secret = secret
		}
	} else {
		// Token-only format: dingtalk://token
		accessToken := host
		if accessToken == "" {
			return fmt.Errorf("missing DingTalk access token")
		}
		d.webhookURL = fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", accessToken)

		// Parse query parameters
		query := serviceURL.Query()

		// Parse secret
		if secret := query.Get("secret"); secret != "" {
			d.secret = secret
		}
	}

	// Parse common query parameters
	query := serviceURL.Query()

	// Parse @all parameter
	if atAllStr := query.Get("atall"); atAllStr != "" {
		d.atAll = atAllStr == "true"
	}

	// Parse @mobile parameter
	if atMobileStr := query.Get("atmobile"); atMobileStr != "" {
		mobiles := strings.Split(atMobileStr, ",")
		d.atMobiles = make([]string, 0, len(mobiles))
		for _, mobile := range mobiles {
			if trimmed := strings.TrimSpace(mobile); trimmed != "" {
				d.atMobiles = append(d.atMobiles, trimmed)
			}
		}
	}

	return nil
}

// Send sends a notification via DingTalk webhook
func (d *DingTalkService) Send(ctx context.Context, req NotificationRequest) error {
	// Build request payload
	payload := d.buildRequest(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build final webhook URL with signature if secret is configured
	webhookURL := d.webhookURL
	if d.secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := d.generateSignature(timestamp, d.secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var dtResp DingTalkResponse
	if err := json.NewDecoder(resp.Body).Decode(&dtResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if dtResp.ErrCode != 0 {
		return fmt.Errorf("dingtalk error: %s (code: %d)", dtResp.ErrMsg, dtResp.ErrCode)
	}

	return nil
}

// buildRequest constructs the DingTalk webhook request
func (d *DingTalkService) buildRequest(req NotificationRequest) DingTalkRequest {
	payload := DingTalkRequest{
		MsgType: "markdown", // Use markdown for rich formatting
	}

	// Build markdown content
	var contentBuilder strings.Builder

	// Add title with emoji indicator
	emoji := d.getEmojiForNotifyType(req.NotifyType)
	if req.Title != "" {
		contentBuilder.WriteString(fmt.Sprintf("## %s %s\n\n", emoji, req.Title))
	} else if req.Body != "" {
		// Use first line of body as title
		lines := strings.Split(req.Body, "\n")
		title := lines[0]
		if len(title) > 100 {
			title = title[:100] + "..."
		}
		contentBuilder.WriteString(fmt.Sprintf("## %s %s\n\n", emoji, title))
	} else {
		contentBuilder.WriteString(fmt.Sprintf("## %s Notification\n\n", emoji))
	}

	// Add body content
	if req.Body != "" {
		contentBuilder.WriteString(req.Body)
		contentBuilder.WriteString("\n\n")
	}

	// Add tags if present
	if len(req.Tags) > 0 {
		contentBuilder.WriteString("**Tags**: ")
		contentBuilder.WriteString(strings.Join(req.Tags, ", "))
		contentBuilder.WriteString("\n\n")
	}

	// Add timestamp
	contentBuilder.WriteString(fmt.Sprintf("*%s*", time.Now().Format("2006-01-02 15:04:05")))

	payload.Markdown = &DingTalkMarkdown{
		Title: d.getMarkdownTitle(req),
		Text:  contentBuilder.String(),
	}

	// Add @mentions if configured
	if d.atAll || len(d.atMobiles) > 0 {
		payload.At = &DingTalkAt{
			IsAtAll:   d.atAll,
			AtMobiles: d.atMobiles,
		}
	}

	return payload
}

// getMarkdownTitle generates a title for the markdown message
func (d *DingTalkService) getMarkdownTitle(req NotificationRequest) string {
	if req.Title != "" {
		return req.Title
	}
	if req.Body != "" {
		// Use first line as title
		lines := strings.Split(req.Body, "\n")
		title := lines[0]
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		return title
	}
	return "Notification"
}

// getEmojiForNotifyType returns an emoji indicator for the notification type
func (d *DingTalkService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "🔴" // Red circle for errors
	case NotifyTypeWarning:
		return "⚠️" // Warning sign
	case NotifyTypeSuccess:
		return "✅" // Check mark for success
	case NotifyTypeInfo:
		return "ℹ️" // Information icon
	default:
		return "📢" // Megaphone for general notifications
	}
}

// generateSignature generates HMAC-SHA256 signature for webhook security
func (d *DingTalkService) generateSignature(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

// TestURL validates the DingTalk service URL format
func (d *DingTalkService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return d.ParseURL(parsedURL)
}

// SupportsAttachments returns false as DingTalk webhook doesn't support direct file attachments
// Note: Can send image links in markdown format
func (d *DingTalkService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 20000 (DingTalk's markdown content limit)
func (d *DingTalkService) GetMaxBodyLength() int {
	return 20000
}
