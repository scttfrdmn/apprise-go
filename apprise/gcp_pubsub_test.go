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

func TestGCPPubSubService_GetServiceID(t *testing.T) {
	service := NewGCPPubSubService()
	if service.GetServiceID() != "pubsub" {
		t.Errorf("Expected service ID 'pubsub', got '%s'", service.GetServiceID())
	}
}

func TestGCPPubSubService_GetDefaultPort(t *testing.T) {
	service := NewGCPPubSubService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestGCPPubSubService_SupportsAttachments(t *testing.T) {
	service := NewGCPPubSubService()
	if service.SupportsAttachments() {
		t.Error("GCP Pub/Sub should not support attachments")
	}
}

func TestGCPPubSubService_GetMaxBodyLength(t *testing.T) {
	service := NewGCPPubSubService()
	expected := 10 * 1024 * 1024 // 10MB
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestGCPPubSubService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedProject  string
		expectedTopic    string
		expectedSA       string
		expectedAPIKey   string
		expectedOrdering string
		expectedAttrs    map[string]string
	}{
		{
			name:            "Valid basic Pub/Sub URL",
			url:             "pubsub://webhook.example.com/pubsub?project_id=my-project&topic=notifications",
			expectError:     false,
			expectedProject: "my-project",
			expectedTopic:   "notifications",
		},
		{
			name:            "Valid Pub/Sub URL with service account",
			url:             "pubsub://webhook.example.com/gcp?project_id=company-project&topic=alerts&service_account=sa@project.iam.gserviceaccount.com",
			expectError:     false,
			expectedProject: "company-project",
			expectedTopic:   "alerts",
			expectedSA:      "sa@project.iam.gserviceaccount.com",
		},
		{
			name:             "Valid Pub/Sub URL with API key and ordering",
			url:              "pubsub://api-key@webhook.example.com/proxy?project_id=test-project&topic=events&ordering_key=region-us",
			expectError:      false,
			expectedAPIKey:   "api-key",
			expectedProject:  "test-project",
			expectedTopic:    "events",
			expectedOrdering: "region-us",
		},
		{
			name:            "Valid Pub/Sub URL with attributes",
			url:             "pubsub://webhook.example.com/pubsub?project_id=my-project&topic=logs&attr_environment=prod&attr_service=api&attr_team=backend",
			expectError:     false,
			expectedProject: "my-project",
			expectedTopic:   "logs",
			expectedAttrs: map[string]string{
				"environment": "prod",
				"service":     "api",
				"team":        "backend",
			},
		},
		{
			name:        "Invalid scheme",
			url:         "http://webhook.example.com/pubsub?project_id=test&topic=test",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "pubsub:///pubsub?project_id=test&topic=test",
			expectError: true,
		},
		{
			name:        "Missing project_id",
			url:         "pubsub://webhook.example.com/pubsub?topic=test",
			expectError: true,
		},
		{
			name:        "Missing topic",
			url:         "pubsub://webhook.example.com/pubsub?project_id=test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGCPPubSubService().(*GCPPubSubService)
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

			if tt.expectedProject != "" && service.projectID != tt.expectedProject {
				t.Errorf("Expected project ID '%s', got '%s'", tt.expectedProject, service.projectID)
			}

			if tt.expectedTopic != "" && service.topicName != tt.expectedTopic {
				t.Errorf("Expected topic '%s', got '%s'", tt.expectedTopic, service.topicName)
			}

			if tt.expectedSA != "" && service.serviceAccount != tt.expectedSA {
				t.Errorf("Expected service account '%s', got '%s'", tt.expectedSA, service.serviceAccount)
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}

			if tt.expectedOrdering != "" && service.orderingKey != tt.expectedOrdering {
				t.Errorf("Expected ordering key '%s', got '%s'", tt.expectedOrdering, service.orderingKey)
			}

			if len(tt.expectedAttrs) > 0 {
				for key, expectedValue := range tt.expectedAttrs {
					if actualValue, exists := service.attributes[key]; !exists {
						t.Errorf("Expected attribute '%s' not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected attribute '%s'='%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestGCPPubSubService_TestURL(t *testing.T) {
	service := NewGCPPubSubService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Pub/Sub URL",
			url:         "pubsub://webhook.example.com/pubsub?project_id=test&topic=notifications",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://webhook.example.com/pubsub",
			expectError: true,
		},
		{
			name:        "Missing required parameters",
			url:         "pubsub://webhook.example.com/pubsub",
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

func TestGCPPubSubService_Send(t *testing.T) {
	// Create mock server
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

		// Check Google Cloud-specific headers
		if r.Header.Get("X-Google-Cloud-Service") != "PubSub" {
			t.Errorf("Expected X-Google-Cloud-Service header to be PubSub, got %s", r.Header.Get("X-Google-Cloud-Service"))
		}

		// Parse and verify request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload["projectId"] == "" {
			t.Error("Expected projectId in payload")
		}

		if payload["topicName"] == "" {
			t.Error("Expected topicName in payload")
		}

		if payload["message"] == nil {
			t.Error("Expected message in payload")
		}

		if payload["attributes"] == nil {
			t.Error("Expected attributes in payload")
		}

		// Verify message structure
		message, ok := payload["message"].(map[string]interface{})
		if !ok {
			t.Error("Expected message to be an object")
		} else {
			if message["data"] == "" {
				t.Error("Expected data in message")
			}
			if message["messageId"] == "" {
				t.Error("Expected messageId in message")
			}
		}

		// Verify attributes structure
		attributes, ok := payload["attributes"].(map[string]interface{})
		if !ok {
			t.Error("Expected attributes to be an object")
		} else {
			if attributes["notificationType"] == "" {
				t.Error("Expected notificationType in attributes")
			}
			if attributes["source"] != "apprise-go" {
				t.Error("Expected source to be apprise-go in attributes")
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messageIds":["12345-67890-abcdef"]}`))
	}))
	defer server.Close()

	// Parse server URL and create Pub/Sub URL with test_mode=true for HTTP
	serverURL, _ := url.Parse(server.URL)
	pubsubURL := "pubsub://" + serverURL.Host + "/webhook?project_id=test-project&topic=notifications&test_mode=true"

	service := NewGCPPubSubService().(*GCPPubSubService)
	parsedURL, _ := url.Parse(pubsubURL)
	_ = service.ParseURL(parsedURL)

	// Test different notification types
	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
	}{
		{
			name:       "Info notification",
			title:      "Test Info",
			body:       "This is an info notification",
			notifyType: NotifyTypeInfo,
		},
		{
			name:       "Success notification",
			title:      "Test Success",
			body:       "This is a success notification",
			notifyType: NotifyTypeSuccess,
		},
		{
			name:       "Warning notification",
			title:      "Test Warning",
			body:       "This is a warning notification",
			notifyType: NotifyTypeWarning,
		},
		{
			name:       "Error notification",
			title:      "Test Error",
			body:       "This is an error notification",
			notifyType: NotifyTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{
				Title:      tt.title,
				Body:       tt.body,
				NotifyType: tt.notifyType,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := service.Send(ctx, req)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}
		})
	}
}

func TestGCPPubSubService_SendWithAPIKey(t *testing.T) {
	// Create mock server that checks for API key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		authHeader := r.Header.Get("Authorization")

		if apiKey != "test-api-key" {
			t.Errorf("Expected X-API-Key 'test-api-key', got '%s'", apiKey)
		}

		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization 'Bearer test-api-key', got '%s'", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"messageIds":["test-message-id"]}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	pubsubURL := "pubsub://test-api-key@" + serverURL.Host + "/webhook?project_id=test-project&topic=notifications&test_mode=true"

	service := NewGCPPubSubService().(*GCPPubSubService)
	parsedURL, _ := url.Parse(pubsubURL)
	_ = service.ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "API Key Test",
		Body:       "Testing API key authentication",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send with API key failed: %v", err)
	}
}

func TestGCPPubSubService_FormatMessage(t *testing.T) {
	service := &GCPPubSubService{
		projectID: "test-project",
		topicName: "test-topic",
	}

	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
		checkJSON  bool
	}{
		{
			name:       "Info message",
			title:      "Test Title",
			body:       "Test Body",
			notifyType: NotifyTypeInfo,
			checkJSON:  true,
		},
		{
			name:       "Error message",
			title:      "Critical Error",
			body:       "System failure detected",
			notifyType: NotifyTypeError,
			checkJSON:  true,
		},
		{
			name:       "Warning message",
			title:      "Performance Warning",
			body:       "High CPU usage detected",
			notifyType: NotifyTypeWarning,
			checkJSON:  true,
		},
		{
			name:       "Success message",
			title:      "Deployment Complete",
			body:       "Version 1.2.3 deployed successfully",
			notifyType: NotifyTypeSuccess,
			checkJSON:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.formatMessage(tt.title, tt.body, tt.notifyType)

			if tt.checkJSON {
				// Should be valid JSON
				var messageData map[string]interface{}
				err := json.Unmarshal([]byte(message), &messageData)
				if err != nil {
					t.Fatalf("Failed to parse JSON message: %v", err)
				}

				// Check required fields
				if messageData["title"] != tt.title {
					t.Errorf("Expected title '%s', got '%s'", tt.title, messageData["title"])
				}

				if messageData["body"] != tt.body {
					t.Errorf("Expected body '%s', got '%s'", tt.body, messageData["body"])
				}

				if messageData["type"] != tt.notifyType.String() {
					t.Errorf("Expected type '%s', got '%s'", tt.notifyType.String(), messageData["type"])
				}

				// Check that timestamp and source are present
				if messageData["timestamp"] == "" {
					t.Error("Expected timestamp to be present")
				}

				if messageData["source"] != "apprise-go" {
					t.Error("Expected source to be apprise-go")
				}

				// Check severity mapping
				expectedSeverity := service.getSeverityLevel(tt.notifyType)
				if messageData["severity"] != expectedSeverity {
					t.Errorf("Expected severity '%s', got '%s'", expectedSeverity, messageData["severity"])
				}

				// Check environment context
				environment, ok := messageData["environment"].(map[string]interface{})
				if !ok {
					t.Error("Expected environment to be an object")
				} else {
					if environment["project"] != service.projectID {
						t.Errorf("Expected environment project '%s', got '%s'", service.projectID, environment["project"])
					}
					if environment["topic"] != service.topicName {
						t.Errorf("Expected environment topic '%s', got '%s'", service.topicName, environment["topic"])
					}
				}
			}
		})
	}
}

func TestGCPPubSubService_BuildAttributes(t *testing.T) {
	service := &GCPPubSubService{
		projectID: "test-project",
		topicName: "test-topic",
		attributes: map[string]string{
			"environment": "production",
			"service":     "web-api",
		},
	}

	attributes := service.buildAttributes(NotifyTypeError)

	// Check custom attributes
	if attributes["environment"] != "production" {
		t.Errorf("Expected environment 'production', got '%s'", attributes["environment"])
	}

	if attributes["service"] != "web-api" {
		t.Errorf("Expected service 'web-api', got '%s'", attributes["service"])
	}

	// Check standard attributes
	if attributes["notificationType"] != "error" {
		t.Errorf("Expected notificationType 'error', got '%s'", attributes["notificationType"])
	}

	if attributes["source"] != "apprise-go" {
		t.Errorf("Expected source 'apprise-go', got '%s'", attributes["source"])
	}

	if attributes["severity"] != "ERROR" {
		t.Errorf("Expected severity 'ERROR' for error type, got '%s'", attributes["severity"])
	}

	if attributes["priority"] != "HIGH" {
		t.Errorf("Expected priority 'HIGH' for error type, got '%s'", attributes["priority"])
	}

	if attributes["alertLevel"] != "CRITICAL" {
		t.Errorf("Expected alertLevel 'CRITICAL' for error type, got '%s'", attributes["alertLevel"])
	}

	// Check routing attributes
	if attributes["topic"] != service.topicName {
		t.Errorf("Expected topic '%s', got '%s'", service.topicName, attributes["topic"])
	}

	if attributes["project"] != service.projectID {
		t.Errorf("Expected project '%s', got '%s'", service.projectID, attributes["project"])
	}
}

func TestGCPPubSubService_BuildMessage(t *testing.T) {
	service := &GCPPubSubService{}

	req := NotificationRequest{
		Title:      "Test Message",
		Body:       "Test body",
		NotifyType: NotifyTypeWarning,
	}

	message := service.buildMessage("test-data", req)

	// Check required fields
	if message["data"] != "test-data" {
		t.Errorf("Expected data 'test-data', got '%s'", message["data"])
	}

	if message["messageId"] == "" {
		t.Error("Expected messageId to be present")
	}

	if message["publishTime"] == "" {
		t.Error("Expected publishTime to be present")
	}

	// Verify messageId format
	messageID, ok := message["messageId"].(string)
	if !ok {
		t.Error("Expected messageId to be a string")
	} else {
		if !strings.HasPrefix(messageID, "apprise-") {
			t.Error("Expected messageId to start with 'apprise-'")
		}
		if !strings.HasSuffix(messageID, "-warning") {
			t.Error("Expected messageId to end with notification type")
		}
	}
}

func TestGCPPubSubService_GetSeverityLevel(t *testing.T) {
	service := &GCPPubSubService{}

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeError, "ERROR"},
		{NotifyTypeWarning, "WARNING"},
		{NotifyTypeSuccess, "INFO"},
		{NotifyTypeInfo, "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			result := service.getSeverityLevel(tt.notifyType)
			if result != tt.expected {
				t.Errorf("Expected severity '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
