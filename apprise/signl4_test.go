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

func TestSIGNL4Service_GetServiceID(t *testing.T) {
	service := NewSIGNL4Service()
	if service.GetServiceID() != "signl4" {
		t.Errorf("Expected service ID 'signl4', got '%s'", service.GetServiceID())
	}
}

func TestSIGNL4Service_GetDefaultPort(t *testing.T) {
	service := NewSIGNL4Service()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestSIGNL4Service_SupportsAttachments(t *testing.T) {
	service := NewSIGNL4Service()
	if service.SupportsAttachments() {
		t.Error("SIGNL4 should not support attachments")
	}
}

func TestSIGNL4Service_ParseURL(t *testing.T) {
	tests := []struct {
		name           string
		inputURL       string
		expectError    bool
		expectedSecret string
	}{
		{
			name:           "valid team secret",
			inputURL:       "signl4://myteamsecret",
			expectError:    false,
			expectedSecret: "myteamsecret",
		},
		{
			name:        "invalid scheme",
			inputURL:    "http://myteamsecret",
			expectError: true,
		},
		{
			name:        "missing team secret",
			inputURL:    "signl4://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSIGNL4Service().(*SIGNL4Service)
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
			if service.teamSecret != tt.expectedSecret {
				t.Errorf("Expected team secret '%s', got '%s'", tt.expectedSecret, service.teamSecret)
			}
		})
	}
}

func TestSIGNL4Service_TestURL(t *testing.T) {
	service := NewSIGNL4Service()

	if err := service.TestURL("signl4://myteamsecret"); err != nil {
		t.Errorf("Valid URL should not error: %v", err)
	}
	if err := service.TestURL("http://myteamsecret"); err == nil {
		t.Error("Invalid URL should error")
	}
}

func TestSIGNL4Service_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		// URL should end with the team secret
		if !strings.HasSuffix(r.URL.Path, "/myteamsecret") {
			t.Errorf("Expected path ending with /myteamsecret, got %s", r.URL.Path)
		}

		var payload SIGNL4Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if payload.Title != "Alert" {
			t.Errorf("Expected title 'Alert', got '%s'", payload.Title)
		}
		if payload.Body != "Something happened" {
			t.Errorf("Expected body 'Something happened', got '%s'", payload.Body)
		}
		if payload.S4Status != "new" {
			t.Errorf("Expected X-S4-Status 'new', got '%s'", payload.S4Status)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"eventId": "abc123"})
	}))
	defer server.Close()

	service := NewSIGNL4Service().(*SIGNL4Service)
	service.teamSecret = "myteamsecret"
	service.client = &http.Client{
		Transport: &redirectTransport{targetURL: server.URL},
	}

	req := NotificationRequest{
		Title:      "Alert",
		Body:       "Something happened",
		NotifyType: NotifyTypeWarning,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Send(ctx, req); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestSIGNL4Service_SendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	service := NewSIGNL4Service().(*SIGNL4Service)
	service.teamSecret = "badsecret"
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
	if !strings.Contains(err.Error(), "signl4 API error") {
		t.Errorf("Expected signl4 API error, got: %v", err)
	}
}
