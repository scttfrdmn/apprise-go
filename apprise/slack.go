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

// SlackService implements Slack webhook and bot API notifications
type SlackService struct {
	// Webhook mode fields
	webhookTokenA string
	webhookTokenB string
	webhookTokenC string
	webhookURL    string

	// Bot mode fields
	botToken string

	// Common fields
	channel   string
	username  string
	iconURL   string
	iconEmoji string
	client    *http.Client
	mode      string // "webhook" or "bot"
}

// NewSlackService creates a new Slack service instance
func NewSlackService() Service {
	return &SlackService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *SlackService) GetServiceID() string {
	return "slack"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (s *SlackService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Slack service URL
// Webhook format: slack://TokenA/TokenB/TokenC/Channel
// Bot format: slack://bottoken/Channel
func (s *SlackService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "slack" {
		return fmt.Errorf("invalid scheme: expected 'slack', got '%s'", serviceURL.Scheme)
	}

	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")

	// Determine if this is webhook mode (3 tokens) or bot mode (1 token)
	if len(pathParts) >= 3 && pathParts[0] != "" && pathParts[1] != "" && pathParts[2] != "" {
		// Webhook mode: slack://TokenA/TokenB/TokenC[/Channel]
		s.mode = "webhook"
		s.webhookTokenA = pathParts[0]
		s.webhookTokenB = pathParts[1]
		s.webhookTokenC = pathParts[2]
		s.webhookURL = fmt.Sprintf("https://hooks.slack.com/services/%s/%s/%s",
			s.webhookTokenA, s.webhookTokenB, s.webhookTokenC)

		if len(pathParts) > 3 && pathParts[3] != "" {
			s.channel = pathParts[3]
		}
	} else if len(pathParts) >= 1 && pathParts[0] != "" {
		// Bot mode: slack://bottoken[/Channel]
		s.mode = "bot"
		s.botToken = pathParts[0]

		if len(pathParts) > 1 && pathParts[1] != "" {
			s.channel = pathParts[1]
		}
	} else {
		return fmt.Errorf("invalid Slack URL: missing required tokens")
	}

	// Parse query parameters for additional options
	query := serviceURL.Query()
	if username := query.Get("username"); username != "" {
		s.username = username
	}
	if iconURL := query.Get("icon_url"); iconURL != "" {
		s.iconURL = iconURL
	}
	if iconEmoji := query.Get("icon_emoji"); iconEmoji != "" {
		s.iconEmoji = iconEmoji
	}
	if channel := query.Get("channel"); channel != "" {
		s.channel = channel
	}

	return nil
}

// SlackWebhookPayload represents the Slack webhook payload structure
type SlackWebhookPayload struct {
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackBotPayload represents the Slack bot API payload structure
type SlackBotPayload struct {
	Channel     string            `json:"channel"`
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color     string                 `json:"color,omitempty"`
	Title     string                 `json:"title,omitempty"`
	Text      string                 `json:"text,omitempty"`
	Footer    string                 `json:"footer,omitempty"`
	Timestamp int64                  `json:"ts,omitempty"`
	Fields    []SlackAttachmentField `json:"fields,omitempty"`
}

// SlackAttachmentField represents a field in a Slack attachment
type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
}

// SlackText represents text in Slack blocks
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Send sends a notification to Slack
func (s *SlackService) Send(ctx context.Context, req NotificationRequest) error {
	if s.mode == "webhook" {
		return s.sendWebhook(ctx, req)
	}
	return s.sendBot(ctx, req)
}

// sendWebhook sends notification via Slack webhook
func (s *SlackService) sendWebhook(ctx context.Context, req NotificationRequest) error {
	color := s.getColorForNotifyType(req.NotifyType)

	payload := SlackWebhookPayload{
		Username:  s.username,
		IconURL:   s.iconURL,
		IconEmoji: s.iconEmoji,
		Channel:   s.channel,
	}

	// Create attachment if we have a title, otherwise use simple text
	if req.Title != "" {
		attachment := SlackAttachment{
			Color:  color,
			Title:  req.Title,
			Text:   req.Body,
			Footer: fmt.Sprintf("Type: %s", req.NotifyType.String()),
		}
		payload.Attachments = []SlackAttachment{attachment}
	} else {
		payload.Text = req.Body
	}

	return s.sendPayload(ctx, s.webhookURL, payload)
}

// sendBot sends notification via Slack bot API
func (s *SlackService) sendBot(ctx context.Context, req NotificationRequest) error {
	color := s.getColorForNotifyType(req.NotifyType)

	payload := SlackBotPayload{
		Channel:   s.channel,
		Username:  s.username,
		IconURL:   s.iconURL,
		IconEmoji: s.iconEmoji,
	}

	// Create attachment if we have a title, otherwise use simple text
	if req.Title != "" {
		attachment := SlackAttachment{
			Color:  color,
			Title:  req.Title,
			Text:   req.Body,
			Footer: fmt.Sprintf("Type: %s", req.NotifyType.String()),
		}
		payload.Attachments = []SlackAttachment{attachment}
	} else {
		payload.Text = req.Body
	}

	apiURL := "https://slack.com/api/chat.postMessage"
	return s.sendBotPayload(ctx, apiURL, payload)
}

// sendPayload sends a webhook payload to Slack
func (s *SlackService) sendPayload(ctx context.Context, webhookURL string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack API error (status %d): %s", resp.StatusCode, string(body))
	}

	// For webhooks, check if response is "ok"
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		return fmt.Errorf("Slack webhook error: %s", string(body))
	}

	return nil
}

// sendBotPayload sends a bot API payload to Slack
func (s *SlackService) sendBotPayload(ctx context.Context, apiURL string, payload SlackBotPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack bot payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.botToken))
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Slack bot notification: %w", err)
	}
	defer resp.Body.Close()

	// Parse response for bot API
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Slack response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("Slack API error: %s", result.Error)
	}

	return nil
}

// TestURL validates a Slack service URL
func (s *SlackService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return s.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Slack supports file attachments
func (s *SlackService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Slack's message length limit
func (s *SlackService) GetMaxBodyLength() int {
	return 4000 // Slack's character limit for messages
}

// getColorForNotifyType returns appropriate color for notification type
func (s *SlackService) getColorForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "good" // Green
	case NotifyTypeWarning:
		return "warning" // Yellow
	case NotifyTypeError:
		return "danger" // Red
	case NotifyTypeInfo:
		fallthrough
	default:
		return "#36a64f" // Blue/Info color
	}
}

// Example usage and URL formats:
// slack://TokenA/TokenB/TokenC                     (webhook to default channel)
// slack://TokenA/TokenB/TokenC/general            (webhook to #general)
// slack://TokenA/TokenB/TokenC/@username          (webhook to user)
// slack://bottoken/general                        (bot to #general)
// slack://bottoken/@username                      (bot to user)
// slack://TokenA/TokenB/TokenC?username=MyBot&icon_emoji=:ghost:
