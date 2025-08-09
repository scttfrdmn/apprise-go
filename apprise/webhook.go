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
	"time"
)

// WebhookService implements generic HTTP webhook notifications
type WebhookService struct {
	webhookURL  string
	method      string
	contentType string
	headers     map[string]string
	template    string
	client      *http.Client
	timeout     time.Duration
}

// JSONService is an alias for WebhookService with JSON content type
type JSONService struct {
	*WebhookService
}

// NewWebhookService creates a new webhook service instance
func NewWebhookService() Service {
	return &WebhookService{
		method:      "POST",
		contentType: "application/json",
		headers:     make(map[string]string),
		client:      &http.Client{},
		timeout:     30 * time.Second,
	}
}

// NewJSONService creates a new JSON webhook service instance
func NewJSONService() Service {
	webhook := &WebhookService{
		method:      "POST",
		contentType: "application/json",
		headers:     make(map[string]string),
		client:      &http.Client{},
		timeout:     30 * time.Second,
	}
	return &JSONService{WebhookService: webhook}
}

// GetServiceID returns the service identifier
func (w *WebhookService) GetServiceID() string {
	return "webhook"
}

// GetServiceID returns the service identifier for JSON service
func (j *JSONService) GetServiceID() string {
	return "json"
}

// GetDefaultPort returns the default port (443 for HTTPS, 80 for HTTP)
func (w *WebhookService) GetDefaultPort() int {
	return 443
}

// GetDefaultPort returns the default port for JSON service
func (j *JSONService) GetDefaultPort() int {
	return j.WebhookService.GetDefaultPort()
}

// ParseURL parses a webhook service URL
// Format: webhooks://hostname/path[?method=POST&header_name=value]
// Format: webhook://hostname/path[?method=POST&header_name=value]
// Format: json://hostname/path[?method=POST&header_name=value]
func (w *WebhookService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "webhook" && scheme != "webhooks" && scheme != "json" {
		return fmt.Errorf("invalid scheme: expected 'webhook', 'webhooks', or 'json', got '%s'", scheme)
	}

	// Determine if HTTPS should be used
	useHTTPS := scheme == "webhooks" || scheme == "json"
	
	// Construct the full webhook URL
	if useHTTPS {
		w.webhookURL = fmt.Sprintf("https://%s%s", serviceURL.Host, serviceURL.Path)
	} else {
		w.webhookURL = fmt.Sprintf("http://%s%s", serviceURL.Host, serviceURL.Path)
	}

	// Parse query parameters
	query := serviceURL.Query()
	
	if method := query.Get("method"); method != "" {
		w.method = strings.ToUpper(method)
	}
	
	if contentType := query.Get("content_type"); contentType != "" {
		w.contentType = contentType
	}
	
	if template := query.Get("template"); template != "" {
		w.template = template
	}

	// Extract custom headers (any parameter starting with "header_")
	for key, values := range query {
		if strings.HasPrefix(key, "header_") && len(values) > 0 {
			headerName := strings.TrimPrefix(key, "header_")
			w.headers[headerName] = values[0]
		}
	}

	// Add authentication headers if provided in URL
	if serviceURL.User != nil {
		username := serviceURL.User.Username()
		if password, hasPassword := serviceURL.User.Password(); hasPassword {
			// Basic auth
			w.headers["Authorization"] = fmt.Sprintf("Basic %s", 
				encodeBasicAuth(username, password))
		} else {
			// Bearer token
			w.headers["Authorization"] = fmt.Sprintf("Bearer %s", username)
		}
	}

	return nil
}

// ParseURL for JSON service
func (j *JSONService) ParseURL(serviceURL *url.URL) error {
	return j.WebhookService.ParseURL(serviceURL)
}

// WebhookPayload represents the default webhook payload structure
type WebhookPayload struct {
	Title     string            `json:"title,omitempty"`
	Message   string            `json:"message"`
	Type      string            `json:"type"`
	Timestamp string            `json:"timestamp"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Send sends a notification via webhook
func (w *WebhookService) Send(ctx context.Context, req NotificationRequest) error {
	// Prepare payload
	payload, err := w.createPayload(req)
	if err != nil {
		return fmt.Errorf("failed to create webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, w.method, w.webhookURL, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", w.contentType)
	httpReq.Header.Set("User-Agent", GetUserAgent())
	
	for key, value := range w.headers {
		httpReq.Header.Set(key, value)
	}

	// Set timeout
	client := &http.Client{Timeout: w.timeout}

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Send for JSON service
func (j *JSONService) Send(ctx context.Context, req NotificationRequest) error {
	return j.WebhookService.Send(ctx, req)
}

// createPayload creates the webhook payload based on content type and template
func (w *WebhookService) createPayload(req NotificationRequest) (io.Reader, error) {
	if w.template != "" {
		// Use custom template
		return w.createTemplatedPayload(req)
	}

	switch w.contentType {
	case "application/json":
		return w.createJSONPayload(req)
	case "application/x-www-form-urlencoded":
		return w.createFormPayload(req)
	case "text/plain":
		return w.createTextPayload(req)
	default:
		return w.createJSONPayload(req) // Default to JSON
	}
}

// createJSONPayload creates a JSON payload
func (w *WebhookService) createJSONPayload(req NotificationRequest) (io.Reader, error) {
	payload := WebhookPayload{
		Title:     req.Title,
		Message:   req.Body,
		Type:      req.NotifyType.String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Tags:      req.Tags,
		Metadata: map[string]string{
			"service": w.GetServiceID(),
			"format":  req.BodyFormat,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(jsonData), nil
}

// createFormPayload creates a form-encoded payload
func (w *WebhookService) createFormPayload(req NotificationRequest) (io.Reader, error) {
	values := url.Values{
		"title":     {req.Title},
		"message":   {req.Body},
		"type":      {req.NotifyType.String()},
		"timestamp": {time.Now().Format(time.RFC3339)},
		"format":    {req.BodyFormat},
	}

	if len(req.Tags) > 0 {
		values.Set("tags", strings.Join(req.Tags, ","))
	}

	return strings.NewReader(values.Encode()), nil
}

// createTextPayload creates a plain text payload
func (w *WebhookService) createTextPayload(req NotificationRequest) (io.Reader, error) {
	var text strings.Builder
	
	if req.Title != "" {
		text.WriteString(fmt.Sprintf("Title: %s\n", req.Title))
	}
	
	text.WriteString(fmt.Sprintf("Message: %s\n", req.Body))
	text.WriteString(fmt.Sprintf("Type: %s\n", req.NotifyType.String()))
	text.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
	
	if len(req.Tags) > 0 {
		text.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(req.Tags, ", ")))
	}

	return strings.NewReader(text.String()), nil
}

// createTemplatedPayload creates a payload using a custom template
func (w *WebhookService) createTemplatedPayload(req NotificationRequest) (io.Reader, error) {
	// Simple template substitution (can be enhanced with proper templating)
	template := w.template
	
	template = strings.ReplaceAll(template, "{{title}}", req.Title)
	template = strings.ReplaceAll(template, "{{message}}", req.Body)
	template = strings.ReplaceAll(template, "{{body}}", req.Body)
	template = strings.ReplaceAll(template, "{{type}}", req.NotifyType.String())
	template = strings.ReplaceAll(template, "{{timestamp}}", time.Now().Format(time.RFC3339))
	template = strings.ReplaceAll(template, "{{format}}", req.BodyFormat)
	
	if len(req.Tags) > 0 {
		template = strings.ReplaceAll(template, "{{tags}}", strings.Join(req.Tags, ","))
	} else {
		template = strings.ReplaceAll(template, "{{tags}}", "")
	}

	return strings.NewReader(template), nil
}

// TestURL validates a webhook service URL
func (w *WebhookService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return w.ParseURL(parsedURL)
}

// TestURL for JSON service
func (j *JSONService) TestURL(serviceURL string) error {
	return j.WebhookService.TestURL(serviceURL)
}

// SupportsAttachments returns false for basic webhooks
func (w *WebhookService) SupportsAttachments() bool {
	return false
}

// SupportsAttachments for JSON service
func (j *JSONService) SupportsAttachments() bool {
	return j.WebhookService.SupportsAttachments()
}

// GetMaxBodyLength returns 0 (no limit for webhooks)
func (w *WebhookService) GetMaxBodyLength() int {
	return 0
}

// GetMaxBodyLength for JSON service
func (j *JSONService) GetMaxBodyLength() int {
	return j.WebhookService.GetMaxBodyLength()
}

// encodeBasicAuth encodes username and password for Basic authentication
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return strings.TrimRight(strings.Replace(
		strings.Replace(auth, "+", "-", -1), "/", "_", -1), "=")
}

// Example usage and URL formats:
// webhook://api.example.com/notify
// webhooks://api.example.com/notify (HTTPS)
// webhooks://token@api.example.com/notify (Bearer token)
// webhooks://user:pass@api.example.com/notify (Basic auth)
// webhook://api.example.com/notify?method=PUT&content_type=text/plain
// json://api.example.com/webhook?header_X-API-Key=secret123
// webhook://api.example.com/notify?template={"text":"{{message}}","level":"{{type}}"}