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

// GCPIoTService implements Google Cloud IoT Core device notifications
type GCPIoTService struct {
	projectID       string // GCP project ID
	region          string // GCP region (us-central1, europe-west1, etc.)
	registryID      string // IoT Core device registry ID
	deviceID        string // Target device ID (optional, for device-specific messages)
	serviceAccount  string // Service account email for authentication
	privateKey      string // Service account private key (PEM format)
	webhookURL      string // Webhook proxy URL for secure credential management
	proxyAPIKey     string // API key for webhook authentication
	messageType     string // Message type: config, state, or event
	client          *http.Client
}

// GCPIoTMessage represents a Google Cloud IoT message
type GCPIoTMessage struct {
	ProjectID    string                 `json:"project_id"`
	Region       string                 `json:"region"`
	RegistryID   string                 `json:"registry_id"`
	DeviceID     string                 `json:"device_id,omitempty"`
	MessageType  string                 `json:"message_type"`
	Payload      map[string]interface{} `json:"payload"`
	Timestamp    string                 `json:"timestamp"`
}

// GCPIoTWebhookPayload represents webhook proxy payload
type GCPIoTWebhookPayload struct {
	Service        string        `json:"service"`
	ProjectID      string        `json:"project_id"`
	Region         string        `json:"region"`
	RegistryID     string        `json:"registry_id"`
	ServiceAccount string        `json:"service_account"`
	PrivateKey     string        `json:"private_key"`
	Message        GCPIoTMessage `json:"iot_message"`
	Timestamp      string        `json:"timestamp"`
	Source         string        `json:"source"`
	Version        string        `json:"version"`
}

// NewGCPIoTService creates a new Google Cloud IoT Core service instance
func NewGCPIoTService() Service {
	return &GCPIoTService{
		client:      GetCloudHTTPClient("gcp-iot"),
		region:      "us-central1", // Default region
		messageType: "event",       // Default message type
	}
}

// GetServiceID returns the service identifier
func (g *GCPIoTService) GetServiceID() string {
	return "gcp-iot"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (g *GCPIoTService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Google Cloud IoT service URL
// Format: gcp-iot://service_account:private_key@cloudiot.googleapis.com/projects/PROJECT_ID/locations/REGION/registries/REGISTRY_ID?device_id=DEVICE_ID&message_type=event
// Format: gcp-iot://proxy-key@webhook.example.com/gcp-iot?project_id=PROJECT&region=REGION&registry_id=REGISTRY&service_account=EMAIL&private_key=KEY
func (g *GCPIoTService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "gcp-iot" {
		return fmt.Errorf("invalid scheme: expected 'gcp-iot', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/gcp-iot") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		g.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			g.proxyAPIKey = serviceURL.User.Username()
		}

		// Get GCP credentials and configuration from query parameters
		g.projectID = query.Get("project_id")
		if g.projectID == "" {
			return fmt.Errorf("project_id parameter is required for webhook mode")
		}

		g.region = query.Get("region")
		if g.region == "" {
			g.region = "us-central1" // Default region
		}
		if !g.isValidRegion(g.region) {
			return fmt.Errorf("invalid GCP region: %s", g.region)
		}

		g.registryID = query.Get("registry_id")
		if g.registryID == "" {
			return fmt.Errorf("registry_id parameter is required for webhook mode")
		}

		g.serviceAccount = query.Get("service_account")
		if g.serviceAccount == "" {
			return fmt.Errorf("service_account parameter is required for webhook mode")
		}

		g.privateKey = query.Get("private_key")
		if g.privateKey == "" {
			return fmt.Errorf("private_key parameter is required for webhook mode")
		}

		// Optional device ID for device-specific messages
		g.deviceID = query.Get("device_id")
	} else {
		// Direct GCP IoT API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: service_account and private_key must be provided")
		}

		g.serviceAccount = serviceURL.User.Username()
		if g.serviceAccount == "" {
			return fmt.Errorf("service account email is required")
		}

		if privateKey, hasKey := serviceURL.User.Password(); hasKey {
			g.privateKey = privateKey
		}
		if g.privateKey == "" {
			return fmt.Errorf("service account private key is required")
		}

		// Parse path to extract project, region, and registry
		// Expected format: /projects/PROJECT_ID/locations/REGION/registries/REGISTRY_ID
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		if len(pathParts) < 6 {
			return fmt.Errorf("invalid path format: expected /projects/PROJECT_ID/locations/REGION/registries/REGISTRY_ID")
		}

		if pathParts[0] != "projects" {
			return fmt.Errorf("path must start with /projects/")
		}
		g.projectID = pathParts[1]

		if pathParts[2] != "locations" {
			return fmt.Errorf("invalid path format: expected 'locations' after project ID")
		}
		g.region = pathParts[3]
		if !g.isValidRegion(g.region) {
			return fmt.Errorf("invalid GCP region: %s", g.region)
		}

		if pathParts[4] != "registries" {
			return fmt.Errorf("invalid path format: expected 'registries' after region")
		}
		g.registryID = pathParts[5]

		// Optional device ID from query
		g.deviceID = query.Get("device_id")
	}

	// Parse optional message type
	if messageType := query.Get("message_type"); messageType != "" {
		if g.isValidMessageType(messageType) {
			g.messageType = messageType
		} else {
			return fmt.Errorf("invalid message type: %s (valid: config, state, event)", messageType)
		}
	}

	return nil
}

// Send sends an IoT notification via Google Cloud IoT Core
func (g *GCPIoTService) Send(ctx context.Context, req NotificationRequest) error {
	// Build IoT message
	message := g.buildIoTMessage(req)

	if g.webhookURL != "" {
		// Send via webhook proxy
		return g.sendViaWebhook(ctx, message)
	} else {
		// Send directly to GCP IoT Core API
		return g.sendToGCPIoTDirectly(ctx, message)
	}
}

// buildIoTMessage creates a GCP IoT message from notification request
func (g *GCPIoTService) buildIoTMessage(req NotificationRequest) GCPIoTMessage {
	payload := map[string]interface{}{
		"title":            req.Title,
		"body":             req.Body,
		"notification_type": req.NotifyType.String(),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"source":           "apprise-go",
	}

	// Add severity and priority mapping
	payload["severity"] = g.getSeverityForNotifyType(req.NotifyType)
	payload["priority"] = g.getPriorityForNotifyType(req.NotifyType)

	// Add tags if present
	if len(req.Tags) > 0 {
		payload["tags"] = req.Tags
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

	return GCPIoTMessage{
		ProjectID:   g.projectID,
		Region:      g.region,
		RegistryID:  g.registryID,
		DeviceID:    g.deviceID,
		MessageType: g.messageType,
		Payload:     payload,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// sendViaWebhook sends message via webhook proxy
func (g *GCPIoTService) sendViaWebhook(ctx context.Context, message GCPIoTMessage) error {
	payload := GCPIoTWebhookPayload{
		Service:        "gcp-iot",
		ProjectID:      g.projectID,
		Region:         g.region,
		RegistryID:     g.registryID,
		ServiceAccount: g.serviceAccount,
		PrivateKey:     g.privateKey,
		Message:        message,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Source:         "apprise-go",
		Version:        GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GCP IoT webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GCP IoT webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if g.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", g.proxyAPIKey)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send GCP IoT webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gcp iot webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendToGCPIoTDirectly sends message directly to GCP IoT Core API
func (g *GCPIoTService) sendToGCPIoTDirectly(ctx context.Context, message GCPIoTMessage) error {
	var apiURL string
	
	switch message.MessageType {
	case "config":
		// Send device configuration
		if message.DeviceID == "" {
			return fmt.Errorf("device_id is required for config messages")
		}
		apiURL = fmt.Sprintf("https://cloudiot.googleapis.com/v1/projects/%s/locations/%s/registries/%s/devices/%s:modifyCloudToDeviceConfig",
			g.projectID, g.region, g.registryID, message.DeviceID)
	case "state":
		// Device state updates (read-only, not supported for sending)
		return fmt.Errorf("state messages are read-only and cannot be sent to devices")
	case "event":
		// Send telemetry event (requires device authentication, not service account)
		return fmt.Errorf("event messages require device authentication - please use webhook proxy mode for device events")
	default:
		return fmt.Errorf("unsupported message type: %s", message.MessageType)
	}

	requestBody := map[string]interface{}{
		"binaryData": message.Payload, // GCP IoT expects base64-encoded binary data
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal GCP IoT request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GCP IoT request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Google Cloud authentication would normally be required here
	// For now, return an error indicating webhook mode should be used
	return fmt.Errorf("direct GCP IoT API access requires Google Cloud authentication - please use webhook proxy mode")
}

// Helper methods

func (g *GCPIoTService) getSeverityForNotifyType(notifyType NotifyType) string {
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

func (g *GCPIoTService) getPriorityForNotifyType(notifyType NotifyType) int {
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

func (g *GCPIoTService) isValidRegion(region string) bool {
	// Common GCP regions that support IoT Core
	validRegions := []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
		"europe-north1", "europe-central2",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1",
	}

	for _, valid := range validRegions {
		if strings.EqualFold(region, valid) {
			return true
		}
	}
	return false
}

func (g *GCPIoTService) isValidMessageType(messageType string) bool {
	validTypes := []string{"config", "state", "event"}
	for _, valid := range validTypes {
		if strings.EqualFold(messageType, valid) {
			return true
		}
	}
	return false
}

// TestURL validates a Google Cloud IoT service URL
func (g *GCPIoTService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return g.ParseURL(parsedURL)
}

// SupportsAttachments returns true (IoT supports attachment metadata)
func (g *GCPIoTService) SupportsAttachments() bool {
	return true // GCP IoT can include attachment metadata in message payload
}

// GetMaxBodyLength returns GCP IoT's message payload limit
func (g *GCPIoTService) GetMaxBodyLength() int {
	return 262144 // GCP IoT Core limit is 256KB for message payload
}

// Example usage and URL formats:
// gcp-iot://service_account:private_key@cloudiot.googleapis.com/projects/PROJECT_ID/locations/us-central1/registries/REGISTRY_ID?device_id=DEVICE_ID&message_type=config
// gcp-iot://proxy-key@webhook.example.com/gcp-iot?project_id=my-project&region=us-central1&registry_id=my-registry&service_account=email@project.iam.gserviceaccount.com&private_key=KEY