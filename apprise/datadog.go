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

// DatadogService implements Datadog monitoring notifications
type DatadogService struct {
	apiKey      string            // Datadog API key
	appKey      string            // Datadog application key (optional)
	region      string            // Datadog region (us, eu, us3, us5, gov, ap1)
	tags        []string          // Default tags to apply
	webhookURL  string            // Webhook proxy URL for secure credential management
	proxyAPIKey string            // API key for webhook authentication
	client      *http.Client
}

// DatadogEvent represents a Datadog event
type DatadogEvent struct {
	Title          string    `json:"title"`
	Text           string    `json:"text"`
	DateHappened   int64     `json:"date_happened,omitempty"`
	Priority       string    `json:"priority,omitempty"`          // "normal" or "low"
	AlertType      string    `json:"alert_type,omitempty"`        // "error", "warning", "info", "success"
	AggregationKey string    `json:"aggregation_key,omitempty"`
	SourceTypeName string    `json:"source_type_name,omitempty"`
	Host           string    `json:"host,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
}

// DatadogMetric represents a Datadog metric submission
type DatadogMetric struct {
	Metric string                 `json:"metric"`
	Points [][]interface{}        `json:"points"`
	Type   string                 `json:"type,omitempty"`    // "count", "rate", "gauge"
	Host   string                 `json:"host,omitempty"`
	Tags   []string               `json:"tags,omitempty"`
}

// DatadogMetricsPayload represents the metrics submission payload
type DatadogMetricsPayload struct {
	Series []DatadogMetric `json:"series"`
}

// DatadogLog represents a Datadog log entry
type DatadogLog struct {
	Message   string                 `json:"message"`
	Level     string                 `json:"level,omitempty"`     // "DEBUG", "INFO", "WARN", "ERROR"
	Timestamp int64                  `json:"timestamp,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	Service   string                 `json:"service,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// DatadogWebhookPayload represents webhook proxy payload
type DatadogWebhookPayload struct {
	Service     string                 `json:"service"`
	Region      string                 `json:"region"`
	Event       *DatadogEvent          `json:"event,omitempty"`
	Metrics     *DatadogMetricsPayload `json:"metrics,omitempty"`
	Log         *DatadogLog            `json:"log,omitempty"`
	Timestamp   string                 `json:"timestamp"`
	Source      string                 `json:"source"`
	Version     string                 `json:"version"`
}

// NewDatadogService creates a new Datadog service instance
func NewDatadogService() Service {
	return &DatadogService{
		client: GetCloudHTTPClient("datadog"),
		region: "us", // Default to US region
	}
}

// GetServiceID returns the service identifier
func (d *DatadogService) GetServiceID() string {
	return "datadog"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (d *DatadogService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Datadog service URL
// Format: datadog://api_key@datadoghq.com/?region=us&tags=env:prod,service:api
// Format: datadog://api_key:app_key@datadoghq.com/?region=eu&tags=team:backend
// Format: datadog://proxy-api-key@webhook.example.com/datadog?api_key=dd_key&region=us3
func (d *DatadogService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "datadog" {
		return fmt.Errorf("invalid scheme: expected 'datadog', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/datadog") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		d.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			d.proxyAPIKey = serviceURL.User.Username()
		}

		// Get Datadog API key from query parameters
		d.apiKey = query.Get("api_key")
		if d.apiKey == "" {
			return fmt.Errorf("api_key parameter is required for webhook mode")
		}

		d.appKey = query.Get("app_key")
	} else {
		// Direct Datadog API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: API key must be provided")
		}

		d.apiKey = serviceURL.User.Username()
		if d.apiKey == "" {
			return fmt.Errorf("Datadog API key is required")
		}

		// Extract app key if provided as password
		if appKey, hasAppKey := serviceURL.User.Password(); hasAppKey {
			d.appKey = appKey
		}

		// Get app key from query if not in user info
		if d.appKey == "" {
			d.appKey = query.Get("app_key")
		}
	}

	// Parse region
	if region := query.Get("region"); region != "" {
		if !d.isValidRegion(region) {
			return fmt.Errorf("invalid region: %s (valid: us, eu, us3, us5, gov, ap1)", region)
		}
		d.region = region
	}

	// Parse tags
	if tagsStr := query.Get("tags"); tagsStr != "" {
		d.tags = strings.Split(tagsStr, ",")
		// Trim whitespace from tags
		for i, tag := range d.tags {
			d.tags[i] = strings.TrimSpace(tag)
		}
	}

	return nil
}

// isValidRegion checks if the region is valid
func (d *DatadogService) isValidRegion(region string) bool {
	validRegions := []string{"us", "eu", "us3", "us5", "gov", "ap1"}
	for _, valid := range validRegions {
		if region == valid {
			return true
		}
	}
	return false
}

// Send sends a notification to Datadog
func (d *DatadogService) Send(ctx context.Context, req NotificationRequest) error {
	// Create Datadog event
	event := d.createEvent(req)

	// Also create a metric for notification count
	metric := d.createMetric(req)

	// Create log entry
	log := d.createLog(req)

	if d.webhookURL != "" {
		// Send via webhook proxy
		return d.sendViaWebhook(ctx, event, metric, log)
	} else {
		// Send directly to Datadog API
		return d.sendDirectly(ctx, event, metric, log)
	}
}

// createEvent creates a Datadog event from notification request
func (d *DatadogService) createEvent(req NotificationRequest) *DatadogEvent {
	event := &DatadogEvent{
		Title:          req.Title,
		Text:           req.Body,
		DateHappened:   time.Now().Unix(),
		Priority:       d.getPriorityForNotifyType(req.NotifyType),
		AlertType:      d.getAlertTypeForNotifyType(req.NotifyType),
		AggregationKey: fmt.Sprintf("apprise_%s", req.NotifyType.String()),
		SourceTypeName: "apprise-go",
		Tags:           d.mergeTags(req.Tags),
	}

	return event
}

// createMetric creates a metric for notification tracking
func (d *DatadogService) createMetric(req NotificationRequest) *DatadogMetric {
	timestamp := float64(time.Now().Unix())
	value := 1.0

	metric := &DatadogMetric{
		Metric: "apprise.notification",
		Points: [][]interface{}{{timestamp, value}},
		Type:   "count",
		Tags:   d.mergeTags(append(req.Tags, fmt.Sprintf("notification_type:%s", req.NotifyType.String()))),
	}

	return metric
}

// createLog creates a log entry
func (d *DatadogService) createLog(req NotificationRequest) *DatadogLog {
	log := &DatadogLog{
		Message:   fmt.Sprintf("%s: %s", req.Title, req.Body),
		Level:     d.getLogLevelForNotifyType(req.NotifyType),
		Timestamp: time.Now().UnixMilli(),
		Service:   "apprise-go",
		Tags:      d.mergeTags(req.Tags),
		Attributes: map[string]interface{}{
			"notification_type": req.NotifyType.String(),
			"title":            req.Title,
			"source":           "apprise-go",
		},
	}

	// Add body format if specified
	if req.BodyFormat != "" {
		log.Attributes["body_format"] = req.BodyFormat
	}

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
func (d *DatadogService) sendViaWebhook(ctx context.Context, event *DatadogEvent, metric *DatadogMetric, log *DatadogLog) error {
	payload := DatadogWebhookPayload{
		Service: "datadog",
		Region:  d.region,
		Event:   event,
		Metrics: &DatadogMetricsPayload{Series: []DatadogMetric{*metric}},
		Log:     log,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Source:    "apprise-go",
		Version:   GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Datadog webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Datadog webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if d.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", d.proxyAPIKey)
	}

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Datadog webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Datadog webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendDirectly sends data directly to Datadog API
func (d *DatadogService) sendDirectly(ctx context.Context, event *DatadogEvent, metric *DatadogMetric, log *DatadogLog) error {
	baseURL := d.getAPIBaseURL()
	logsURL := d.getLogsAPIURL()

	// Send event
	if err := d.sendEvent(ctx, baseURL, event); err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}

	// Send metric
	if err := d.sendMetric(ctx, baseURL, metric); err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	// Send log
	if err := d.sendLogToURL(ctx, logsURL, log); err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}

	return nil
}

// sendEvent sends an event to Datadog
func (d *DatadogService) sendEvent(ctx context.Context, baseURL string, event *DatadogEvent) error {
	eventURL := fmt.Sprintf("%s/api/v1/events", baseURL)

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", eventURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create event request: %w", err)
	}

	d.setAuthHeaders(httpReq)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Datadog event API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendMetric sends a metric to Datadog
func (d *DatadogService) sendMetric(ctx context.Context, baseURL string, metric *DatadogMetric) error {
	metricsURL := fmt.Sprintf("%s/api/v1/series", baseURL)

	payload := DatadogMetricsPayload{
		Series: []DatadogMetric{*metric},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", metricsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metrics request: %w", err)
	}

	d.setAuthHeaders(httpReq)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Datadog metrics API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendLogToURL sends a log to Datadog logs API
func (d *DatadogService) sendLogToURL(ctx context.Context, logsBaseURL string, log *DatadogLog) error {
	logsURL := fmt.Sprintf("%s/v1/input/%s", logsBaseURL, d.apiKey)

	jsonData, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", logsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create log request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send log: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Datadog logs API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (d *DatadogService) getAPIBaseURL() string {
	switch d.region {
	case "eu":
		return "https://api.datadoghq.eu"
	case "us3":
		return "https://api.us3.datadoghq.com"
	case "us5":
		return "https://api.us5.datadoghq.com"
	case "gov":
		return "https://api.ddog-gov.com"
	case "ap1":
		return "https://api.ap1.datadoghq.com"
	default:
		return "https://api.datadoghq.com" // us region
	}
}

func (d *DatadogService) getLogsAPIURL() string {
	switch d.region {
	case "eu":
		return "https://http-intake.logs.datadoghq.eu"
	case "us3":
		return "https://http-intake.logs.us3.datadoghq.com"
	case "us5":
		return "https://http-intake.logs.us5.datadoghq.com"
	case "gov":
		return "https://http-intake.logs.ddog-gov.com"
	case "ap1":
		return "https://http-intake.logs.ap1.datadoghq.com"
	default:
		return "https://http-intake.logs.datadoghq.com" // us region
	}
}

func (d *DatadogService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("DD-API-KEY", d.apiKey)
	
	if d.appKey != "" {
		req.Header.Set("DD-APPLICATION-KEY", d.appKey)
	}
}

func (d *DatadogService) mergeTags(requestTags []string) []string {
	// Merge default tags with request tags
	allTags := make([]string, 0, len(d.tags)+len(requestTags)+1)
	allTags = append(allTags, d.tags...)
	allTags = append(allTags, requestTags...)
	allTags = append(allTags, "source:apprise-go")

	return allTags
}

func (d *DatadogService) getPriorityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError, NotifyTypeWarning:
		return "normal"
	default:
		return "low"
	}
}

func (d *DatadogService) getAlertTypeForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "error"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeSuccess:
		return "success"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "info"
	}
}

func (d *DatadogService) getLogLevelForNotifyType(notifyType NotifyType) string {
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

// TestURL validates a Datadog service URL
func (d *DatadogService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return d.ParseURL(parsedURL)
}

// SupportsAttachments returns true for Datadog (supports metadata in logs/events)
func (d *DatadogService) SupportsAttachments() bool {
	return true // Datadog supports attachment metadata in events and logs
}

// GetMaxBodyLength returns Datadog's message length limit
func (d *DatadogService) GetMaxBodyLength() int {
	return 8192 // Datadog supports reasonably large event text (8KB)
}

// Example usage and URL formats:
// datadog://api_key@datadoghq.com/?region=us&tags=env:prod,service:api
// datadog://api_key:app_key@datadoghq.com/?region=eu&tags=team:backend
// datadog://proxy-api-key@webhook.example.com/datadog?api_key=dd_key&region=us3&tags=service:alerts