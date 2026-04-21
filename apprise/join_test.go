package apprise

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestJoinService_GetServiceID(t *testing.T) {
	service := NewJoinService()
	if service.GetServiceID() != "join" {
		t.Errorf("Expected service ID 'join', got '%s'", service.GetServiceID())
	}
}

func TestJoinService_GetDefaultPort(t *testing.T) {
	service := NewJoinService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestJoinService_SupportsAttachments(t *testing.T) {
	service := NewJoinService()
	if service.SupportsAttachments() {
		t.Error("Join should not support attachments")
	}
}

func TestJoinService_ParseURL(t *testing.T) {
	tests := []struct {
		name            string
		inputURL        string
		expectError     bool
		expectedKey     string
		expectedDevices []string
	}{
		{
			name:            "single device",
			inputURL:        "join://myapikey/device1",
			expectError:     false,
			expectedKey:     "myapikey",
			expectedDevices: []string{"device1"},
		},
		{
			name:            "multiple devices",
			inputURL:        "join://myapikey/device1/device2/device3",
			expectError:     false,
			expectedKey:     "myapikey",
			expectedDevices: []string{"device1", "device2", "device3"},
		},
		{
			name:        "invalid scheme",
			inputURL:    "http://myapikey/device1",
			expectError: true,
		},
		{
			name:        "missing API key",
			inputURL:    "join:///device1",
			expectError: true,
		},
		{
			name:        "missing device ID",
			inputURL:    "join://myapikey/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewJoinService().(*JoinService)
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
			if len(service.deviceIDs) != len(tt.expectedDevices) {
				t.Errorf("Expected %d devices, got %d", len(tt.expectedDevices), len(service.deviceIDs))
			} else {
				for i, d := range tt.expectedDevices {
					if service.deviceIDs[i] != d {
						t.Errorf("Expected device[%d]='%s', got '%s'", i, d, service.deviceIDs[i])
					}
				}
			}
		})
	}
}

func TestJoinService_TestURL(t *testing.T) {
	service := NewJoinService()

	valid := []string{"join://apikey/device1", "join://apikey/device1/device2"}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://apikey/device1", "join:///device1", "join://apikey/"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestJoinService_Send(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		query := r.URL.Query()
		if query.Get("apikey") != "myapikey" {
			t.Errorf("Expected apikey=myapikey, got %s", query.Get("apikey"))
		}
		if query.Get("title") != "Test Title" {
			t.Errorf("Expected title='Test Title', got '%s'", query.Get("title"))
		}
		if query.Get("text") != "Test body" {
			t.Errorf("Expected text='Test body', got '%s'", query.Get("text"))
		}

		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewJoinService().(*JoinService)
	service.apiKey = "myapikey"
	service.deviceIDs = []string{"device1", "device2"}
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

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (one per device), got %d", requestCount)
	}
}

func TestJoinService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	service := NewJoinService().(*JoinService)
	service.apiKey = "badkey"
	service.deviceIDs = []string{"device1"}
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
	if !strings.Contains(err.Error(), "join API error") {
		t.Errorf("Expected join API error, got: %v", err)
	}
}
