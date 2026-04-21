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

func TestZulipService_GetServiceID(t *testing.T) {
	service := NewZulipService()
	if service.GetServiceID() != "zulip" {
		t.Errorf("Expected service ID 'zulip', got '%s'", service.GetServiceID())
	}
}

func TestZulipService_GetDefaultPort(t *testing.T) {
	service := NewZulipService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestZulipService_SupportsAttachments(t *testing.T) {
	service := NewZulipService()
	if service.SupportsAttachments() {
		t.Error("Zulip should not support attachments")
	}
}

func TestZulipService_GetMaxBodyLength(t *testing.T) {
	service := NewZulipService()
	if service.GetMaxBodyLength() != 10000 {
		t.Errorf("Expected max body length 10000, got %d", service.GetMaxBodyLength())
	}
}

func TestZulipService_ParseURL(t *testing.T) {
	tests := []struct {
		name           string
		inputURL       string
		expectError    bool
		expectedEmail  string
		expectedDomain string
		expectedKey    string
		expectedStream string
		expectedTopic  string
	}{
		{
			name:           "full URL",
			inputURL:       "zulip://botname@myorg.zulipchat.com/myapikey/general/notifications",
			expectError:    false,
			expectedEmail:  "botname@myorg.zulipchat.com",
			expectedDomain: "myorg.zulipchat.com",
			expectedKey:    "myapikey",
			expectedStream: "general",
			expectedTopic:  "notifications",
		},
		{
			name:           "defaults for stream and topic",
			inputURL:       "zulip://botname@myorg.zulipchat.com/myapikey",
			expectError:    false,
			expectedEmail:  "botname@myorg.zulipchat.com",
			expectedKey:    "myapikey",
			expectedStream: "general",
			expectedTopic:  "notifications",
		},
		{
			name:        "missing bot name",
			inputURL:    "zulip://myorg.zulipchat.com/myapikey/general/topic",
			expectError: true,
		},
		{
			name:        "missing domain",
			inputURL:    "zulip://botname@/myapikey/general/topic",
			expectError: true,
		},
		{
			name:        "missing API key",
			inputURL:    "zulip://botname@myorg.zulipchat.com/",
			expectError: true,
		},
		{
			name:        "invalid scheme",
			inputURL:    "https://botname@myorg.zulipchat.com/myapikey",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewZulipService().(*ZulipService)
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
			if tt.expectedEmail != "" && service.botEmail != tt.expectedEmail {
				t.Errorf("Expected botEmail '%s', got '%s'", tt.expectedEmail, service.botEmail)
			}
			if tt.expectedKey != "" && service.apiKey != tt.expectedKey {
				t.Errorf("Expected apiKey '%s', got '%s'", tt.expectedKey, service.apiKey)
			}
			if tt.expectedStream != "" && service.stream != tt.expectedStream {
				t.Errorf("Expected stream '%s', got '%s'", tt.expectedStream, service.stream)
			}
			if tt.expectedTopic != "" && service.topic != tt.expectedTopic {
				t.Errorf("Expected topic '%s', got '%s'", tt.expectedTopic, service.topic)
			}
		})
	}
}

func TestZulipService_TestURL(t *testing.T) {
	service := NewZulipService()

	valid := []string{
		"zulip://bot@myorg.zulipchat.com/apikey/general/topic",
		"zulip://bot@myorg.zulipchat.com/apikey",
	}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://bot@domain/key", "zulip://domain/key"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestZulipService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Check Basic auth
		username, _, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected Basic auth")
		}
		if !strings.Contains(username, "@") {
			t.Errorf("Expected bot email as username, got '%s'", username)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		if r.FormValue("type") != "stream" {
			t.Errorf("Expected type=stream, got '%s'", r.FormValue("type"))
		}
		if r.FormValue("to") != "general" {
			t.Errorf("Expected to=general, got '%s'", r.FormValue("to"))
		}
		if r.FormValue("topic") != "alerts" {
			t.Errorf("Expected topic=alerts, got '%s'", r.FormValue("topic"))
		}
		content := r.FormValue("content")
		if !strings.Contains(content, "Test Title") || !strings.Contains(content, "Test body") {
			t.Errorf("Expected content to contain title and body, got '%s'", content)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success","msg":"","id":12345}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)

	service := NewZulipService().(*ZulipService)
	service.domain = serverURL.Host
	service.botEmail = "bot@" + serverURL.Host
	service.apiKey = "myapikey"
	service.stream = "general"
	service.topic = "alerts"
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

func TestZulipService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewZulipService().(*ZulipService)
	service.domain = serverURL.Host
	service.botEmail = "bot@" + serverURL.Host
	service.apiKey = "badkey"
	service.stream = "general"
	service.topic = "alerts"
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
	if !strings.Contains(err.Error(), "zulip API error") {
		t.Errorf("Expected zulip API error, got: %v", err)
	}
}
