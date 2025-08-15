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

func TestAzureServiceBusService_GetServiceID(t *testing.T) {
	service := NewAzureServiceBusService()
	if service.GetServiceID() != "azuresb" {
		t.Errorf("Expected service ID 'azuresb', got '%s'", service.GetServiceID())
	}
}

func TestAzureServiceBusService_GetDefaultPort(t *testing.T) {
	service := NewAzureServiceBusService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAzureServiceBusService_SupportsAttachments(t *testing.T) {
	service := NewAzureServiceBusService()
	if service.SupportsAttachments() {
		t.Error("Azure Service Bus should not support attachments")
	}
}

func TestAzureServiceBusService_GetMaxBodyLength(t *testing.T) {
	service := NewAzureServiceBusService()
	expected := 256 * 1024 // 256KB
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestAzureServiceBusService_ParseURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedNamespace string
		expectedQueue     string
		expectedTopic     string
		expectedSASKey    string
		expectedAPIKey    string
	}{
		{
			name:              "Valid queue URL with SAS",
			url:               "azuresb://webhook.example.com/sb?namespace=mybus&queue=notifications&sas_key_name=SendKey&sas_key=abc123",
			expectError:       false,
			expectedNamespace: "mybus",
			expectedQueue:     "notifications",
			expectedSASKey:    "abc123",
		},
		{
			name:              "Valid topic URL with API key",
			url:               "azuresb://api-key@webhook.example.com/servicebus?namespace=company-bus&topic=alerts&subscription=email-processor",
			expectError:       false,
			expectedAPIKey:    "api-key",
			expectedNamespace: "company-bus",
			expectedTopic:     "alerts",
		},
		{
			name:              "Valid connection string URL",
			url:               "azuresb://webhook.example.com/proxy?connection_string=Endpoint%3Dsb%3A//mybus.servicebus.windows.net/&queue=messages",
			expectError:       false,
			expectedNamespace: "mybus",
			expectedQueue:     "messages",
		},
		{
			name:        "Invalid scheme",
			url:         "http://webhook.example.com/sb?namespace=test&queue=test",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "azuresb:///sb?namespace=test&queue=test",
			expectError: true,
		},
		{
			name:        "Missing namespace and connection_string",
			url:         "azuresb://webhook.example.com/sb?queue=test",
			expectError: true,
		},
		{
			name:        "Missing queue and topic",
			url:         "azuresb://webhook.example.com/sb?namespace=test",
			expectError: true,
		},
		{
			name:        "Missing SAS key and API key",
			url:         "azuresb://webhook.example.com/sb?namespace=test&queue=test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAzureServiceBusService().(*AzureServiceBusService)
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

			if tt.expectedNamespace != "" && service.namespace != tt.expectedNamespace {
				t.Errorf("Expected namespace '%s', got '%s'", tt.expectedNamespace, service.namespace)
			}

			if tt.expectedQueue != "" && service.queueName != tt.expectedQueue {
				t.Errorf("Expected queue '%s', got '%s'", tt.expectedQueue, service.queueName)
			}

			if tt.expectedTopic != "" && service.topicName != tt.expectedTopic {
				t.Errorf("Expected topic '%s', got '%s'", tt.expectedTopic, service.topicName)
			}

			if tt.expectedSASKey != "" && service.sasKey != tt.expectedSASKey {
				t.Errorf("Expected SAS key '%s', got '%s'", tt.expectedSASKey, service.sasKey)
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}
		})
	}
}

func TestAzureServiceBusService_TestURL(t *testing.T) {
	service := NewAzureServiceBusService()
	
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Service Bus URL",
			url:         "azuresb://webhook.example.com/sb?namespace=test&queue=notifications&sas_key=abc123",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://webhook.example.com/sb",
			expectError: true,
		},
		{
			name:        "Missing required parameters",
			url:         "azuresb://webhook.example.com/sb",
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

func TestAzureServiceBusService_Send(t *testing.T) {
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

		// Check Azure-specific headers
		if r.Header.Get("X-Azure-Service") != "ServiceBus" {
			t.Errorf("Expected X-Azure-Service header to be ServiceBus, got %s", r.Header.Get("X-Azure-Service"))
		}

		// Parse and verify request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload["namespace"] == "" {
			t.Error("Expected namespace in payload")
		}

		if payload["authentication"] == nil {
			t.Error("Expected authentication in payload")
		}

		if payload["destination"] == nil {
			t.Error("Expected destination in payload")
		}

		if payload["message"] == nil {
			t.Error("Expected message in payload")
		}

		// Verify authentication structure
		auth, ok := payload["authentication"].(map[string]interface{})
		if !ok {
			t.Error("Expected authentication to be an object")
		} else {
			if auth["type"] == "" {
				t.Error("Expected type in authentication")
			}
		}

		// Verify destination structure
		destination, ok := payload["destination"].(map[string]interface{})
		if !ok {
			t.Error("Expected destination to be an object")
		} else {
			if destination["type"] == "" {
				t.Error("Expected type in destination")
			}
			if destination["name"] == "" {
				t.Error("Expected name in destination")
			}
		}

		// Verify message structure
		message, ok := payload["message"].(map[string]interface{})
		if !ok {
			t.Error("Expected message to be an object")
		} else {
			if message["body"] == "" {
				t.Error("Expected body in message")
			}
			if message["label"] == "" {
				t.Error("Expected label in message")
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MessageId":"12345-67890-abcdef","SequenceNumber":12345}`))
	}))
	defer server.Close()

	// Parse server URL and create Service Bus URL with test_mode=true for HTTP
	serverURL, _ := url.Parse(server.URL)
	sbURL := "azuresb://" + serverURL.Host + "/webhook?namespace=test-bus&queue=notifications&sas_key=test-key&test_mode=true"

	service := NewAzureServiceBusService().(*AzureServiceBusService)
	parsedURL, _ := url.Parse(sbURL)
	service.ParseURL(parsedURL)

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

func TestAzureServiceBusService_SendWithAPIKey(t *testing.T) {
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
		w.Write([]byte(`{"MessageId":"test-message-id"}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	sbURL := "azuresb://test-api-key@" + serverURL.Host + "/webhook?namespace=test-bus&queue=notifications&test_mode=true"

	service := NewAzureServiceBusService().(*AzureServiceBusService)
	parsedURL, _ := url.Parse(sbURL)
	service.ParseURL(parsedURL)

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

func TestAzureServiceBusService_FormatMessage(t *testing.T) {
	service := &AzureServiceBusService{}
	
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

				// Check that timestamp is present
				if messageData["timestamp"] == "" {
					t.Error("Expected timestamp to be present")
				}

				// Check severity mapping
				expectedSeverity := service.getSeverityLevel(tt.notifyType)
				if messageData["severity"] != expectedSeverity {
					t.Errorf("Expected severity '%s', got '%s'", expectedSeverity, messageData["severity"])
				}
			}
		})
	}
}

func TestAzureServiceBusService_BuildAuthentication(t *testing.T) {
	tests := []struct {
		name             string
		service          *AzureServiceBusService
		expectedType     string
		expectedKeyName  string
	}{
		{
			name:            "SAS authentication",
			service:         &AzureServiceBusService{sasKeyName: "SendKey", sasKey: "abc123"},
			expectedType:    "sas",
			expectedKeyName: "SendKey",
		},
		{
			name:         "Managed identity",
			service:      &AzureServiceBusService{},
			expectedType: "managed_identity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := tt.service.buildAuthentication()

			if auth["type"] != tt.expectedType {
				t.Errorf("Expected auth type '%s', got '%s'", tt.expectedType, auth["type"])
			}

			if tt.expectedKeyName != "" {
				if auth["keyName"] != tt.expectedKeyName {
					t.Errorf("Expected keyName '%s', got '%s'", tt.expectedKeyName, auth["keyName"])
				}
			}
		})
	}
}

func TestAzureServiceBusService_BuildDestination(t *testing.T) {
	tests := []struct {
		name             string
		service          *AzureServiceBusService
		expectedType     string
		expectedName     string
		expectedSubscription string
	}{
		{
			name:         "Queue destination",
			service:      &AzureServiceBusService{queueName: "notifications"},
			expectedType: "queue",
			expectedName: "notifications",
		},
		{
			name:         "Topic destination",
			service:      &AzureServiceBusService{topicName: "alerts"},
			expectedType: "topic",
			expectedName: "alerts",
		},
		{
			name:                 "Topic with subscription",
			service:              &AzureServiceBusService{topicName: "alerts", subscriptionName: "email-processor"},
			expectedType:         "topic",
			expectedName:         "alerts",
			expectedSubscription: "email-processor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destination := tt.service.buildDestination()

			if destination["type"] != tt.expectedType {
				t.Errorf("Expected destination type '%s', got '%s'", tt.expectedType, destination["type"])
			}

			if destination["name"] != tt.expectedName {
				t.Errorf("Expected destination name '%s', got '%s'", tt.expectedName, destination["name"])
			}

			if tt.expectedSubscription != "" {
				if destination["subscription"] != tt.expectedSubscription {
					t.Errorf("Expected subscription '%s', got '%s'", tt.expectedSubscription, destination["subscription"])
				}
			}
		})
	}
}

func TestAzureServiceBusService_BuildMessageProperties(t *testing.T) {
	service := &AzureServiceBusService{
		messageProperties: map[string]interface{}{
			"Environment": "production",
			"Service":     "web-api",
		},
	}

	properties := service.buildMessageProperties(NotifyTypeError)

	// Check custom properties
	if properties["Environment"] != "production" {
		t.Errorf("Expected Environment 'production', got '%v'", properties["Environment"])
	}

	if properties["Service"] != "web-api" {
		t.Errorf("Expected Service 'web-api', got '%v'", properties["Service"])
	}

	// Check standard properties
	if properties["NotificationType"] != "error" {
		t.Errorf("Expected NotificationType 'error', got '%v'", properties["NotificationType"])
	}

	if properties["Source"] != "apprise-go" {
		t.Errorf("Expected Source 'apprise-go', got '%v'", properties["Source"])
	}

	if properties["Priority"] != "High" {
		t.Errorf("Expected Priority 'High' for error type, got '%v'", properties["Priority"])
	}

	if properties["Severity"] != "Critical" {
		t.Errorf("Expected Severity 'Critical' for error type, got '%v'", properties["Severity"])
	}
}

func TestAzureServiceBusService_GetSeverityLevel(t *testing.T) {
	service := &AzureServiceBusService{}
	
	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeError, "Critical"},
		{NotifyTypeWarning, "Warning"},
		{NotifyTypeSuccess, "Informational"},
		{NotifyTypeInfo, "Informational"},
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

func TestAzureServiceBusService_ParseInt(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"-1", -1, false},
		{"abc", 0, true},
		{"", 0, true},
		{"123.45", 123, false}, // Should parse integer part
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseInt(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}