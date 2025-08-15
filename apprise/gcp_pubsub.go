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

// GCPPubSubService implements Google Cloud Pub/Sub notifications via webhook
type GCPPubSubService struct {
	webhookURL    string
	projectID     string
	topicName     string
	serviceAccount string
	orderingKey   string
	attributes    map[string]string
	apiKey        string
	client        *http.Client
}

// NewGCPPubSubService creates a new Google Cloud Pub/Sub service instance
func NewGCPPubSubService() Service {
	return &GCPPubSubService{
		client:     GetCloudHTTPClient("gcp-pubsub"),
		attributes: make(map[string]string),
	}
}

// GetServiceID returns the service identifier
func (g *GCPPubSubService) GetServiceID() string {
	return "pubsub"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (g *GCPPubSubService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Google Cloud Pub/Sub service URL
// Format: pubsub://webhook.url/pubsub-proxy?project_id=my-project&topic=notifications&service_account=sa@project.iam.gserviceaccount.com
// Format: pubsub://api-key@webhook.url/gcp?project_id=company-project&topic=alerts&ordering_key=region
// Format: pubsub://webhook.url/proxy?project_id=my-project&topic=events&attr_environment=prod&attr_service=api
func (g *GCPPubSubService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "pubsub" {
		return fmt.Errorf("invalid scheme: expected 'pubsub', got '%s'", serviceURL.Scheme)
	}

	// Extract API key from userinfo if provided
	if serviceURL.User != nil {
		g.apiKey = serviceURL.User.Username()
	}

	if serviceURL.Host == "" {
		return fmt.Errorf("webhook host is required")
	}

	// Build webhook URL - use HTTP for testing if specified
	scheme := "https" // Default to HTTPS for production
	if serviceURL.Query().Get("test_mode") == "true" {
		scheme = "http" // Use HTTP for testing
	}
	g.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

	// Parse query parameters
	queryParams := serviceURL.Query()

	// Required: project ID
	projectID := queryParams.Get("project_id")
	if projectID == "" {
		return fmt.Errorf("project_id parameter is required")
	}
	g.projectID = projectID

	// Required: topic name
	topicName := queryParams.Get("topic")
	if topicName == "" {
		return fmt.Errorf("topic parameter is required")
	}
	g.topicName = topicName

	// Optional: service account email
	if serviceAccount := queryParams.Get("service_account"); serviceAccount != "" {
		g.serviceAccount = serviceAccount
	}

	// Optional: ordering key for ordered delivery
	if orderingKey := queryParams.Get("ordering_key"); orderingKey != "" {
		g.orderingKey = orderingKey
	}

	// Parse message attributes (prefix: attr_)
	for key, values := range queryParams {
		if strings.HasPrefix(key, "attr_") && len(values) > 0 {
			attrKey := strings.TrimPrefix(key, "attr_")
			g.attributes[attrKey] = values[0]
		}
	}

	return nil
}

// TestURL validates the URL format
func (g *GCPPubSubService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	tempService := NewGCPPubSubService().(*GCPPubSubService)
	return tempService.ParseURL(parsedURL)
}

// SupportsAttachments returns whether this service supports attachments
func (g *GCPPubSubService) SupportsAttachments() bool {
	return false // Pub/Sub messages are primarily for structured data
}

// GetMaxBodyLength returns the maximum body length (10MB for Pub/Sub)
func (g *GCPPubSubService) GetMaxBodyLength() int {
	return 10 * 1024 * 1024 // 10MB
}

// Send sends a notification via Google Cloud Pub/Sub webhook
func (g *GCPPubSubService) Send(ctx context.Context, req NotificationRequest) error {
	// Prepare the message
	messageData := g.formatMessage(req.Title, req.Body, req.NotifyType)

	// Truncate message if too long
	maxLength := g.GetMaxBodyLength()
	if len(messageData) > maxLength {
		messageData = messageData[:maxLength-3] + "..."
	}

	// Create payload for Pub/Sub webhook
	payload := map[string]interface{}{
		"projectId":    g.projectID,
		"topicName":    g.topicName,
		"message":      g.buildMessage(messageData, req),
		"attributes":   g.buildAttributes(req.NotifyType),
	}

	// Add optional fields
	if g.serviceAccount != "" {
		payload["serviceAccount"] = g.serviceAccount
	}

	if g.orderingKey != "" {
		payload["orderingKey"] = g.orderingKey
	}

	return g.sendWebhookRequest(ctx, payload)
}

// formatMessage creates a structured message for Pub/Sub
func (g *GCPPubSubService) formatMessage(title, body string, notifyType NotifyType) string {
	// Create structured message for Pub/Sub
	messageData := map[string]interface{}{
		"title":       title,
		"body":        body,
		"type":        notifyType.String(),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"source":      "apprise-go",
		"version":     GetVersion(),
		"severity":    g.getSeverityLevel(notifyType),
		"category":    "notification",
	}

	// Add visual indicators
	switch notifyType {
	case NotifyTypeSuccess:
		messageData["emoji"] = "✅"
		messageData["color"] = "#28a745"
		messageData["priority"] = "low"
	case NotifyTypeWarning:
		messageData["emoji"] = "⚠️"
		messageData["color"] = "#ffc107"
		messageData["priority"] = "medium"
	case NotifyTypeError:
		messageData["emoji"] = "❌"
		messageData["color"] = "#dc3545"
		messageData["priority"] = "high"
	default:
		messageData["emoji"] = "ℹ️"
		messageData["color"] = "#17a2b8"
		messageData["priority"] = "normal"
	}

	// Add environment context
	messageData["environment"] = map[string]interface{}{
		"project":     g.projectID,
		"topic":       g.topicName,
		"orderingKey": g.orderingKey,
	}

	// Convert to JSON string for Pub/Sub message data
	if jsonBytes, err := json.Marshal(messageData); err == nil {
		return string(jsonBytes)
	}

	// Fallback to simple text format
	var message strings.Builder
	if title != "" {
		message.WriteString(title)
		if body != "" {
			message.WriteString(" - ")
		}
	}
	if body != "" {
		message.WriteString(body)
	}
	return message.String()
}

// getSeverityLevel maps notification types to severity levels for GCP
func (g *GCPPubSubService) getSeverityLevel(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "ERROR"
	case NotifyTypeWarning:
		return "WARNING"
	case NotifyTypeSuccess:
		return "INFO"
	default:
		return "INFO"
	}
}

// buildMessage constructs the Pub/Sub message object
func (g *GCPPubSubService) buildMessage(data string, req NotificationRequest) map[string]interface{} {
	message := map[string]interface{}{
		"data": data, // Base64 encoding will be handled by the webhook
	}

	// Add message ID for deduplication
	message["messageId"] = fmt.Sprintf("apprise-%d-%s", time.Now().Unix(), req.NotifyType.String())

	// Add publish time
	message["publishTime"] = time.Now().UTC().Format(time.RFC3339)

	return message
}

// buildAttributes creates message attributes for filtering and routing
func (g *GCPPubSubService) buildAttributes(notifyType NotifyType) map[string]string {
	attributes := make(map[string]string)

	// Add custom attributes from URL
	for key, value := range g.attributes {
		attributes[key] = value
	}

	// Add standard attributes for Pub/Sub filtering
	attributes["notificationType"] = notifyType.String()
	attributes["source"] = "apprise-go"
	attributes["version"] = GetVersion()
	attributes["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	// Add severity and priority for subscriber filtering
	switch notifyType {
	case NotifyTypeError:
		attributes["severity"] = "ERROR"
		attributes["priority"] = "HIGH"
		attributes["alertLevel"] = "CRITICAL"
	case NotifyTypeWarning:
		attributes["severity"] = "WARNING"
		attributes["priority"] = "MEDIUM"
		attributes["alertLevel"] = "WARNING"
	case NotifyTypeSuccess:
		attributes["severity"] = "INFO"
		attributes["priority"] = "LOW"
		attributes["alertLevel"] = "INFO"
	default:
		attributes["severity"] = "INFO"
		attributes["priority"] = "NORMAL"
		attributes["alertLevel"] = "INFO"
	}

	// Add routing attributes
	attributes["topic"] = g.topicName
	attributes["project"] = g.projectID

	// Add ordering key if specified (for ordered delivery)
	if g.orderingKey != "" {
		attributes["orderingKey"] = g.orderingKey
	}

	return attributes
}

// sendWebhookRequest sends the webhook request to the Pub/Sub gateway
func (g *GCPPubSubService) sendWebhookRequest(ctx context.Context, payload map[string]interface{}) error {
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())

	// Add API key if provided
	if g.apiKey != "" {
		req.Header.Set("X-API-Key", g.apiKey)
		req.Header.Set("Authorization", "Bearer "+g.apiKey)
	}

	// Add Google Cloud-specific headers
	req.Header.Set("X-Google-Cloud-Service", "PubSub")
	req.Header.Set("X-Message-Source", "Apprise-Go")

	// Send the request
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Pub/Sub webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Pub/Sub webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}