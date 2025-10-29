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
)

// BarkService implements Bark iOS push notification service
type BarkService struct {
	deviceKey string
	hostname  string
	port      int
	secure    bool // Use HTTPS (barks://) vs HTTP (bark://)

	// Bark optional parameters
	icon     string // Icon URL (new in v1.9.5)
	sound    string // Sound name
	badge    int    // Badge count
	url      string // URL to open when tapped
	category string // Notification category
	group    string // Notification group

	client *http.Client
}

// BarkPayload represents the Bark API request payload
type BarkPayload struct {
	Title    string `json:"title,omitempty"`
	Body     string `json:"body"`
	DeviceKey string `json:"device_key"`
	Icon     string `json:"icon,omitempty"`      // Icon URL (v1.9.5+)
	Sound    string `json:"sound,omitempty"`     // Sound name
	Badge    int    `json:"badge,omitempty"`     // Badge count
	URL      string `json:"url,omitempty"`       // Action URL
	Category string `json:"category,omitempty"`  // Category
	Group    string `json:"group,omitempty"`     // Group
}

// NewBarkService creates a new Bark service instance
func NewBarkService() Service {
	return &BarkService{
		client: &http.Client{},
		port:   -1, // Use default port based on scheme
	}
}

// GetServiceID returns the service identifier
func (b *BarkService) GetServiceID() string {
	return "bark"
}

// GetDefaultPort returns the default port
func (b *BarkService) GetDefaultPort() int {
	if b.secure {
		return 443
	}
	return 80
}

// ParseURL parses a Bark service URL
// Format: bark://devicekey@hostname[:port]/
// Format: barks://devicekey@hostname[:port]/?icon=url&sound=sound
func (b *BarkService) ParseURL(serviceURL *url.URL) error {
	// Check scheme
	if serviceURL.Scheme == "barks" {
		b.secure = true
	} else if serviceURL.Scheme == "bark" {
		b.secure = false
	} else {
		return fmt.Errorf("invalid scheme: expected 'bark' or 'barks', got '%s'", serviceURL.Scheme)
	}

	// Extract device key from user info
	if serviceURL.User == nil || serviceURL.User.Username() == "" {
		return fmt.Errorf("bark device key is required (format: bark://devicekey@hostname)")
	}
	b.deviceKey = serviceURL.User.Username()

	// Extract hostname
	b.hostname = serviceURL.Hostname()
	if b.hostname == "" {
		return fmt.Errorf("bark server hostname is required")
	}

	// Extract port if specified
	if portStr := serviceURL.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		b.port = port
	}

	// Parse query parameters
	query := serviceURL.Query()

	if icon := query.Get("icon"); icon != "" {
		b.icon = icon
	}

	if sound := query.Get("sound"); sound != "" {
		b.sound = sound
	}

	if badgeStr := query.Get("badge"); badgeStr != "" {
		if badge, err := strconv.Atoi(badgeStr); err == nil {
			b.badge = badge
		}
	}

	if actionURL := query.Get("url"); actionURL != "" {
		b.url = actionURL
	}

	if category := query.Get("category"); category != "" {
		b.category = category
	}

	if group := query.Get("group"); group != "" {
		b.group = group
	}

	return nil
}

// Send sends a notification via Bark
func (b *BarkService) Send(ctx context.Context, req NotificationRequest) error {
	// Build the Bark server URL
	scheme := "http"
	if b.secure {
		scheme = "https"
	}

	port := b.port
	if port == -1 {
		port = b.GetDefaultPort()
	}

	var serverURL string
	if (b.secure && port == 443) || (!b.secure && port == 80) {
		// Use default port - omit from URL
		serverURL = fmt.Sprintf("%s://%s/push", scheme, b.hostname)
	} else {
		serverURL = fmt.Sprintf("%s://%s:%d/push", scheme, b.hostname, port)
	}

	// Build payload
	payload := BarkPayload{
		DeviceKey: b.deviceKey,
		Title:     req.Title,
		Body:      req.Body,
		Icon:      b.icon,
		Sound:     b.sound,
		Badge:     b.badge,
		URL:       b.url,
		Category:  b.category,
		Group:     b.group,
	}

	// If no title, use body as message
	if payload.Title == "" && payload.Body != "" {
		payload.Title = payload.Body
		payload.Body = ""
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Bark payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Send request
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Bark notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bark API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to check for Bark-specific errors
	var barkResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Non-fatal - response was successful HTTP status
		return nil
	}

	if err := json.Unmarshal(body, &barkResp); err != nil {
		// Non-fatal - response was successful HTTP status
		return nil
	}

	// Check Bark API response code (200 = success)
	if barkResp.Code != 200 {
		return fmt.Errorf("bark API returned error code %d: %s", barkResp.Code, barkResp.Message)
	}

	return nil
}

// TestURL validates a Bark service URL
func (b *BarkService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return b.ParseURL(parsedURL)
}

// SupportsAttachments returns false (Bark doesn't support file attachments)
func (b *BarkService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns Bark's message length limit
func (b *BarkService) GetMaxBodyLength() int {
	return 4096 // Reasonable limit for push notifications
}

// Example usage and URL formats:
// bark://devicekey@api.day.app/
// bark://devicekey@api.day.app/?icon=https://example.com/icon.png
// barks://devicekey@bark.example.com:8080/?sound=alarm&badge=5
// bark://devicekey@localhost:8080/?url=https://example.com&category=news
