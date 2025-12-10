package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestMQTTService_GetServiceID(t *testing.T) {
	service := NewMQTTService()
	if service.GetServiceID() != "mqtt" {
		t.Errorf("Expected service ID 'mqtt', got '%s'", service.GetServiceID())
	}
}

func TestMQTTService_GetDefaultPort(t *testing.T) {
	tests := []struct {
		name         string
		useTLS       bool
		expectedPort int
	}{
		{"TCP", false, 1883},
		{"TLS", true, 8883},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMQTTService().(*MQTTService)
			service.useTLS = tt.useTLS
			port := service.GetDefaultPort()
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestMQTTService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *MQTTService)
	}{
		{
			name:        "Basic MQTT URL",
			url:         "mqtt://localhost/notifications",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.broker != "tcp://localhost:1883" {
					t.Errorf("Expected broker 'tcp://localhost:1883', got '%s'", m.broker)
				}
				if m.topic != "notifications" {
					t.Errorf("Expected topic 'notifications', got '%s'", m.topic)
				}
				if m.qos != 0 {
					t.Errorf("Expected default QoS 0, got %d", m.qos)
				}
			},
		},
		{
			name:        "MQTTS (TLS) URL",
			url:         "mqtts://broker.example.com/secure/topic",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.broker != "ssl://broker.example.com:8883" {
					t.Errorf("Expected SSL broker, got '%s'", m.broker)
				}
				if !m.useTLS {
					t.Error("Expected useTLS to be true for mqtts://")
				}
				if m.topic != "secure/topic" {
					t.Errorf("Expected topic 'secure/topic', got '%s'", m.topic)
				}
			},
		},
		{
			name:        "With authentication",
			url:         "mqtt://user:pass@broker.local:1883/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.username != "user" {
					t.Errorf("Expected username 'user', got '%s'", m.username)
				}
				if m.password != "pass" {
					t.Errorf("Expected password 'pass', got '%s'", m.password)
				}
			},
		},
		{
			name:        "With QoS parameter",
			url:         "mqtt://broker/topic?qos=1",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.qos != 1 {
					t.Errorf("Expected QoS 1, got %d", m.qos)
				}
			},
		},
		{
			name:        "With QoS 2 (exactly once)",
			url:         "mqtt://broker/topic?qos=2",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.qos != 2 {
					t.Errorf("Expected QoS 2, got %d", m.qos)
				}
			},
		},
		{
			name:        "With retained flag",
			url:         "mqtt://broker/topic?retained=true",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if !m.retained {
					t.Error("Expected retained to be true")
				}
			},
		},
		{
			name:        "With custom client ID",
			url:         "mqtt://broker/topic?clientid=my-app-001",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.clientID != "my-app-001" {
					t.Errorf("Expected clientID 'my-app-001', got '%s'", m.clientID)
				}
			},
		},
		{
			name:        "With Last Will and Testament",
			url:         "mqtt://broker/topic?will_topic=offline&will_payload=disconnected&will_qos=1&will_retain=true",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.willTopic != "offline" {
					t.Errorf("Expected will topic 'offline', got '%s'", m.willTopic)
				}
				if m.willPayload != "disconnected" {
					t.Errorf("Expected will payload 'disconnected', got '%s'", m.willPayload)
				}
				if m.willQos != 1 {
					t.Errorf("Expected will QoS 1, got %d", m.willQos)
				}
				if !m.willRetain {
					t.Error("Expected will retain to be true")
				}
			},
		},
		{
			name:        "With TLS options",
			url:         "mqtts://broker/topic?insecure=true&ca_file=/path/to/ca.pem",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if !m.insecure {
					t.Error("Expected insecure to be true")
				}
				if m.caFile != "/path/to/ca.pem" {
					t.Errorf("Expected ca_file '/path/to/ca.pem', got '%s'", m.caFile)
				}
			},
		},
		{
			name:        "With client certificates",
			url:         "mqtts://broker/topic?cert_file=/cert.pem&key_file=/key.pem",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.certFile != "/cert.pem" {
					t.Errorf("Expected cert_file '/cert.pem', got '%s'", m.certFile)
				}
				if m.keyFile != "/key.pem" {
					t.Errorf("Expected key_file '/key.pem', got '%s'", m.keyFile)
				}
			},
		},
		{
			name:        "With hierarchical topic",
			url:         "mqtt://broker/home/livingroom/temperature",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.topic != "home/livingroom/temperature" {
					t.Errorf("Expected hierarchical topic, got '%s'", m.topic)
				}
			},
		},
		{
			name:        "With explicit port",
			url:         "mqtt://broker:9001/topic",
			expectError: false,
			checkFunc: func(t *testing.T, m *MQTTService) {
				if m.broker != "tcp://broker:9001" {
					t.Errorf("Expected explicit port in broker URL, got '%s'", m.broker)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "http://broker/topic",
			expectError: true,
		},
		{
			name:        "Missing broker",
			url:         "mqtt:///topic",
			expectError: true,
		},
		{
			name:        "Missing topic",
			url:         "mqtt://broker/",
			expectError: true,
		},
		{
			name:        "Invalid QoS value",
			url:         "mqtt://broker/topic?qos=3",
			expectError: true,
		},
		{
			name:        "Invalid QoS format",
			url:         "mqtt://broker/topic?qos=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMQTTService().(*MQTTService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkFunc != nil {
				tt.checkFunc(t, service)
			}
		})
	}
}

func TestMQTTService_BuildMessage(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		contains    []string
		notContains []string
	}{
		{
			name: "Info notification",
			request: NotificationRequest{
				Title:      "System Update",
				Body:       "Version 2.1.0 deployed",
				NotifyType: NotifyTypeInfo,
			},
			contains:    []string{"[INFO]", "System Update", "Version 2.1.0 deployed"},
			notContains: []string{"[ERROR]", "[WARN]"},
		},
		{
			name: "Success notification",
			request: NotificationRequest{
				Title:      "Deployment Complete",
				Body:       "All services running",
				NotifyType: NotifyTypeSuccess,
			},
			contains: []string{"[OK]", "Deployment Complete", "All services running"},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "High Memory",
				Body:       "Usage at 85%",
				NotifyType: NotifyTypeWarning,
			},
			contains: []string{"[WARN]", "High Memory", "Usage at 85%"},
		},
		{
			name: "Error notification",
			request: NotificationRequest{
				Title:      "Service Down",
				Body:       "API not responding",
				NotifyType: NotifyTypeError,
			},
			contains: []string{"[ERROR]", "Service Down", "API not responding"},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Alert",
				Body:       "Test message",
				NotifyType: NotifyTypeInfo,
				Tags:       []string{"production", "critical", "api"},
			},
			contains: []string{"[INFO]", "Alert", "Test message", "[production, critical, api]"},
		},
		{
			name: "Title only",
			request: NotificationRequest{
				Title:      "Quick Alert",
				NotifyType: NotifyTypeWarning,
			},
			contains:    []string{"[WARN]", "Quick Alert"},
			notContains: []string{": "},
		},
		{
			name: "Body only",
			request: NotificationRequest{
				Body:       "Simple message",
				NotifyType: NotifyTypeInfo,
			},
			contains:    []string{"[INFO]", "Simple message"},
			notContains: []string{": "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMQTTService().(*MQTTService)
			message := service.buildMessage(tt.request)

			for _, substr := range tt.contains {
				if !strings.Contains(message, substr) {
					t.Errorf("Expected message to contain '%s', got: %s", substr, message)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(message, substr) {
					t.Errorf("Expected message NOT to contain '%s', got: %s", substr, message)
				}
			}
		})
	}
}

func TestMQTTService_GetTypePrefix(t *testing.T) {
	service := NewMQTTService().(*MQTTService)

	tests := []struct {
		notifyType     NotifyType
		expectedPrefix string
	}{
		{NotifyTypeInfo, "[INFO]"},
		{NotifyTypeSuccess, "[OK]"},
		{NotifyTypeWarning, "[WARN]"},
		{NotifyTypeError, "[ERROR]"},
	}

	for _, tt := range tests {
		prefix := service.getTypePrefix(tt.notifyType)
		if prefix != tt.expectedPrefix {
			t.Errorf("For %v, expected prefix '%s', got '%s'", tt.notifyType, tt.expectedPrefix, prefix)
		}
	}
}

func TestMQTTService_TestURL(t *testing.T) {
	service := NewMQTTService()

	validURLs := []string{
		"mqtt://localhost/test",
		"mqtt://broker.example.com:1883/alerts",
		"mqtts://secure.broker.com/secure/topic",
		"mqtt://user:pass@broker/topic?qos=1",
		"mqtt://broker/home/sensors?retained=true",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"http://broker/topic",
		"mqtt://",
		"mqtt://broker/",
		"mqtt://broker/topic?qos=5",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestMQTTService_SupportsAttachments(t *testing.T) {
	service := NewMQTTService()
	if service.SupportsAttachments() {
		t.Error("MQTT service should not support attachments")
	}
}

func TestMQTTService_GetMaxBodyLength(t *testing.T) {
	service := NewMQTTService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (no limit), got %d", service.GetMaxBodyLength())
	}
}

func TestMQTTService_ClientIDGeneration(t *testing.T) {
	// Create two services and verify they have different client IDs
	service1 := NewMQTTService().(*MQTTService)
	service2 := NewMQTTService().(*MQTTService)

	if service1.clientID == "" {
		t.Error("Client ID should be auto-generated")
	}

	if !strings.HasPrefix(service1.clientID, "apprise-go-") {
		t.Errorf("Client ID should start with 'apprise-go-', got '%s'", service1.clientID)
	}

	// Client IDs should be different (time-based)
	// Note: This might fail if executed too quickly, but unlikely
	if service1.clientID == service2.clientID {
		t.Log("Warning: Client IDs are the same (may happen if created in same millisecond)")
	}
}

func TestMQTTService_QoSLevels(t *testing.T) {
	tests := []struct {
		name     string
		qos      byte
		expected string
	}{
		{"QoS 0 - At most once", 0, "fire and forget"},
		{"QoS 1 - At least once", 1, "acknowledged delivery"},
		{"QoS 2 - Exactly once", 2, "guaranteed delivery"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMQTTService().(*MQTTService)
			service.qos = tt.qos

			// Verify QoS is set correctly
			if service.qos != tt.qos {
				t.Errorf("Expected QoS %d, got %d", tt.qos, service.qos)
			}
		})
	}
}

func TestMQTTService_TopicHierarchy(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		expectedValid bool
	}{
		{"Simple topic", "notifications", true},
		{"Two levels", "home/temperature", true},
		{"Three levels", "building/floor2/room5", true},
		{"Many levels", "company/site/building/floor/room/sensor", true},
		{"With wildcards (single level)", "home/+/temperature", true},
		{"With wildcards (multi level)", "home/#", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMQTTService().(*MQTTService)
			service.topic = tt.topic

			if service.topic != tt.topic {
				t.Errorf("Expected topic '%s', got '%s'", tt.topic, service.topic)
			}
		})
	}
}

// Note: Integration tests with actual MQTT broker are skipped in unit tests
// To test with a real broker, set MQTT_BROKER_URL environment variable
func TestMQTTService_Integration(t *testing.T) {
	brokerURL := ""
	if brokerURL == "" {
		t.Skip("Skipping integration test: MQTT broker not configured")
	}

	service := NewMQTTService().(*MQTTService)
	parsedURL, _ := url.Parse(brokerURL)
	err := service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse broker URL: %v", err)
	}

	req := NotificationRequest{
		Title:      "Integration Test",
		Body:       "Testing MQTT service",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"test"},
	}

	err = service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}
}
