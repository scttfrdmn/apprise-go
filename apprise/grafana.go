package apprise

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GrafanaService implements Grafana webhook notifications
// Grafana is the leading open-source observability platform
// Supports alerting webhooks with rich alert metadata
type GrafanaService struct {
	webhookURL     string
	method         string // POST or PUT
	username       string // For HTTP Basic Auth
	password       string // For HTTP Basic Auth
	authHeader     string // Custom Authorization header
	hmacSecret     string // For HMAC signature validation
	maxAlerts      int    // Limit number of alerts in payload
	customHeaders  map[string]string
	client         *http.Client
}

// GrafanaWebhookPayload represents the Grafana alerting webhook payload
// This matches the format sent by Grafana Alerting (v9.0+)
type GrafanaWebhookPayload struct {
	Receiver          string                 `json:"receiver"`
	Status            string                 `json:"status"` // "firing" or "resolved"
	OrgID             int64                  `json:"orgId"`
	Alerts            []GrafanaAlert         `json:"alerts"`
	GroupLabels       map[string]string      `json:"groupLabels"`
	CommonLabels      map[string]string      `json:"commonLabels"`
	CommonAnnotations map[string]string      `json:"commonAnnotations"`
	ExternalURL       string                 `json:"externalURL"`
	Version           string                 `json:"version,omitempty"`
	GroupKey          string                 `json:"groupKey,omitempty"`
	TruncatedAlerts   int                    `json:"truncatedAlerts,omitempty"`
	Title             string                 `json:"title,omitempty"`
	State             string                 `json:"state,omitempty"`
	Message           string                 `json:"message,omitempty"`
}

// GrafanaAlert represents an individual alert in the Grafana payload
type GrafanaAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Fingerprint  string            `json:"fingerprint,omitempty"`
	SilenceURL   string            `json:"silenceURL,omitempty"`
	DashboardURL string            `json:"dashboardURL,omitempty"`
	PanelURL     string            `json:"panelURL,omitempty"`
	Values       map[string]string `json:"values,omitempty"`
}

// NewGrafanaService creates a new Grafana service instance
func NewGrafanaService() Service {
	return &GrafanaService{
		method:        "POST",
		maxAlerts:     0, // No limit by default
		customHeaders: make(map[string]string),
		client:        GetWebhookHTTPClient("grafana"),
	}
}

// GetServiceID returns the service identifier
func (g *GrafanaService) GetServiceID() string {
	return "grafana"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (g *GrafanaService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Grafana webhook URL
// Format: grafana://webhook_url
// Format: grafana://username:password@webhook_url (Basic Auth)
// Format: grafana://token@webhook_url (Bearer token)
// Format: grafanas://webhook_url (explicit HTTPS)
//
// Query Parameters:
//   - method=POST|PUT - HTTP method (default: POST)
//   - max_alerts=N - Limit number of alerts (default: unlimited)
//   - hmac_secret=secret - HMAC signature secret for validation
//   - header_X-Custom-Header=value - Custom headers
//
// Examples:
//   grafana://alerts.example.com/webhook
//   grafana://user:pass@alerts.example.com/webhook
//   grafana://token@alerts.example.com/webhook?method=PUT
//   grafanas://alerts.example.com/webhook?max_alerts=100
func (g *GrafanaService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "grafana" && scheme != "grafanas" {
		return fmt.Errorf("invalid scheme: expected 'grafana' or 'grafanas', got '%s'", scheme)
	}

	// Determine if HTTPS should be used
	useHTTPS := scheme == "grafanas" || strings.Contains(serviceURL.Host, "443")

	// Construct the full webhook URL
	if useHTTPS {
		g.webhookURL = fmt.Sprintf("https://%s%s", serviceURL.Host, serviceURL.Path)
	} else {
		// Check if it's explicitly HTTP
		if strings.HasPrefix(serviceURL.Host, "http://") {
			g.webhookURL = fmt.Sprintf("%s%s", serviceURL.Host, serviceURL.Path)
		} else {
			g.webhookURL = fmt.Sprintf("https://%s%s", serviceURL.Host, serviceURL.Path)
		}
	}

	// Parse authentication from URL
	if serviceURL.User != nil {
		username := serviceURL.User.Username()
		if password, hasPassword := serviceURL.User.Password(); hasPassword {
			// HTTP Basic Authentication
			g.username = username
			g.password = password
		} else {
			// Bearer token in Authorization header
			g.authHeader = fmt.Sprintf("Bearer %s", username)
		}
	}

	// Parse query parameters
	query := serviceURL.Query()

	if method := query.Get("method"); method != "" {
		g.method = strings.ToUpper(method)
		if g.method != "POST" && g.method != "PUT" {
			return fmt.Errorf("invalid method: %s (must be POST or PUT)", g.method)
		}
	}

	if maxAlerts := query.Get("max_alerts"); maxAlerts != "" {
		max, err := strconv.Atoi(maxAlerts)
		if err != nil {
			return fmt.Errorf("invalid max_alerts value: %s", maxAlerts)
		}
		g.maxAlerts = max
	}

	if hmacSecret := query.Get("hmac_secret"); hmacSecret != "" {
		g.hmacSecret = hmacSecret
	}

	// Extract custom headers (any parameter starting with "header_")
	for key, values := range query {
		if strings.HasPrefix(key, "header_") && len(values) > 0 {
			headerName := strings.TrimPrefix(key, "header_")
			g.customHeaders[headerName] = values[0]
		}
	}

	return nil
}

// Send sends a notification via Grafana webhook
// Converts the Apprise NotificationRequest into Grafana's webhook format
func (g *GrafanaService) Send(ctx context.Context, req NotificationRequest) error {
	// Build Grafana-compatible payload
	payload := g.buildPayload(req)

	// Apply max alerts limit if configured
	if g.maxAlerts > 0 && len(payload.Alerts) > g.maxAlerts {
		truncated := len(payload.Alerts) - g.maxAlerts
		payload.Alerts = payload.Alerts[:g.maxAlerts]
		payload.TruncatedAlerts = truncated
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Grafana payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, g.method, g.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Add authentication
	if g.username != "" && g.password != "" {
		httpReq.SetBasicAuth(g.username, g.password)
	} else if g.authHeader != "" {
		httpReq.Header.Set("Authorization", g.authHeader)
	}

	// Add HMAC signature if configured
	if g.hmacSecret != "" {
		signature := g.generateHMACSignature(jsonData)
		httpReq.Header.Set("X-Grafana-Alerting-Signature", signature)
		httpReq.Header.Set("X-Grafana-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	}

	// Add custom headers
	for key, value := range g.customHeaders {
		httpReq.Header.Set(key, value)
	}

	// Send request
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPayload converts NotificationRequest to Grafana webhook format
func (g *GrafanaService) buildPayload(req NotificationRequest) GrafanaWebhookPayload {
	// Determine status based on notification type
	status := "firing"
	if req.NotifyType == NotifyTypeSuccess {
		status = "resolved"
	}

	// Build alert object
	alert := GrafanaAlert{
		Status: status,
		Labels: map[string]string{
			"alertname": req.Title,
			"severity":  g.mapNotifyTypeToSeverity(req.NotifyType),
		},
		Annotations: map[string]string{
			"summary":     req.Title,
			"description": req.Body,
		},
		StartsAt: time.Now(),
		Values:   make(map[string]string),
	}

	// Add resolved timestamp for success notifications
	if status == "resolved" {
		alert.EndsAt = time.Now()
	}

	// Add tags as labels
	for _, tag := range req.Tags {
		alert.Labels[fmt.Sprintf("tag_%s", tag)] = "true"
	}

	// Build main payload
	payload := GrafanaWebhookPayload{
		Receiver: "apprise-go",
		Status:   status,
		OrgID:    1,
		Alerts:   []GrafanaAlert{alert},
		GroupLabels: map[string]string{
			"alertname": req.Title,
		},
		CommonLabels: alert.Labels,
		CommonAnnotations: map[string]string{
			"summary":     req.Title,
			"description": req.Body,
		},
		ExternalURL: "apprise-go",
		Version:     "1.9.5-1",
		Title:       req.Title,
		State:       status,
		Message:     req.Body,
	}

	return payload
}

// mapNotifyTypeToSeverity maps Apprise notification types to Grafana severity levels
func (g *GrafanaService) mapNotifyTypeToSeverity(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeSuccess:
		return "ok"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeError:
		return "critical"
	default:
		return "info"
	}
}

// generateHMACSignature generates an HMAC-SHA256 signature for the payload
func (g *GrafanaService) generateHMACSignature(data []byte) string {
	h := hmac.New(sha256.New, []byte(g.hmacSecret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// TestURL validates the Grafana webhook URL format
func (g *GrafanaService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return g.ParseURL(parsedURL)
}

// SupportsAttachments returns false as Grafana webhooks don't support attachments
func (g *GrafanaService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 0 (unlimited) as Grafana can handle large payloads
func (g *GrafanaService) GetMaxBodyLength() int {
	return 0 // Unlimited
}
