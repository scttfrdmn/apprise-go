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

func TestDatadogService_GetServiceID(t *testing.T) {
	service := NewDatadogService()
	if service.GetServiceID() != "datadog" {
		t.Errorf("Expected service ID 'datadog', got '%s'", service.GetServiceID())
	}
}

func TestDatadogService_GetDefaultPort(t *testing.T) {
	service := NewDatadogService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestDatadogService_SupportsAttachments(t *testing.T) {
	service := NewDatadogService()
	if !service.SupportsAttachments() {
		t.Error("Datadog should support attachments (metadata)")
	}
}

func TestDatadogService_GetMaxBodyLength(t *testing.T) {
	service := NewDatadogService()
	expected := 8192
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestDatadogService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedAPIKey   string
		expectedAppKey   string
		expectedRegion   string
		expectedTags     []string
		expectedWebhook  string
		expectedProxyKey string
	}{
		{
			name:           "Basic API key",
			url:            "datadog://abc123@datadoghq.com/",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
		},
		{
			name:           "API key with app key",
			url:            "datadog://api_key:app_key@datadoghq.com/",
			expectError:    false,
			expectedAPIKey: "api_key",
			expectedAppKey: "app_key",
			expectedRegion: "us",
		},
		{
			name:           "EU region with tags",
			url:            "datadog://abc123@datadoghq.com/?region=eu&tags=env:prod,service:api",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "eu",
			expectedTags:   []string{"env:prod", "service:api"},
		},
		{
			name:           "App key in query parameter",
			url:            "datadog://abc123@datadoghq.com/?app_key=def456&region=us3",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedAppKey: "def456",
			expectedRegion: "us3",
		},
		{
			name:             "Webhook proxy mode",
			url:              "datadog://proxy-key@webhook.example.com/datadog?api_key=dd_key&region=us5",
			expectError:      false,
			expectedWebhook:  "https://webhook.example.com/datadog",
			expectedProxyKey: "proxy-key",
			expectedAPIKey:   "dd_key",
			expectedRegion:   "us5",
		},
		{
			name:             "Webhook with tags",
			url:              "datadog://proxy@webhook.example.com/datadog?api_key=key&region=gov&tags=team:ops,env:staging",
			expectError:      false,
			expectedWebhook:  "https://webhook.example.com/datadog",
			expectedProxyKey: "proxy",
			expectedAPIKey:   "key",
			expectedRegion:   "gov",
			expectedTags:     []string{"team:ops", "env:staging"},
		},
		{
			name:           "AP1 region",
			url:            "datadog://abc123@datadoghq.com/?region=ap1",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "ap1",
		},
		{
			name:        "Invalid scheme",
			url:         "http://abc123@datadoghq.com/",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "datadog://@datadoghq.com/",
			expectError: true,
		},
		{
			name:        "Invalid region",
			url:         "datadog://abc123@datadoghq.com/?region=invalid",
			expectError: true,
		},
		{
			name:        "Webhook missing API key",
			url:         "datadog://proxy@webhook.example.com/datadog?region=us",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewDatadogService().(*DatadogService)
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

			if tt.expectedAppKey != "" && service.appKey != tt.expectedAppKey {
				t.Errorf("Expected app key '%s', got '%s'", tt.expectedAppKey, service.appKey)
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

			if len(tt.expectedTags) > 0 {
				if len(service.tags) != len(tt.expectedTags) {
					t.Errorf("Expected %d tags, got %d", len(tt.expectedTags), len(service.tags))
				}
				for i, expectedTag := range tt.expectedTags {
					if i < len(service.tags) && service.tags[i] != expectedTag {
						t.Errorf("Expected tag[%d] '%s', got '%s'", i, expectedTag, service.tags[i])
					}
				}
			}
		})
	}
}

func TestDatadogService_IsValidRegion(t *testing.T) {
	service := &DatadogService{}

	validRegions := []string{"us", "eu", "us3", "us5", "gov", "ap1"}
	for _, region := range validRegions {
		if !service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be valid", region)
		}
	}

	invalidRegions := []string{"invalid", "asia", "canada", ""}
	for _, region := range invalidRegions {
		if service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be invalid", region)
		}
	}
}

func TestDatadogService_TestURL(t *testing.T) {
	service := NewDatadogService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Datadog URL",
			url:         "datadog://abc123@datadoghq.com/?region=us",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "datadog://proxy@webhook.example.com/datadog?api_key=key",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://abc123@datadoghq.com/",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "datadog://@datadoghq.com/",
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

func TestDatadogService_SendWebhook(t *testing.T) {
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
		var payload DatadogWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "datadog" {
			t.Errorf("Expected service 'datadog', got '%s'", payload.Service)
		}

		if payload.Region != "us" {
			t.Errorf("Expected region 'us', got '%s'", payload.Region)
		}

		if payload.Event == nil {
			t.Error("Expected event to be present")
		}

		if payload.Metrics == nil {
			t.Error("Expected metrics to be present")
		}

		if payload.Log == nil {
			t.Error("Expected log to be present")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewDatadogService().(*DatadogService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.apiKey = "dd-api-key"
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
			title:      "High Memory Usage",
			body:       "Memory usage is above 80%",
			notifyType: NotifyTypeWarning,
		},
		{
			name:       "Error notification",
			title:      "Service Down",
			body:       "API service is not responding",
			notifyType: NotifyTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{
				Title:      tt.title,
				Body:       tt.body,
				NotifyType: tt.notifyType,
				Tags:       []string{"test", "automation"},
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

func TestDatadogService_SendAPI(t *testing.T) {
	// Create mock API server for webhook mode (simpler to test)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers for webhook
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify webhook authentication
		if r.Header.Get("X-API-Key") != "test-proxy-key" {
			t.Errorf("Expected X-API-Key 'test-proxy-key', got '%s'", r.Header.Get("X-API-Key"))
		}

		// Parse and verify webhook payload
		var payload DatadogWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode webhook payload: %v", err)
		}

		// Verify payload components
		if payload.Service != "datadog" {
			t.Errorf("Expected service 'datadog', got '%s'", payload.Service)
		}

		if payload.Event == nil {
			t.Error("Expected event to be present")
		} else if payload.Event.Title != "Webhook Test" {
			t.Errorf("Expected event title 'Webhook Test', got '%s'", payload.Event.Title)
		}

		if payload.Metrics == nil || len(payload.Metrics.Series) == 0 {
			t.Error("Expected metrics to be present")
		}

		if payload.Log == nil {
			t.Error("Expected log to be present")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Configure service for webhook mode
	service := NewDatadogService().(*DatadogService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.apiKey = "test-api-key"
	service.region = "us"

	req := NotificationRequest{
		Title:      "Webhook Test",
		Body:       "Testing webhook integration",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Webhook Send failed: %v", err)
	}
}

func TestDatadogService_CreateEvent(t *testing.T) {
	service := &DatadogService{
		tags: []string{"env:test", "service:apprise"},
	}

	req := NotificationRequest{
		Title:      "Test Event",
		Body:       "Test event body",
		NotifyType: NotifyTypeWarning,
		Tags:       []string{"custom:tag"},
	}

	event := service.createEvent(req)

	if event.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, event.Title)
	}

	if event.Text != req.Body {
		t.Errorf("Expected text '%s', got '%s'", req.Body, event.Text)
	}

	if event.AlertType != "warning" {
		t.Errorf("Expected alert type 'warning', got '%s'", event.AlertType)
	}

	if event.Priority != "normal" {
		t.Errorf("Expected priority 'normal', got '%s'", event.Priority)
	}

	if event.SourceTypeName != "apprise-go" {
		t.Errorf("Expected source type 'apprise-go', got '%s'", event.SourceTypeName)
	}

	// Check that tags are merged
	expectedTags := []string{"env:test", "service:apprise", "custom:tag", "source:apprise-go"}
	if len(event.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(event.Tags))
	}
}

func TestDatadogService_CreateMetric(t *testing.T) {
	service := &DatadogService{
		tags: []string{"env:test"},
	}

	req := NotificationRequest{
		Title:      "Test Metric",
		Body:       "Test metric body",
		NotifyType: NotifyTypeSuccess,
		Tags:       []string{"custom:tag"},
	}

	metric := service.createMetric(req)

	if metric.Metric != "apprise.notification" {
		t.Errorf("Expected metric name 'apprise.notification', got '%s'", metric.Metric)
	}

	if metric.Type != "count" {
		t.Errorf("Expected metric type 'count', got '%s'", metric.Type)
	}

	if len(metric.Points) == 0 {
		t.Error("Expected metric points to be present")
	}

	// Check tags include notification type
	found := false
	for _, tag := range metric.Tags {
		if tag == "notification_type:success" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find notification_type:success tag")
	}
}

func TestDatadogService_CreateLog(t *testing.T) {
	service := &DatadogService{
		tags: []string{"env:test"},
	}

	req := NotificationRequest{
		Title:      "Test Log",
		Body:       "Test log body",
		NotifyType: NotifyTypeError,
		Tags:       []string{"custom:tag"},
		BodyFormat: "html",
	}

	log := service.createLog(req)

	expectedMessage := "Test Log: Test log body"
	if log.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, log.Message)
	}

	if log.Level != "ERROR" {
		t.Errorf("Expected level 'ERROR', got '%s'", log.Level)
	}

	if log.Service != "apprise-go" {
		t.Errorf("Expected service 'apprise-go', got '%s'", log.Service)
	}

	// Check attributes
	if log.Attributes["notification_type"] != "error" {
		t.Errorf("Expected notification_type 'error', got '%v'", log.Attributes["notification_type"])
	}

	if log.Attributes["body_format"] != "html" {
		t.Errorf("Expected body_format 'html', got '%v'", log.Attributes["body_format"])
	}
}

func TestDatadogService_HelperMethods(t *testing.T) {
	service := &DatadogService{}

	// Test priority mapping
	tests := []struct {
		notifyType       NotifyType
		expectedPriority string
		expectedAlert    string
		expectedLogLevel string
	}{
		{NotifyTypeInfo, "low", "info", "INFO"},
		{NotifyTypeSuccess, "low", "success", "INFO"},
		{NotifyTypeWarning, "normal", "warning", "WARN"},
		{NotifyTypeError, "normal", "error", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if priority := service.getPriorityForNotifyType(tt.notifyType); priority != tt.expectedPriority {
				t.Errorf("Expected priority '%s', got '%s'", tt.expectedPriority, priority)
			}

			if alert := service.getAlertTypeForNotifyType(tt.notifyType); alert != tt.expectedAlert {
				t.Errorf("Expected alert type '%s', got '%s'", tt.expectedAlert, alert)
			}

			if logLevel := service.getLogLevelForNotifyType(tt.notifyType); logLevel != tt.expectedLogLevel {
				t.Errorf("Expected log level '%s', got '%s'", tt.expectedLogLevel, logLevel)
			}
		})
	}
}

func TestDatadogService_APIURLs(t *testing.T) {
	tests := []struct {
		region          string
		expectedAPIURL  string
		expectedLogsURL string
	}{
		{"us", "https://api.datadoghq.com", "https://http-intake.logs.datadoghq.com"},
		{"eu", "https://api.datadoghq.eu", "https://http-intake.logs.datadoghq.eu"},
		{"us3", "https://api.us3.datadoghq.com", "https://http-intake.logs.us3.datadoghq.com"},
		{"us5", "https://api.us5.datadoghq.com", "https://http-intake.logs.us5.datadoghq.com"},
		{"gov", "https://api.ddog-gov.com", "https://http-intake.logs.ddog-gov.com"},
		{"ap1", "https://api.ap1.datadoghq.com", "https://http-intake.logs.ap1.datadoghq.com"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			service := &DatadogService{region: tt.region}

			if apiURL := service.getAPIBaseURL(); apiURL != tt.expectedAPIURL {
				t.Errorf("Expected API URL '%s', got '%s'", tt.expectedAPIURL, apiURL)
			}

			if logsURL := service.getLogsAPIURL(); logsURL != tt.expectedLogsURL {
				t.Errorf("Expected logs URL '%s', got '%s'", tt.expectedLogsURL, logsURL)
			}
		})
	}
}

func TestDatadogService_WithAttachments(t *testing.T) {
	service := &DatadogService{
		tags: []string{"env:test"},
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test data"), "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	log := service.createLog(req)

	// Should have attachment information in attributes
	attachments, ok := log.Attributes["attachments"]
	if !ok {
		t.Error("Expected attachments to be present in log attributes")
	}

	attachmentList, ok := attachments.([]map[string]string)
	if !ok {
		t.Fatal("Expected attachments to be array of maps")
	}

	if len(attachmentList) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachmentList))
	}

	if attachmentList[0]["name"] != "test.txt" {
		t.Errorf("Expected attachment name 'test.txt', got '%s'", attachmentList[0]["name"])
	}

	if attachmentList[0]["mime_type"] != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", attachmentList[0]["mime_type"])
	}
}
