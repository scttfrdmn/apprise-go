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

func TestSentryService_GetServiceID(t *testing.T) {
	service := NewSentryService()
	if service.GetServiceID() != "sentry" {
		t.Errorf("Expected service ID 'sentry', got '%s'", service.GetServiceID())
	}
}

func TestSentryService_GetDefaultPort(t *testing.T) {
	tests := []struct {
		name         string
		protocol     string
		expectedPort int
	}{
		{"HTTPS", "https", 443},
		{"HTTP", "http", 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSentryService().(*SentryService)
			service.protocol = tt.protocol
			port := service.GetDefaultPort()
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestSentryService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *SentryService)
	}{
		{
			name:        "Basic Sentry DSN",
			url:         "sentry://abc123def456@o123456.ingest.sentry.io/789012",
			expectError: false,
			checkFunc: func(t *testing.T, s *SentryService) {
				if s.protocol != "https" {
					t.Errorf("Expected protocol 'https', got '%s'", s.protocol)
				}
				if s.publicKey != "abc123def456" {
					t.Errorf("Expected public key 'abc123def456', got '%s'", s.publicKey)
				}
				if s.host != "o123456.ingest.sentry.io" {
					t.Errorf("Expected host 'o123456.ingest.sentry.io', got '%s'", s.host)
				}
				if s.projectID != "789012" {
					t.Errorf("Expected project ID '789012', got '%s'", s.projectID)
				}
			},
		},
		{
			name:        "HTTPS DSN format",
			url:         "https://public_key@sentry.example.com/1",
			expectError: false,
			checkFunc: func(t *testing.T, s *SentryService) {
				if s.protocol != "https" {
					t.Errorf("Expected protocol 'https', got '%s'", s.protocol)
				}
				if s.publicKey != "public_key" {
					t.Errorf("Expected public key 'public_key', got '%s'", s.publicKey)
				}
				if s.host != "sentry.example.com" {
					t.Errorf("Expected host 'sentry.example.com', got '%s'", s.host)
				}
				if s.projectID != "1" {
					t.Errorf("Expected project ID '1', got '%s'", s.projectID)
				}
			},
		},
		{
			name:        "Self-hosted with port",
			url:         "sentry://key123@sentry.internal.com:8080/project-42",
			expectError: false,
			checkFunc: func(t *testing.T, s *SentryService) {
				if s.host != "sentry.internal.com:8080" {
					t.Errorf("Expected host with port, got '%s'", s.host)
				}
				if s.projectID != "project-42" {
					t.Errorf("Expected project ID 'project-42', got '%s'", s.projectID)
				}
			},
		},
		{
			name:        "HTTP scheme (self-hosted)",
			url:         "http://mykey@localhost:9000/123",
			expectError: false,
			checkFunc: func(t *testing.T, s *SentryService) {
				if s.protocol != "http" {
					t.Errorf("Expected protocol 'http', got '%s'", s.protocol)
				}
			},
		},
		{
			name:        "Sentries scheme (plural)",
			url:         "sentries://key@host.com/999",
			expectError: false,
			checkFunc: func(t *testing.T, s *SentryService) {
				if s.protocol != "https" {
					t.Errorf("Expected protocol 'https' for sentries scheme, got '%s'", s.protocol)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "ftp://key@host.com/123",
			expectError: true,
		},
		{
			name:        "Missing public key",
			url:         "sentry://host.com/123",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "sentry://key@/123",
			expectError: true,
		},
		{
			name:        "Missing project ID",
			url:         "sentry://key@host.com/",
			expectError: true,
		},
		{
			name:        "Empty project ID",
			url:         "sentry://key@host.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSentryService().(*SentryService)
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

func TestSentryService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		checkFunc   func(*testing.T, []byte)
		expectError bool
	}{
		{
			name: "Info notification",
			request: NotificationRequest{
				Title:      "Database Connection",
				Body:       "Successfully connected to PostgreSQL",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, body []byte) {
				// Parse envelope
				lines := strings.Split(string(body), "\n")
				if len(lines) < 4 {
					t.Errorf("Expected at least 4 lines in envelope, got %d", len(lines))
					return
				}

				// Parse event (third line)
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Level != "info" {
					t.Errorf("Expected level 'info', got '%s'", event.Level)
				}
				if event.Platform != "go" {
					t.Errorf("Expected platform 'go', got '%s'", event.Platform)
				}
				if event.Logger != "apprise-go" {
					t.Errorf("Expected logger 'apprise-go', got '%s'", event.Logger)
				}
				if event.Message == nil {
					t.Error("Expected message to be present")
				} else {
					expected := "Database Connection: Successfully connected to PostgreSQL"
					if event.Message.Formatted != expected {
						t.Errorf("Expected message '%s', got '%s'", expected, event.Message.Formatted)
					}
				}
			},
		},
		{
			name: "Error notification",
			request: NotificationRequest{
				Title:      "API Error",
				Body:       "Failed to connect to external service",
				NotifyType: NotifyTypeError,
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Level != "error" {
					t.Errorf("Expected level 'error', got '%s'", event.Level)
				}
			},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "High Memory Usage",
				Body:       "Memory usage at 85%",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Level != "warning" {
					t.Errorf("Expected level 'warning', got '%s'", event.Level)
				}
			},
		},
		{
			name: "Success notification (mapped to info)",
			request: NotificationRequest{
				Title:      "Deployment Complete",
				Body:       "Version 2.0 deployed",
				NotifyType: NotifyTypeSuccess,
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Level != "info" {
					t.Errorf("Expected level 'info' for success, got '%s'", event.Level)
				}
			},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Error",
				Body:       "Something went wrong",
				NotifyType: NotifyTypeError,
				Tags:       []string{"production", "api", "critical"},
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Tags == nil {
					t.Error("Expected tags to be present")
					return
				}

				if len(event.Tags) != 3 {
					t.Errorf("Expected 3 tags, got %d", len(event.Tags))
				}
			},
		},
		{
			name: "Title only",
			request: NotificationRequest{
				Title:      "Quick Alert",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Message.Formatted != "Quick Alert" {
					t.Errorf("Expected message 'Quick Alert', got '%s'", event.Message.Formatted)
				}
			},
		},
		{
			name: "Body only",
			request: NotificationRequest{
				Body:       "Simple message",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, body []byte) {
				lines := strings.Split(string(body), "\n")
				var event SentryEvent
				if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
					t.Errorf("Failed to parse event: %v", err)
					return
				}

				if event.Message.Formatted != "Simple message" {
					t.Errorf("Expected message 'Simple message', got '%s'", event.Message.Formatted)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			var receivedBody []byte
			var receivedHeaders http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedHeaders = r.Header.Clone()
				body, _ := io.ReadAll(r.Body)
				receivedBody = body

				// Check auth header
				authHeader := r.Header.Get("X-Sentry-Auth")
				if !strings.HasPrefix(authHeader, "Sentry sentry_version=7") {
					t.Errorf("Invalid auth header: %s", authHeader)
				}

				// Check content type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/x-sentry-envelope" {
					t.Errorf("Expected content type 'application/x-sentry-envelope', got '%s'", contentType)
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Parse server URL to get host
			serverURL, _ := url.Parse(server.URL)

			// Create and configure service
			service := NewSentryService().(*SentryService)
			service.protocol = "http"
			service.publicKey = "test_key"
			service.host = serverURL.Host
			service.projectID = "123"

			// Send notification
			err := service.Send(context.Background(), tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkFunc != nil {
				tt.checkFunc(t, receivedBody)
			}

			_ = receivedHeaders // Suppress unused warning
		})
	}
}

func TestSentryService_EnvelopeFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)

		// Check envelope structure
		lines := strings.Split(bodyStr, "\n")
		if len(lines) < 4 {
			t.Errorf("Expected at least 4 lines in envelope, got %d", len(lines))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate envelope header (line 0)
		var envelopeHeader SentryEnvelopeHeader
		if err := json.Unmarshal([]byte(lines[0]), &envelopeHeader); err != nil {
			t.Errorf("Failed to parse envelope header: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if envelopeHeader.EventID == "" {
			t.Error("Envelope header missing event_id")
		}

		// Validate item header (line 1)
		var itemHeader SentryItemHeader
		if err := json.Unmarshal([]byte(lines[1]), &itemHeader); err != nil {
			t.Errorf("Failed to parse item header: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if itemHeader.Type != "event" {
			t.Errorf("Expected item type 'event', got '%s'", itemHeader.Type)
		}

		// Validate event payload (line 2)
		var event SentryEvent
		if err := json.Unmarshal([]byte(lines[2]), &event); err != nil {
			t.Errorf("Failed to parse event payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if event.EventID == "" {
			t.Error("Event missing event_id")
		}
		if event.Timestamp == "" {
			t.Error("Event missing timestamp")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	service := NewSentryService().(*SentryService)
	service.protocol = "http"
	service.publicKey = "test_key"
	service.host = serverURL.Host
	service.projectID = "123"

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Envelope format test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Failed to send: %v", err)
	}
}

func TestSentryService_TestURL(t *testing.T) {
	service := NewSentryService()

	validURLs := []string{
		"sentry://key@o123456.ingest.sentry.io/789012",
		"https://public_key@sentry.example.com/1",
		"sentry://key@localhost:8080/project",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"sentry://host.com/123",        // Missing public key
		"sentry://@host.com/123",       // Empty public key
		"sentry://key@/123",            // Missing host
		"sentry://key@host.com/",       // Missing project ID
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestSentryService_SupportsAttachments(t *testing.T) {
	service := NewSentryService()
	if service.SupportsAttachments() {
		t.Error("Sentry service should not support attachments in basic notification mode")
	}
}

func TestSentryService_GetMaxBodyLength(t *testing.T) {
	service := NewSentryService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (no limit), got %d", service.GetMaxBodyLength())
	}
}

func TestSentryService_MapNotifyTypeToLevel(t *testing.T) {
	service := NewSentryService().(*SentryService)

	tests := []struct {
		notifyType    NotifyType
		expectedLevel string
	}{
		{NotifyTypeInfo, "info"},
		{NotifyTypeSuccess, "info"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "error"},
	}

	for _, tt := range tests {
		level := service.mapNotifyTypeToLevel(tt.notifyType)
		if level != tt.expectedLevel {
			t.Errorf("For %v, expected level '%s', got '%s'", tt.notifyType, tt.expectedLevel, level)
		}
	}
}

func TestGenerateEventID(t *testing.T) {
	// Generate multiple event IDs
	id1, err := generateEventID()
	if err != nil {
		t.Fatalf("Failed to generate event ID: %v", err)
	}

	id2, err := generateEventID()
	if err != nil {
		t.Fatalf("Failed to generate event ID: %v", err)
	}

	// Check format (32 hex characters)
	if len(id1) != 32 {
		t.Errorf("Expected event ID length 32, got %d", len(id1))
	}

	// Check uniqueness
	if id1 == id2 {
		t.Error("Event IDs should be unique")
	}

	// Check hex format
	for _, c := range id1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Event ID contains invalid hex character: %c", c)
		}
	}
}

func TestSentryService_BuildAuthHeader(t *testing.T) {
	service := NewSentryService().(*SentryService)
	service.publicKey = "test_public_key"

	authHeader := service.buildAuthHeader("event123")

	// Check format
	if !strings.HasPrefix(authHeader, "Sentry sentry_version=7") {
		t.Errorf("Auth header should start with 'Sentry sentry_version=7', got: %s", authHeader)
	}

	// Check for required fields
	if !strings.Contains(authHeader, "sentry_key=test_public_key") {
		t.Error("Auth header should contain sentry_key")
	}
	if !strings.Contains(authHeader, "sentry_timestamp=") {
		t.Error("Auth header should contain sentry_timestamp")
	}
	if !strings.Contains(authHeader, "sentry_client=apprise-go") {
		t.Error("Auth header should contain sentry_client")
	}
}

func TestSentryService_BuildMessage(t *testing.T) {
	service := NewSentryService().(*SentryService)

	tests := []struct {
		name     string
		request  NotificationRequest
		expected string
	}{
		{
			name: "Title and body",
			request: NotificationRequest{
				Title: "Error Occurred",
				Body:  "Database connection failed",
			},
			expected: "Error Occurred: Database connection failed",
		},
		{
			name: "Title only",
			request: NotificationRequest{
				Title: "Alert",
			},
			expected: "Alert",
		},
		{
			name: "Body only",
			request: NotificationRequest{
				Body: "Something happened",
			},
			expected: "Something happened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildMessage(tt.request)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
