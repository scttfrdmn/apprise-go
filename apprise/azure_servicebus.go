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

// AzureServiceBusService implements Azure Service Bus notifications via webhook
type AzureServiceBusService struct {
	webhookURL       string
	namespace        string
	queueName        string
	topicName        string
	subscriptionName string
	sasKeyName       string
	sasKey           string
	connectionString string
	messageProperties map[string]interface{}
	timeToLive       int // TTL in seconds
	apiKey           string
	client           *http.Client
}

// NewAzureServiceBusService creates a new Azure Service Bus service instance
func NewAzureServiceBusService() Service {
	return &AzureServiceBusService{
		client:            GetCloudHTTPClient("azure-servicebus"),
		messageProperties: make(map[string]interface{}),
		timeToLive:        3600, // Default 1 hour TTL
	}
}

// GetServiceID returns the service identifier
func (a *AzureServiceBusService) GetServiceID() string {
	return "azuresb"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (a *AzureServiceBusService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an Azure Service Bus service URL
// Format: azuresb://webhook.url/servicebus?namespace=mybus&queue=notifications&sas_key_name=RootManageSharedAccessKey&sas_key=base64key
// Format: azuresb://api-key@webhook.url/sb?namespace=company-bus&topic=alerts&subscription=email-processor
// Format: azuresb://webhook.url/proxy?connection_string=Endpoint%3Dsb%3A//...&queue=messages
func (a *AzureServiceBusService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "azuresb" {
		return fmt.Errorf("invalid scheme: expected 'azuresb', got '%s'", serviceURL.Scheme)
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

	// Connection string (preferred method)
	if connStr := queryParams.Get("connection_string"); connStr != "" {
		a.connectionString = connStr
		// Extract namespace from connection string if possible
		if strings.Contains(connStr, "Endpoint=sb://") {
			parts := strings.Split(connStr, "sb://")
			if len(parts) > 1 {
				endpointPart := strings.Split(parts[1], ".")[0]
				a.namespace = endpointPart
			}
		}
	} else {
		// Individual parameters method
		namespace := queryParams.Get("namespace")
		if namespace == "" {
			return fmt.Errorf("namespace or connection_string parameter is required")
		}
		a.namespace = namespace

		// SAS authentication
		a.sasKeyName = queryParams.Get("sas_key_name")
		a.sasKey = queryParams.Get("sas_key")
		if a.sasKeyName == "" {
			a.sasKeyName = "RootManageSharedAccessKey" // Default key name
		}
		if a.sasKey == "" && a.apiKey == "" {
			return fmt.Errorf("sas_key parameter or API key authentication is required")
		}
	}

	// Destination: either queue or topic+subscription
	if queueName := queryParams.Get("queue"); queueName != "" {
		a.queueName = queueName
	} else if topicName := queryParams.Get("topic"); topicName != "" {
		a.topicName = topicName
		// Subscription is optional for topics (broadcast)
		if subscription := queryParams.Get("subscription"); subscription != "" {
			a.subscriptionName = subscription
		}
	} else {
		return fmt.Errorf("queue or topic parameter is required")
	}

	// Optional: Time to Live (TTL)
	if ttl := queryParams.Get("ttl"); ttl != "" {
		if ttlSeconds, err := parseInt(ttl); err == nil && ttlSeconds > 0 {
			a.timeToLive = ttlSeconds
		}
	}

	// Parse message properties (prefix: prop_)
	for key, values := range queryParams {
		if strings.HasPrefix(key, "prop_") && len(values) > 0 {
			propKey := strings.TrimPrefix(key, "prop_")
			a.messageProperties[propKey] = values[0]
		}
	}

	return nil
}

// parseInt safely converts string to int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// TestURL validates the URL format
func (a *AzureServiceBusService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	tempService := NewAzureServiceBusService().(*AzureServiceBusService)
	return tempService.ParseURL(parsedURL)
}

// SupportsAttachments returns whether this service supports attachments
func (a *AzureServiceBusService) SupportsAttachments() bool {
	return false // Service Bus messages are text/JSON based
}

// GetMaxBodyLength returns the maximum body length (256KB for Service Bus)
func (a *AzureServiceBusService) GetMaxBodyLength() int {
	return 256 * 1024 // 256KB
}

// Send sends a notification via Azure Service Bus webhook
func (a *AzureServiceBusService) Send(ctx context.Context, req NotificationRequest) error {
	// Prepare the message
	message := a.formatMessage(req.Title, req.Body, req.NotifyType)

	// Truncate message if too long
	maxLength := a.GetMaxBodyLength()
	if len(message) > maxLength {
		message = message[:maxLength-3] + "..."
	}

	// Create payload for Service Bus webhook
	payload := map[string]interface{}{
		"namespace":       a.namespace,
		"authentication":  a.buildAuthentication(),
		"destination":     a.buildDestination(),
		"message":         a.buildMessage(message, req),
		"messageProperties": a.buildMessageProperties(req.NotifyType),
	}

	// Add connection string if available
	if a.connectionString != "" {
		payload["connectionString"] = a.connectionString
	}

	return a.sendWebhookRequest(ctx, payload)
}

// formatMessage formats the message based on the notification type and Service Bus requirements
func (a *AzureServiceBusService) formatMessage(title, body string, notifyType NotifyType) string {
	// Create structured message for Service Bus
	messageData := map[string]interface{}{
		"title":       title,
		"body":        body,
		"type":        notifyType.String(),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"source":      "apprise-go",
		"severity":    a.getSeverityLevel(notifyType),
		"category":    "notification",
	}

	// Add emoji for visual context
	switch notifyType {
	case NotifyTypeSuccess:
		messageData["emoji"] = "✅"
		messageData["color"] = "#28a745"
	case NotifyTypeWarning:
		messageData["emoji"] = "⚠️"
		messageData["color"] = "#ffc107"
	case NotifyTypeError:
		messageData["emoji"] = "❌"
		messageData["color"] = "#dc3545"
	default:
		messageData["emoji"] = "ℹ️"
		messageData["color"] = "#17a2b8"
	}

	// Convert to JSON string for Service Bus message body
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

// getSeverityLevel maps notification types to severity levels
func (a *AzureServiceBusService) getSeverityLevel(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "Critical"
	case NotifyTypeWarning:
		return "Warning"
	case NotifyTypeSuccess:
		return "Informational"
	default:
		return "Informational"
	}
}

// buildAuthentication constructs the authentication object for Service Bus
func (a *AzureServiceBusService) buildAuthentication() map[string]interface{} {
	auth := make(map[string]interface{})
	
	if a.sasKeyName != "" && a.sasKey != "" {
		auth["type"] = "sas"
		auth["keyName"] = a.sasKeyName
		auth["key"] = a.sasKey
	} else {
		auth["type"] = "managed_identity" // Assume managed identity
	}
	
	return auth
}

// buildDestination constructs the destination object (queue or topic)
func (a *AzureServiceBusService) buildDestination() map[string]interface{} {
	destination := make(map[string]interface{})
	
	if a.queueName != "" {
		destination["type"] = "queue"
		destination["name"] = a.queueName
	} else if a.topicName != "" {
		destination["type"] = "topic"
		destination["name"] = a.topicName
		if a.subscriptionName != "" {
			destination["subscription"] = a.subscriptionName
		}
	}
	
	return destination
}

// buildMessage constructs the Service Bus message object
func (a *AzureServiceBusService) buildMessage(content string, req NotificationRequest) map[string]interface{} {
	message := map[string]interface{}{
		"body":        content,
		"contentType": "application/json",
		"timeToLive":  a.timeToLive,
	}

	// Add message label (subject) for routing
	label := req.Title
	if label == "" {
		label = fmt.Sprintf("Apprise Notification (%s)", req.NotifyType.String())
	}
	message["label"] = label

	// Add correlation ID for tracking
	message["correlationId"] = fmt.Sprintf("apprise-%d", time.Now().Unix())

	// Add reply-to for two-way messaging scenarios
	message["replyTo"] = a.queueName // Reply to same queue by default

	return message
}

// buildMessageProperties creates custom message properties
func (a *AzureServiceBusService) buildMessageProperties(notifyType NotifyType) map[string]interface{} {
	properties := make(map[string]interface{})

	// Add custom properties from URL
	for key, value := range a.messageProperties {
		properties[key] = value
	}

	// Add standard properties
	properties["NotificationType"] = notifyType.String()
	properties["Source"] = "apprise-go"
	properties["Timestamp"] = time.Now().UTC().Format(time.RFC3339)
	properties["Version"] = GetVersion()

	// Add severity and priority
	switch notifyType {
	case NotifyTypeError:
		properties["Priority"] = "High"
		properties["Severity"] = "Critical"
	case NotifyTypeWarning:
		properties["Priority"] = "Medium"
		properties["Severity"] = "Warning"
	case NotifyTypeSuccess:
		properties["Priority"] = "Low"
		properties["Severity"] = "Informational"
	default:
		properties["Priority"] = "Normal"
		properties["Severity"] = "Informational"
	}

	return properties
}

// sendWebhookRequest sends the webhook request to the Service Bus gateway
func (a *AzureServiceBusService) sendWebhookRequest(ctx context.Context, payload map[string]interface{}) error {
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
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	// Add Azure-specific headers
	req.Header.Set("X-Azure-Service", "ServiceBus")
	req.Header.Set("X-Message-Source", "Apprise-Go")

	// Send the request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Service Bus webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Service Bus webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}