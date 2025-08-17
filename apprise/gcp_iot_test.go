package apprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGCPIoTService_GetServiceID(t *testing.T) {
	service := NewGCPIoTService()
	if service.GetServiceID() != "gcp-iot" {
		t.Errorf("Expected service ID 'gcp-iot', got '%s'", service.GetServiceID())
	}
}

func TestGCPIoTService_GetDefaultPort(t *testing.T) {
	service := NewGCPIoTService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestGCPIoTService_SupportsAttachments(t *testing.T) {
	service := NewGCPIoTService()
	if !service.SupportsAttachments() {
		t.Error("GCP IoT should support attachments (metadata)")
	}
}

func TestGCPIoTService_GetMaxBodyLength(t *testing.T) {
	service := NewGCPIoTService()
	expected := 262144
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestGCPIoTService_ParseURL(t *testing.T) {
	tests := []struct {
		name                   string
		url                    string
		expectError            bool
		expectedProjectID      string
		expectedRegion         string
		expectedRegistryID     string
		expectedDeviceID       string
		expectedServiceAccount string
		expectedPrivateKey     string
		expectedMessageType    string
		expectedWebhook        string
		expectedProxyKey       string
	}{
		{
			name:                   "Basic GCP IoT URL",
			url:                    "gcp-iot://service@project.iam.gserviceaccount.com:private_key_data@cloudiot.googleapis.com/projects/my-project/locations/us-central1/registries/my-registry",
			expectError:            false,
			expectedProjectID:      "my-project",
			expectedRegion:         "us-central1",
			expectedRegistryID:     "my-registry",
			expectedServiceAccount: "service@project.iam.gserviceaccount.com",
			expectedPrivateKey:     "private_key_data",
			expectedMessageType:    "event", // default
		},
		{
			name:                   "With device ID and message type",
			url:                    "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/test-project/locations/europe-west1/registries/test-registry?device_id=sensor-001&message_type=config",
			expectError:            false,
			expectedProjectID:      "test-project",
			expectedRegion:         "europe-west1",
			expectedRegistryID:     "test-registry",
			expectedDeviceID:       "sensor-001",
			expectedServiceAccount: "service@project.iam.gserviceaccount.com",
			expectedPrivateKey:     "key",
			expectedMessageType:    "config",
		},
		{
			name:                   "Webhook proxy mode",
			url:                    "gcp-iot://proxy-key@webhook.example.com/gcp-iot?project_id=my-project&region=us-central1&registry_id=my-registry&service_account=service@project.iam.gserviceaccount.com&private_key=key",
			expectError:            false,
			expectedWebhook:        "https://webhook.example.com/gcp-iot",
			expectedProxyKey:       "proxy-key",
			expectedProjectID:      "my-project",
			expectedRegion:         "us-central1",
			expectedRegistryID:     "my-registry",
			expectedServiceAccount: "service@project.iam.gserviceaccount.com",
			expectedPrivateKey:     "key",
			expectedMessageType:    "event", // default
		},
		{
			name:                   "Webhook with full parameters",
			url:                    "gcp-iot://proxy@webhook.example.com/gcp-iot?project_id=test&region=asia-east1&registry_id=sensors&service_account=iot@test.iam.gserviceaccount.com&private_key=secret&device_id=device-123&message_type=config",
			expectError:            false,
			expectedWebhook:        "https://webhook.example.com/gcp-iot",
			expectedProxyKey:       "proxy",
			expectedProjectID:      "test",
			expectedRegion:         "asia-east1",
			expectedRegistryID:     "sensors",
			expectedDeviceID:       "device-123",
			expectedServiceAccount: "iot@test.iam.gserviceaccount.com",
			expectedPrivateKey:     "secret",
			expectedMessageType:    "config",
		},
		{
			name:        "Invalid scheme",
			url:         "http://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Missing service account",
			url:         "gcp-iot://:key@cloudiot.googleapis.com/projects/project/locations/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Missing private key",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com@cloudiot.googleapis.com/projects/project/locations/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Invalid path format - missing projects",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/project/locations/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Invalid path format - missing locations",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Invalid path format - missing registries",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/region/registry",
			expectError: true,
		},
		{
			name:        "Invalid region",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/invalid-region/registries/registry",
			expectError: true,
		},
		{
			name:        "Invalid message type",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/us-central1/registries/registry?message_type=invalid",
			expectError: true,
		},
		{
			name:        "Webhook missing project_id",
			url:         "gcp-iot://proxy@webhook.example.com/gcp-iot?region=us-central1&registry_id=registry&service_account=service@project.iam.gserviceaccount.com&private_key=key",
			expectError: true,
		},
		{
			name:        "Webhook missing registry_id",
			url:         "gcp-iot://proxy@webhook.example.com/gcp-iot?project_id=project&region=us-central1&service_account=service@project.iam.gserviceaccount.com&private_key=key",
			expectError: true,
		},
		{
			name:        "Webhook missing service_account",
			url:         "gcp-iot://proxy@webhook.example.com/gcp-iot?project_id=project&region=us-central1&registry_id=registry&private_key=key",
			expectError: true,
		},
		{
			name:        "Webhook missing private_key",
			url:         "gcp-iot://proxy@webhook.example.com/gcp-iot?project_id=project&region=us-central1&registry_id=registry&service_account=service@project.iam.gserviceaccount.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGCPIoTService().(*GCPIoTService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse URL: %v", err)
				}
				return
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.projectID != tt.expectedProjectID {
				t.Errorf("Expected project ID '%s', got '%s'", tt.expectedProjectID, service.projectID)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
			}

			if service.registryID != tt.expectedRegistryID {
				t.Errorf("Expected registry ID '%s', got '%s'", tt.expectedRegistryID, service.registryID)
			}

			if tt.expectedDeviceID != "" && service.deviceID != tt.expectedDeviceID {
				t.Errorf("Expected device ID '%s', got '%s'", tt.expectedDeviceID, service.deviceID)
			}

			if service.serviceAccount != tt.expectedServiceAccount {
				t.Errorf("Expected service account '%s', got '%s'", tt.expectedServiceAccount, service.serviceAccount)
			}

			if service.privateKey != tt.expectedPrivateKey {
				t.Errorf("Expected private key '%s', got '%s'", tt.expectedPrivateKey, service.privateKey)
			}

			if service.messageType != tt.expectedMessageType {
				t.Errorf("Expected message type '%s', got '%s'", tt.expectedMessageType, service.messageType)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedProxyKey != "" && service.proxyAPIKey != tt.expectedProxyKey {
				t.Errorf("Expected proxy key '%s', got '%s'", tt.expectedProxyKey, service.proxyAPIKey)
			}
		})
	}
}

func TestGCPIoTService_TestURL(t *testing.T) {
	service := NewGCPIoTService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid GCP IoT URL",
			url:         "gcp-iot://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/us-central1/registries/registry",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "gcp-iot://proxy@webhook.example.com/gcp-iot?project_id=project&region=us-central1&registry_id=registry&service_account=service@project.iam.gserviceaccount.com&private_key=key",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://service@project.iam.gserviceaccount.com:key@cloudiot.googleapis.com/projects/project/locations/region/registries/registry",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "gcp-iot://cloudiot.googleapis.com/projects/project/locations/region/registries/registry",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGCPIoTService_SendWebhook(t *testing.T) {
	// Create mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if !strings.Contains(r.Header.Get("User-Agent"), "Apprise-Go") {
			t.Errorf("Expected User-Agent to contain Apprise-Go, got %s", r.Header.Get("User-Agent"))
		}

		// Verify authentication
		if r.Header.Get("X-API-Key") != "test-proxy-key" {
			t.Errorf("Expected X-API-Key 'test-proxy-key', got '%s'", r.Header.Get("X-API-Key"))
		}

		// Parse and verify request body
		var payload GCPIoTWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "gcp-iot" {
			t.Errorf("Expected service 'gcp-iot', got '%s'", payload.Service)
		}

		if payload.ProjectID != "test-project" {
			t.Errorf("Expected project ID 'test-project', got '%s'", payload.ProjectID)
		}

		if payload.Region != "us-central1" {
			t.Errorf("Expected region 'us-central1', got '%s'", payload.Region)
		}

		if payload.RegistryID != "test-registry" {
			t.Errorf("Expected registry ID 'test-registry', got '%s'", payload.RegistryID)
		}

		if payload.ServiceAccount != "service@test-project.iam.gserviceaccount.com" {
			t.Errorf("Expected service account 'service@test-project.iam.gserviceaccount.com', got '%s'", payload.ServiceAccount)
		}

		if payload.PrivateKey != "private-key-data" {
			t.Errorf("Expected private key 'private-key-data', got '%s'", payload.PrivateKey)
		}

		// Verify message structure
		if payload.Message.ProjectID != "test-project" {
			t.Errorf("Expected message project ID 'test-project', got '%s'", payload.Message.ProjectID)
		}

		if payload.Message.RegistryID != "test-registry" {
			t.Errorf("Expected message registry ID 'test-registry', got '%s'", payload.Message.RegistryID)
		}

		if payload.Message.DeviceID != "sensor-001" {
			t.Errorf("Expected message device ID 'sensor-001', got '%s'", payload.Message.DeviceID)
		}

		if payload.Message.MessageType != "event" {
			t.Errorf("Expected message type 'event', got '%s'", payload.Message.MessageType)
		}

		// Verify message payload
		if payload.Message.Payload["title"] != "IoT Sensor Alert" {
			t.Errorf("Expected title 'IoT Sensor Alert', got '%v'", payload.Message.Payload["title"])
		}

		if payload.Message.Payload["body"] != "Temperature sensor reading anomaly detected" {
			t.Errorf("Expected body to match, got '%v'", payload.Message.Payload["body"])
		}

		if payload.Message.Payload["notification_type"] != "warning" {
			t.Errorf("Expected notification_type 'warning', got '%v'", payload.Message.Payload["notification_type"])
		}

		if payload.Message.Payload["severity"] != "warning" {
			t.Errorf("Expected severity 'warning', got '%v'", payload.Message.Payload["severity"])
		}

		if priority, ok := payload.Message.Payload["priority"].(float64); !ok || int(priority) != 2 {
			t.Errorf("Expected priority 2, got %v", payload.Message.Payload["priority"])
		}

		if payload.Message.Payload["source"] != "apprise-go" {
			t.Errorf("Expected source 'apprise-go', got '%v'", payload.Message.Payload["source"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true, "message_id": "projects/test-project/locations/us-central1/registries/test-registry/devices/sensor-001"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewGCPIoTService().(*GCPIoTService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.projectID = "test-project"
	service.region = "us-central1"
	service.registryID = "test-registry"
	service.deviceID = "sensor-001"
	service.serviceAccount = "service@test-project.iam.gserviceaccount.com"
	service.privateKey = "private-key-data"
	service.messageType = "event"

	req := NotificationRequest{
		Title:      "IoT Sensor Alert",
		Body:       "Temperature sensor reading anomaly detected",
		NotifyType: NotifyTypeWarning,
		Tags:       []string{"temperature", "sensor", "anomaly"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestGCPIoTService_BuildIoTMessage(t *testing.T) {
	service := &GCPIoTService{
		projectID:   "test-project",
		region:      "us-central1",
		registryID:  "sensors",
		deviceID:    "temp-sensor-01",
		messageType: "config",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("config data"), "config.json", "application/json")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Device Configuration Update",
		Body:          "Updated temperature threshold settings",
		NotifyType:    NotifyTypeSuccess,
		Tags:          []string{"config", "temperature", "threshold"},
		BodyFormat:    "json",
		URL:           "https://console.cloud.google.com/iot/registries/sensors",
		AttachmentMgr: attachmentMgr,
	}

	message := service.buildIoTMessage(req)

	// Check basic fields
	if message.ProjectID != "test-project" {
		t.Errorf("Expected project ID 'test-project', got '%s'", message.ProjectID)
	}

	if message.Region != "us-central1" {
		t.Errorf("Expected region 'us-central1', got '%s'", message.Region)
	}

	if message.RegistryID != "sensors" {
		t.Errorf("Expected registry ID 'sensors', got '%s'", message.RegistryID)
	}

	if message.DeviceID != "temp-sensor-01" {
		t.Errorf("Expected device ID 'temp-sensor-01', got '%s'", message.DeviceID)
	}

	if message.MessageType != "config" {
		t.Errorf("Expected message type 'config', got '%s'", message.MessageType)
	}

	// Check payload fields
	if message.Payload["title"] != req.Title {
		t.Errorf("Expected title '%s', got '%v'", req.Title, message.Payload["title"])
	}

	if message.Payload["body"] != req.Body {
		t.Errorf("Expected body '%s', got '%v'", req.Body, message.Payload["body"])
	}

	if message.Payload["notification_type"] != "success" {
		t.Errorf("Expected notification_type 'success', got '%v'", message.Payload["notification_type"])
	}

	if message.Payload["severity"] != "info" {
		t.Errorf("Expected severity 'info', got '%v'", message.Payload["severity"])
	}

	if priority, ok := message.Payload["priority"].(int); !ok || priority != 3 {
		t.Errorf("Expected priority 3, got %v", message.Payload["priority"])
	}

	if message.Payload["body_format"] != "json" {
		t.Errorf("Expected body_format 'json', got '%v'", message.Payload["body_format"])
	}

	if message.Payload["url"] != req.URL {
		t.Errorf("Expected url '%s', got '%v'", req.URL, message.Payload["url"])
	}

	if message.Payload["source"] != "apprise-go" {
		t.Errorf("Expected source 'apprise-go', got '%v'", message.Payload["source"])
	}

	// Check tags
	tags, ok := message.Payload["tags"].([]string)
	if !ok {
		t.Error("Expected tags to be string array")
	} else {
		expectedTags := []string{"config", "temperature", "threshold"}
		if len(tags) != len(expectedTags) {
			t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
		}
		for i, expectedTag := range expectedTags {
			if i < len(tags) && tags[i] != expectedTag {
				t.Errorf("Expected tag[%d] '%s', got '%s'", i, expectedTag, tags[i])
			}
		}
	}

	// Check attachments
	if attachmentCount, ok := message.Payload["attachment_count"]; !ok || attachmentCount != 1 {
		t.Errorf("Expected attachment_count 1, got %v", attachmentCount)
	}

	attachments, ok := message.Payload["attachments"].([]map[string]interface{})
	if !ok {
		t.Error("Expected attachments to be array of maps")
	} else {
		if len(attachments) != 1 {
			t.Errorf("Expected 1 attachment, got %d", len(attachments))
		} else {
			attachment := attachments[0]
			if attachment["name"] != "config.json" {
				t.Errorf("Expected attachment name 'config.json', got '%v'", attachment["name"])
			}
			if attachment["mime_type"] != "application/json" {
				t.Errorf("Expected MIME type 'application/json', got '%v'", attachment["mime_type"])
			}
		}
	}
}

func TestGCPIoTService_HelperMethods(t *testing.T) {
	service := &GCPIoTService{}

	// Test severity mapping
	tests := []struct {
		notifyType       NotifyType
		expectedSeverity string
		expectedPriority int
	}{
		{NotifyTypeError, "critical", 1},
		{NotifyTypeWarning, "warning", 2},
		{NotifyTypeSuccess, "info", 3},
		{NotifyTypeInfo, "info", 3},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if severity := service.getSeverityForNotifyType(tt.notifyType); severity != tt.expectedSeverity {
				t.Errorf("Expected severity '%s', got '%s'", tt.expectedSeverity, severity)
			}

			if priority := service.getPriorityForNotifyType(tt.notifyType); priority != tt.expectedPriority {
				t.Errorf("Expected priority %d, got %d", tt.expectedPriority, priority)
			}
		})
	}

	// Test region validation
	validRegions := []string{"us-central1", "us-east1", "europe-west1", "asia-east1"}
	for _, region := range validRegions {
		if !service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be valid", region)
		}
	}

	invalidRegions := []string{"invalid", "us-invalid-1", "europe-invalid", ""}
	for _, region := range invalidRegions {
		if service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be invalid", region)
		}
	}

	// Test message type validation
	validTypes := []string{"config", "state", "event"}
	for _, msgType := range validTypes {
		if !service.isValidMessageType(msgType) {
			t.Errorf("Expected message type '%s' to be valid", msgType)
		}
	}

	invalidTypes := []string{"invalid", "telemetry", "command", ""}
	for _, msgType := range invalidTypes {
		if service.isValidMessageType(msgType) {
			t.Errorf("Expected message type '%s' to be invalid", msgType)
		}
	}
}

func TestGCPIoTService_SendToGCPIoTDirectly(t *testing.T) {
	service := &GCPIoTService{
		projectID:  "test-project",
		region:     "us-central1",
		registryID: "test-registry",
	}

	tests := []struct {
		name           string
		messageType    string
		deviceID       string
		expectedError  string
	}{
		{
			name:          "Config message without device ID",
			messageType:   "config",
			deviceID:      "",
			expectedError: "device_id is required for config messages",
		},
		{
			name:          "State message",
			messageType:   "state",
			deviceID:      "device-123",
			expectedError: "state messages are read-only and cannot be sent to devices",
		},
		{
			name:          "Event message",
			messageType:   "event",
			deviceID:      "device-123",
			expectedError: "event messages require device authentication - please use webhook proxy mode for device events",
		},
		{
			name:          "Config message with GCP auth required",
			messageType:   "config",
			deviceID:      "device-123",
			expectedError: "direct GCP IoT API access requires Google Cloud authentication - please use webhook proxy mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := GCPIoTMessage{
				ProjectID:   service.projectID,
				Region:      service.region,
				RegistryID:  service.registryID,
				DeviceID:    tt.deviceID,
				MessageType: tt.messageType,
				Payload:     map[string]interface{}{"test": "data"},
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := service.sendToGCPIoTDirectly(ctx, message)
			if err == nil {
				t.Error("Expected error for direct API call")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error message to contain '%s', got: %v", tt.expectedError, err)
			}
		})
	}
}