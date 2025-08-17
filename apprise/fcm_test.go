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

func TestFCMService_GetServiceID(t *testing.T) {
	service := NewFCMService()
	if service.GetServiceID() != "fcm" {
		t.Errorf("Expected service ID 'fcm', got '%s'", service.GetServiceID())
	}
}

func TestFCMService_GetDefaultPort(t *testing.T) {
	service := NewFCMService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestFCMService_SupportsAttachments(t *testing.T) {
	service := NewFCMService()
	if !service.SupportsAttachments() {
		t.Error("FCM should support attachments (images in notifications)")
	}
}

func TestFCMService_GetMaxBodyLength(t *testing.T) {
	service := NewFCMService()
	expected := 4096
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestFCMService_ParseURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedProject   string
		expectedServerKey string
		expectedAPIKey    string
	}{
		{
			name:              "Valid FCM URL with server key",
			url:               "fcm://webhook.example.com/firebase?project_id=my-project&server_key=AAAA1234567890",
			expectError:       false,
			expectedProject:   "my-project",
			expectedServerKey: "AAAA1234567890",
		},
		{
			name:            "Valid FCM URL with service account",
			url:             "fcm://webhook.example.com/proxy?project_id=test-project&service_account=path/to/sa.json",
			expectError:     false,
			expectedProject: "test-project",
		},
		{
			name:              "Valid FCM URL with API key authentication",
			url:               "fcm://api-key-123@webhook.example.com/fcm?project_id=company-project&server_key=legacy-key",
			expectError:       false,
			expectedAPIKey:    "api-key-123",
			expectedProject:   "company-project",
			expectedServerKey: "legacy-key",
		},
		{
			name:        "Invalid scheme",
			url:         "http://webhook.example.com/fcm?project_id=test&server_key=key",
			expectError: true,
		},
		{
			name:        "Missing project_id",
			url:         "fcm://webhook.example.com/fcm?server_key=key",
			expectError: true,
		},
		{
			name:        "Missing authentication (no server_key or service_account)",
			url:         "fcm://webhook.example.com/fcm?project_id=test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewFCMService().(*FCMService)
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

			if tt.expectedServerKey != "" && service.serverKey != tt.expectedServerKey {
				t.Errorf("Expected server key '%s', got '%s'", tt.expectedServerKey, service.serverKey)
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}
		})
	}
}

func TestFCMService_TestURL(t *testing.T) {
	service := NewFCMService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid FCM URL",
			url:         "fcm://webhook.example.com/fcm?project_id=test&server_key=key",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://webhook.example.com/fcm",
			expectError: true,
		},
		{
			name:        "Missing required parameters",
			url:         "fcm://webhook.example.com/fcm",
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

func TestFCMService_Send(t *testing.T) {
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
		if payload["service"] != "fcm" {
			t.Error("Expected service to be 'fcm'")
		}

		if payload["projectId"] == "" {
			t.Error("Expected projectId in payload")
		}

		if payload["message"] == nil {
			t.Error("Expected message in payload")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"projects/test/messages/12345"}`))
	}))
	defer server.Close()

	// Parse server URL and create FCM URL
	serverURL, _ := url.Parse(server.URL)
	fcmURL := "fcm://" + serverURL.Host + "/webhook?project_id=test-project&server_key=test-key"

	service := NewFCMService().(*FCMService)
	parsedURL, _ := url.Parse(fcmURL)
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

func TestFCMService_SendWithAPIKey(t *testing.T) {
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
		w.Write([]byte(`{"name":"projects/test/messages/67890"}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	fcmURL := "fcm://test-api-key@" + serverURL.Host + "/webhook?project_id=test-project&server_key=test-key"

	service := NewFCMService().(*FCMService)
	parsedURL, _ := url.Parse(fcmURL)
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

func TestFCMService_CreateMessage(t *testing.T) {
	service := &FCMService{
		projectID: "test-project",
	}

	req := NotificationRequest{
		Title:      "Test Message",
		Body:       "Test body content",
		NotifyType: NotifyTypeWarning,
	}

	message := service.createMessage(req)

	// Check basic message structure
	if message.Notification.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, message.Notification.Title)
	}

	if message.Notification.Body != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, message.Notification.Body)
	}

	// Check data payload
	if message.Data["notification_type"] != req.NotifyType.String() {
		t.Errorf("Expected notification_type '%s', got '%s'", req.NotifyType.String(), message.Data["notification_type"])
	}

	if message.Data["source"] != "apprise-go" {
		t.Error("Expected source to be 'apprise-go'")
	}

	// Check platform-specific configurations
	if message.Android == nil {
		t.Error("Expected Android configuration to be present")
	}

	if message.APNS == nil {
		t.Error("Expected APNS configuration to be present")
	}

	if message.WebPush == nil {
		t.Error("Expected WebPush configuration to be present")
	}
}

func TestFCMService_CreateAndroidConfig(t *testing.T) {
	service := &FCMService{}

	req := NotificationRequest{
		Title:      "Android Test",
		Body:       "Android notification test",
		NotifyType: NotifyTypeError,
	}

	config := service.createAndroidConfig(req)

	if config.Priority != "high" {
		t.Errorf("Expected high priority for error notification, got '%s'", config.Priority)
	}

	if config.Notification.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, config.Notification.Title)
	}

	if config.Notification.Body != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, config.Notification.Body)
	}

	if config.Notification.Color != "#FF0000" {
		t.Errorf("Expected red color for error, got '%s'", config.Notification.Color)
	}

	if config.Notification.ChannelID != "error_notifications" {
		t.Errorf("Expected error_notifications channel, got '%s'", config.Notification.ChannelID)
	}
}

func TestFCMService_CreateAPNSConfig(t *testing.T) {
	service := &FCMService{}

	req := NotificationRequest{
		Title:      "iOS Test",
		Body:       "iOS notification test",
		NotifyType: NotifyTypeSuccess,
	}

	config := service.createAPNSConfig(req)

	if config.Headers["apns-priority"] != "5" {
		t.Errorf("Expected normal priority for success notification, got '%s'", config.Headers["apns-priority"])
	}

	// Check payload structure
	payload, ok := config.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("Expected payload to be a map")
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected aps to be a map")
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected alert to be a map")
	}

	if alert["title"] != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, alert["title"])
	}

	if alert["body"] != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, alert["body"])
	}
}

func TestFCMService_CreateWebPushConfig(t *testing.T) {
	service := &FCMService{}

	req := NotificationRequest{
		Title:      "Web Test",
		Body:       "Web notification test",
		NotifyType: NotifyTypeInfo,
	}

	config := service.createWebPushConfig(req)

	if config.Notification.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, config.Notification.Title)
	}

	if config.Notification.Body != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, config.Notification.Body)
	}

	if config.Data["notification_type"] != req.NotifyType.String() {
		t.Errorf("Expected notification_type '%s', got '%s'", req.NotifyType.String(), config.Data["notification_type"])
	}

	if config.Notification.Tag != "apprise_info" {
		t.Errorf("Expected tag 'apprise_info', got '%s'", config.Notification.Tag)
	}

	// Info notifications should not require interaction
	if config.Notification.RequireInteraction {
		t.Error("Info notifications should not require interaction")
	}
}

func TestFCMService_HelperMethods(t *testing.T) {
	service := &FCMService{}

	// Test priority mapping
	tests := []struct {
		notifyType       NotifyType
		expectedPriority string
		expectedColor    string
		expectedSound    string
	}{
		{NotifyTypeInfo, "normal", "#007BFF", "default"},
		{NotifyTypeSuccess, "normal", "#00FF00", "default"},
		{NotifyTypeWarning, "high", "#FFA500", "warning"},
		{NotifyTypeError, "high", "#FF0000", "urgent"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if priority := service.getPriorityForNotifyType(tt.notifyType); priority != tt.expectedPriority {
				t.Errorf("Expected priority '%s', got '%s'", tt.expectedPriority, priority)
			}

			if color := service.getColorForNotifyType(tt.notifyType); color != tt.expectedColor {
				t.Errorf("Expected color '%s', got '%s'", tt.expectedColor, color)
			}

			if sound := service.getSoundForNotifyType(tt.notifyType); sound != tt.expectedSound {
				t.Errorf("Expected sound '%s', got '%s'", tt.expectedSound, sound)
			}
		})
	}
}

func TestFCMService_ChannelIDMapping(t *testing.T) {
	service := &FCMService{}

	tests := []struct {
		notifyType        NotifyType
		expectedChannelID string
	}{
		{NotifyTypeInfo, "default_notifications"},
		{NotifyTypeSuccess, "success_notifications"},
		{NotifyTypeWarning, "warning_notifications"},
		{NotifyTypeError, "error_notifications"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			channelID := service.getChannelIDForNotifyType(tt.notifyType)
			if channelID != tt.expectedChannelID {
				t.Errorf("Expected channel ID '%s', got '%s'", tt.expectedChannelID, channelID)
			}
		})
	}
}

func TestFCMService_APNSPriorityMapping(t *testing.T) {
	service := &FCMService{}

	tests := []struct {
		notifyType       NotifyType
		expectedPriority string
		expectedSound    string
	}{
		{NotifyTypeInfo, "5", "default"},
		{NotifyTypeSuccess, "5", "default"},
		{NotifyTypeWarning, "10", "warning.wav"},
		{NotifyTypeError, "10", "critical.wav"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			priority := service.getAPNSPriorityForNotifyType(tt.notifyType)
			if priority != tt.expectedPriority {
				t.Errorf("Expected APNS priority '%s', got '%s'", tt.expectedPriority, priority)
			}

			sound := service.getAPNSSoundForNotifyType(tt.notifyType)
			if sound != tt.expectedSound {
				t.Errorf("Expected APNS sound '%s', got '%s'", tt.expectedSound, sound)
			}
		})
	}
}
