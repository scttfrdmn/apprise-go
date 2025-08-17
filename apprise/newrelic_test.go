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

func TestNewRelicService_GetServiceID(t *testing.T) {
	service := NewNewRelicService()
	if service.GetServiceID() != "newrelic" {
		t.Errorf("Expected service ID 'newrelic', got '%s'", service.GetServiceID())
	}
}

func TestNewRelicService_GetDefaultPort(t *testing.T) {
	service := NewNewRelicService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestNewRelicService_SupportsAttachments(t *testing.T) {
	service := NewNewRelicService()
	if !service.SupportsAttachments() {
		t.Error("New Relic should support attachments (metadata)")
	}
}

func TestNewRelicService_GetMaxBodyLength(t *testing.T) {
	service := NewNewRelicService()
	expected := 4096
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestNewRelicService_ParseURL(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAPIKey     string
		expectedAccountID  string
		expectedRegion     string
		expectedWebhook    string
		expectedProxyKey   string
	}{
		{
			name:              "Basic API key with account ID",
			url:               "newrelic://api_key@newrelic.com/?account_id=123456",
			expectError:       false,
			expectedAPIKey:    "api_key",
			expectedAccountID: "123456",
			expectedRegion:    "us",
		},
		{
			name:              "EU region",
			url:               "newrelic://api_key@newrelic.com/?account_id=123456&region=eu",
			expectError:       false,
			expectedAPIKey:    "api_key",
			expectedAccountID: "123456",
			expectedRegion:    "eu",
		},
		{
			name:              "Webhook proxy mode",
			url:               "newrelic://proxy-key@webhook.example.com/newrelic?api_key=nr_key&account_id=789012",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/newrelic",
			expectedProxyKey:  "proxy-key",
			expectedAPIKey:    "nr_key",
			expectedAccountID: "789012",
			expectedRegion:    "us",
		},
		{
			name:              "Webhook with EU region",
			url:               "newrelic://proxy@webhook.example.com/newrelic?api_key=key&account_id=456789&region=eu",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/newrelic",
			expectedProxyKey:  "proxy",
			expectedAPIKey:    "key",
			expectedAccountID: "456789",
			expectedRegion:    "eu",
		},
		{
			name:        "Invalid scheme",
			url:         "http://api_key@newrelic.com/?account_id=123456",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "newrelic://@newrelic.com/?account_id=123456",
			expectError: true,
		},
		{
			name:        "Missing account ID",
			url:         "newrelic://api_key@newrelic.com/",
			expectError: true,
		},
		{
			name:        "Invalid region",
			url:         "newrelic://api_key@newrelic.com/?account_id=123456&region=invalid",
			expectError: true,
		},
		{
			name:        "Webhook missing API key",
			url:         "newrelic://proxy@webhook.example.com/newrelic?account_id=123456",
			expectError: true,
		},
		{
			name:        "Webhook missing account ID",
			url:         "newrelic://proxy@webhook.example.com/newrelic?api_key=key",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNewRelicService().(*NewRelicService)
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

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}

			if service.accountID != tt.expectedAccountID {
				t.Errorf("Expected account ID '%s', got '%s'", tt.expectedAccountID, service.accountID)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
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

func TestNewRelicService_IsValidRegion(t *testing.T) {
	service := &NewRelicService{}

	validRegions := []string{"us", "eu"}
	for _, region := range validRegions {
		if !service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be valid", region)
		}
	}

	invalidRegions := []string{"invalid", "ap", "gov", ""}
	for _, region := range invalidRegions {
		if service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be invalid", region)
		}
	}
}

func TestNewRelicService_TestURL(t *testing.T) {
	service := NewNewRelicService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid New Relic URL",
			url:         "newrelic://api_key@newrelic.com/?account_id=123456",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "newrelic://proxy@webhook.example.com/newrelic?api_key=key&account_id=123456",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://api_key@newrelic.com/?account_id=123456",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "newrelic://@newrelic.com/?account_id=123456",
			expectError: true,
		},
		{
			name:        "Missing account ID",
			url:         "newrelic://api_key@newrelic.com/",
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

func TestNewRelicService_SendWebhook(t *testing.T) {
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
		var payload NewRelicWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "newrelic" {
			t.Errorf("Expected service 'newrelic', got '%s'", payload.Service)
		}

		if payload.AccountID != "123456" {
			t.Errorf("Expected account ID '123456', got '%s'", payload.AccountID)
		}

		if payload.Region != "us" {
			t.Errorf("Expected region 'us', got '%s'", payload.Region)
		}

		if payload.Events == nil || len(payload.Events.Events) == 0 {
			t.Error("Expected events to be present")
		}

		if payload.Metrics == nil || len(payload.Metrics.Metrics) == 0 {
			t.Error("Expected metrics to be present")
		}

		if payload.Logs == nil || len(payload.Logs.Logs) == 0 {
			t.Error("Expected logs to be present")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewNewRelicService().(*NewRelicService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.apiKey = "nr-api-key"
	service.accountID = "123456"
	service.region = "us"

	// Test different notification types
	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
	}{
		{
			name:       "Info notification",
			title:      "System Status",
			body:       "All systems operational",
			notifyType: NotifyTypeInfo,
		},
		{
			name:       "Success notification",
			title:      "Deployment Complete",
			body:       "Application deployed successfully",
			notifyType: NotifyTypeSuccess,
		},
		{
			name:       "Warning notification",
			title:      "High CPU Usage",
			body:       "CPU usage is above 80%",
			notifyType: NotifyTypeWarning,
		},
		{
			name:       "Error notification",
			title:      "Service Failure",
			body:       "Database connection failed",
			notifyType: NotifyTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{
				Title:      tt.title,
				Body:       tt.body,
				NotifyType: tt.notifyType,
				Tags:       []string{"env:test", "service:api"},
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

func TestNewRelicService_CreateEvent(t *testing.T) {
	service := &NewRelicService{}

	req := NotificationRequest{
		Title:      "Test Event",
		Body:       "Test event body",
		NotifyType: NotifyTypeWarning,
		Tags:       []string{"env:test", "service:api", "single_tag"},
		BodyFormat: "markdown",
	}

	event := service.createEvent(req)

	if event.EventType != "AppriseNotification" {
		t.Errorf("Expected event type 'AppriseNotification', got '%s'", event.EventType)
	}

	if event.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, event.Title)
	}

	if event.Message != req.Body {
		t.Errorf("Expected message '%s', got '%s'", req.Body, event.Message)
	}

	if event.NotificationType != "warning" {
		t.Errorf("Expected notification type 'warning', got '%s'", event.NotificationType)
	}

	if event.Severity != "WARNING" {
		t.Errorf("Expected severity 'WARNING', got '%s'", event.Severity)
	}

	if event.Source != "apprise-go" {
		t.Errorf("Expected source 'apprise-go', got '%s'", event.Source)
	}

	// Check tags
	if event.Tags["env"] != "test" {
		t.Errorf("Expected env tag 'test', got '%s'", event.Tags["env"])
	}

	if event.Tags["service"] != "api" {
		t.Errorf("Expected service tag 'api', got '%s'", event.Tags["service"])
	}

	if event.Tags["single_tag"] != "true" {
		t.Errorf("Expected single_tag 'true', got '%s'", event.Tags["single_tag"])
	}

	if event.Tags["source"] != "apprise-go" {
		t.Error("Expected source tag to be set")
	}

	// Check attributes
	if event.Attributes["notification_type"] != "warning" {
		t.Errorf("Expected notification_type attribute 'warning', got '%v'", event.Attributes["notification_type"])
	}

	if event.Attributes["body_format"] != "markdown" {
		t.Errorf("Expected body_format attribute 'markdown', got '%v'", event.Attributes["body_format"])
	}

	if event.Attributes["body_length"] != len(req.Body) {
		t.Errorf("Expected body_length %d, got %v", len(req.Body), event.Attributes["body_length"])
	}
}

func TestNewRelicService_CreateMetric(t *testing.T) {
	service := &NewRelicService{}

	req := NotificationRequest{
		Title:      "Test Metric",
		Body:       "Test metric body",
		NotifyType: NotifyTypeSuccess,
		Tags:       []string{"env:prod", "app:web"},
	}

	metric := service.createMetric(req)

	if metric.Name != "apprise.notification.count" {
		t.Errorf("Expected metric name 'apprise.notification.count', got '%s'", metric.Name)
	}

	if metric.Type != "count" {
		t.Errorf("Expected metric type 'count', got '%s'", metric.Type)
	}

	if metric.Value != 1 {
		t.Errorf("Expected metric value 1, got %v", metric.Value)
	}

	// Check attributes
	if metric.Attributes["notification_type"] != "success" {
		t.Errorf("Expected notification_type attribute 'success', got '%v'", metric.Attributes["notification_type"])
	}

	if metric.Attributes["source"] != "apprise-go" {
		t.Errorf("Expected source attribute 'apprise-go', got '%v'", metric.Attributes["source"])
	}

	// Check tag attributes
	if metric.Attributes["tag.env"] != "prod" {
		t.Errorf("Expected tag.env attribute 'prod', got '%v'", metric.Attributes["tag.env"])
	}

	if metric.Attributes["tag.app"] != "web" {
		t.Errorf("Expected tag.app attribute 'web', got '%v'", metric.Attributes["tag.app"])
	}
}

func TestNewRelicService_CreateLog(t *testing.T) {
	service := &NewRelicService{}

	req := NotificationRequest{
		Title:      "Test Log",
		Body:       "Test log body",
		NotifyType: NotifyTypeError,
		Tags:       []string{"env:staging", "component:auth"},
		BodyFormat: "text",
	}

	log := service.createLog(req)

	expectedMessage := "[ERROR] Test Log: Test log body"
	if log.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, log.Message)
	}

	if log.LogLevel != "ERROR" {
		t.Errorf("Expected log level 'ERROR', got '%s'", log.LogLevel)
	}

	if log.Service != "apprise-go" {
		t.Errorf("Expected service 'apprise-go', got '%s'", log.Service)
	}

	// Check tags
	if log.Tags["env"] != "staging" {
		t.Errorf("Expected env tag 'staging', got '%s'", log.Tags["env"])
	}

	if log.Tags["component"] != "auth" {
		t.Errorf("Expected component tag 'auth', got '%s'", log.Tags["component"])
	}

	if log.Tags["source"] != "apprise-go" {
		t.Error("Expected source tag to be set")
	}

	// Check attributes
	if log.Attributes["notification_type"] != "error" {
		t.Errorf("Expected notification_type attribute 'error', got '%v'", log.Attributes["notification_type"])
	}

	if log.Attributes["body_format"] != "text" {
		t.Errorf("Expected body_format attribute 'text', got '%v'", log.Attributes["body_format"])
	}
}

func TestNewRelicService_HelperMethods(t *testing.T) {
	service := &NewRelicService{}

	// Test severity and log level mapping
	tests := []struct {
		notifyType       NotifyType
		expectedSeverity string
		expectedLogLevel string
	}{
		{NotifyTypeInfo, "INFO", "INFO"},
		{NotifyTypeSuccess, "INFO", "INFO"},
		{NotifyTypeWarning, "WARNING", "WARN"},
		{NotifyTypeError, "CRITICAL", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if severity := service.getSeverityForNotifyType(tt.notifyType); severity != tt.expectedSeverity {
				t.Errorf("Expected severity '%s', got '%s'", tt.expectedSeverity, severity)
			}

			if logLevel := service.getLogLevelForNotifyType(tt.notifyType); logLevel != tt.expectedLogLevel {
				t.Errorf("Expected log level '%s', got '%s'", tt.expectedLogLevel, logLevel)
			}
		})
	}
}

func TestNewRelicService_APIURLs(t *testing.T) {
	tests := []struct {
		region      string
		expectedURL string
	}{
		{"us", "https://insights-api.newrelic.com"},
		{"eu", "https://insights-api.eu01.nr-data.net"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			service := &NewRelicService{region: tt.region}

			if apiURL := service.getAPIBaseURL(); apiURL != tt.expectedURL {
				t.Errorf("Expected API URL '%s', got '%s'", tt.expectedURL, apiURL)
			}
		})
	}
}

func TestNewRelicService_WithAttachments(t *testing.T) {
	service := &NewRelicService{}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test data"), "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	err = attachmentMgr.AddData([]byte("image data"), "test.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	// Test event
	event := service.createEvent(req)

	if event.Attributes["attachment_count"] != 2 {
		t.Errorf("Expected attachment_count 2, got %v", event.Attributes["attachment_count"])
	}

	attachmentTypes, ok := event.Attributes["attachment_types"].(string)
	if !ok {
		t.Fatal("Expected attachment_types to be string")
	}

	if !strings.Contains(attachmentTypes, "text/plain") {
		t.Error("Expected attachment_types to contain 'text/plain'")
	}

	if !strings.Contains(attachmentTypes, "image/jpeg") {
		t.Error("Expected attachment_types to contain 'image/jpeg'")
	}

	// Test log
	log := service.createLog(req)

	attachments, ok := log.Attributes["attachments"]
	if !ok {
		t.Error("Expected attachments to be present in log attributes")
	}

	attachmentList, ok := attachments.([]map[string]string)
	if !ok {
		t.Fatal("Expected attachments to be array of maps")
	}

	if len(attachmentList) != 2 {
		t.Errorf("Expected 2 attachments, got %d", len(attachmentList))
	}

	// Check first attachment
	if attachmentList[0]["name"] != "test.txt" {
		t.Errorf("Expected attachment name 'test.txt', got '%s'", attachmentList[0]["name"])
	}

	if attachmentList[0]["mime_type"] != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", attachmentList[0]["mime_type"])
	}
}