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

func TestRevoltService_GetServiceID(t *testing.T) {
	service := NewRevoltService()
	if service.GetServiceID() != "revolt" {
		t.Errorf("Expected service ID 'revolt', got '%s'", service.GetServiceID())
	}
}

func TestRevoltService_GetDefaultPort(t *testing.T) {
	service := NewRevoltService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestRevoltService_SupportsAttachments(t *testing.T) {
	service := NewRevoltService()
	if service.SupportsAttachments() {
		t.Error("Revolt should not support attachments")
	}
}

func TestRevoltService_GetMaxBodyLength(t *testing.T) {
	service := NewRevoltService()
	if service.GetMaxBodyLength() != 2000 {
		t.Errorf("Expected max body length 2000, got %d", service.GetMaxBodyLength())
	}
}

func TestRevoltService_ParseURL(t *testing.T) {
	tests := []struct {
		name          string
		inputURL      string
		expectError   bool
		expectedID    string
		expectedToken string
	}{
		{
			name:          "valid webhook URL",
			inputURL:      "revolt://webhookid/webhooktoken",
			expectError:   false,
			expectedID:    "webhookid",
			expectedToken: "webhooktoken",
		},
		{
			name:        "invalid scheme",
			inputURL:    "https://webhookid/webhooktoken",
			expectError: true,
		},
		{
			name:        "missing webhook ID",
			inputURL:    "revolt:///webhooktoken",
			expectError: true,
		},
		{
			name:        "missing webhook token",
			inputURL:    "revolt://webhookid/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRevoltService().(*RevoltService)
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
			if service.webhookID != tt.expectedID {
				t.Errorf("Expected webhook ID '%s', got '%s'", tt.expectedID, service.webhookID)
			}
			if service.webhookToken != tt.expectedToken {
				t.Errorf("Expected webhook token '%s', got '%s'", tt.expectedToken, service.webhookToken)
			}
		})
	}
}

func TestRevoltService_TestURL(t *testing.T) {
	service := NewRevoltService()

	if err := service.TestURL("revolt://myid/mytoken"); err != nil {
		t.Errorf("Valid URL should not error: %v", err)
	}
	if err := service.TestURL("http://myid/mytoken"); err == nil {
		t.Error("Invalid URL should error")
	}
}

func TestRevoltService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		// Path should include webhook ID and token
		if !strings.Contains(r.URL.Path, "webhookid") {
			t.Errorf("Expected path to contain webhook ID, got %s", r.URL.Path)
		}

		var payload RevoltPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if !strings.Contains(payload.Content, "Test Title") {
			t.Errorf("Expected content to contain 'Test Title', got '%s'", payload.Content)
		}
		if !strings.Contains(payload.Content, "Test body") {
			t.Errorf("Expected content to contain 'Test body', got '%s'", payload.Content)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewRevoltService().(*RevoltService)
	service.webhookID = "webhookid"
	service.webhookToken = "webhooktoken"
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

func TestRevoltService_SendNoTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload RevoltPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if payload.Content != "Body only" {
			t.Errorf("Expected content 'Body only', got '%s'", payload.Content)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := NewRevoltService().(*RevoltService)
	service.webhookID = "webhookid"
	service.webhookToken = "webhooktoken"
	service.client = &http.Client{
		Transport: &redirectTransport{targetURL: server.URL},
	}

	req := NotificationRequest{Body: "Body only", NotifyType: NotifyTypeInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Send(ctx, req); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestRevoltService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	service := NewRevoltService().(*RevoltService)
	service.webhookID = "badid"
	service.webhookToken = "badtoken"
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
	if !strings.Contains(err.Error(), "revolt API error") {
		t.Errorf("Expected revolt API error, got: %v", err)
	}
}
