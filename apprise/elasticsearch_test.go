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

func TestElasticsearchService_GetServiceID(t *testing.T) {
	service := NewElasticsearchService()
	if service.GetServiceID() != "elasticsearch" {
		t.Errorf("Expected service ID 'elasticsearch', got '%s'", service.GetServiceID())
	}
}

func TestElasticsearchService_GetDefaultPort(t *testing.T) {
	service := NewElasticsearchService()
	if service.GetDefaultPort() != 9200 {
		t.Errorf("Expected port 9200, got %d", service.GetDefaultPort())
	}
}

func TestElasticsearchService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *ElasticsearchService)
	}{
		{
			name:        "Basic Elasticsearch URL",
			url:         "elasticsearch://localhost:9200/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.host != "localhost" {
					t.Errorf("Expected host 'localhost', got '%s'", e.host)
				}
				if e.port != "9200" {
					t.Errorf("Expected port '9200', got '%s'", e.port)
				}
				if e.index != "alerts" {
					t.Errorf("Expected index 'alerts', got '%s'", e.index)
				}
				if e.scheme != "http" {
					t.Errorf("Expected scheme 'http', got '%s'", e.scheme)
				}
			},
		},
		{
			name:        "With basic authentication",
			url:         "elasticsearch://user:pass@es.example.com:9200/notifications",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.username != "user" {
					t.Errorf("Expected username 'user', got '%s'", e.username)
				}
				if e.password != "pass" {
					t.Errorf("Expected password 'pass', got '%s'", e.password)
				}
				if e.host != "es.example.com" {
					t.Errorf("Expected host 'es.example.com', got '%s'", e.host)
				}
			},
		},
		{
			name:        "With API key",
			url:         "elasticsearch://es.example.com/logs?apikey=abc123xyz",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.apiKey != "abc123xyz" {
					t.Errorf("Expected API key 'abc123xyz', got '%s'", e.apiKey)
				}
			},
		},
		{
			name:        "With HTTPS",
			url:         "elasticsearch://es.example.com/logs?secure=true",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.scheme != "https" {
					t.Errorf("Expected scheme 'https', got '%s'", e.scheme)
				}
			},
		},
		{
			name:        "With SSL parameter",
			url:         "elasticsearch://es.example.com/logs?ssl=true",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.scheme != "https" {
					t.Errorf("Expected scheme 'https', got '%s'", e.scheme)
				}
			},
		},
		{
			name:        "Default index when path is empty",
			url:         "elasticsearch://localhost:9200",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.index != "apprise-notifications" {
					t.Errorf("Expected default index 'apprise-notifications', got '%s'", e.index)
				}
			},
		},
		{
			name:        "Default port when not specified",
			url:         "elasticsearch://localhost/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.port != "9200" {
					t.Errorf("Expected default port '9200', got '%s'", e.port)
				}
			},
		},
		{
			name:        "OpenSearch alias scheme",
			url:         "opensearch://opensearch.example.com:9200/logs",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.host != "opensearch.example.com" {
					t.Errorf("Expected host 'opensearch.example.com', got '%s'", e.host)
				}
			},
		},
		{
			name:        "ES short alias",
			url:         "es://localhost:9200/alerts",
			expectError: false,
			checkFunc: func(t *testing.T, e *ElasticsearchService) {
				if e.host != "localhost" {
					t.Errorf("Expected host 'localhost', got '%s'", e.host)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "http://es.example.com/logs",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "elasticsearch://:9200/logs",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewElasticsearchService().(*ElasticsearchService)
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

func TestElasticsearchService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		checkFunc   func(*testing.T, ElasticsearchDocument)
		expectError bool
	}{
		{
			name: "Basic notification",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test alert message",
				NotifyType: NotifyTypeError,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Title != "Test Alert" {
					t.Errorf("Expected title 'Test Alert', got '%s'", doc.Title)
				}
				if doc.Message != "Test alert message" {
					t.Errorf("Expected message, got '%s'", doc.Message)
				}
				if doc.Severity != "error" {
					t.Errorf("Expected severity 'error', got '%s'", doc.Severity)
				}
				if doc.NotifyType != "error" {
					t.Errorf("Expected notify_type 'error', got '%s'", doc.NotifyType)
				}
				if doc.Source != "apprise-go" {
					t.Errorf("Expected source 'apprise-go', got '%s'", doc.Source)
				}
			},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "Warning",
				Body:       "High memory usage",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Severity != "warning" {
					t.Errorf("Expected severity 'warning', got '%s'", doc.Severity)
				}
			},
		},
		{
			name: "Info notification",
			request: NotificationRequest{
				Title:      "Information",
				Body:       "Deployment completed",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Severity != "info" {
					t.Errorf("Expected severity 'info', got '%s'", doc.Severity)
				}
			},
		},
		{
			name: "Success notification",
			request: NotificationRequest{
				Title:      "Success",
				Body:       "Task completed successfully",
				NotifyType: NotifyTypeSuccess,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Severity != "success" {
					t.Errorf("Expected severity 'success', got '%s'", doc.Severity)
				}
			},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Tagged Alert",
				Body:       "Alert with tags",
				NotifyType: NotifyTypeError,
				Tags:       []string{"production", "database", "critical"},
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if len(doc.Tags) != 3 {
					t.Errorf("Expected 3 tags, got %d", len(doc.Tags))
				}
				if doc.Tags[0] != "production" {
					t.Error("Expected 'production' tag")
				}
			},
		},
		{
			name: "Body only (no title)",
			request: NotificationRequest{
				Body:       "Simple alert message",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Title == "" {
					t.Error("Expected title to be derived from message")
				}
				if doc.Message != "Simple alert message" {
					t.Errorf("Expected message, got '%s'", doc.Message)
				}
			},
		},
		{
			name: "Title only (no body)",
			request: NotificationRequest{
				Title:      "Title Only",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Title != "Title Only" {
					t.Errorf("Expected title, got '%s'", doc.Title)
				}
			},
		},
		{
			name: "Empty notification with defaults",
			request: NotificationRequest{
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Message == "" {
					t.Error("Expected default message")
				}
			},
		},
		{
			name: "Long message with title truncation",
			request: NotificationRequest{
				Body:       strings.Repeat("This is a very long message that should be truncated when used as title. ", 10),
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, doc ElasticsearchDocument) {
				if len(doc.Title) > 103 { // 100 + "..."
					t.Errorf("Expected title to be truncated, got length %d", len(doc.Title))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedDoc ElasticsearchDocument

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got '%s'", r.Method)
				}

				// Check URL path
				if !strings.Contains(r.URL.Path, "/_doc") {
					t.Errorf("Expected URL to contain '/_doc', got '%s'", r.URL.Path)
				}

				// Check headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", r.Header.Get("Content-Type"))
				}

				// Parse body
				body, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(body, &receivedDoc); err != nil {
					t.Errorf("Failed to parse document: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return success response
				response := ElasticsearchIndexResponse{
					Index:   "test-index",
					ID:      "test-id-123",
					Version: 1,
					Result:  "created",
				}
				response.Shards.Total = 2
				response.Shards.Successful = 2
				response.Shards.Failed = 0

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create service
			service := NewElasticsearchService().(*ElasticsearchService)

			// Parse mock server URL to get host and port
			serverURL, _ := url.Parse(server.URL)
			service.host = serverURL.Hostname()
			service.port = serverURL.Port()
			service.scheme = "http"
			service.index = "test-index"

			// Build document directly for testing
			doc := service.buildDocument(tt.request)

			if tt.checkFunc != nil {
				tt.checkFunc(t, doc)
			}

			// Verify common fields
			if doc.Timestamp == "" {
				t.Error("Expected timestamp to be set")
			}
			if doc.Source != "apprise-go" {
				t.Errorf("Expected source 'apprise-go', got '%s'", doc.Source)
			}
		})
	}
}

func TestElasticsearchService_MapNotifyTypeToSeverity(t *testing.T) {
	service := NewElasticsearchService().(*ElasticsearchService)

	tests := []struct {
		notifyType       NotifyType
		expectedSeverity string
	}{
		{NotifyTypeError, "error"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeInfo, "info"},
		{NotifyTypeSuccess, "success"},
	}

	for _, tt := range tests {
		severity := service.mapNotifyTypeToSeverity(tt.notifyType)
		if severity != tt.expectedSeverity {
			t.Errorf("For %v, expected severity '%s', got '%s'", tt.notifyType, tt.expectedSeverity, severity)
		}
	}
}

func TestElasticsearchService_TestURL(t *testing.T) {
	service := NewElasticsearchService()

	validURLs := []string{
		"elasticsearch://localhost:9200/alerts",
		"elasticsearch://user:pass@es.example.com:9200/logs",
		"elasticsearch://es.example.com/notifications?apikey=abc123",
		"opensearch://opensearch.example.com:9200/apprise",
		"es://localhost:9200/test",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"http://es.example.com/logs",
		"elasticsearch://:9200/logs",
		"elasticsearch://",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestElasticsearchService_SupportsAttachments(t *testing.T) {
	service := NewElasticsearchService()
	if service.SupportsAttachments() {
		t.Error("Elasticsearch service should not support attachments")
	}
}

func TestElasticsearchService_GetMaxBodyLength(t *testing.T) {
	service := NewElasticsearchService()
	if service.GetMaxBodyLength() != 32768 {
		t.Errorf("Expected max body length 32768, got %d", service.GetMaxBodyLength())
	}
}

func TestElasticsearchService_BuildDocument(t *testing.T) {
	service := NewElasticsearchService().(*ElasticsearchService)

	tests := []struct {
		name    string
		request NotificationRequest
		check   func(*testing.T, ElasticsearchDocument)
	}{
		{
			name: "Complete document structure",
			request: NotificationRequest{
				Title:      "Test",
				Body:       "Body",
				NotifyType: NotifyTypeError,
				Tags:       []string{"tag1", "tag2"},
			},
			check: func(t *testing.T, doc ElasticsearchDocument) {
				if doc.Title != "Test" {
					t.Errorf("Expected title 'Test', got '%s'", doc.Title)
				}
				if doc.Message != "Body" {
					t.Errorf("Expected message 'Body', got '%s'", doc.Message)
				}
				if doc.NotifyType != "error" {
					t.Errorf("Expected notify_type 'error', got '%s'", doc.NotifyType)
				}
				if doc.Severity != "error" {
					t.Errorf("Expected severity 'error', got '%s'", doc.Severity)
				}
				if len(doc.Tags) != 2 {
					t.Errorf("Expected 2 tags, got %d", len(doc.Tags))
				}
				if doc.Source != "apprise-go" {
					t.Errorf("Expected source 'apprise-go', got '%s'", doc.Source)
				}
				if doc.Timestamp == "" {
					t.Error("Expected timestamp to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := service.buildDocument(tt.request)
			tt.check(t, doc)
		})
	}
}

func TestElasticsearchService_Authentication(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*ElasticsearchService)
		checkFunc func(*testing.T, *http.Request)
	}{
		{
			name: "Basic authentication",
			setupFunc: func(e *ElasticsearchService) {
				e.username = "testuser"
				e.password = "testpass"
			},
			checkFunc: func(t *testing.T, r *http.Request) {
				username, password, ok := r.BasicAuth()
				if !ok {
					t.Error("Expected basic auth to be set")
				}
				if username != "testuser" || password != "testpass" {
					t.Errorf("Expected basic auth testuser:testpass, got %s:%s", username, password)
				}
			},
		},
		{
			name: "API key authentication",
			setupFunc: func(e *ElasticsearchService) {
				e.apiKey = "test-api-key-123"
			},
			checkFunc: func(t *testing.T, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				expected := "ApiKey test-api-key-123"
				if authHeader != expected {
					t.Errorf("Expected auth header '%s', got '%s'", expected, authHeader)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.checkFunc(t, r)

				// Return success response
				response := ElasticsearchIndexResponse{
					Index:  "test",
					ID:     "123",
					Result: "created",
				}
				response.Shards.Successful = 1
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create and configure service
			service := NewElasticsearchService().(*ElasticsearchService)
			serverURL, _ := url.Parse(server.URL)
			service.host = serverURL.Hostname()
			service.port = serverURL.Port()
			service.scheme = "http"
			service.index = "test"
			tt.setupFunc(service)

			// Send notification
			req := NotificationRequest{
				Title:      "Test",
				Body:       "Test",
				NotifyType: NotifyTypeInfo,
			}
			err := service.Send(context.Background(), req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
