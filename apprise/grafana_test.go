package apprise

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGrafanaService_GetServiceID(t *testing.T) {
	service := NewGrafanaService()
	if service.GetServiceID() != "grafana" {
		t.Errorf("Expected service ID 'grafana', got '%s'", service.GetServiceID())
	}
}

func TestGrafanaService_GetDefaultPort(t *testing.T) {
	service := NewGrafanaService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestGrafanaService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *GrafanaService)
	}{
		{
			name:        "Basic URL",
			url:         "grafana://alerts.example.com/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.webhookURL != "https://alerts.example.com/webhook" {
					t.Errorf("Expected webhookURL 'https://alerts.example.com/webhook', got '%s'", g.webhookURL)
				}
				if g.method != "POST" {
					t.Errorf("Expected method 'POST', got '%s'", g.method)
				}
			},
		},
		{
			name:        "HTTPS explicit",
			url:         "grafanas://alerts.example.com/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.webhookURL != "https://alerts.example.com/webhook" {
					t.Errorf("Expected HTTPS URL, got '%s'", g.webhookURL)
				}
			},
		},
		{
			name:        "With Basic Auth",
			url:         "grafana://user:pass@alerts.example.com/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.username != "user" {
					t.Errorf("Expected username 'user', got '%s'", g.username)
				}
				if g.password != "pass" {
					t.Errorf("Expected password 'pass', got '%s'", g.password)
				}
			},
		},
		{
			name:        "With Bearer Token",
			url:         "grafana://token123@alerts.example.com/webhook",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.authHeader != "Bearer token123" {
					t.Errorf("Expected auth header 'Bearer token123', got '%s'", g.authHeader)
				}
			},
		},
		{
			name:        "With method parameter",
			url:         "grafana://alerts.example.com/webhook?method=PUT",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.method != "PUT" {
					t.Errorf("Expected method 'PUT', got '%s'", g.method)
				}
			},
		},
		{
			name:        "With max_alerts parameter",
			url:         "grafana://alerts.example.com/webhook?max_alerts=50",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.maxAlerts != 50 {
					t.Errorf("Expected max_alerts 50, got %d", g.maxAlerts)
				}
			},
		},
		{
			name:        "With HMAC secret",
			url:         "grafana://alerts.example.com/webhook?hmac_secret=mysecret",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.hmacSecret != "mysecret" {
					t.Errorf("Expected hmac_secret 'mysecret', got '%s'", g.hmacSecret)
				}
			},
		},
		{
			name:        "With custom headers",
			url:         "grafana://alerts.example.com/webhook?header_X-Custom=value1&header_X-Another=value2",
			expectError: false,
			checkFunc: func(t *testing.T, g *GrafanaService) {
				if g.customHeaders["X-Custom"] != "value1" {
					t.Errorf("Expected custom header X-Custom='value1', got '%s'", g.customHeaders["X-Custom"])
				}
				if g.customHeaders["X-Another"] != "value2" {
					t.Errorf("Expected custom header X-Another='value2', got '%s'", g.customHeaders["X-Another"])
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "http://alerts.example.com/webhook",
			expectError: true,
		},
		{
			name:        "Invalid method",
			url:         "grafana://alerts.example.com/webhook?method=DELETE",
			expectError: true,
		},
		{
			name:        "Invalid max_alerts",
			url:         "grafana://alerts.example.com/webhook?max_alerts=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGrafanaService().(*GrafanaService)
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

func TestGrafanaService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		setupServer func(*testing.T) *httptest.Server
		expectError bool
		checkFunc   func(*testing.T, *http.Request, GrafanaWebhookPayload)
	}{
		{
			name: "Info notification",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "This is a test alert",
				NotifyType: NotifyTypeInfo,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload GrafanaWebhookPayload) {
				if payload.Status != "firing" {
					t.Errorf("Expected status 'firing', got '%s'", payload.Status)
				}
				if payload.Title != "Test Alert" {
					t.Errorf("Expected title 'Test Alert', got '%s'", payload.Title)
				}
				if len(payload.Alerts) != 1 {
					t.Errorf("Expected 1 alert, got %d", len(payload.Alerts))
				}
				if payload.Alerts[0].Labels["severity"] != "info" {
					t.Errorf("Expected severity 'info', got '%s'", payload.Alerts[0].Labels["severity"])
				}
			},
		},
		{
			name: "Success notification (resolved)",
			request: NotificationRequest{
				Title:      "Issue Resolved",
				Body:       "The issue has been resolved",
				NotifyType: NotifyTypeSuccess,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload GrafanaWebhookPayload) {
				if payload.Status != "resolved" {
					t.Errorf("Expected status 'resolved', got '%s'", payload.Status)
				}
				if payload.Alerts[0].Status != "resolved" {
					t.Errorf("Expected alert status 'resolved', got '%s'", payload.Alerts[0].Status)
				}
				if payload.Alerts[0].Labels["severity"] != "ok" {
					t.Errorf("Expected severity 'ok', got '%s'", payload.Alerts[0].Labels["severity"])
				}
			},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "High CPU Usage",
				Body:       "CPU usage is above 80%",
				NotifyType: NotifyTypeWarning,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload GrafanaWebhookPayload) {
				if payload.Alerts[0].Labels["severity"] != "warning" {
					t.Errorf("Expected severity 'warning', got '%s'", payload.Alerts[0].Labels["severity"])
				}
			},
		},
		{
			name: "Error notification (critical)",
			request: NotificationRequest{
				Title:      "Service Down",
				Body:       "Service is not responding",
				NotifyType: NotifyTypeError,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload GrafanaWebhookPayload) {
				if payload.Alerts[0].Labels["severity"] != "critical" {
					t.Errorf("Expected severity 'critical', got '%s'", payload.Alerts[0].Labels["severity"])
				}
			},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test with tags",
				NotifyType: NotifyTypeInfo,
				Tags:       []string{"production", "web-server"},
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload GrafanaWebhookPayload) {
				if payload.Alerts[0].Labels["tag_production"] != "true" {
					t.Error("Expected tag_production label")
				}
				if payload.Alerts[0].Labels["tag_web-server"] != "true" {
					t.Error("Expected tag_web-server label")
				}
			},
		},
		{
			name: "Server returns error",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test",
				NotifyType: NotifyTypeInfo,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer(t)
			defer server.Close()

			service := NewGrafanaService().(*GrafanaService)

			// Parse the test server URL
			serviceURL, _ := url.Parse("grafana://" + strings.TrimPrefix(server.URL, "http://") + "/webhook")
			service.ParseURL(serviceURL)

			// Override the webhookURL with the test server URL
			service.webhookURL = server.URL + "/webhook"

			// Capture request for validation
			var capturedRequest *http.Request
			var capturedPayload GrafanaWebhookPayload

			// Wrap the test server to capture requests
			if !tt.expectError && tt.checkFunc != nil {
				originalHandler := server.Config.Handler
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedRequest = r
					body, _ := io.ReadAll(r.Body)
					json.Unmarshal(body, &capturedPayload)
					r.Body = io.NopCloser(strings.NewReader(string(body)))
					originalHandler.ServeHTTP(w, r)
				})
			}

			// Send notification
			err := service.Send(context.Background(), tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkFunc != nil {
				tt.checkFunc(t, capturedRequest, capturedPayload)
			}
		})
	}
}

func TestGrafanaService_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewGrafanaService().(*GrafanaService)
	serviceURL, _ := url.Parse("grafana://testuser:testpass@" + strings.TrimPrefix(server.URL, "http://") + "/webhook")
	service.ParseURL(serviceURL)
	service.webhookURL = server.URL + "/webhook"

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGrafanaService_BearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewGrafanaService().(*GrafanaService)
	serviceURL, _ := url.Parse("grafana://mytoken123@" + strings.TrimPrefix(server.URL, "http://") + "/webhook")
	service.ParseURL(serviceURL)
	service.webhookURL = server.URL + "/webhook"

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGrafanaService_HMACSignature(t *testing.T) {
	hmacSecret := "mysecretkey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("X-Grafana-Alerting-Signature")
		timestamp := r.Header.Get("X-Grafana-Timestamp")

		if signature == "" || timestamp == "" {
			t.Error("Missing HMAC signature or timestamp headers")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewGrafanaService().(*GrafanaService)
	serviceURL, _ := url.Parse("grafana://" + strings.TrimPrefix(server.URL, "http://") + "/webhook?hmac_secret=" + hmacSecret)
	service.ParseURL(serviceURL)
	service.webhookURL = server.URL + "/webhook"

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGrafanaService_MaxAlerts(t *testing.T) {
	service := NewGrafanaService().(*GrafanaService)
	service.maxAlerts = 2

	// Create a request that would generate 1 alert
	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	payload := service.buildPayload(req)

	// Manually add more alerts to test truncation
	for i := 0; i < 5; i++ {
		payload.Alerts = append(payload.Alerts, GrafanaAlert{
			Status: "firing",
			Labels: map[string]string{"test": "true"},
		})
	}

	// Apply max alerts limit
	if len(payload.Alerts) > service.maxAlerts {
		truncated := len(payload.Alerts) - service.maxAlerts
		payload.Alerts = payload.Alerts[:service.maxAlerts]
		payload.TruncatedAlerts = truncated
	}

	if len(payload.Alerts) != 2 {
		t.Errorf("Expected 2 alerts after truncation, got %d", len(payload.Alerts))
	}

	if payload.TruncatedAlerts != 4 {
		t.Errorf("Expected 4 truncated alerts, got %d", payload.TruncatedAlerts)
	}
}

func TestGrafanaService_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Error("Expected custom header X-Custom-Header")
		}
		if r.Header.Get("X-Another") != "another-value" {
			t.Error("Expected custom header X-Another")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewGrafanaService().(*GrafanaService)
	serviceURL, _ := url.Parse("grafana://" + strings.TrimPrefix(server.URL, "http://") + "/webhook?header_X-Custom-Header=custom-value&header_X-Another=another-value")
	service.ParseURL(serviceURL)
	service.webhookURL = server.URL + "/webhook"

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGrafanaService_TestURL(t *testing.T) {
	service := NewGrafanaService()

	validURLs := []string{
		"grafana://alerts.example.com/webhook",
		"grafanas://alerts.example.com/webhook",
		"grafana://user:pass@alerts.example.com/webhook",
		"grafana://token@alerts.example.com/webhook",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"http://alerts.example.com/webhook",
		"slack://alerts.example.com/webhook",
		"not-a-url",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestGrafanaService_SupportsAttachments(t *testing.T) {
	service := NewGrafanaService()
	if service.SupportsAttachments() {
		t.Error("Grafana service should not support attachments")
	}
}

func TestGrafanaService_GetMaxBodyLength(t *testing.T) {
	service := NewGrafanaService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected unlimited body length (0), got %d", service.GetMaxBodyLength())
	}
}

func TestGrafanaService_PayloadStructure(t *testing.T) {
	service := NewGrafanaService().(*GrafanaService)

	req := NotificationRequest{
		Title:      "Critical Alert",
		Body:       "Database connection lost",
		NotifyType: NotifyTypeError,
		Tags:       []string{"database", "production"},
	}

	payload := service.buildPayload(req)

	// Validate payload structure
	if payload.Receiver != "apprise-go" {
		t.Errorf("Expected receiver 'apprise-go', got '%s'", payload.Receiver)
	}

	if payload.Version != "1.9.5-1" {
		t.Errorf("Expected version '1.9.5-1', got '%s'", payload.Version)
	}

	if len(payload.Alerts) == 0 {
		t.Fatal("Expected at least one alert")
	}

	alert := payload.Alerts[0]

	if alert.Annotations["summary"] != "Critical Alert" {
		t.Errorf("Expected summary 'Critical Alert', got '%s'", alert.Annotations["summary"])
	}

	if alert.Annotations["description"] != "Database connection lost" {
		t.Errorf("Expected description 'Database connection lost', got '%s'", alert.Annotations["description"])
	}

	if alert.StartsAt.IsZero() {
		t.Error("Expected StartsAt to be set")
	}

	// Validate JSON marshaling
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	var unmarshaledPayload GrafanaWebhookPayload
	err = json.Unmarshal(jsonData, &unmarshaledPayload)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if unmarshaledPayload.Title != payload.Title {
		t.Error("Payload lost data during marshal/unmarshal")
	}
}
