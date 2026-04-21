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

func TestWebexService_GetServiceID(t *testing.T) {
	service := NewWebexService()
	if service.GetServiceID() != "webex" {
		t.Errorf("Expected service ID 'webex', got '%s'", service.GetServiceID())
	}
}

func TestWebexService_GetDefaultPort(t *testing.T) {
	service := NewWebexService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestWebexService_SupportsAttachments(t *testing.T) {
	service := NewWebexService()
	if service.SupportsAttachments() {
		t.Error("Webex should not support attachments")
	}
}

func TestWebexService_GetMaxBodyLength(t *testing.T) {
	service := NewWebexService()
	if service.GetMaxBodyLength() != 7439 {
		t.Errorf("Expected max body length 7439, got %d", service.GetMaxBodyLength())
	}
}

func TestWebexService_ParseURL(t *testing.T) {
	tests := []struct {
		name           string
		inputURL       string
		expectError    bool
		expectedToken  string
		expectedRoomID string
	}{
		{
			name:           "webex scheme",
			inputURL:       "webex://mytoken@myroomid",
			expectError:    false,
			expectedToken:  "mytoken",
			expectedRoomID: "myroomid",
		},
		{
			name:           "wxteams scheme",
			inputURL:       "wxteams://mytoken@myroomid",
			expectError:    false,
			expectedToken:  "mytoken",
			expectedRoomID: "myroomid",
		},
		{
			name:        "invalid scheme",
			inputURL:    "https://mytoken@myroomid",
			expectError: true,
		},
		{
			name:        "missing token",
			inputURL:    "webex://myroomid",
			expectError: true,
		},
		{
			name:        "missing room ID",
			inputURL:    "webex://mytoken@",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewWebexService().(*WebexService)
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
			if service.token != tt.expectedToken {
				t.Errorf("Expected token '%s', got '%s'", tt.expectedToken, service.token)
			}
			if service.roomID != tt.expectedRoomID {
				t.Errorf("Expected roomID '%s', got '%s'", tt.expectedRoomID, service.roomID)
			}
		})
	}
}

func TestWebexService_TestURL(t *testing.T) {
	service := NewWebexService()

	valid := []string{
		"webex://mytoken@myroomid",
		"wxteams://mytoken@myroomid",
	}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://token@roomid", "webex://roomid"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestWebexService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Bearer auth header, got '%s'", authHeader)
		}
		if !strings.Contains(authHeader, "mytoken") {
			t.Errorf("Expected token in auth header, got '%s'", authHeader)
		}

		var payload WebexPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if payload.RoomID != "myroomid" {
			t.Errorf("Expected roomId 'myroomid', got '%s'", payload.RoomID)
		}
		if !strings.Contains(payload.Text, "Test Title") || !strings.Contains(payload.Text, "Test body") {
			t.Errorf("Expected text to contain title and body, got '%s'", payload.Text)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "msg123", "roomId": "myroomid"})
	}))
	defer server.Close()

	service := NewWebexService().(*WebexService)
	service.token = "mytoken"
	service.roomID = "myroomid"
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

func TestWebexService_SendMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebexPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if payload.Text != "" {
			t.Errorf("Expected no plain text when using markdown, got '%s'", payload.Text)
		}
		if !strings.Contains(payload.Markdown, "**Test Title**") {
			t.Errorf("Expected markdown with bold title, got '%s'", payload.Markdown)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "msg123"})
	}))
	defer server.Close()

	service := NewWebexService().(*WebexService)
	service.token = "mytoken"
	service.roomID = "myroomid"
	service.client = &http.Client{
		Transport: &redirectTransport{targetURL: server.URL},
	}

	req := NotificationRequest{
		Title:      "Test Title",
		Body:       "Test body",
		NotifyType: NotifyTypeInfo,
		BodyFormat: "markdown",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Send(ctx, req); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestWebexService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	service := NewWebexService().(*WebexService)
	service.token = "badtoken"
	service.roomID = "myroomid"
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
	if !strings.Contains(err.Error(), "webex API error") {
		t.Errorf("Expected webex API error, got: %v", err)
	}
}
