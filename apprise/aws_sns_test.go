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

func TestAWSSNSService_GetServiceID(t *testing.T) {
	service := NewAWSSNSService()
	if service.GetServiceID() != "sns" {
		t.Errorf("Expected service ID 'sns', got '%s'", service.GetServiceID())
	}
}

func TestAWSSNSService_GetDefaultPort(t *testing.T) {
	service := NewAWSSNSService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAWSSNSService_SupportsAttachments(t *testing.T) {
	service := NewAWSSNSService()
	if service.SupportsAttachments() {
		t.Error("AWS SNS should not support attachments")
	}
}

func TestAWSSNSService_GetMaxBodyLength(t *testing.T) {
	service := NewAWSSNSService()
	expected := 256 * 1024 // 256KB
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestAWSSNSService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		expectedARN string
		expectedRegion string
		expectedAPIKey string
	}{
		{
			name:        "Valid webhook URL with topic ARN",
			url:         "sns://api.example.com/sns-webhook?topic_arn=arn:aws:sns:us-east-1:123456789:my-topic",
			expectError: false,
			expectedARN: "arn:aws:sns:us-east-1:123456789:my-topic",
			expectedRegion: "us-east-1",
		},
		{
			name:        "Valid webhook URL with topic components",
			url:         "sns://api.example.com/sns?topic=alerts&region=eu-west-1&account=987654321",
			expectError: false,
			expectedARN: "arn:aws:sns:eu-west-1:987654321:alerts",
			expectedRegion: "eu-west-1",
		},
		{
			name:        "Valid webhook URL with API key",
			url:         "sns://abc123@api.example.com/webhook?topic=notifications&account=123456789",
			expectError: false,
			expectedAPIKey: "abc123",
			expectedARN: "arn:aws:sns:us-east-1:123456789:notifications",
		},
		{
			name:        "Valid webhook URL with topic name only",
			url:         "sns://webhook.example.com/sns?topic=my-topic",
			expectError: false,
			expectedARN: "my-topic", // Should be just the topic name
		},
		{
			name:        "Invalid scheme",
			url:         "http://api.example.com/sns",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "sns:///webhook",
			expectError: true,
		},
		{
			name:        "Missing topic",
			url:         "sns://api.example.com/webhook",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAWSSNSService().(*AWSSNSService)
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

			if tt.expectedARN != "" && service.topicArn != tt.expectedARN {
				t.Errorf("Expected topic ARN '%s', got '%s'", tt.expectedARN, service.topicArn)
			}

			if tt.expectedRegion != "" && service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}
		})
	}
}

func TestAWSSNSService_TestURL(t *testing.T) {
	service := NewAWSSNSService()
	
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid SNS URL",
			url:         "sns://api.example.com/webhook?topic_arn=arn:aws:sns:us-east-1:123:topic",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://api.example.com/webhook",
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

func TestAWSSNSService_Send(t *testing.T) {
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

		// Parse and verify request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload["topicArn"] == "" {
			t.Error("Expected topicArn in payload")
		}

		if payload["message"] == "" {
			t.Error("Expected message in payload")
		}

		if payload["subject"] == "" {
			t.Error("Expected subject in payload")
		}

		if payload["messageAttributes"] == nil {
			t.Error("Expected messageAttributes in payload")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"MessageId":"12345-67890-abcdef"}`))
	}))
	defer server.Close()

	// Parse server URL and create SNS URL with test_mode=true for HTTP
	serverURL, _ := url.Parse(server.URL)
	snsURL := "sns://" + serverURL.Host + "/webhook?topic_arn=arn:aws:sns:us-east-1:123456789:test-topic&test_mode=true"

	service := NewAWSSNSService().(*AWSSNSService)
	parsedURL, _ := url.Parse(snsURL)
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

func TestAWSSNSService_SendWithAPIKey(t *testing.T) {
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

	// Parse server URL and create SNS URL with API key and test_mode=true for HTTP
	serverURL, _ := url.Parse(server.URL)
	snsURL := "sns://test-api-key@" + serverURL.Host + "/webhook?topic=test-topic&test_mode=true"

	service := NewAWSSNSService().(*AWSSNSService)
	parsedURL, _ := url.Parse(snsURL)
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

func TestAWSSNSService_SendError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"Error":{"Code":"InvalidParameter","Message":"Invalid topic ARN"}}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	snsURL := "sns://" + serverURL.Host + "/webhook?topic_arn=invalid-arn&test_mode=true"

	service := NewAWSSNSService().(*AWSSNSService)
	parsedURL, _ := url.Parse(snsURL)
	service.ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Error Test",
		Body:       "This should fail",
		NotifyType: NotifyTypeError,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error, but got none")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to contain status code 400, got: %v", err)
	}
}

func TestAWSSNSService_FormatMessage(t *testing.T) {
	tests := []struct {
		name           string
		service        *AWSSNSService
		title          string
		body           string
		notifyType     NotifyType
		expectedEmoji  string
		expectedFormat string
	}{
		{
			name:          "Text format info",
			service:       &AWSSNSService{messageFormat: "text"},
			title:         "Test Title",
			body:          "Test Body",
			notifyType:    NotifyTypeInfo,
			expectedEmoji: "ℹ️",
			expectedFormat: "text",
		},
		{
			name:          "Text format success",
			service:       &AWSSNSService{messageFormat: "text"},
			title:         "Success",
			body:          "Operation completed",
			notifyType:    NotifyTypeSuccess,
			expectedEmoji: "✅",
			expectedFormat: "text",
		},
		{
			name:          "JSON format warning",
			service:       &AWSSNSService{messageFormat: "json"},
			title:         "Warning",
			body:          "Something needs attention",
			notifyType:    NotifyTypeWarning,
			expectedEmoji: "⚠️",
			expectedFormat: "json",
		},
		{
			name:          "JSON format error",
			service:       &AWSSNSService{messageFormat: "json"},
			title:         "Error",
			body:          "Something went wrong",
			notifyType:    NotifyTypeError,
			expectedEmoji: "❌",
			expectedFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.service.formatMessage(tt.title, tt.body, tt.notifyType)

			if tt.expectedFormat == "json" {
				// For JSON format, parse and verify structure
				var messageData map[string]interface{}
				err := json.Unmarshal([]byte(message), &messageData)
				if err != nil {
					t.Fatalf("Failed to parse JSON message: %v", err)
				}

				if messageData["title"] != tt.title {
					t.Errorf("Expected title '%s', got '%s'", tt.title, messageData["title"])
				}

				if messageData["body"] != tt.body {
					t.Errorf("Expected body '%s', got '%s'", tt.body, messageData["body"])
				}

				if messageData["type"] != tt.notifyType.String() {
					t.Errorf("Expected type '%s', got '%s'", tt.notifyType.String(), messageData["type"])
				}

				if messageData["emoji"] != tt.expectedEmoji {
					t.Errorf("Expected emoji '%s', got '%s'", tt.expectedEmoji, messageData["emoji"])
				}
			} else {
				// For text format, check emoji prefix and content
				if !strings.HasPrefix(message, tt.expectedEmoji) {
					t.Errorf("Expected message to start with '%s', got: %s", tt.expectedEmoji, message)
				}

				if !strings.Contains(message, tt.title) {
					t.Errorf("Expected message to contain title '%s', got: %s", tt.title, message)
				}

				if !strings.Contains(message, tt.body) {
					t.Errorf("Expected message to contain body '%s', got: %s", tt.body, message)
				}
			}
		})
	}
}

func TestAWSSNSService_MessageAttributes(t *testing.T) {
	service := &AWSSNSService{
		attributes: map[string]string{
			"Environment": "production",
			"Service":     "web-api",
		},
	}

	attrs := service.buildMessageAttributes(NotifyTypeError)

	// Check custom attributes
	if env, exists := attrs["Environment"]; !exists {
		t.Error("Expected Environment attribute")
	} else if envMap, ok := env.(map[string]string); !ok {
		t.Error("Expected Environment attribute to be map[string]string")
	} else if envMap["StringValue"] != "production" {
		t.Errorf("Expected Environment value 'production', got '%s'", envMap["StringValue"])
	}

	// Check notification type attribute
	if notifyType, exists := attrs["NotificationType"]; !exists {
		t.Error("Expected NotificationType attribute")
	} else if typeMap, ok := notifyType.(map[string]string); !ok {
		t.Error("Expected NotificationType attribute to be map[string]string")
	} else if typeMap["StringValue"] != "error" {
		t.Errorf("Expected NotificationType value 'error', got '%s'", typeMap["StringValue"])
	}

	// Check source attribute
	if source, exists := attrs["Source"]; !exists {
		t.Error("Expected Source attribute")
	} else if sourceMap, ok := source.(map[string]string); !ok {
		t.Error("Expected Source attribute to be map[string]string")
	} else if sourceMap["StringValue"] != "apprise-go" {
		t.Errorf("Expected Source value 'apprise-go', got '%s'", sourceMap["StringValue"])
	}
}

func TestAWSSNSService_LongMessage(t *testing.T) {
	service := NewAWSSNSService().(*AWSSNSService)
	
	// Create a body longer than the max length
	title := "Long Message Test"
	maxLength := service.GetMaxBodyLength()
	longBody := strings.Repeat("a", maxLength+100)
	
	req := NotificationRequest{
		Title:      title,
		Body:       longBody,
		NotifyType: NotifyTypeInfo,
	}

	// Apply the truncation logic from Send method
	body := req.Body
	if len(req.Title)+len(body) > maxLength-100 {
		availableLength := maxLength - len(req.Title) - 100
		if availableLength > 0 {
			body = body[:availableLength] + "..."
		}
	}

	message := service.formatMessage(req.Title, body, req.NotifyType)
	
	// The message should now be within reasonable bounds
	if len(message) > maxLength {
		t.Errorf("Message should be truncated to reasonable length, got length %d", len(message))
	}
	
	// Verify the body was actually truncated
	if !strings.HasSuffix(body, "...") {
		t.Error("Expected body to be truncated with '...' suffix")
	}
}