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

func TestAWSIoTService_GetServiceID(t *testing.T) {
	service := NewAWSIoTService()
	if service.GetServiceID() != "aws-iot" {
		t.Errorf("Expected service ID 'aws-iot', got '%s'", service.GetServiceID())
	}
}

func TestAWSIoTService_GetDefaultPort(t *testing.T) {
	service := NewAWSIoTService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAWSIoTService_SupportsAttachments(t *testing.T) {
	service := NewAWSIoTService()
	if !service.SupportsAttachments() {
		t.Error("AWS IoT should support attachments (metadata)")
	}
}

func TestAWSIoTService_GetMaxBodyLength(t *testing.T) {
	service := NewAWSIoTService()
	expected := 131072
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestAWSIoTService_ParseURL(t *testing.T) {
	tests := []struct {
		name                    string
		url                     string
		expectError             bool
		expectedAccessKeyID     string
		expectedSecretAccessKey string
		expectedRegion          string
		expectedEndpoint        string
		expectedTopicName       string
		expectedQoS             int
		expectedDeviceType      string
		expectedWebhook         string
		expectedProxyKey        string
	}{
		{
			name:                    "Basic AWS IoT URL",
			url:                     "aws-iot://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY@a1b2c3d4e5f6g7-ats.iot.us-east-1.amazonaws.com/device/notifications",
			expectError:             false,
			expectedAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			expectedSecretAccessKey: "wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY",
			expectedRegion:          "us-east-1",
			expectedEndpoint:        "a1b2c3d4e5f6g7-ats.iot.us-east-1.amazonaws.com",
			expectedTopicName:       "device/notifications",
			expectedQoS:             1, // default
		},
		{
			name:                    "With QoS and device type parameters",
			url:                     "aws-iot://AKIATEST:secret@endpoint.iot.eu-west-1.amazonaws.com/sensors/alerts?qos=2&device_type=temperature",
			expectError:             false,
			expectedAccessKeyID:     "AKIATEST",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "eu-west-1",
			expectedEndpoint:        "endpoint.iot.eu-west-1.amazonaws.com",
			expectedTopicName:       "sensors/alerts",
			expectedQoS:             2,
			expectedDeviceType:      "temperature",
		},
		{
			name:                    "Webhook proxy mode",
			url:                     "aws-iot://proxy-key@webhook.example.com/aws-iot?access_key=AKIATEST&secret_key=secret&region=us-west-2&endpoint=test.iot.us-west-2.amazonaws.com&topic=notifications",
			expectError:             false,
			expectedWebhook:         "https://webhook.example.com/aws-iot",
			expectedProxyKey:        "proxy-key",
			expectedAccessKeyID:     "AKIATEST",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "us-west-2",
			expectedEndpoint:        "test.iot.us-west-2.amazonaws.com",
			expectedTopicName:       "notifications",
			expectedQoS:             1, // default
		},
		{
			name:                    "Webhook with full parameters",
			url:                     "aws-iot://proxy@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=ap-southeast-1&endpoint=iot.example.com&topic=device/status&qos=0&device_type=sensor",
			expectError:             false,
			expectedWebhook:         "https://webhook.example.com/aws-iot",
			expectedProxyKey:        "proxy",
			expectedAccessKeyID:     "key",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "ap-southeast-1",
			expectedEndpoint:        "iot.example.com",
			expectedTopicName:       "device/status",
			expectedQoS:             0,
			expectedDeviceType:      "sensor",
		},
		{
			name:        "Invalid scheme",
			url:         "http://key:secret@endpoint.iot.us-east-1.amazonaws.com/topic",
			expectError: true,
		},
		{
			name:        "Missing access key",
			url:         "aws-iot://:secret@endpoint.iot.us-east-1.amazonaws.com/topic",
			expectError: true,
		},
		{
			name:        "Missing secret key",
			url:         "aws-iot://key@endpoint.iot.us-east-1.amazonaws.com/topic",
			expectError: true,
		},
		{
			name:        "Missing endpoint",
			url:         "aws-iot://key:secret@/topic",
			expectError: true,
		},
		{
			name:        "Missing topic",
			url:         "aws-iot://key:secret@endpoint.iot.us-east-1.amazonaws.com/",
			expectError: true,
		},
		{
			name:        "Invalid QoS",
			url:         "aws-iot://key:secret@endpoint.iot.us-east-1.amazonaws.com/topic?qos=3",
			expectError: true,
		},
		{
			name:        "Webhook missing access key",
			url:         "aws-iot://proxy@webhook.example.com/aws-iot?secret_key=secret&region=us-east-1&endpoint=endpoint&topic=topic",
			expectError: true,
		},
		{
			name:        "Webhook missing secret key",
			url:         "aws-iot://proxy@webhook.example.com/aws-iot?access_key=key&region=us-east-1&endpoint=endpoint&topic=topic",
			expectError: true,
		},
		{
			name:        "Webhook missing endpoint",
			url:         "aws-iot://proxy@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=us-east-1&topic=topic",
			expectError: true,
		},
		{
			name:        "Webhook missing topic",
			url:         "aws-iot://proxy@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=us-east-1&endpoint=endpoint",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAWSIoTService().(*AWSIoTService)
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

			if service.accessKeyID != tt.expectedAccessKeyID {
				t.Errorf("Expected access key ID '%s', got '%s'", tt.expectedAccessKeyID, service.accessKeyID)
			}

			if service.secretAccessKey != tt.expectedSecretAccessKey {
				t.Errorf("Expected secret access key '%s', got '%s'", tt.expectedSecretAccessKey, service.secretAccessKey)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
			}

			if service.endpoint != tt.expectedEndpoint {
				t.Errorf("Expected endpoint '%s', got '%s'", tt.expectedEndpoint, service.endpoint)
			}

			if service.topicName != tt.expectedTopicName {
				t.Errorf("Expected topic name '%s', got '%s'", tt.expectedTopicName, service.topicName)
			}

			if service.qos != tt.expectedQoS {
				t.Errorf("Expected QoS %d, got %d", tt.expectedQoS, service.qos)
			}

			if tt.expectedDeviceType != "" && service.deviceType != tt.expectedDeviceType {
				t.Errorf("Expected device type '%s', got '%s'", tt.expectedDeviceType, service.deviceType)
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

func TestAWSIoTService_TestURL(t *testing.T) {
	service := NewAWSIoTService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid AWS IoT URL",
			url:         "aws-iot://key:secret@endpoint.iot.us-east-1.amazonaws.com/device/notifications",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "aws-iot://proxy@webhook.example.com/aws-iot?access_key=key&secret_key=secret&region=us-east-1&endpoint=endpoint&topic=notifications",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://key:secret@endpoint.iot.us-east-1.amazonaws.com/topic",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "aws-iot://endpoint.iot.us-east-1.amazonaws.com/topic",
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

func TestAWSIoTService_SendWebhook(t *testing.T) {
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
		var payload AWSIoTWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "aws-iot" {
			t.Errorf("Expected service 'aws-iot', got '%s'", payload.Service)
		}

		if payload.Region != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got '%s'", payload.Region)
		}

		if payload.Endpoint != "test.iot.us-east-1.amazonaws.com" {
			t.Errorf("Expected endpoint 'test.iot.us-east-1.amazonaws.com', got '%s'", payload.Endpoint)
		}

		if payload.AccessKeyID != "AKIATEST" {
			t.Errorf("Expected access key 'AKIATEST', got '%s'", payload.AccessKeyID)
		}

		if payload.SecretAccessKey != "secret" {
			t.Errorf("Expected secret key 'secret', got '%s'", payload.SecretAccessKey)
		}

		if payload.Message.Topic != "device/alerts" {
			t.Errorf("Expected topic 'device/alerts', got '%s'", payload.Message.Topic)
		}

		if payload.Message.QoS != 1 {
			t.Errorf("Expected QoS 1, got %d", payload.Message.QoS)
		}

		// Verify message payload
		if payload.Message.Payload["title"] != "IoT Device Alert" {
			t.Errorf("Expected title 'IoT Device Alert', got '%v'", payload.Message.Payload["title"])
		}

		if payload.Message.Payload["body"] != "Sensor reading exceeded threshold" {
			t.Errorf("Expected body to match, got '%v'", payload.Message.Payload["body"])
		}

		if payload.Message.Payload["notification_type"] != "error" {
			t.Errorf("Expected notification_type 'error', got '%v'", payload.Message.Payload["notification_type"])
		}

		if payload.Message.Payload["severity"] != "critical" {
			t.Errorf("Expected severity 'critical', got '%v'", payload.Message.Payload["severity"])
		}

		if priority, ok := payload.Message.Payload["priority"].(float64); !ok || int(priority) != 1 {
			t.Errorf("Expected priority 1, got %v", payload.Message.Payload["priority"])
		}

		if payload.Message.Payload["source"] != "apprise-go" {
			t.Errorf("Expected source 'apprise-go', got '%v'", payload.Message.Payload["source"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true, "message_id": "abc123"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewAWSIoTService().(*AWSIoTService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.accessKeyID = "AKIATEST"
	service.secretAccessKey = "secret"
	service.region = "us-east-1"
	service.endpoint = "test.iot.us-east-1.amazonaws.com"
	service.topicName = "device/alerts"
	service.qos = 1

	req := NotificationRequest{
		Title:      "IoT Device Alert",
		Body:       "Sensor reading exceeded threshold",
		NotifyType: NotifyTypeError,
		Tags:       []string{"sensor", "critical"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestAWSIoTService_BuildIoTMessage(t *testing.T) {
	service := &AWSIoTService{
		topicName:  "device/notifications",
		qos:        2,
		deviceType: "temperature-sensor",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("sensor data"), "reading.json", "application/json")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Temperature Alert",
		Body:          "Temperature exceeded safe limits",
		NotifyType:    NotifyTypeWarning,
		Tags:          []string{"temperature", "safety"},
		BodyFormat:    "text",
		URL:           "https://dashboard.example.com/sensors/temp-01",
		AttachmentMgr: attachmentMgr,
	}

	message := service.buildIoTMessage(req)

	// Check basic fields
	if message.Topic != "device/notifications" {
		t.Errorf("Expected topic 'device/notifications', got '%s'", message.Topic)
	}

	if message.QoS != 2 {
		t.Errorf("Expected QoS 2, got %d", message.QoS)
	}

	// Check payload fields
	if message.Payload["title"] != req.Title {
		t.Errorf("Expected title '%s', got '%v'", req.Title, message.Payload["title"])
	}

	if message.Payload["body"] != req.Body {
		t.Errorf("Expected body '%s', got '%v'", req.Body, message.Payload["body"])
	}

	if message.Payload["notification_type"] != "warning" {
		t.Errorf("Expected notification_type 'warning', got '%v'", message.Payload["notification_type"])
	}

	if message.Payload["severity"] != "warning" {
		t.Errorf("Expected severity 'warning', got '%v'", message.Payload["severity"])
	}

	if priority, ok := message.Payload["priority"].(int); !ok || priority != 2 {
		t.Errorf("Expected priority 2, got %v", message.Payload["priority"])
	}

	if message.Payload["device_type"] != "temperature-sensor" {
		t.Errorf("Expected device_type 'temperature-sensor', got '%v'", message.Payload["device_type"])
	}

	if message.Payload["body_format"] != "text" {
		t.Errorf("Expected body_format 'text', got '%v'", message.Payload["body_format"])
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
		expectedTags := []string{"temperature", "safety"}
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
			if attachment["name"] != "reading.json" {
				t.Errorf("Expected attachment name 'reading.json', got '%v'", attachment["name"])
			}
			if attachment["mime_type"] != "application/json" {
				t.Errorf("Expected MIME type 'application/json', got '%v'", attachment["mime_type"])
			}
		}
	}
}

func TestAWSIoTService_HelperMethods(t *testing.T) {
	service := &AWSIoTService{}

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
	validRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	for _, region := range validRegions {
		if !service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be valid", region)
		}
	}

	invalidRegions := []string{"invalid", "us-invalid-1", "eu-invalid", ""}
	for _, region := range invalidRegions {
		if service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be invalid", region)
		}
	}

	// Test topic validation
	validTopics := []string{"device/notifications", "sensors/data", "alerts"}
	for _, topic := range validTopics {
		if !service.isValidTopic(topic) {
			t.Errorf("Expected topic '%s' to be valid", topic)
		}
	}

	invalidTopics := []string{"", "/invalid", "invalid/", "device/#", "sensors/+", "$aws/thing"}
	for _, topic := range invalidTopics {
		if service.isValidTopic(topic) {
			t.Errorf("Expected topic '%s' to be invalid", topic)
		}
	}
}

func TestAWSIoTService_SendToIoTDirectly(t *testing.T) {
	service := &AWSIoTService{
		endpoint: "test.iot.us-east-1.amazonaws.com",
	}

	message := AWSIoTMessage{
		Topic:     "test/topic",
		Payload:   map[string]interface{}{"test": "data"},
		QoS:       1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should return an error indicating AWS Signature V4 is required
	err := service.sendToIoTDirectly(ctx, message)
	if err == nil {
		t.Error("Expected error for direct API call without AWS Signature V4")
	}

	if !strings.Contains(err.Error(), "AWS Signature V4") {
		t.Errorf("Expected error message about AWS Signature V4, got: %v", err)
	}
}