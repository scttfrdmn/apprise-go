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

func TestSynologyService_GetServiceID(t *testing.T) {
	service := NewSynologyService()
	if service.GetServiceID() != "synology" {
		t.Errorf("Expected service ID 'synology', got '%s'", service.GetServiceID())
	}
}

func TestSynologyService_GetDefaultPort(t *testing.T) {
	service := NewSynologyService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestSynologyService_SupportsAttachments(t *testing.T) {
	service := NewSynologyService()
	if service.SupportsAttachments() {
		t.Error("Synology Chat should not support attachments")
	}
}

func TestSynologyService_ParseURL(t *testing.T) {
	tests := []struct {
		name          string
		inputURL      string
		expectError   bool
		expectedHost  string
		expectedToken string
		expectedPort  int
		expectedSecure bool
	}{
		{
			name:          "basic HTTP URL",
			inputURL:      "synology://nas.local/mytoken",
			expectError:   false,
			expectedHost:  "nas.local",
			expectedToken: "mytoken",
			expectedSecure: false,
		},
		{
			name:           "HTTPS URL",
			inputURL:       "synologys://nas.local/mytoken",
			expectError:    false,
			expectedHost:   "nas.local",
			expectedToken:  "mytoken",
			expectedSecure: true,
		},
		{
			name:          "with custom port",
			inputURL:      "synology://nas.local:5001/mytoken",
			expectError:   false,
			expectedHost:  "nas.local",
			expectedToken: "mytoken",
			expectedPort:  5001,
		},
		{
			name:        "invalid scheme",
			inputURL:    "http://nas.local/mytoken",
			expectError: true,
		},
		{
			name:        "missing hostname",
			inputURL:    "synology:///mytoken",
			expectError: true,
		},
		{
			name:        "missing token",
			inputURL:    "synology://nas.local/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSynologyService().(*SynologyService)
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
			if service.hostname != tt.expectedHost {
				t.Errorf("Expected hostname '%s', got '%s'", tt.expectedHost, service.hostname)
			}
			if service.token != tt.expectedToken {
				t.Errorf("Expected token '%s', got '%s'", tt.expectedToken, service.token)
			}
			if tt.expectedPort > 0 && service.port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, service.port)
			}
			if service.secure != tt.expectedSecure {
				t.Errorf("Expected secure=%v, got %v", tt.expectedSecure, service.secure)
			}
		})
	}
}

func TestSynologyService_TestURL(t *testing.T) {
	service := NewSynologyService()

	valid := []string{
		"synology://nas.local/mytoken",
		"synologys://nas.local/mytoken",
		"synology://nas.local:5000/mytoken",
	}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://nas.local/token", "synology:///token"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestSynologyService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify query parameters
		query := r.URL.Query()
		if query.Get("api") != "SYNO.Chat.External" {
			t.Errorf("Expected api=SYNO.Chat.External, got %s", query.Get("api"))
		}
		if query.Get("method") != "incoming" {
			t.Errorf("Expected method=incoming, got %s", query.Get("method"))
		}
		if query.Get("token") != "mytoken" {
			t.Errorf("Expected token=mytoken, got %s", query.Get("token"))
		}

		// Verify payload form field
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}
		payloadStr := r.FormValue("payload")
		if !strings.Contains(payloadStr, "Test body") {
			t.Errorf("Expected payload to contain 'Test body', got '%s'", payloadStr)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewSynologyService().(*SynologyService)
	service.hostname = serverURL.Hostname()
	service.port, _ = func() (int, error) {
		portStr := serverURL.Port()
		if portStr == "" {
			return -1, nil
		}
		var p int
		_, err := url.Parse("http://host:" + portStr)
		if err == nil {
			p = 0
			for _, c := range portStr {
				p = p*10 + int(c-'0')
			}
		}
		return p, err
	}()
	service.token = "mytoken"

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

func TestSynologyService_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewSynologyService().(*SynologyService)
	service.hostname = serverURL.Hostname()
	port := 0
	for _, c := range serverURL.Port() {
		port = port*10 + int(c-'0')
	}
	service.port = port
	service.token = "badtoken"

	req := NotificationRequest{Body: "test", NotifyType: NotifyTypeInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for non-2xx response")
	}
	if !strings.Contains(err.Error(), "synology chat API error") {
		t.Errorf("Expected synology chat API error, got: %v", err)
	}
}
