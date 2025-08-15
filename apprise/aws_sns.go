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

// AWSSNSService implements Amazon SNS notifications via API Gateway webhook
type AWSSNSService struct {
	webhookURL    string
	topicArn      string
	region        string
	subject       string
	messageFormat string // "json" or "text"
	attributes    map[string]string
	apiKey        string
	client        *http.Client
}

// NewAWSSNSService creates a new AWS SNS service instance
func NewAWSSNSService() Service {
	return &AWSSNSService{
		client:        &http.Client{},
		region:        "us-east-1",
		messageFormat: "text",
		attributes:    make(map[string]string),
	}
}

// GetServiceID returns the service identifier
func (a *AWSSNSService) GetServiceID() string {
	return "sns"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (a *AWSSNSService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an AWS SNS service URL
// Format: sns://api.gateway.url/path/to/endpoint?topic_arn=arn:aws:sns:region:account:topic
// Format: sns://apikey@api.gateway.url/path?topic_arn=arn:aws:sns:region:account:topic
// Format: sns://webhook.url/sns?topic=my-topic&region=us-east-1&account=123456789
func (a *AWSSNSService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "sns" {
		return fmt.Errorf("invalid scheme: expected 'sns', got '%s'", serviceURL.Scheme)
	}

	// Extract API key from userinfo if provided
	if serviceURL.User != nil {
		a.apiKey = serviceURL.User.Username()
	}

	if serviceURL.Host == "" {
		return fmt.Errorf("webhook host is required")
	}

	// Build webhook URL - use HTTP for testing if specified
	scheme := "https" // Default to HTTPS for production
	if serviceURL.Query().Get("test_mode") == "true" {
		scheme = "http" // Use HTTP for testing
	}
	a.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

	// Parse query parameters
	queryParams := serviceURL.Query()

	// Get topic ARN directly or build it from components
	if topicArn := queryParams.Get("topic_arn"); topicArn != "" {
		a.topicArn = topicArn
		// Extract region from ARN
		parts := strings.Split(topicArn, ":")
		if len(parts) >= 4 {
			a.region = parts[3]
		}
	} else {
		// Build ARN from components
		topic := queryParams.Get("topic")
		region := queryParams.Get("region")
		account := queryParams.Get("account")

		if topic == "" {
			return fmt.Errorf("topic_arn or topic parameter is required")
		}

		if region != "" {
			a.region = region
		}

		if account != "" {
			a.topicArn = fmt.Sprintf("arn:aws:sns:%s:%s:%s", a.region, account, topic)
		} else {
			// Leave topicArn empty and let the webhook handle topic resolution
			a.topicArn = topic
		}
	}

	if subject := queryParams.Get("subject"); subject != "" {
		a.subject = subject
	}

	if format := queryParams.Get("format"); format != "" {
		if format == "json" || format == "text" {
			a.messageFormat = format
		}
	}

	// Parse message attributes (prefix: attr_)
	for key, values := range queryParams {
		if strings.HasPrefix(key, "attr_") && len(values) > 0 {
			attrName := strings.TrimPrefix(key, "attr_")
			a.attributes[attrName] = values[0]
		}
	}

	return nil
}

// TestURL validates the URL format
func (a *AWSSNSService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	tempService := NewAWSSNSService().(*AWSSNSService)
	return tempService.ParseURL(parsedURL)
}

// SupportsAttachments returns whether this service supports attachments
func (a *AWSSNSService) SupportsAttachments() bool {
	return false // SNS doesn't support attachments directly
}

// GetMaxBodyLength returns the maximum body length (256KB for SNS)
func (a *AWSSNSService) GetMaxBodyLength() int {
	return 256 * 1024 // 256KB
}

// Send sends a notification via AWS SNS webhook
func (a *AWSSNSService) Send(ctx context.Context, req NotificationRequest) error {
	// Truncate body if too long (leave room for title, formatting, etc.)
	body := req.Body
	maxLength := a.GetMaxBodyLength()
	if len(req.Title)+len(body) > maxLength-100 { // Leave 100 chars for formatting
		availableLength := maxLength - len(req.Title) - 100
		if availableLength > 0 {
			body = body[:availableLength] + "..."
		}
	}

	// Prepare the message with potentially truncated body
	message := a.formatMessage(req.Title, body, req.NotifyType)

	// Prepare subject
	subject := a.subject
	if subject == "" {
		subject = req.Title
		if subject == "" {
			subject = fmt.Sprintf("Notification (%s)", req.NotifyType.String())
		}
	}

	// Create payload for SNS webhook
	payload := map[string]interface{}{
		"topicArn": a.topicArn,
		"message":  message,
		"subject":  subject,
		"messageAttributes": a.buildMessageAttributes(req.NotifyType),
	}

	// Add region if available
	if a.region != "" {
		payload["region"] = a.region
	}

	return a.sendWebhookRequest(ctx, payload)
}

// formatMessage formats the message based on the notification type
func (a *AWSSNSService) formatMessage(title, body string, notifyType NotifyType) string {
	if a.messageFormat == "json" {
		// Create structured JSON message
		messageData := map[string]interface{}{
			"title":     title,
			"body":      body,
			"type":      notifyType.String(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		// Add notification type emoji
		switch notifyType {
		case NotifyTypeSuccess:
			messageData["emoji"] = "✅"
		case NotifyTypeWarning:
			messageData["emoji"] = "⚠️"
		case NotifyTypeError:
			messageData["emoji"] = "❌"
		default:
			messageData["emoji"] = "ℹ️"
		}

		if jsonBytes, err := json.Marshal(messageData); err == nil {
			return string(jsonBytes)
		}
	}

	// Default text format
	var message strings.Builder

	// Add emoji prefix based on type
	switch notifyType {
	case NotifyTypeSuccess:
		message.WriteString("✅ ")
	case NotifyTypeWarning:
		message.WriteString("⚠️ ")
	case NotifyTypeError:
		message.WriteString("❌ ")
	case NotifyTypeInfo:
		message.WriteString("ℹ️ ")
	}

	if title != "" {
		message.WriteString(title)
		if body != "" {
			message.WriteString("\n\n")
		}
	}

	if body != "" {
		message.WriteString(body)
	}

	return message.String()
}

// buildMessageAttributes creates SNS message attributes
func (a *AWSSNSService) buildMessageAttributes(notifyType NotifyType) map[string]interface{} {
	attributes := make(map[string]interface{})

	// Add custom attributes
	for key, value := range a.attributes {
		attributes[key] = map[string]string{
			"DataType":    "String",
			"StringValue": value,
		}
	}

	// Add notification type attribute
	attributes["NotificationType"] = map[string]string{
		"DataType":    "String",
		"StringValue": notifyType.String(),
	}

	// Add source attribute
	attributes["Source"] = map[string]string{
		"DataType":    "String",
		"StringValue": "apprise-go",
	}

	return attributes
}

// sendWebhookRequest sends the webhook request to the SNS gateway
func (a *AWSSNSService) sendWebhookRequest(ctx context.Context, payload map[string]interface{}) error {
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())

	// Add API key if provided
	if a.apiKey != "" {
		req.Header.Set("X-API-Key", a.apiKey)
		// Also support Authorization header
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	// Send the request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SNS webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SNS webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}