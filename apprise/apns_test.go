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

func TestAPNSService_GetServiceID(t *testing.T) {
	service := NewAPNSService()
	if service.GetServiceID() != "apns" {
		t.Errorf("Expected service ID 'apns', got '%s'", service.GetServiceID())
	}
}

func TestAPNSService_GetDefaultPort(t *testing.T) {
	service := NewAPNSService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAPNSService_SupportsAttachments(t *testing.T) {
	service := NewAPNSService()
	if !service.SupportsAttachments() {
		t.Error("APNS should support attachments (rich notifications)")
	}
}

func TestAPNSService_GetMaxBodyLength(t *testing.T) {
	service := NewAPNSService()
	expected := 4096
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestAPNSService_ParseURL(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		expectError         bool
		expectedBundleID    string
		expectedEnvironment string
		expectedKeyID       string
		expectedTeamID      string
		expectedAPIKey      string
	}{
		{
			name:                "Valid APNS URL with JWT authentication",
			url:                 "apns://webhook.example.com/apns?bundle_id=com.example.app&key_id=ABC123&team_id=DEF456&key_path=path/to/key.p8",
			expectError:         false,
			expectedBundleID:    "com.example.app",
			expectedEnvironment: "production",
			expectedKeyID:       "ABC123",
			expectedTeamID:      "DEF456",
		},
		{
			name:                "Valid APNS URL with certificate authentication",
			url:                 "apns://webhook.example.com/proxy?bundle_id=com.test.app&cert_path=cert.p12&cert_pass=password",
			expectError:         false,
			expectedBundleID:    "com.test.app",
			expectedEnvironment: "production",
		},
		{
			name:                "Valid APNS URL with sandbox environment",
			url:                 "apns://webhook.example.com/apns?bundle_id=com.dev.app&environment=sandbox&key_id=KEY&team_id=TEAM&key_path=key.p8",
			expectError:         false,
			expectedBundleID:    "com.dev.app",
			expectedEnvironment: "sandbox",
			expectedKeyID:       "KEY",
			expectedTeamID:      "TEAM",
		},
		{
			name:             "Valid APNS URL with API key authentication",
			url:              "apns://api-key-123@webhook.example.com/apns?bundle_id=com.company.app&key_id=XYZ&team_id=ABC&key_path=auth.p8",
			expectError:      false,
			expectedAPIKey:   "api-key-123",
			expectedBundleID: "com.company.app",
			expectedKeyID:    "XYZ",
			expectedTeamID:   "ABC",
		},
		{
			name:        "Invalid scheme",
			url:         "http://webhook.example.com/apns?bundle_id=com.app&key_id=key&team_id=team",
			expectError: true,
		},
		{
			name:        "Missing bundle_id",
			url:         "apns://webhook.example.com/apns?key_id=key&team_id=team",
			expectError: true,
		},
		{
			name:        "Invalid environment",
			url:         "apns://webhook.example.com/apns?bundle_id=com.app&environment=invalid&key_id=key&team_id=team",
			expectError: true,
		},
		{
			name:        "Missing authentication (no JWT or certificate)",
			url:         "apns://webhook.example.com/apns?bundle_id=com.app",
			expectError: true,
		},
		{
			name:        "Incomplete JWT authentication (missing key_path)",
			url:         "apns://webhook.example.com/apns?bundle_id=com.app&key_id=key&team_id=team",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAPNSService().(*APNSService)
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

			if tt.expectedBundleID != "" && service.bundleID != tt.expectedBundleID {
				t.Errorf("Expected bundle ID '%s', got '%s'", tt.expectedBundleID, service.bundleID)
			}

			if tt.expectedEnvironment != "" && service.environment != tt.expectedEnvironment {
				t.Errorf("Expected environment '%s', got '%s'", tt.expectedEnvironment, service.environment)
			}

			if tt.expectedKeyID != "" && service.keyID != tt.expectedKeyID {
				t.Errorf("Expected key ID '%s', got '%s'", tt.expectedKeyID, service.keyID)
			}

			if tt.expectedTeamID != "" && service.teamID != tt.expectedTeamID {
				t.Errorf("Expected team ID '%s', got '%s'", tt.expectedTeamID, service.teamID)
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}
		})
	}
}

func TestAPNSService_TestURL(t *testing.T) {
	service := NewAPNSService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid APNS URL",
			url:         "apns://webhook.example.com/apns?bundle_id=com.app&key_id=key&team_id=team&key_path=key.p8",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://webhook.example.com/apns",
			expectError: true,
		},
		{
			name:        "Missing required parameters",
			url:         "apns://webhook.example.com/apns",
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

func TestAPNSService_Send(t *testing.T) {
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
		var apnsReq APNSRequest
		if err := json.NewDecoder(r.Body).Decode(&apnsReq); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify request structure
		if apnsReq.DeviceToken != "webhook-managed" {
			t.Errorf("Expected device token 'webhook-managed', got '%s'", apnsReq.DeviceToken)
		}

		if apnsReq.Source != "apprise-go" {
			t.Error("Expected source to be 'apprise-go'")
		}

		if apnsReq.Environment == "" {
			t.Error("Expected environment to be set")
		}

		// Verify payload structure
		if apnsReq.Payload.APS == nil {
			t.Error("Expected APS payload to be present")
		}

		if apnsReq.Payload.APS.Alert == nil {
			t.Error("Expected alert to be present")
		}

		// Verify headers
		if len(apnsReq.Headers) == 0 {
			t.Error("Expected APNS headers to be present")
		}

		if apnsReq.Headers["apns-topic"] == "" {
			t.Error("Expected apns-topic header")
		}

		// Verify authentication
		if apnsReq.Authentication.Method == "" {
			t.Error("Expected authentication method")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"uuid":"550e8400-e29b-41d4-a716-446655440000"}`))
	}))
	defer server.Close()

	// Parse server URL and create APNS URL
	serverURL, _ := url.Parse(server.URL)
	apnsURL := "apns://" + serverURL.Host + "/webhook?bundle_id=com.test.app&key_id=test-key&team_id=test-team&key_path=test.p8"

	service := NewAPNSService().(*APNSService)
	parsedURL, _ := url.Parse(apnsURL)
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

func TestAPNSService_SendWithAPIKey(t *testing.T) {
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
		_, _ = w.Write([]byte(`{"uuid":"test-uuid"}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	apnsURL := "apns://test-api-key@" + serverURL.Host + "/webhook?bundle_id=com.test.app&key_id=key&team_id=team&key_path=key.p8"

	service := NewAPNSService().(*APNSService)
	parsedURL, _ := url.Parse(apnsURL)
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

func TestAPNSService_CreatePayload(t *testing.T) {
	service := &APNSService{
		bundleID: "com.test.app",
	}

	req := NotificationRequest{
		Title:      "Test Message",
		Body:       "Test body content",
		NotifyType: NotifyTypeWarning,
	}

	payload := service.createPayload(req)

	// Check APS structure
	if payload.APS == nil {
		t.Fatal("Expected APS payload to be present")
	}

	// Check alert
	alert, ok := payload.APS.Alert.(APNSAlert)
	if !ok {
		t.Fatal("Expected alert to be APNSAlert type")
	}

	if alert.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, alert.Title)
	}

	if alert.Body != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, alert.Body)
	}

	// Check data payload
	if payload.Data["notification_type"] != req.NotifyType.String() {
		t.Errorf("Expected notification_type '%s', got '%s'", req.NotifyType.String(), payload.Data["notification_type"])
	}

	if payload.Data["source"] != "apprise-go" {
		t.Error("Expected source to be 'apprise-go'")
	}

	// Check interruption level for warning
	if payload.APS.InterruptionLevel != "time-sensitive" {
		t.Errorf("Expected time-sensitive interruption level for warning, got '%s'", payload.APS.InterruptionLevel)
	}
}

func TestAPNSService_CreateAuthentication(t *testing.T) {
	// Test JWT authentication
	service := &APNSService{
		keyID:    "ABC123",
		teamID:   "DEF456",
		bundleID: "com.test.app",
		keyPath:  "path/to/key.p8",
	}

	auth := service.createAuthentication()

	if auth.Method != "jwt" {
		t.Errorf("Expected JWT authentication method, got '%s'", auth.Method)
	}

	if auth.KeyID != service.keyID {
		t.Errorf("Expected key ID '%s', got '%s'", service.keyID, auth.KeyID)
	}

	if auth.TeamID != service.teamID {
		t.Errorf("Expected team ID '%s', got '%s'", service.teamID, auth.TeamID)
	}

	// Test certificate authentication
	service2 := &APNSService{
		bundleID:        "com.test.app",
		certificatePath: "path/to/cert.p12",
		certificatePass: "password",
	}

	auth2 := service2.createAuthentication()

	if auth2.Method != "certificate" {
		t.Errorf("Expected certificate authentication method, got '%s'", auth2.Method)
	}

	if auth2.CertificatePath != service2.certificatePath {
		t.Errorf("Expected certificate path '%s', got '%s'", service2.certificatePath, auth2.CertificatePath)
	}
}

func TestAPNSService_CreateHeaders(t *testing.T) {
	service := &APNSService{
		bundleID: "com.test.app",
	}

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test body",
		NotifyType: NotifyTypeError,
	}

	headers := service.createHeaders(req)

	// Check required headers
	if headers["apns-topic"] != service.bundleID {
		t.Errorf("Expected apns-topic '%s', got '%s'", service.bundleID, headers["apns-topic"])
	}

	if headers["apns-priority"] != "10" {
		t.Errorf("Expected high priority for error notification, got '%s'", headers["apns-priority"])
	}

	if headers["apns-push-type"] != "alert" {
		t.Errorf("Expected push type 'alert', got '%s'", headers["apns-push-type"])
	}

	// Check collapse ID for error notifications
	expectedCollapseID := "apprise-error"
	if headers["apns-collapse-id"] != expectedCollapseID {
		t.Errorf("Expected collapse ID '%s', got '%s'", expectedCollapseID, headers["apns-collapse-id"])
	}
}

func TestAPNSService_HelperMethods(t *testing.T) {
	service := &APNSService{}

	// Test sound configurations
	tests := []struct {
		notifyType        NotifyType
		expectedPriority  string
		expectedCategory  string
		expectedInterrupt string
		expectedRelevance float64
	}{
		{NotifyTypeInfo, "5", "INFO_CATEGORY", "passive", 0.4},
		{NotifyTypeSuccess, "5", "SUCCESS_CATEGORY", "active", 0.6},
		{NotifyTypeWarning, "10", "WARNING_CATEGORY", "time-sensitive", 0.8},
		{NotifyTypeError, "10", "ERROR_CATEGORY", "critical", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if priority := service.getPriorityForNotifyType(tt.notifyType); priority != tt.expectedPriority {
				t.Errorf("Expected priority '%s', got '%s'", tt.expectedPriority, priority)
			}

			if category := service.getCategoryForNotifyType(tt.notifyType); category != tt.expectedCategory {
				t.Errorf("Expected category '%s', got '%s'", tt.expectedCategory, category)
			}

			if interrupt := service.getInterruptionLevelForNotifyType(tt.notifyType); interrupt != tt.expectedInterrupt {
				t.Errorf("Expected interruption level '%s', got '%s'", tt.expectedInterrupt, interrupt)
			}

			if relevance := service.getRelevanceScoreForNotifyType(tt.notifyType); relevance != tt.expectedRelevance {
				t.Errorf("Expected relevance score %f, got %f", tt.expectedRelevance, relevance)
			}
		})
	}
}

func TestAPNSService_SoundConfiguration(t *testing.T) {
	service := &APNSService{}

	// Test error sound (critical)
	errorSound := service.getSoundForNotifyType(NotifyTypeError)
	apnsSound, ok := errorSound.(APNSSound)
	if !ok {
		t.Fatal("Expected error sound to be APNSSound type")
	}

	if apnsSound.Critical != 1 {
		t.Error("Expected error sound to be critical")
	}

	if apnsSound.Name != "critical.wav" {
		t.Errorf("Expected critical sound name, got '%s'", apnsSound.Name)
	}

	// Test warning sound
	warningSound := service.getSoundForNotifyType(NotifyTypeWarning)
	warningSoundStruct, ok := warningSound.(APNSSound)
	if !ok {
		t.Fatal("Expected warning sound to be APNSSound type")
	}

	if warningSoundStruct.Name != "warning.wav" {
		t.Errorf("Expected warning sound name, got '%s'", warningSoundStruct.Name)
	}

	// Test success sound (string)
	successSound := service.getSoundForNotifyType(NotifyTypeSuccess)
	successSoundStr, ok := successSound.(string)
	if !ok {
		t.Fatal("Expected success sound to be string type")
	}

	if successSoundStr != "success.wav" {
		t.Errorf("Expected success sound 'success.wav', got '%s'", successSoundStr)
	}

	// Test info sound (default)
	infoSound := service.getSoundForNotifyType(NotifyTypeInfo)
	infoSoundStr, ok := infoSound.(string)
	if !ok {
		t.Fatal("Expected info sound to be string type")
	}

	if infoSoundStr != "default" {
		t.Errorf("Expected default sound for info, got '%s'", infoSoundStr)
	}
}

func TestAPNSService_WithAttachments(t *testing.T) {
	service := &APNSService{
		bundleID: "com.test.app",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test image data"), "test.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This message has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	payload := service.createPayload(req)

	// Should have mutable-content set for rich notifications
	if payload.APS.MutableContent != 1 {
		t.Error("Expected mutable-content to be 1 for messages with attachments")
	}

	// Should have attachment information in data
	attachments, ok := payload.Data["attachments"]
	if !ok {
		t.Error("Expected attachments data to be present")
	}

	attachmentList, ok := attachments.([]map[string]string)
	if !ok {
		t.Fatal("Expected attachments to be array of maps")
	}

	if len(attachmentList) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachmentList))
	}

	if attachmentList[0]["name"] != "test.jpg" {
		t.Errorf("Expected attachment name 'test.jpg', got '%s'", attachmentList[0]["name"])
	}

	if attachmentList[0]["mime_type"] != "image/jpeg" {
		t.Errorf("Expected MIME type 'image/jpeg', got '%s'", attachmentList[0]["mime_type"])
	}
}
