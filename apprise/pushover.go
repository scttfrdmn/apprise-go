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

// PushoverService implements Pushover push notifications
type PushoverService struct {
	token    string
	userKey  string
	devices  []string
	priority int
	sound    string
	retry    int
	expire   int
	client   *http.Client
}

// NewPushoverService creates a new Pushover service instance
func NewPushoverService() Service {
	return &PushoverService{
		client:   &http.Client{},
		priority: 0,          // Normal priority
		sound:    "pushover", // Default sound
	}
}

// GetServiceID returns the service identifier
func (p *PushoverService) GetServiceID() string {
	return "pushover"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PushoverService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Pushover service URL
// Format: pover://token@user[/device1/device2/...]
// Format: pushover://token@user[/device1/device2/...]
func (p *PushoverService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "pover" && scheme != "pushover" {
		return fmt.Errorf("invalid scheme: expected 'pover' or 'pushover', got '%s'", scheme)
	}

	// Extract token and user key
	if serviceURL.User == nil {
		return fmt.Errorf("Pushover token and user key are required")
	}

	p.token = serviceURL.User.Username()
	if password, hasPassword := serviceURL.User.Password(); hasPassword {
		// Format: pushover://token:userkey@host
		p.userKey = password
	} else {
		// Format: pushover://token@userkey
		p.userKey = serviceURL.Host
	}

	if p.token == "" || p.userKey == "" {
		return fmt.Errorf("both Pushover token and user key are required")
	}

	// Extract devices from path
	if serviceURL.Path != "" {
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, part := range pathParts {
			if part != "" {
				p.devices = append(p.devices, part)
			}
		}
	}

	// Parse query parameters
	query := serviceURL.Query()

	if priority := query.Get("priority"); priority != "" {
		if prio, err := strconv.Atoi(priority); err == nil && prio >= -2 && prio <= 2 {
			p.priority = prio
		}
	}

	if sound := query.Get("sound"); sound != "" {
		p.sound = sound
	}

	if retry := query.Get("retry"); retry != "" && p.priority == 2 {
		if r, err := strconv.Atoi(retry); err == nil && r >= 30 {
			p.retry = r
		} else {
			p.retry = 60 // Default retry for emergency priority
		}
	}

	if expire := query.Get("expire"); expire != "" && p.priority == 2 {
		if e, err := strconv.Atoi(expire); err == nil && e <= 10800 {
			p.expire = e
		} else {
			p.expire = 3600 // Default expire for emergency priority
		}
	}

	// Set defaults for emergency priority
	if p.priority == 2 {
		if p.retry == 0 {
			p.retry = 60 // Default retry interval
		}
		if p.expire == 0 {
			p.expire = 3600 // Default expiration
		}
	}

	return nil
}

// PushoverPayload represents the Pushover API payload structure
type PushoverPayload struct {
	Token    string `json:"token"`
	User     string `json:"user"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Sound    string `json:"sound,omitempty"`
	Device   string `json:"device,omitempty"`
	Retry    int    `json:"retry,omitempty"`
	Expire   int    `json:"expire,omitempty"`
	URL      string `json:"url,omitempty"`
	URLTitle string `json:"url_title,omitempty"`
}

// PushoverResponse represents the Pushover API response
type PushoverResponse struct {
	Status  int      `json:"status"`
	Request string   `json:"request"`
	Errors  []string `json:"errors,omitempty"`
	Receipt string   `json:"receipt,omitempty"`
}

// Send sends a notification to Pushover
func (p *PushoverService) Send(ctx context.Context, req NotificationRequest) error {
	apiURL := "https://api.pushover.net/1/messages.json"

	// If no devices specified, send to all user's devices
	if len(p.devices) == 0 {
		return p.sendToDevice(ctx, apiURL, "", req)
	}

	// Send to each specified device
	var lastError error
	successCount := 0

	for _, device := range p.devices {
		if err := p.sendToDevice(ctx, apiURL, device, req); err != nil {
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

// sendToDevice sends a notification to a specific device
func (p *PushoverService) sendToDevice(ctx context.Context, apiURL, device string, req NotificationRequest) error {
	payload := PushoverPayload{
		Token:    p.token,
		User:     p.userKey,
		Message:  req.Body,
		Title:    req.Title,
		Priority: p.priority,
		Sound:    p.sound,
		Device:   device,
	}

	// Set retry and expire for emergency priority
	if p.priority == 2 {
		payload.Retry = p.retry
		payload.Expire = p.expire
	}

	// Add emoji based on notification type
	emoji := p.getEmojiForNotifyType(req.NotifyType)
	if emoji != "" && payload.Title != "" {
		payload.Title = fmt.Sprintf("%s %s", emoji, payload.Title)
	} else if emoji != "" && payload.Title == "" {
		payload.Title = emoji
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Pushover payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var result PushoverResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Pushover response: %w", err)
	}

	if result.Status != 1 {
		if len(result.Errors) > 0 {
			return fmt.Errorf("Pushover API error: %s", strings.Join(result.Errors, ", "))
		}
		return fmt.Errorf("Pushover API error: status %d", result.Status)
	}

	return nil
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (p *PushoverService) getEmojiForNotifyType(notifyType NotifyType) string {
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

// TestURL validates a Pushover service URL
func (p *PushoverService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Pushover supports image attachments
func (p *PushoverService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Pushover's message length limit
func (p *PushoverService) GetMaxBodyLength() int {
	return 1024 // Pushover's character limit for messages
}

// Example usage and URL formats:
// pushover://token@userkey
// pover://token@userkey/device1/device2
// pushover://token@userkey?priority=1&sound=cosmic
// pushover://token@userkey?priority=2&retry=60&expire=3600 (emergency)
// pushover://token:userkey@host (alternative format)
