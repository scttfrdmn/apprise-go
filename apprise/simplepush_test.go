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

func TestSimplePushService_GetServiceID(t *testing.T) {
	service := NewSimplePushService()
	if service.GetServiceID() != "spush" {
		t.Errorf("Expected service ID 'spush', got '%s'", service.GetServiceID())
	}
}

func TestSimplePushService_GetDefaultPort(t *testing.T) {
	service := NewSimplePushService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestSimplePushService_SupportsAttachments(t *testing.T) {
	service := NewSimplePushService()
	if service.SupportsAttachments() {
		t.Error("SimplePush should not support attachments")
	}
}

func TestSimplePushService_GetMaxBodyLength(t *testing.T) {
	service := NewSimplePushService()
	if service.GetMaxBodyLength() != 10000 {
		t.Errorf("Expected max body length 10000, got %d", service.GetMaxBodyLength())
	}
}

func TestSimplePushService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		expectError bool
		expectedKey string
	}{
		{
			name:        "spush scheme",
			inputURL:    "spush://myapikey123",
			expectError: false,
			expectedKey: "myapikey123",
		},
		{
			name:        "simplepush scheme",
			inputURL:    "simplepush://myapikey123",
			expectError: false,
			expectedKey: "myapikey123",
		},
		{
			name:        "invalid scheme",
			inputURL:    "http://myapikey123",
			expectError: true,
		},
		{
			name:        "missing API key",
			inputURL:    "spush://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSimplePushService().(*SimplePushService)
			parsedURL, err := url.Parse(tt.inputURL)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse URL: %v", err)
				}
				return
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if service.apiKey != tt.expectedKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedKey, service.apiKey)
			}
		})
	}
}

func TestSimplePushService_TestURL(t *testing.T) {
	service := NewSimplePushService()

	valid := []string{"spush://mykey", "simplepush://mykey"}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://mykey", "spush://"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestSimplePushService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var payload SimplePushPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if payload.Key != "testkey" {
			t.Errorf("Expected key 'testkey', got '%s'", payload.Key)
		}
		if payload.Title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got '%s'", payload.Title)
		}
		if payload.Msg != "Test body" {
			t.Errorf("Expected msg 'Test body', got '%s'", payload.Msg)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewSimplePushService().(*SimplePushService)
	service.apiKey = "testkey"

	// Override the API URL for testing by pointing client to test server
	origURL := simplePushAPIURL
	_ = origURL // reference to avoid unused warning; we test the full flow with a fresh client
	service.client = server.Client()

	// We need to intercept the URL. Instead, let's verify the payload construction
	// by manually calling through with the test server URL substituted.
	// Use a custom RoundTripper to redirect requests.
	service.client = &http.Client{
		Transport: &redirectTransport{targetURL: server.URL},
	}

	req := NotificationRequest{
		Title:      "Test Title",
		Body:       "Test body",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Send(ctx, req); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestSimplePushService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	service := NewSimplePushService().(*SimplePushService)
	service.apiKey = "badkey"
	service.client = &http.Client{
		Transport: &redirectTransport{targetURL: server.URL},
	}

	req := NotificationRequest{Body: "test", NotifyType: NotifyTypeInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for non-2xx response")
	}
	if !strings.Contains(err.Error(), "simplepush API error") {
		t.Errorf("Expected simplepush API error, got: %v", err)
	}
}

// redirectTransport redirects all HTTP requests to a given base URL (for testing)
type redirectTransport struct {
	targetURL string
	base      http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, _ := url.Parse(rt.targetURL)
	req = req.Clone(req.Context())
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	if rt.base != nil {
		return rt.base.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
