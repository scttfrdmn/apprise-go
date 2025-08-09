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

// DiscordService implements Discord webhook notifications
type DiscordService struct {
	webhookID    string
	webhookToken string
	avatar       string
	username     string
	client       *http.Client
}

// NewDiscordService creates a new Discord service instance
func NewDiscordService() Service {
	return &DiscordService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (d *DiscordService) GetServiceID() string {
	return "discord"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (d *DiscordService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Discord webhook URL
// Format: discord://avatar@webhook_id/webhook_token
// Format: discord://webhook_id/webhook_token
func (d *DiscordService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "discord" {
		return fmt.Errorf("invalid scheme: expected 'discord', got '%s'", serviceURL.Scheme)
	}

	// Extract avatar from user info if present
	if serviceURL.User != nil {
		d.avatar = serviceURL.User.Username()
	}

	// Extract webhook ID and token from path
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return fmt.Errorf("invalid Discord URL: missing webhook_id and/or webhook_token")
	}

	d.webhookID = pathParts[0]
	d.webhookToken = pathParts[1]

	// Parse query parameters for additional options
	query := serviceURL.Query()
	if username := query.Get("username"); username != "" {
		d.username = username
	}
	if avatar := query.Get("avatar"); avatar != "" {
		d.avatar = avatar
	}

	if d.webhookID == "" || d.webhookToken == "" {
		return fmt.Errorf("Discord webhook ID and token are required")
	}

	return nil
}

// DiscordWebhookPayload represents the Discord webhook payload structure
type DiscordWebhookPayload struct {
	Content   string                   `json:"content,omitempty"`
	Username  string                   `json:"username,omitempty"`
	AvatarURL string                   `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed          `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed object
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedFooter represents the footer of a Discord embed
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedAuthor represents the author of a Discord embed
type DiscordEmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Send sends a notification to Discord
func (d *DiscordService) Send(ctx context.Context, req NotificationRequest) error {
	webhookURL := fmt.Sprintf("https://discord.com/api/webhooks/%s/%s", d.webhookID, d.webhookToken)

	// Determine embed color based on notification type
	color := d.getColorForNotifyType(req.NotifyType)

	payload := DiscordWebhookPayload{
		Username:  d.username,
		AvatarURL: d.avatar,
	}

	// Create embed if we have a title, otherwise use simple content
	if req.Title != "" {
		embed := DiscordEmbed{
			Title:       req.Title,
			Description: req.Body,
			Color:       color,
		}

		// Add notification type as footer
		embed.Footer = &DiscordEmbedFooter{
			Text: fmt.Sprintf("Type: %s", req.NotifyType.String()),
		}

		payload.Embeds = []DiscordEmbed{embed}
	} else {
		// Use simple content for body-only messages
		payload.Content = req.Body
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Go-Apprise/1.0")

	// Send request
	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Discord API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a Discord service URL
func (d *DiscordService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return d.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Discord supports file attachments
func (d *DiscordService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Discord's message length limit
func (d *DiscordService) GetMaxBodyLength() int {
	return 2000 // Discord's character limit for messages
}

// getColorForNotifyType returns appropriate color for notification type
func (d *DiscordService) getColorForNotifyType(notifyType NotifyType) int {
	switch notifyType {
	case NotifyTypeSuccess:
		return 0x00FF00 // Green
	case NotifyTypeWarning:
		return 0xFFFF00 // Yellow
	case NotifyTypeError:
		return 0xFF0000 // Red
	case NotifyTypeInfo:
		fallthrough
	default:
		return 0x0099FF // Blue
	}
}

// Example usage and URL formats:
// discord://webhook_id/webhook_token
// discord://avatar@webhook_id/webhook_token
// discord://webhook_id/webhook_token?username=MyBot
// discord://avatar@webhook_id/webhook_token?username=MyBot&avatar=https://example.com/avatar.png