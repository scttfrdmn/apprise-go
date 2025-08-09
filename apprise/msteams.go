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
	version      int    // 1 = legacy, 2 = modern, 3 = version 3
	includeImage bool
	client       *http.Client
}

// NewMSTeamsService creates a new Microsoft Teams service instance
func NewMSTeamsService() Service {
	return &MSTeamsService{
		client:       &http.Client{},
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
	Type        string                `json:"@type"`
	Context     string                `json:"@context"`
	Summary     string                `json:"summary"`
	ThemeColor  string                `json:"themeColor"`
	Sections    []MSTeamsSection      `json:"sections"`
	PotentialAction []MSTeamsAction   `json:"potentialAction,omitempty"`
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
	Type    string              `json:"@type"`
	Name    string              `json:"name"`
	Targets []MSTeamsActionTarget `json:"targets,omitempty"`
}

// MSTeamsActionTarget represents an action target
type MSTeamsActionTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// Send sends a notification to Microsoft Teams
func (m *MSTeamsService) Send(ctx context.Context, req NotificationRequest) error {
	// Create the Teams message payload
	payload := MSTeamsPayload{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    m.createSummary(req.Title, req.Body),
		ThemeColor: m.getColorForNotifyType(req.NotifyType),
		Sections:   []MSTeamsSection{m.createSection(req)},
	}

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
	httpReq.Header.Set("User-Agent", "Go-Apprise/1.0")

	// Send request
	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Teams API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Teams typically returns "1" for success
	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "1" {
		return fmt.Errorf("Teams webhook error: %s", string(body))
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

// SupportsAttachments returns false for Teams webhooks (basic implementation)
func (m *MSTeamsService) SupportsAttachments() bool {
	return false // Can be extended with adaptive cards
}

// GetMaxBodyLength returns Teams' message length limit
func (m *MSTeamsService) GetMaxBodyLength() int {
	return 28000 // Teams has a high character limit
}

// Example usage and URL formats:
// msteams://team_name/token_a/token_b/token_c
// msteams://team_name/token_a/token_b/token_c/token_d (version 3)
// msteams://token_a/token_b/token_c (legacy)
// msteams://team_name/token_a/token_b/token_c?image=no