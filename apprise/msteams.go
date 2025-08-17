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

// MSTeamsService implements Microsoft Teams webhook notifications
type MSTeamsService struct {
	teamName     string
	tokenA       string
	tokenB       string
	tokenC       string
	tokenD       string // Optional for version 3
	webhookURL   string
	version      int // 1 = legacy, 2 = modern, 3 = version 3
	includeImage bool
	client       *http.Client
}

// NewMSTeamsService creates a new Microsoft Teams service instance
func NewMSTeamsService() Service {
	return &MSTeamsService{
		client:       GetWebhookHTTPClient("msteams"),
		includeImage: true,
		version:      2, // Default to modern version
	}
}

// GetServiceID returns the service identifier
func (m *MSTeamsService) GetServiceID() string {
	return "msteams"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (m *MSTeamsService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Microsoft Teams webhook URL
// Legacy format: msteams://token_a/token_b/token_c
// Modern format: msteams://team_name/token_a/token_b/token_c
// Version 3 format: msteams://team_name/token_a/token_b/token_c/token_d
func (m *MSTeamsService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "msteams" {
		return fmt.Errorf("invalid scheme: expected 'msteams', got '%s'", serviceURL.Scheme)
	}

	// Extract tokens from path
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")

	// Determine version based on path structure and host
	if serviceURL.Host != "" {
		// Modern format: host is team name
		m.teamName = serviceURL.Host
		m.version = 2

		if len(pathParts) < 3 {
			return fmt.Errorf("insufficient tokens for modern MS Teams format")
		}

		m.tokenA = pathParts[0]
		m.tokenB = pathParts[1]
		m.tokenC = pathParts[2]

		// Check for version 3 (4th token)
		if len(pathParts) >= 4 && pathParts[3] != "" {
			m.tokenD = pathParts[3]
			m.version = 3
		}

	} else {
		// Legacy format: no host, tokens start from beginning
		m.version = 1

		if len(pathParts) < 3 {
			return fmt.Errorf("insufficient tokens for legacy MS Teams format")
		}

		m.tokenA = pathParts[0]
		m.tokenB = pathParts[1]
		m.tokenC = pathParts[2]
	}

	// Build webhook URL based on version
	m.webhookURL = m.buildWebhookURL()

	// Parse query parameters
	query := serviceURL.Query()
	if includeImage := query.Get("image"); includeImage != "" {
		m.includeImage = strings.ToLower(includeImage) == "yes" || strings.ToLower(includeImage) == "true"
	}

	return nil
}

// buildWebhookURL constructs the appropriate webhook URL based on version
func (m *MSTeamsService) buildWebhookURL() string {
	switch m.version {
	case 1:
		// Legacy format
		return fmt.Sprintf("https://outlook.office.com/webhook/%s/IncomingWebhook/%s/%s",
			m.tokenA, m.tokenB, m.tokenC)
	case 2:
		// Modern format
		return fmt.Sprintf("https://%s.webhook.office.com/webhookb2/%s/IncomingWebhook/%s/%s",
			m.teamName, m.tokenA, m.tokenB, m.tokenC)
	case 3:
		// Version 3 with additional token
		return fmt.Sprintf("https://%s.webhook.office.com/webhookb2/%s/IncomingWebhook/%s/%s/%s",
			m.teamName, m.tokenA, m.tokenB, m.tokenC, m.tokenD)
	default:
		return ""
	}
}

// MSTeamsPayload represents the Microsoft Teams webhook payload structure
type MSTeamsPayload struct {
	Type            string              `json:"@type"`
	Context         string              `json:"@context"`
	Summary         string              `json:"summary"`
	ThemeColor      string              `json:"themeColor"`
	Sections        []MSTeamsSection    `json:"sections"`
	PotentialAction []MSTeamsAction     `json:"potentialAction,omitempty"`
	Attachments     []MSTeamsAttachment `json:"attachments,omitempty"`
}

// MSTeamsSection represents a section in the Teams message
type MSTeamsSection struct {
	ActivityTitle    string `json:"activityTitle,omitempty"`
	ActivitySubtitle string `json:"activitySubtitle,omitempty"`
	ActivityImage    string `json:"activityImage,omitempty"`
	Text             string `json:"text"`
	Markdown         bool   `json:"markdown,omitempty"`
}

// MSTeamsAction represents a potential action in the Teams message
type MSTeamsAction struct {
	Type    string                `json:"@type"`
	Name    string                `json:"name"`
	Targets []MSTeamsActionTarget `json:"targets,omitempty"`
}

// MSTeamsActionTarget represents an action target
type MSTeamsActionTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// MSTeamsAttachment represents a file attachment in Teams message
type MSTeamsAttachment struct {
	ContentType  string      `json:"contentType"`
	ContentURL   string      `json:"contentUrl,omitempty"`
	Name         string      `json:"name"`
	ThumbnailURL string      `json:"thumbnailUrl,omitempty"`
	Content      interface{} `json:"content,omitempty"`
}

// MSTeamsAdaptiveCard represents an Adaptive Card for rich content
type MSTeamsAdaptiveCard struct {
	Type    string               `json:"$schema"`
	Version string               `json:"version"`
	Body    []MSTeamsCardElement `json:"body"`
}

// MSTeamsCardElement represents an element in an Adaptive Card
type MSTeamsCardElement struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	URL     string `json:"url,omitempty"`
	AltText string `json:"altText,omitempty"`
	Size    string `json:"size,omitempty"`
	Style   string `json:"style,omitempty"`
	Weight  string `json:"weight,omitempty"`
}

// Send sends a notification to Microsoft Teams
func (m *MSTeamsService) Send(ctx context.Context, req NotificationRequest) error {
	// Check if we have attachments
	hasAttachments := req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0

	if hasAttachments {
		// Send with attachments using Adaptive Cards
		return m.sendWithAttachments(ctx, req)
	}

	// Send standard message without attachments
	return m.sendStandardMessage(ctx, req)
}

// sendStandardMessage sends a regular Teams message without attachments
func (m *MSTeamsService) sendStandardMessage(ctx context.Context, req NotificationRequest) error {
	// Create the Teams message payload
	payload := MSTeamsPayload{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    m.createSummary(req.Title, req.Body),
		ThemeColor: m.getColorForNotifyType(req.NotifyType),
		Sections:   []MSTeamsSection{m.createSection(req)},
	}

	return m.sendPayload(ctx, payload)
}

// sendWithAttachments sends a Teams message with file attachments using Adaptive Cards
func (m *MSTeamsService) sendWithAttachments(ctx context.Context, req NotificationRequest) error {
	// Create Adaptive Card with attachments
	card := m.createAdaptiveCard(req)

	// Create attachment containing the Adaptive Card
	attachment := MSTeamsAttachment{
		ContentType: "application/vnd.microsoft.card.adaptive",
		Content:     card,
	}

	// Create payload with Adaptive Card attachment
	payload := MSTeamsPayload{
		Type:        "message",
		Summary:     m.createSummary(req.Title, req.Body),
		Attachments: []MSTeamsAttachment{attachment},
	}

	// Add file attachments as additional attachments
	if req.AttachmentMgr != nil {
		fileAttachments, err := m.createFileAttachments(req.AttachmentMgr)
		if err != nil {
			return fmt.Errorf("failed to create file attachments: %w", err)
		}
		payload.Attachments = append(payload.Attachments, fileAttachments...)
	}

	return m.sendPayload(ctx, payload)
}

// sendPayload sends the payload to Teams webhook
func (m *MSTeamsService) sendPayload(ctx context.Context, payload MSTeamsPayload) error {
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Teams payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Send request
	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("teams API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Teams typically returns "1" for success
	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "1" {
		return fmt.Errorf("teams webhook error: %s", string(body))
	}

	return nil
}

// createSection creates a Teams message section
func (m *MSTeamsService) createSection(req NotificationRequest) MSTeamsSection {
	section := MSTeamsSection{
		Text:     req.Body,
		Markdown: req.BodyFormat == "markdown",
	}

	if req.Title != "" {
		section.ActivityTitle = req.Title
	}

	// Add activity image based on notification type if enabled
	if m.includeImage {
		section.ActivityImage = m.getImageForNotifyType(req.NotifyType)
	}

	return section
}

// createSummary creates a summary for the Teams message
func (m *MSTeamsService) createSummary(title, body string) string {
	if title != "" {
		return title
	}

	// Truncate body for summary if no title
	const maxSummaryLength = 100
	if len(body) <= maxSummaryLength {
		return body
	}

	return body[:maxSummaryLength] + "..."
}

// getColorForNotifyType returns appropriate theme color for notification type
func (m *MSTeamsService) getColorForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "00FF00" // Green
	case NotifyTypeWarning:
		return "FFFF00" // Yellow
	case NotifyTypeError:
		return "FF0000" // Red
	case NotifyTypeInfo:
		fallthrough
	default:
		return "0078D4" // Microsoft Blue
	}
}

// getImageForNotifyType returns appropriate activity image URL for notification type
func (m *MSTeamsService) getImageForNotifyType(notifyType NotifyType) string {
	// Using Microsoft's own icon set from their CDN
	baseURL := "https://cdn.jsdelivr.net/gh/microsoft/fluentui-emoji@main/assets"

	switch notifyType {
	case NotifyTypeSuccess:
		return baseURL + "/Check-mark-button/3D/check_mark_button_3d.png"
	case NotifyTypeWarning:
		return baseURL + "/Warning/3D/warning_3d.png"
	case NotifyTypeError:
		return baseURL + "/Cross-mark/3D/cross_mark_3d.png"
	case NotifyTypeInfo:
		fallthrough
	default:
		return baseURL + "/Information/3D/information_3d.png"
	}
}

// TestURL validates a Microsoft Teams service URL
func (m *MSTeamsService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return m.ParseURL(parsedURL)
}

// SupportsAttachments returns true for Teams webhooks with Adaptive Cards
func (m *MSTeamsService) SupportsAttachments() bool {
	return true // Supported via Adaptive Cards and file attachments
}

// GetMaxBodyLength returns Teams' message length limit
func (m *MSTeamsService) GetMaxBodyLength() int {
	return 28000 // Teams has a high character limit
}

// createAdaptiveCard creates an Adaptive Card for rich content with attachments
func (m *MSTeamsService) createAdaptiveCard(req NotificationRequest) MSTeamsAdaptiveCard {
	card := MSTeamsAdaptiveCard{
		Type:    "http://adaptivecards.io/schemas/adaptive-card.json",
		Version: "1.2",
		Body:    []MSTeamsCardElement{},
	}

	// Add title if present
	if req.Title != "" {
		titleElement := MSTeamsCardElement{
			Type:   "TextBlock",
			Text:   req.Title,
			Weight: "Bolder",
			Size:   "Medium",
		}
		card.Body = append(card.Body, titleElement)
	}

	// Add body text
	if req.Body != "" {
		bodyElement := MSTeamsCardElement{
			Type: "TextBlock",
			Text: req.Body,
		}
		card.Body = append(card.Body, bodyElement)
	}

	// Add notification type indicator
	emoji := m.getEmojiForNotifyType(req.NotifyType)
	typeElement := MSTeamsCardElement{
		Type:   "TextBlock",
		Text:   fmt.Sprintf("%s **%s**", emoji, req.NotifyType.String()),
		Style:  "emphasis",
		Weight: "Lighter",
		Size:   "Small",
	}
	card.Body = append(card.Body, typeElement)

	return card
}

// createFileAttachments creates file attachments from the attachment manager
func (m *MSTeamsService) createFileAttachments(attachmentMgr *AttachmentManager) ([]MSTeamsAttachment, error) {
	var attachments []MSTeamsAttachment

	if attachmentMgr == nil {
		return attachments, nil
	}

	files := attachmentMgr.GetAll()
	for _, file := range files {
		if !file.Exists() {
			continue // Skip non-existent files
		}

		// Create Teams attachment
		attachment := MSTeamsAttachment{
			Name:        file.GetName(),
			ContentType: file.GetMimeType(),
		}

		// For images, add as inline content with data URL
		if strings.HasPrefix(file.GetMimeType(), "image/") {
			base64Content, err := file.Base64()
			if err != nil {
				return nil, fmt.Errorf("failed to encode image %s: %w", file.GetName(), err)
			}

			// Create data URL for image
			dataURL := fmt.Sprintf("data:%s;base64,%s", file.GetMimeType(), base64Content)
			attachment.ContentURL = dataURL

			// Add image element to show inline
			attachment.Content = map[string]interface{}{
				"type":    "Image",
				"url":     dataURL,
				"altText": file.GetName(),
				"size":    "Medium",
			}
		} else {
			// For non-image files, create a text representation
			attachment.Content = map[string]interface{}{
				"type":   "TextBlock",
				"text":   fmt.Sprintf("📎 **%s** (%s)", file.GetName(), file.GetMimeType()),
				"weight": "Bolder",
			}
		}

		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (m *MSTeamsService) getEmojiForNotifyType(notifyType NotifyType) string {
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
		return "📢"
	}
}

// Example usage and URL formats:
// msteams://team_name/token_a/token_b/token_c
// msteams://team_name/token_a/token_b/token_c/token_d (version 3)
// msteams://token_a/token_b/token_c (legacy)
// msteams://team_name/token_a/token_b/token_c?image=no
