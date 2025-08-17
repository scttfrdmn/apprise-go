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

// AWSIoTService implements AWS IoT Core device notifications
type AWSIoTService struct {
	accessKeyID     string // AWS Access Key ID
	secretAccessKey string // AWS Secret Access Key
	region          string // AWS region (us-east-1, eu-west-1, etc.)
	endpoint        string // IoT Core endpoint (e.g., xxxxxxxxxxxxx-ats.iot.region.amazonaws.com)
	topicName       string // MQTT topic name for publishing messages
	qos             int    // Quality of Service level (0, 1, or 2)
	webhookURL      string // Webhook proxy URL for secure credential management
	proxyAPIKey     string // API key for webhook authentication
	deviceType      string // Device type filter (optional)
	client          *http.Client
}

// AWSIoTMessage represents an IoT message payload
type AWSIoTMessage struct {
	Topic     string                 `json:"topic"`
	Payload   map[string]interface{} `json:"payload"`
	QoS       int                    `json:"qos"`
	Timestamp string                 `json:"timestamp"`
}

// AWSIoTWebhookPayload represents webhook proxy payload
type AWSIoTWebhookPayload struct {
	Service         string        `json:"service"`
	Region          string        `json:"region"`
	Endpoint        string        `json:"endpoint"`
	AccessKeyID     string        `json:"access_key_id"`
	SecretAccessKey string        `json:"secret_access_key"`
	Message         AWSIoTMessage `json:"iot_message"`
	DeviceType      string        `json:"device_type,omitempty"`
	Timestamp       string        `json:"timestamp"`
	Source          string        `json:"source"`
	Version         string        `json:"version"`
}

// NewAWSIoTService creates a new AWS IoT Core service instance
func NewAWSIoTService() Service {
	return &AWSIoTService{
		client: GetCloudHTTPClient("aws-iot"),
		region: "us-east-1", // Default region
		qos:    1,           // Default QoS level
	}
}

// GetServiceID returns the service identifier
func (a *AWSIoTService) GetServiceID() string {
	return "aws-iot"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (a *AWSIoTService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an AWS IoT service URL
// Format: aws-iot://access_key:secret_key@endpoint.iot.us-east-1.amazonaws.com/topic/device/notifications?qos=1&device_type=sensor
// Format: aws-iot://proxy-key@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=us-east-1&endpoint=xxx-ats.iot.region.amazonaws.com&topic=notifications
func (a *AWSIoTService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "aws-iot" {
		return fmt.Errorf("invalid scheme: expected 'aws-iot', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/aws-iot") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		a.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			a.proxyAPIKey = serviceURL.User.Username()
		}

		// Get AWS credentials from query parameters
		a.accessKeyID = query.Get("access_key")
		if a.accessKeyID == "" {
			return fmt.Errorf("access_key parameter is required for webhook mode")
		}

		a.secretAccessKey = query.Get("secret_key")
		if a.secretAccessKey == "" {
			return fmt.Errorf("secret_key parameter is required for webhook mode")
		}

		// Get region
		if region := query.Get("region"); region != "" {
			if a.isValidRegion(region) {
				a.region = region
			} else {
				return fmt.Errorf("invalid AWS region: %s", region)
			}
		}

		// Get IoT endpoint
		a.endpoint = query.Get("endpoint")
		if a.endpoint == "" {
			return fmt.Errorf("endpoint parameter is required for webhook mode")
		}

		// Get topic name
		a.topicName = query.Get("topic")
		if a.topicName == "" {
			return fmt.Errorf("topic parameter is required for webhook mode")
		}
	} else {
		// Direct AWS IoT API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: access_key and secret_key must be provided")
		}

		a.accessKeyID = serviceURL.User.Username()
		if a.accessKeyID == "" {
			return fmt.Errorf("AWS access key is required")
		}

		if secretKey, hasSecret := serviceURL.User.Password(); hasSecret {
			a.secretAccessKey = secretKey
		}
		if a.secretAccessKey == "" {
			return fmt.Errorf("AWS secret key is required")
		}

		// Extract IoT endpoint and region from host
		if serviceURL.Host == "" {
			return fmt.Errorf("IoT endpoint is required")
		}

		a.endpoint = serviceURL.Host

		// Extract region from IoT endpoint if it follows AWS pattern
		// Example: xxxxxxxxxxxxx-ats.iot.us-east-1.amazonaws.com
		hostParts := strings.Split(serviceURL.Host, ".")
		for i, part := range hostParts {
			if part == "iot" && i+1 < len(hostParts) {
				if a.isValidRegion(hostParts[i+1]) {
					a.region = hostParts[i+1]
					break
				}
			}
		}

		// Extract topic from path
		if serviceURL.Path == "" || serviceURL.Path == "/" {
			return fmt.Errorf("topic name is required in path")
		}

		a.topicName = strings.Trim(serviceURL.Path, "/")
		if a.topicName == "" {
			return fmt.Errorf("topic name cannot be empty")
		}
	}

	// Parse optional parameters
	if qosStr := query.Get("qos"); qosStr != "" {
		switch qosStr {
		case "0":
			a.qos = 0
		case "1":
			a.qos = 1
		case "2":
			a.qos = 2
		default:
			return fmt.Errorf("invalid QoS level: %s (valid: 0, 1, 2)", qosStr)
		}
	}

	// Device type filter
	a.deviceType = query.Get("device_type")

	return nil
}

// Send sends an IoT notification via AWS IoT Core
func (a *AWSIoTService) Send(ctx context.Context, req NotificationRequest) error {
	// Build IoT message payload
	message := a.buildIoTMessage(req)

	if a.webhookURL != "" {
		// Send via webhook proxy
		return a.sendViaWebhook(ctx, message)
	} else {
		// Send directly to AWS IoT Core API
		return a.sendToIoTDirectly(ctx, message)
	}
}

// buildIoTMessage creates an IoT message from notification request
func (a *AWSIoTService) buildIoTMessage(req NotificationRequest) AWSIoTMessage {
	payload := map[string]interface{}{
		"title":            req.Title,
		"body":             req.Body,
		"notification_type": req.NotifyType.String(),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"source":           "apprise-go",
	}

	// Add severity mapping
	payload["severity"] = a.getSeverityForNotifyType(req.NotifyType)
	payload["priority"] = a.getPriorityForNotifyType(req.NotifyType)

	// Add tags if present
	if len(req.Tags) > 0 {
		payload["tags"] = req.Tags
	}

	// Add device type if configured
	if a.deviceType != "" {
		payload["device_type"] = a.deviceType
	}

	// Add body format if specified
	if req.BodyFormat != "" {
		payload["body_format"] = req.BodyFormat
	}

	// Add attachment info if present
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		attachmentInfo := make([]map[string]interface{}, 0, len(attachments))
		
		for _, attachment := range attachments {
			info := map[string]interface{}{
				"name":      attachment.GetName(),
				"mime_type": attachment.GetMimeType(),
				"size":      attachment.GetSize(),
			}
			attachmentInfo = append(attachmentInfo, info)
		}
		
		payload["attachments"] = attachmentInfo
		payload["attachment_count"] = len(attachments)
	}

	// Add URL if present
	if req.URL != "" {
		payload["url"] = req.URL
	}

	return AWSIoTMessage{
		Topic:     a.topicName,
		Payload:   payload,
		QoS:       a.qos,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// sendViaWebhook sends message via webhook proxy
func (a *AWSIoTService) sendViaWebhook(ctx context.Context, message AWSIoTMessage) error {
	payload := AWSIoTWebhookPayload{
		Service:         "aws-iot",
		Region:          a.region,
		Endpoint:        a.endpoint,
		AccessKeyID:     a.accessKeyID,
		SecretAccessKey: a.secretAccessKey,
		Message:         message,
		DeviceType:      a.deviceType,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Source:          "apprise-go",
		Version:         GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal AWS IoT webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create AWS IoT webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if a.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", a.proxyAPIKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send AWS IoT webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("aws iot webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendToIoTDirectly sends message directly to AWS IoT Core API
func (a *AWSIoTService) sendToIoTDirectly(ctx context.Context, message AWSIoTMessage) error {
	// AWS IoT Core data endpoint URL
	iotURL := fmt.Sprintf("https://%s/topics/%s", a.endpoint, message.Topic)

	jsonData, err := json.Marshal(message.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal IoT message payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", iotURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create IoT request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Add QoS header if not default
	if message.QoS != 0 {
		httpReq.Header.Set("x-amz-qos", fmt.Sprintf("%d", message.QoS))
	}

	// AWS Signature V4 would normally be required here
	// For now, return an error indicating webhook mode should be used
	return fmt.Errorf("direct AWS IoT API access requires AWS Signature V4 authentication - please use webhook proxy mode")
}

// Helper methods

func (a *AWSIoTService) getSeverityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "critical"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeSuccess:
		return "info"
	case NotifyTypeInfo:
		return "info"
	default:
		return "info"
	}
}

func (a *AWSIoTService) getPriorityForNotifyType(notifyType NotifyType) int {
	switch notifyType {
	case NotifyTypeError:
		return 1 // High priority
	case NotifyTypeWarning:
		return 2 // Medium priority
	case NotifyTypeSuccess:
		return 3 // Low priority
	case NotifyTypeInfo:
		return 3 // Low priority
	default:
		return 3
	}
}

func (a *AWSIoTService) isValidRegion(region string) bool {
	// Common AWS regions that support IoT Core
	validRegions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		"ap-south-1", "ca-central-1", "sa-east-1",
		"ap-east-1", "me-south-1", "af-south-1",
	}

	for _, valid := range validRegions {
		if strings.EqualFold(region, valid) {
			return true
		}
	}
	return false
}

func (a *AWSIoTService) isValidTopic(topic string) bool {
	// Basic IoT topic validation
	if topic == "" {
		return false
	}

	// Topics cannot start or end with /
	if strings.HasPrefix(topic, "/") || strings.HasSuffix(topic, "/") {
		return false
	}

	// Check for invalid characters
	invalidChars := []string{"#", "+", "$"}
	for _, char := range invalidChars {
		if strings.Contains(topic, char) {
			return false
		}
	}

	return true
}

// TestURL validates an AWS IoT service URL
func (a *AWSIoTService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return a.ParseURL(parsedURL)
}

// SupportsAttachments returns true (IoT supports attachment metadata)
func (a *AWSIoTService) SupportsAttachments() bool {
	return true // IoT can include attachment metadata in message payload
}

// GetMaxBodyLength returns AWS IoT's message payload limit
func (a *AWSIoTService) GetMaxBodyLength() int {
	return 131072 // AWS IoT Core limit is 128KB for message payload
}

// Example usage and URL formats:
// aws-iot://access_key:secret_key@xxxxxxxxxxxxx-ats.iot.us-east-1.amazonaws.com/device/notifications?qos=1&device_type=sensor
// aws-iot://proxy-key@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=us-east-1&endpoint=xxx-ats.iot.region.amazonaws.com&topic=alerts