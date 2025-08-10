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

// PushbulletService implements Pushbullet push notifications
type PushbulletService struct {
	accessToken string
	devices     []string
	emails      []string
	channels    []string
	client      *http.Client
}

// NewPushbulletService creates a new Pushbullet service instance
func NewPushbulletService() Service {
	return &PushbulletService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (p *PushbulletService) GetServiceID() string {
	return "pushbullet"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PushbulletService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Pushbullet service URL
// Format: pball://access_token
// Format: pball://access_token/device_id[/device_id2/...]
// Format: pushbullet://access_token/email@domain.com
// Format: pball://access_token?device=device1&email=user@domain.com&channel=channel1
func (p *PushbulletService) ParseURL(serviceURL *url.URL) error {
	if err := p.validateScheme(serviceURL.Scheme); err != nil {
		return err
	}
	if err := p.parseAccessToken(serviceURL); err != nil {
		return err
	}
	if err := p.parsePathTargets(serviceURL); err != nil {
		return err
	}
	if err := p.parseQueryTargets(serviceURL); err != nil {
		return err
	}
	if err := p.parseFragmentTargets(serviceURL); err != nil {
		return err
	}
	return nil
}

func (p *PushbulletService) validateScheme(scheme string) error {
	if scheme != "pball" && scheme != "pushbullet" {
		return fmt.Errorf("invalid scheme: expected 'pball' or 'pushbullet', got '%s'", scheme)
	}
	return nil
}

func (p *PushbulletService) parseAccessToken(serviceURL *url.URL) error {
	p.accessToken = serviceURL.Host
	if p.accessToken == "" {
		return fmt.Errorf("pushbullet access token is required")
	}
	return nil
}

func (p *PushbulletService) parsePathTargets(serviceURL *url.URL) error {
	if serviceURL.Path == "" {
		return nil
	}
	
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	for _, part := range pathParts {
		if part != "" {
			p.addTarget(part)
		}
	}
	return nil
}

func (p *PushbulletService) parseQueryTargets(serviceURL *url.URL) error {
	query := serviceURL.Query()
	
	p.parseCommaSeparatedTargets(query.Get("device"), "device")
	p.parseCommaSeparatedTargets(query.Get("email"), "email")
	p.parseCommaSeparatedTargets(query.Get("channel"), "channel")
	
	return nil
}

func (p *PushbulletService) parseCommaSeparatedTargets(value, targetType string) {
	if value == "" {
		return
	}
	
	parts := strings.Split(value, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		switch targetType {
		case "device":
			p.devices = append(p.devices, part)
		case "email":
			if strings.Contains(part, "@") {
				p.emails = append(p.emails, part)
			}
		case "channel":
			p.channels = append(p.channels, part)
		}
	}
}

func (p *PushbulletService) parseFragmentTargets(serviceURL *url.URL) error {
	if serviceURL.Fragment == "" {
		return nil
	}
	
	fragmentParts := strings.Split(serviceURL.Fragment, "/")
	for i, part := range fragmentParts {
		if part == "" {
			continue
		}
		
		if strings.Contains(part, "@") {
			p.emails = append(p.emails, part)
		} else if len(fragmentParts) == 1 {
			// Single fragment part is assumed to be a channel
			p.channels = append(p.channels, part)
		} else {
			// Multiple fragment parts: first is channel, rest are devices
			if i == 0 {
				p.channels = append(p.channels, part)
			} else {
				p.devices = append(p.devices, part)
			}
		}
	}
	return nil
}

func (p *PushbulletService) addTarget(part string) {
	if strings.Contains(part, "@") {
		p.emails = append(p.emails, part)
	} else if strings.HasPrefix(part, "#") {
		p.channels = append(p.channels, strings.TrimPrefix(part, "#"))
	} else {
		p.devices = append(p.devices, part)
	}
}

// PushbulletPayload represents the Pushbullet API payload structure
type PushbulletPayload struct {
	Type             string `json:"type"`
	Title            string `json:"title,omitempty"`
	Body             string `json:"body,omitempty"`
	DeviceIden       string `json:"device_iden,omitempty"`
	Email            string `json:"email,omitempty"`
	ChannelTag       string `json:"channel_tag,omitempty"`
	SourceDeviceIden string `json:"source_device_iden,omitempty"`
}

// PushbulletResponse represents the Pushbullet API response
type PushbulletResponse struct {
	Active    bool    `json:"active"`
	Created   float64 `json:"created"`
	Direction string  `json:"direction"`
	Dismissed bool    `json:"dismissed"`
	Iden      string  `json:"iden"`
	Modified  float64 `json:"modified"`
	Type      string  `json:"type"`

	// Error fields
	Error     *PushbulletError `json:"error,omitempty"`
	ErrorCode string           `json:"error_code,omitempty"`
}

// PushbulletError represents a Pushbullet API error
type PushbulletError struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Cat     string `json:"cat,omitempty"`
}

// Send sends a notification to Pushbullet
func (p *PushbulletService) Send(ctx context.Context, req NotificationRequest) error {
	apiURL := "https://api.pushbullet.com/v2/pushes"

	// If no specific targets, send to all devices
	if len(p.devices) == 0 && len(p.emails) == 0 && len(p.channels) == 0 {
		return p.sendPush(ctx, apiURL, req, "", "", "")
	}

	// Send to each target
	var lastError error
	successCount := 0

	// Send to devices
	for _, device := range p.devices {
		if err := p.sendPush(ctx, apiURL, req, device, "", ""); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Send to emails
	for _, email := range p.emails {
		if err := p.sendPush(ctx, apiURL, req, "", email, ""); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Send to channels
	for _, channel := range p.channels {
		if err := p.sendPush(ctx, apiURL, req, "", "", channel); err != nil {
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

// sendPush sends a single push notification
func (p *PushbulletService) sendPush(ctx context.Context, apiURL string, req NotificationRequest, device, email, channel string) error {
	payload := PushbulletPayload{
		Type:  "note",
		Title: req.Title,
		Body:  req.Body,
	}

	// Set target
	if device != "" {
		payload.DeviceIden = device
	} else if email != "" {
		payload.Email = email
	} else if channel != "" {
		payload.ChannelTag = channel
	}

	// Add emoji based on notification type
	emoji := p.getEmojiForNotifyType(req.NotifyType)
	if emoji != "" {
		if payload.Title != "" {
			payload.Title = fmt.Sprintf("%s %s", emoji, payload.Title)
		} else {
			payload.Title = emoji
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Pushbullet payload: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Access-Token", p.accessToken)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Pushbullet notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var result PushbulletResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Pushbullet response: %w", err)
	}

	// Check for errors
	if result.Error != nil {
		return fmt.Errorf("pushbullet API error: %s - %s", result.Error.Code, result.Error.Message)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushbullet API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (p *PushbulletService) getEmojiForNotifyType(notifyType NotifyType) string {
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

// TestURL validates a Pushbullet service URL
func (p *PushbulletService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Pushbullet supports file attachments
func (p *PushbulletService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Pushbullet's message length limit
func (p *PushbulletService) GetMaxBodyLength() int {
	return 0 // Pushbullet doesn't have a strict limit
}

// Example usage and URL formats:
// pball://access_token
// pushbullet://access_token/device_id
// pball://access_token/user@email.com
// pball://access_token/#channel_name
// pball://access_token?device=device1,device2&email=user@domain.com
