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

// NtfyService implements Ntfy push notification service
type NtfyService struct {
	baseURL    string
	topic      string
	username   string
	password   string
	token      string
	priority   int
	tags       []string
	delay      string
	actions    []string
	attach     string
	filename   string
	click      string
	email      string
	client     *http.Client
}

// NewNtfyService creates a new Ntfy service instance
func NewNtfyService() Service {
	return &NtfyService{
		client:   &http.Client{},
		priority: 3, // Default priority (normal)
	}
}

// GetServiceID returns the service identifier
func (n *NtfyService) GetServiceID() string {
	return "ntfy"
}

// GetDefaultPort returns the default port (443 for HTTPS, 80 for HTTP)
func (n *NtfyService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Ntfy service URL
// Format: ntfy://[username:password@]server[:port]/topic
// Format: ntfys://[username:password@]server[:port]/topic  (HTTPS)
// Format: ntfy://[token@]server[:port]/topic
func (n *NtfyService) ParseURL(serviceURL *url.URL) error {
	scheme := strings.ToLower(serviceURL.Scheme)
	if scheme != "ntfy" && scheme != "ntfys" {
		return fmt.Errorf("invalid scheme: expected 'ntfy' or 'ntfys', got '%s'", serviceURL.Scheme)
	}

	// Build base URL
	protocol := "http"
	if scheme == "ntfys" {
		protocol = "https"
	}

	host := serviceURL.Host
	if host == "" {
		return fmt.Errorf("ntfy server host is required")
	}

	// If no port specified, use scheme default
	if !strings.Contains(host, ":") {
		if scheme == "ntfys" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	n.baseURL = fmt.Sprintf("%s://%s", protocol, host)

	// Parse authentication
	if serviceURL.User != nil {
		username := serviceURL.User.Username()
		password, hasPassword := serviceURL.User.Password()

		if hasPassword {
			// Username:password authentication
			n.username = username
			n.password = password
		} else {
			// Token authentication (token as username)
			n.token = username
		}
	}

	// Extract topic from path
	if serviceURL.Path == "" || serviceURL.Path == "/" {
		return fmt.Errorf("ntfy topic is required")
	}

	n.topic = strings.Trim(serviceURL.Path, "/")
	if n.topic == "" {
		return fmt.Errorf("ntfy topic is required")
	}

	// Parse query parameters
	query := serviceURL.Query()

	if token := query.Get("token"); token != "" {
		n.token = token
	}

	if priorityStr := query.Get("priority"); priorityStr != "" {
		priority, err := strconv.Atoi(priorityStr)
		if err != nil || priority < 1 || priority > 5 {
			return fmt.Errorf("invalid priority: must be 1-5, got '%s'", priorityStr)
		}
		n.priority = priority
	}

	if tags := query.Get("tags"); tags != "" {
		n.tags = strings.Split(tags, ",")
		// Trim whitespace from tags
		for i, tag := range n.tags {
			n.tags[i] = strings.TrimSpace(tag)
		}
	}

	if delay := query.Get("delay"); delay != "" {
		n.delay = delay
	}

	if actions := query.Get("actions"); actions != "" {
		n.actions = strings.Split(actions, ",")
		// Trim whitespace from actions
		for i, action := range n.actions {
			n.actions[i] = strings.TrimSpace(action)
		}
	}

	if attach := query.Get("attach"); attach != "" {
		n.attach = attach
	}

	if filename := query.Get("filename"); filename != "" {
		n.filename = filename
	}

	if click := query.Get("click"); click != "" {
		n.click = click
	}

	if email := query.Get("email"); email != "" {
		n.email = email
	}

	return nil
}

// NtfyMessage represents a Ntfy notification payload
type NtfyMessage struct {
	Topic    string   `json:"topic"`
	Title    string   `json:"title,omitempty"`
	Message  string   `json:"message"`
	Priority int      `json:"priority,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Delay    string   `json:"delay,omitempty"`
	Actions  []string `json:"actions,omitempty"`
	Attach   string   `json:"attach,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Click    string   `json:"click,omitempty"`
	Email    string   `json:"email,omitempty"`
}

// Send sends a notification to Ntfy
func (n *NtfyService) Send(ctx context.Context, req NotificationRequest) error {
	message := NtfyMessage{
		Topic:    n.topic,
		Title:    req.Title,
		Message:  req.Body,
		Priority: n.priority,
		Tags:     n.tags,
		Delay:    n.delay,
		Actions:  n.actions,
		Attach:   n.attach,
		Filename: n.filename,
		Click:    n.click,
		Email:    n.email,
	}

	// Add notification type as emoji tag if no custom tags
	if len(n.tags) == 0 {
		message.Tags = []string{n.getEmojiForNotifyType(req.NotifyType)}
	}

	// Adjust priority based on notification type if using default priority
	if n.priority == 3 {
		message.Priority = n.mapNotifyTypeToPriority(req.NotifyType)
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Ntfy message: %w", err)
	}

	// Send via JSON API
	apiURL := n.baseURL + "/v1/publish"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Set authentication
	if n.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+n.token)
	} else if n.username != "" && n.password != "" {
		httpReq.SetBasicAuth(n.username, n.password)
	}

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Ntfy notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response for error details
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// getEmojiForNotifyType returns an emoji tag for the notification type
func (n *NtfyService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "white_check_mark"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeError:
		return "rotating_light"
	default:
		return "information_source"
	}
}

// mapNotifyTypeToPriority maps notification types to Ntfy priority levels
func (n *NtfyService) mapNotifyTypeToPriority(notifyType NotifyType) int {
	switch notifyType {
	case NotifyTypeSuccess:
		return 3 // Normal
	case NotifyTypeWarning:
		return 4 // High
	case NotifyTypeError:
		return 5 // Max
	default:
		return 3 // Normal (info)
	}
}

// TestURL validates a Ntfy service URL
func (n *NtfyService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return n.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Ntfy supports file attachments via URL
func (n *NtfyService) SupportsAttachments() bool {
	return true // Via attach parameter
}

// GetMaxBodyLength returns Ntfy's message length limit
func (n *NtfyService) GetMaxBodyLength() int {
	return 4096 // Ntfy message limit is 4096 characters
}

// Example usage and URL formats:
// ntfy://ntfy.sh/my-topic
// ntfys://ntfy.example.com/alerts
// ntfy://username:password@ntfy.example.com:8080/notifications
// ntfy://token@ntfy.sh/alerts?priority=5&tags=urgent,production
// ntfy://ntfy.sh/alerts?delay=30min&email=admin@example.com
// ntfy://ntfy.sh/alerts?attach=https://example.com/file.pdf&filename=report.pdf
// ntfy://ntfy.sh/alerts?click=https://example.com&actions=view,View Dashboard,https://dashboard.example.com