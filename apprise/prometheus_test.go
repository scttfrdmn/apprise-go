package apprise

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPrometheusService_GetServiceID(t *testing.T) {
	service := NewPrometheusService()
	if service.GetServiceID() != "prometheus" {
		t.Errorf("Expected service ID 'prometheus', got '%s'", service.GetServiceID())
	}
}

func TestPrometheusService_GetDefaultPort(t *testing.T) {
	service := NewPrometheusService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected port 443, got %d", service.GetDefaultPort())
	}
}

func TestPrometheusService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *PrometheusService)
	}{
		{
			name:        "Basic Prometheus URL with default path",
			url:         "prometheus://alertmanager.example.com",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				expected := "http://alertmanager.example.com/api/v1/webhook"
				if p.webhookURL != expected {
					t.Errorf("Expected webhook URL '%s', got '%s'", expected, p.webhookURL)
				}
				if !p.sendResolved {
					t.Error("Expected sendResolved to be true by default")
				}
			},
		},
		{
			name:        "Prometheus URL with custom path",
			url:         "prometheus://alertmanager.example.com/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				expected := "http://alertmanager.example.com/alerts"
				if p.webhookURL != expected {
					t.Errorf("Expected webhook URL '%s', got '%s'", expected, p.webhookURL)
				}
			},
		},
		{
			name:        "Prometheus URL with port",
			url:         "prometheus://alertmanager.example.com:9093/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				expected := "http://alertmanager.example.com:9093/webhook"
				if p.webhookURL != expected {
					t.Errorf("Expected webhook URL '%s', got '%s'", expected, p.webhookURL)
				}
			},
		},
		{
			name:        "Prometheus URL with HTTPS (port 443)",
			url:         "prometheus://alertmanager.example.com:443/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				expected := "https://alertmanager.example.com:443/alerts"
				if p.webhookURL != expected {
					t.Errorf("Expected webhook URL '%s', got '%s'", expected, p.webhookURL)
				}
			},
		},
		{
			name:        "Prometheus URL with secure parameter",
			url:         "prometheus://alertmanager.example.com/alerts?secure=true",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				if !strings.HasPrefix(p.webhookURL, "https://") {
					t.Errorf("Expected HTTPS URL with secure=true, got '%s'", p.webhookURL)
				}
			},
		},
		{
			name:        "Prometheus URL with send_resolved=false",
			url:         "prometheus://alertmanager.example.com/alerts?send_resolved=false",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				if p.sendResolved {
					t.Error("Expected sendResolved to be false")
				}
			},
		},
		{
			name:        "Prometheus URL with IP address",
			url:         "prometheus://10.0.0.5:9093/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				expected := "http://10.0.0.5:9093/webhook"
				if p.webhookURL != expected {
					t.Errorf("Expected webhook URL '%s', got '%s'", expected, p.webhookURL)
				}
			},
		},
		{
			name:        "PrometheusAM alias scheme",
			url:         "prometheusam://alertmanager.example.com/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, p *PrometheusService) {
				if !strings.Contains(p.webhookURL, "alertmanager.example.com") {
					t.Errorf("Expected URL to contain host, got '%s'", p.webhookURL)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "http://alertmanager.example.com/alerts",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "prometheus:///alerts",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPrometheusService().(*PrometheusService)
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

func TestPrometheusService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		checkFunc   func(*testing.T, PrometheusWebhookPayload)
		expectError bool
	}{
		{
			name: "Basic firing alert",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test alert message",
				NotifyType: NotifyTypeError,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				if payload.Status != "firing" {
					t.Errorf("Expected status 'firing', got '%s'", payload.Status)
				}
				if len(payload.Alerts) != 1 {
					t.Errorf("Expected 1 alert, got %d", len(payload.Alerts))
				}
				alert := payload.Alerts[0]
				if alert.Status != "firing" {
					t.Errorf("Expected alert status 'firing', got '%s'", alert.Status)
				}
				if alert.Labels["severity"] != "critical" {
					t.Errorf("Expected severity 'critical', got '%s'", alert.Labels["severity"])
				}
				if alert.Annotations["summary"] != "Test Alert" {
					t.Errorf("Expected summary 'Test Alert', got '%s'", alert.Annotations["summary"])
				}
				if alert.Annotations["description"] != "Test alert message" {
					t.Errorf("Expected description, got '%s'", alert.Annotations["description"])
				}
			},
		},
		{
			name: "Resolved alert (success notification)",
			request: NotificationRequest{
				Title:      "Issue Resolved",
				Body:       "The problem has been fixed",
				NotifyType: NotifyTypeSuccess,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				if payload.Status != "resolved" {
					t.Errorf("Expected status 'resolved', got '%s'", payload.Status)
				}
				alert := payload.Alerts[0]
				if alert.Status != "resolved" {
					t.Errorf("Expected alert status 'resolved', got '%s'", alert.Status)
				}
				if alert.EndsAt == "" {
					t.Error("Expected endsAt to be set for resolved alert")
				}
			},
		},
		{
			name: "Warning alert",
			request: NotificationRequest{
				Title:      "Warning",
				Body:       "High memory usage detected",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				if payload.Status != "firing" {
					t.Errorf("Expected status 'firing', got '%s'", payload.Status)
				}
				alert := payload.Alerts[0]
				if alert.Labels["severity"] != "warning" {
					t.Errorf("Expected severity 'warning', got '%s'", alert.Labels["severity"])
				}
			},
		},
		{
			name: "Info alert",
			request: NotificationRequest{
				Title:      "Information",
				Body:       "Deployment completed successfully",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				alert := payload.Alerts[0]
				if alert.Labels["severity"] != "info" {
					t.Errorf("Expected severity 'info', got '%s'", alert.Labels["severity"])
				}
			},
		},
		{
			name: "Alert with tags",
			request: NotificationRequest{
				Title:      "Tagged Alert",
				Body:       "Alert with custom tags",
				NotifyType: NotifyTypeError,
				Tags:       []string{"production", "database", "critical"},
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				alert := payload.Alerts[0]
				if alert.Labels["production"] != "true" {
					t.Error("Expected 'production' tag in labels")
				}
				if alert.Labels["database"] != "true" {
					t.Error("Expected 'database' tag in labels")
				}
				if alert.Labels["critical"] != "true" {
					t.Error("Expected 'critical' tag in labels")
				}
			},
		},
		{
			name: "Alert with body only",
			request: NotificationRequest{
				Body:       "Simple alert message",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				alert := payload.Alerts[0]
				if alert.Annotations["summary"] != "Simple alert message" {
					t.Errorf("Expected body as summary, got '%s'", alert.Annotations["summary"])
				}
				if alert.Annotations["description"] != "Simple alert message" {
					t.Errorf("Expected body as description, got '%s'", alert.Annotations["description"])
				}
			},
		},
		{
			name: "Alert with title only",
			request: NotificationRequest{
				Title:      "Title Only Alert",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				alert := payload.Alerts[0]
				if alert.Annotations["summary"] != "Title Only Alert" {
					t.Errorf("Expected title as summary, got '%s'", alert.Annotations["summary"])
				}
			},
		},
		{
			name: "Empty notification with default",
			request: NotificationRequest{
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, payload PrometheusWebhookPayload) {
				alert := payload.Alerts[0]
				if alert.Annotations["summary"] == "" {
					t.Error("Expected default summary for empty notification")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPayload PrometheusWebhookPayload

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got '%s'", r.Method)
				}

				// Check headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", r.Header.Get("Content-Type"))
				}

				// Parse body
				body, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(body, &receivedPayload); err != nil {
					t.Errorf("Failed to parse webhook payload: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return success response
				response := PrometheusWebhookResponse{
					Status:  "success",
					Message: "Alert received",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create service
			service := NewPrometheusService().(*PrometheusService)
			service.webhookURL = server.URL

			// Build payload directly for testing
			payload := service.buildWebhookPayload(tt.request)

			if tt.checkFunc != nil {
				tt.checkFunc(t, payload)
			}

			// Verify common fields
			if payload.Version != "4" {
				t.Errorf("Expected version '4', got '%s'", payload.Version)
			}
			if payload.Receiver != "apprise-go" {
				t.Errorf("Expected receiver 'apprise-go', got '%s'", payload.Receiver)
			}
			if len(payload.Alerts) == 0 {
				t.Error("Expected at least one alert in payload")
			}
			if len(payload.Alerts) > 0 {
				alert := payload.Alerts[0]
				if alert.Labels["alertname"] != "apprise_notification" {
					t.Errorf("Expected alertname 'apprise_notification', got '%s'", alert.Labels["alertname"])
				}
				if alert.Labels["source"] != "apprise-go" {
					t.Errorf("Expected source 'apprise-go', got '%s'", alert.Labels["source"])
				}
				if alert.StartsAt == "" {
					t.Error("Expected startsAt to be set")
				}
				if alert.Fingerprint == "" {
					t.Error("Expected fingerprint to be set")
				}
			}
		})
	}
}

func TestPrometheusService_MapNotifyTypeToStatus(t *testing.T) {
	service := NewPrometheusService().(*PrometheusService)

	tests := []struct {
		notifyType     NotifyType
		expectedStatus string
	}{
		{NotifyTypeError, "firing"},
		{NotifyTypeWarning, "firing"},
		{NotifyTypeInfo, "firing"},
		{NotifyTypeSuccess, "resolved"},
	}

	for _, tt := range tests {
		status := service.mapNotifyTypeToStatus(tt.notifyType)
		if status != tt.expectedStatus {
			t.Errorf("For %v, expected status '%s', got '%s'", tt.notifyType, tt.expectedStatus, status)
		}
	}
}

func TestPrometheusService_MapNotifyTypeToSeverity(t *testing.T) {
	service := NewPrometheusService().(*PrometheusService)

	tests := []struct {
		notifyType       NotifyType
		expectedSeverity string
	}{
		{NotifyTypeError, "critical"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeInfo, "info"},
		{NotifyTypeSuccess, "info"},
	}

	for _, tt := range tests {
		severity := service.mapNotifyTypeToSeverity(tt.notifyType)
		if severity != tt.expectedSeverity {
			t.Errorf("For %v, expected severity '%s', got '%s'", tt.notifyType, tt.expectedSeverity, severity)
		}
	}
}

func TestPrometheusService_TestURL(t *testing.T) {
	service := NewPrometheusService()

	validURLs := []string{
		"prometheus://alertmanager.example.com",
		"prometheus://alertmanager.example.com:9093/webhook",
		"prometheus://10.0.0.5:9093/alerts",
		"prometheusam://alertmanager.example.com/alerts",
		"prometheus://host.com/path?send_resolved=false",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"http://alertmanager.example.com",
		"prometheus:///webhook",
		"prometheusam://",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestPrometheusService_SupportsAttachments(t *testing.T) {
	service := NewPrometheusService()
	if service.SupportsAttachments() {
		t.Error("Prometheus service should not support attachments")
	}
}

func TestPrometheusService_GetMaxBodyLength(t *testing.T) {
	service := NewPrometheusService()
	if service.GetMaxBodyLength() != 8192 {
		t.Errorf("Expected max body length 8192, got %d", service.GetMaxBodyLength())
	}
}

func TestPrometheusService_GenerateFingerprint(t *testing.T) {
	labels1 := map[string]string{
		"alertname": "test_alert",
		"severity":  "critical",
	}

	labels2 := map[string]string{
		"alertname": "test_alert",
		"severity":  "warning",
	}

	fingerprint1 := generateFingerprint(labels1)
	fingerprint2 := generateFingerprint(labels2)

	if fingerprint1 == "" {
		t.Error("Expected non-empty fingerprint")
	}

	// Different labels should produce different fingerprints
	if fingerprint1 == fingerprint2 {
		t.Error("Different labels should produce different fingerprints")
	}
}

func TestPrometheusService_BuildWebhookPayload(t *testing.T) {
	service := NewPrometheusService().(*PrometheusService)

	tests := []struct {
		name    string
		request NotificationRequest
		check   func(*testing.T, PrometheusWebhookPayload)
	}{
		{
			name: "Complete payload structure",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test body",
				NotifyType: NotifyTypeError,
				Tags:       []string{"env_prod", "team_ops"},
			},
			check: func(t *testing.T, payload PrometheusWebhookPayload) {
				if payload.Status == "" {
					t.Error("Expected status to be set")
				}
				if len(payload.Alerts) != 1 {
					t.Errorf("Expected 1 alert, got %d", len(payload.Alerts))
				}
				if payload.GroupLabels == nil {
					t.Error("Expected group labels to be set")
				}
				if payload.CommonLabels == nil {
					t.Error("Expected common labels to be set")
				}
				if payload.CommonAnnotations == nil {
					t.Error("Expected common annotations to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := service.buildWebhookPayload(tt.request)
			tt.check(t, payload)
		})
	}
}
