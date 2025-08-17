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

// NewRelicService implements New Relic monitoring notifications
type NewRelicService struct {
	apiKey        string            // New Relic API key (Ingest - License or User API Key)
	accountID     string            // New Relic account ID
	region        string            // New Relic region (us, eu)
	webhookURL    string            // Webhook proxy URL for secure credential management
	proxyAPIKey   string            // API key for webhook authentication
	client        *http.Client
}

// NewRelicEvent represents a New Relic custom event
type NewRelicEvent struct {
	EventType      string                 `json:"eventType"`
	Timestamp      int64                  `json:"timestamp,omitempty"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	NotificationType string               `json:"notificationType"`
	Source         string                 `json:"source"`
	Severity       string                 `json:"severity"`
	Tags           map[string]string      `json:"tags,omitempty"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
}

// NewRelicLogEntry represents a New Relic log entry
type NewRelicLogEntry struct {
	Message    string                 `json:"message"`
	Timestamp  int64                  `json:"timestamp,omitempty"`
	LogLevel   string                 `json:"logLevel,omitempty"`
	Service    string                 `json:"service,omitempty"`
	Hostname   string                 `json:"hostname,omitempty"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// NewRelicMetric represents a New Relic metric
type NewRelicMetric struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`        // "gauge", "count", "summary"
	Value      interface{}            `json:"value"`
	Timestamp  int64                  `json:"timestamp,omitempty"`
	Interval   int64                  `json:"interval,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// NewRelicMetricsPayload represents metrics batch payload
type NewRelicMetricsPayload struct {
	Metrics []NewRelicMetric `json:"metrics"`
}

// NewRelicEventsPayload represents events batch payload
type NewRelicEventsPayload struct {
	Events []NewRelicEvent `json:"events"`
}

// NewRelicLogsPayload represents logs batch payload
type NewRelicLogsPayload struct {
	Logs []NewRelicLogEntry `json:"logs"`
}

// NewRelicWebhookPayload represents webhook proxy payload
type NewRelicWebhookPayload struct {
	Service     string                  `json:"service"`
	AccountID   string                  `json:"account_id"`
	Region      string                  `json:"region"`
	Events      *NewRelicEventsPayload  `json:"events,omitempty"`
	Metrics     *NewRelicMetricsPayload `json:"metrics,omitempty"`
	Logs        *NewRelicLogsPayload    `json:"logs,omitempty"`
	Timestamp   string                  `json:"timestamp"`
	Source      string                  `json:"source"`
	Version     string                  `json:"version"`
}

// NewNewRelicService creates a new New Relic service instance
func NewNewRelicService() Service {
	return &NewRelicService{
		client: GetCloudHTTPClient("newrelic"),
		region: "us", // Default to US region
	}
}

// GetServiceID returns the service identifier
func (n *NewRelicService) GetServiceID() string {
	return "newrelic"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (n *NewRelicService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a New Relic service URL
// Format: newrelic://api_key@newrelic.com/?account_id=123456&region=us
// Format: newrelic://proxy-key@webhook.example.com/newrelic?api_key=nr_key&account_id=123456&region=eu
func (n *NewRelicService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "newrelic" {
		return fmt.Errorf("invalid scheme: expected 'newrelic', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/newrelic") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		n.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			n.proxyAPIKey = serviceURL.User.Username()
		}

		// Get New Relic API key from query parameters
		n.apiKey = query.Get("api_key")
		if n.apiKey == "" {
			return fmt.Errorf("api_key parameter is required for webhook mode")
		}

		// Get account ID from query
		n.accountID = query.Get("account_id")
		if n.accountID == "" {
			return fmt.Errorf("account_id parameter is required")
		}
	} else {
		// Direct New Relic API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: API key must be provided")
		}

		n.apiKey = serviceURL.User.Username()
		if n.apiKey == "" {
			return fmt.Errorf("New Relic API key is required")
		}

		// Get account ID from query
		n.accountID = query.Get("account_id")
		if n.accountID == "" {
			return fmt.Errorf("account_id parameter is required")
		}
	}

	// Parse region
	if region := query.Get("region"); region != "" {
		if !n.isValidRegion(region) {
			return fmt.Errorf("invalid region: %s (valid: us, eu)", region)
		}
		n.region = region
	}

	return nil
}

// isValidRegion checks if the region is valid
func (n *NewRelicService) isValidRegion(region string) bool {
	validRegions := []string{"us", "eu"}
	for _, valid := range validRegions {
		if region == valid {
			return true
		}
	}
	return false
}

// Send sends a notification to New Relic
func (n *NewRelicService) Send(ctx context.Context, req NotificationRequest) error {
	// Create New Relic event
	event := n.createEvent(req)

	// Create metric for notification count
	metric := n.createMetric(req)

	// Create log entry
	log := n.createLog(req)

	if n.webhookURL != "" {
		// Send via webhook proxy
		return n.sendViaWebhook(ctx, event, metric, log)
	} else {
		// Send directly to New Relic API
		return n.sendDirectly(ctx, event, metric, log)
	}
}

// createEvent creates a New Relic event from notification request
func (n *NewRelicService) createEvent(req NotificationRequest) *NewRelicEvent {
	event := &NewRelicEvent{
		EventType:            "AppriseNotification",
		Timestamp:            time.Now().Unix() * 1000, // New Relic expects milliseconds
		Title:                req.Title,
		Message:              req.Body,
		NotificationType:     req.NotifyType.String(),
		Source:               "apprise-go",
		Severity:             n.getSeverityForNotifyType(req.NotifyType),
		Tags:                 make(map[string]string),
		Attributes:           make(map[string]interface{}),
	}

	// Convert tags to map format
	for _, tag := range req.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			event.Tags[parts[0]] = parts[1]
		} else {
			event.Tags[tag] = "true"
		}
	}

	// Add source tag
	event.Tags["source"] = "apprise-go"

	// Add attributes
	event.Attributes["notification_type"] = req.NotifyType.String()
	event.Attributes["title"] = req.Title
	event.Attributes["body_length"] = len(req.Body)

	if req.BodyFormat != "" {
		event.Attributes["body_format"] = req.BodyFormat
	}

	// Add attachment info
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		event.Attributes["attachment_count"] = req.AttachmentMgr.Count()
		
		attachments := req.AttachmentMgr.GetAll()
		attachmentTypes := make([]string, len(attachments))
		for i, attachment := range attachments {
			attachmentTypes[i] = attachment.GetMimeType()
		}
		event.Attributes["attachment_types"] = strings.Join(attachmentTypes, ",")
	}

	return event
}

// createMetric creates a metric for notification tracking
func (n *NewRelicService) createMetric(req NotificationRequest) *NewRelicMetric {
	metric := &NewRelicMetric{
		Name:      "apprise.notification.count",
		Type:      "count",
		Value:     1,
		Timestamp: time.Now().Unix() * 1000,
		Attributes: map[string]interface{}{
			"notification_type": req.NotifyType.String(),
			"source":           "apprise-go",
		},
	}

	// Add tag attributes
	for _, tag := range req.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			metric.Attributes[fmt.Sprintf("tag.%s", parts[0])] = parts[1]
		}
	}

	return metric
}

// createLog creates a log entry
func (n *NewRelicService) createLog(req NotificationRequest) *NewRelicLogEntry {
	log := &NewRelicLogEntry{
		Message:    fmt.Sprintf("[%s] %s: %s", strings.ToUpper(req.NotifyType.String()), req.Title, req.Body),
		Timestamp:  time.Now().UnixMilli(),
		LogLevel:   n.getLogLevelForNotifyType(req.NotifyType),
		Service:    "apprise-go",
		Tags:       make(map[string]string),
		Attributes: make(map[string]interface{}),
	}

	// Convert tags to map format
	for _, tag := range req.Tags {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			log.Tags[parts[0]] = parts[1]
		} else {
			log.Tags[tag] = "true"
		}
	}

	// Add source tag
	log.Tags["source"] = "apprise-go"

	// Add attributes
	log.Attributes["notification_type"] = req.NotifyType.String()
	log.Attributes["title"] = req.Title
	log.Attributes["body_format"] = req.BodyFormat

	// Add attachment info
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		attachmentInfo := make([]map[string]string, len(attachments))
		
		for i, attachment := range attachments {
			attachmentInfo[i] = map[string]string{
				"name":      attachment.GetName(),
				"mime_type": attachment.GetMimeType(),
			}
		}
		
		log.Attributes["attachments"] = attachmentInfo
	}

	return log
}

// sendViaWebhook sends data via webhook proxy
func (n *NewRelicService) sendViaWebhook(ctx context.Context, event *NewRelicEvent, metric *NewRelicMetric, log *NewRelicLogEntry) error {
	payload := NewRelicWebhookPayload{
		Service:   "newrelic",
		AccountID: n.accountID,
		Region:    n.region,
		Events:    &NewRelicEventsPayload{Events: []NewRelicEvent{*event}},
		Metrics:   &NewRelicMetricsPayload{Metrics: []NewRelicMetric{*metric}},
		Logs:      &NewRelicLogsPayload{Logs: []NewRelicLogEntry{*log}},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Source:    "apprise-go",
		Version:   GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal New Relic webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create New Relic webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if n.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", n.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", n.proxyAPIKey)
	}

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send New Relic webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("New Relic webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendDirectly sends data directly to New Relic API
func (n *NewRelicService) sendDirectly(ctx context.Context, event *NewRelicEvent, metric *NewRelicMetric, log *NewRelicLogEntry) error {
	// Send event
	if err := n.sendEvents(ctx, []NewRelicEvent{*event}); err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}

	// Send metric
	if err := n.sendMetrics(ctx, []NewRelicMetric{*metric}); err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	// Send log
	if err := n.sendLogs(ctx, []NewRelicLogEntry{*log}); err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}

	return nil
}

// sendEvents sends events to New Relic
func (n *NewRelicService) sendEvents(ctx context.Context, events []NewRelicEvent) error {
	eventsURL := fmt.Sprintf("%s/v1/accounts/%s/events", n.getAPIBaseURL(), n.accountID)

	payload := NewRelicEventsPayload{Events: events}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", eventsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create events request: %w", err)
	}

	n.setAuthHeaders(httpReq)

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send events: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("New Relic events API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendMetrics sends metrics to New Relic
func (n *NewRelicService) sendMetrics(ctx context.Context, metrics []NewRelicMetric) error {
	metricsURL := fmt.Sprintf("%s/metric/v1", n.getAPIBaseURL())

	payload := NewRelicMetricsPayload{Metrics: metrics}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", metricsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metrics request: %w", err)
	}

	n.setAuthHeaders(httpReq)

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("New Relic metrics API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendLogs sends logs to New Relic
func (n *NewRelicService) sendLogs(ctx context.Context, logs []NewRelicLogEntry) error {
	logsURL := fmt.Sprintf("%s/log/v1", n.getAPIBaseURL())

	payload := NewRelicLogsPayload{Logs: logs}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", logsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create logs request: %w", err)
	}

	n.setAuthHeaders(httpReq)

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send logs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("New Relic logs API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (n *NewRelicService) getAPIBaseURL() string {
	switch n.region {
	case "eu":
		return "https://insights-api.eu01.nr-data.net"
	default:
		return "https://insights-api.newrelic.com" // us region
	}
}

func (n *NewRelicService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("Api-Key", n.apiKey)
}

func (n *NewRelicService) getSeverityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "CRITICAL"
	case NotifyTypeWarning:
		return "WARNING"
	case NotifyTypeSuccess:
		return "INFO"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "INFO"
	}
}

func (n *NewRelicService) getLogLevelForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "ERROR"
	case NotifyTypeWarning:
		return "WARN"
	case NotifyTypeSuccess, NotifyTypeInfo:
		fallthrough
	default:
		return "INFO"
	}
}

// TestURL validates a New Relic service URL
func (n *NewRelicService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return n.ParseURL(parsedURL)
}

// SupportsAttachments returns true for New Relic (supports metadata in events/logs)
func (n *NewRelicService) SupportsAttachments() bool {
	return true // New Relic supports attachment metadata in events and logs
}

// GetMaxBodyLength returns New Relic's message length limit
func (n *NewRelicService) GetMaxBodyLength() int {
	return 4096 // New Relic supports reasonably large event data (4KB)
}

// Example usage and URL formats:
// newrelic://api_key@newrelic.com/?account_id=123456&region=us
// newrelic://proxy-key@webhook.example.com/newrelic?api_key=nr_key&account_id=123456&region=eu