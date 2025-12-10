package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PrometheusService implements Prometheus AlertManager webhook service
// Prometheus AlertManager handles alerts from Prometheus and routes them to receivers
// Used by thousands of organizations for production monitoring and incident response
type PrometheusService struct {
	webhookURL  string // Full webhook URL to receive alerts
	sendResolved bool   // Whether to send resolved alerts (default: true)
	client      *http.Client
}

// PrometheusAlert represents a single alert in the AlertManager payload
type PrometheusAlert struct {
	Status       string            `json:"status"`       // "firing" or "resolved"
	Labels       map[string]string `json:"labels"`       // Alert labels (alertname, severity, etc.)
	Annotations  map[string]string `json:"annotations"`  // Human-readable info (summary, description)
	StartsAt     string            `json:"startsAt"`     // RFC3339 timestamp
	EndsAt       string            `json:"endsAt"`       // RFC3339 timestamp
	GeneratorURL string            `json:"generatorURL"` // Link to Prometheus expression
	Fingerprint  string            `json:"fingerprint"`  // Unique alert identifier
}

// PrometheusWebhookPayload represents the AlertManager webhook payload
type PrometheusWebhookPayload struct {
	Receiver          string              `json:"receiver"`
	Status            string              `json:"status"` // Overall status: "firing" or "resolved"
	Alerts            []PrometheusAlert   `json:"alerts"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	Version           string              `json:"version"`
}

// PrometheusWebhookResponse represents the response from the webhook endpoint
type PrometheusWebhookResponse struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewPrometheusService creates a new Prometheus AlertManager service instance
func NewPrometheusService() Service {
	return &PrometheusService{
		sendResolved: true, // Default: send both firing and resolved alerts
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (p *PrometheusService) GetServiceID() string {
	return "prometheus"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PrometheusService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Prometheus AlertManager service URL
// Format: prometheus://webhook-url
// Format: prometheus://host:port/path
// Format: prometheusam://webhook-url (alias)
//
// Query Parameters:
//   - send_resolved=true|false - Send resolved alerts (default: true)
//
// Examples:
//   prometheus://alertmanager.example.com/api/v1/webhook
//   prometheus://host:9093/alerts?send_resolved=false
//   prometheusam://10.0.0.5:9093/webhook
func (p *PrometheusService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "prometheus" && scheme != "prometheusam" {
		return fmt.Errorf("invalid scheme: expected 'prometheus' or 'prometheusam', got '%s'", scheme)
	}

	// Build webhook URL
	// The webhook URL should be HTTP(S), not the prometheus:// scheme
	var webhookScheme string
	if serviceURL.Port() == "443" || serviceURL.Query().Get("secure") == "true" {
		webhookScheme = "https"
	} else {
		webhookScheme = "http"
	}

	// Reconstruct the webhook URL
	host := serviceURL.Host
	if host == "" {
		return fmt.Errorf("missing host in URL")
	}

	path := serviceURL.Path
	if path == "" {
		path = "/api/v1/webhook" // Default AlertManager webhook path
	}

	p.webhookURL = fmt.Sprintf("%s://%s%s", webhookScheme, host, path)

	// Parse query parameters
	query := serviceURL.Query()

	// Parse send_resolved parameter
	if sendResolvedStr := query.Get("send_resolved"); sendResolvedStr != "" {
		p.sendResolved = sendResolvedStr == "true"
	}

	return nil
}

// Send sends an alert notification to Prometheus AlertManager webhook
func (p *PrometheusService) Send(ctx context.Context, req NotificationRequest) error {
	// Build AlertManager webhook payload
	payload := p.buildWebhookPayload(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "apprise-go/1.0")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errorResp PrometheusWebhookResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Message != "" {
			return fmt.Errorf("webhook returned error: %s (status: %d)", errorResp.Message, resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// buildWebhookPayload constructs the Prometheus AlertManager webhook payload
func (p *PrometheusService) buildWebhookPayload(req NotificationRequest) PrometheusWebhookPayload {
	// Determine alert status based on notification type
	status := p.mapNotifyTypeToStatus(req.NotifyType)

	// Build alert labels
	labels := map[string]string{
		"alertname": "apprise_notification",
		"severity":  p.mapNotifyTypeToSeverity(req.NotifyType),
		"source":    "apprise-go",
	}

	// Add tags as labels
	for _, tag := range req.Tags {
		// Convert tags to label format (lowercase, underscores)
		labelKey := strings.ToLower(strings.ReplaceAll(tag, " ", "_"))
		labels[labelKey] = "true"
	}

	// Build annotations
	annotations := map[string]string{}
	if req.Title != "" {
		annotations["summary"] = req.Title
	}
	if req.Body != "" {
		annotations["description"] = req.Body
	}

	// If only body is provided, use it as summary
	if req.Title == "" && req.Body != "" {
		annotations["summary"] = req.Body
	}

	// Add default summary if none provided
	if annotations["summary"] == "" {
		annotations["summary"] = "Notification from apprise-go"
	}

	// Build alert
	alert := PrometheusAlert{
		Status:      status,
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    time.Now().UTC().Format(time.RFC3339),
		Fingerprint: generateFingerprint(labels),
	}

	// Set endsAt for resolved alerts
	if status == "resolved" {
		alert.EndsAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Build webhook payload
	payload := PrometheusWebhookPayload{
		Receiver:          "apprise-go",
		Status:            status,
		Alerts:            []PrometheusAlert{alert},
		GroupLabels:       labels,
		CommonLabels:      labels,
		CommonAnnotations: annotations,
		ExternalURL:       "",
		Version:           "4",
	}

	return payload
}

// mapNotifyTypeToStatus maps notification types to AlertManager status
func (p *PrometheusService) mapNotifyTypeToStatus(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "resolved" // Success = problem resolved
	default:
		return "firing" // All other types = firing alert
	}
}

// mapNotifyTypeToSeverity maps notification types to Prometheus severity levels
func (p *PrometheusService) mapNotifyTypeToSeverity(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "critical"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeSuccess:
		return "info"
	default:
		return "info"
	}
}

// generateFingerprint creates a unique fingerprint for an alert based on labels
func generateFingerprint(labels map[string]string) string {
	// Simple fingerprint generation - in production AlertManager generates this
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return fmt.Sprintf("%x", len(strings.Join(parts, ":")))
}

// TestURL validates the Prometheus AlertManager service URL format
func (p *PrometheusService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns false as AlertManager webhooks don't support attachments
func (p *PrometheusService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 8192 (reasonable limit for alert descriptions)
func (p *PrometheusService) GetMaxBodyLength() int {
	return 8192
}
