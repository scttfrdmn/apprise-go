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

func TestKodiService_GetServiceID(t *testing.T) {
	service := NewKodiService()
	if service.GetServiceID() != "kodi" {
		t.Errorf("Expected service ID 'kodi', got '%s'", service.GetServiceID())
	}
}

func TestKodiService_GetDefaultPort(t *testing.T) {
	tests := []struct {
		secure   bool
		expected int
	}{
		{false, 8080},
		{true, 443},
	}
	for _, tt := range tests {
		service := &KodiService{secure: tt.secure}
		if service.GetDefaultPort() != tt.expected {
			t.Errorf("secure=%v: expected port %d, got %d", tt.secure, tt.expected, service.GetDefaultPort())
		}
	}
}

func TestKodiService_SupportsAttachments(t *testing.T) {
	service := NewKodiService()
	if service.SupportsAttachments() {
		t.Error("Kodi should not support attachments")
	}
}

func TestKodiService_GetMaxBodyLength(t *testing.T) {
	service := NewKodiService()
	if service.GetMaxBodyLength() != 250 {
		t.Errorf("Expected max body length 250, got %d", service.GetMaxBodyLength())
	}
}

func TestKodiService_ParseURL(t *testing.T) {
	tests := []struct {
		name           string
		inputURL       string
		expectError    bool
		expectedHost   string
		expectedUser   string
		expectedPass   string
		expectedPort   int
		expectedSecure bool
	}{
		{
			name:         "basic kodi URL no auth",
			inputURL:     "kodi://mykodi.local/",
			expectError:  false,
			expectedHost: "mykodi.local",
		},
		{
			name:         "kodi URL with credentials",
			inputURL:     "kodi://admin:password@mykodi.local/",
			expectError:  false,
			expectedHost: "mykodi.local",
			expectedUser: "admin",
			expectedPass: "password",
		},
		{
			name:           "kodis for HTTPS",
			inputURL:       "kodis://admin:pass@mykodi.local/",
			expectError:    false,
			expectedHost:   "mykodi.local",
			expectedSecure: true,
		},
		{
			name:         "with custom port",
			inputURL:     "kodi://mykodi.local:9090/",
			expectError:  false,
			expectedHost: "mykodi.local",
			expectedPort: 9090,
		},
		{
			name:        "invalid scheme",
			inputURL:    "http://mykodi.local/",
			expectError: true,
		},
		{
			name:        "missing hostname",
			inputURL:    "kodi:///",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewKodiService().(*KodiService)
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
			if tt.expectedUser != "" && service.username != tt.expectedUser {
				t.Errorf("Expected username '%s', got '%s'", tt.expectedUser, service.username)
			}
			if tt.expectedPass != "" && service.password != tt.expectedPass {
				t.Errorf("Expected password '%s', got '%s'", tt.expectedPass, service.password)
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

func TestKodiService_TestURL(t *testing.T) {
	service := NewKodiService()

	valid := []string{
		"kodi://mykodi.local/",
		"kodis://admin:pass@mykodi.local/",
		"kodi://admin:pass@mykodi.local:9090/",
	}
	for _, u := range valid {
		if err := service.TestURL(u); err != nil {
			t.Errorf("Valid URL %q should not error: %v", u, err)
		}
	}

	invalid := []string{"http://mykodi.local/", "kodi:///"}
	for _, u := range invalid {
		if err := service.TestURL(u); err == nil {
			t.Errorf("Invalid URL %q should error", u)
		}
	}
}

func TestKodiService_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/jsonrpc" {
			t.Errorf("Expected path /jsonrpc, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		var rpcReq KodiRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
			t.Fatalf("Failed to decode body: %v", err)
		}
		if rpcReq.Method != "GUI.ShowNotification" {
			t.Errorf("Expected method GUI.ShowNotification, got %s", rpcReq.Method)
		}
		if rpcReq.JSONRPC != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %s", rpcReq.JSONRPC)
		}
		if rpcReq.Params["title"] != "Test Alert" {
			t.Errorf("Expected title 'Test Alert', got '%v'", rpcReq.Params["title"])
		}
		if rpcReq.Params["message"] != "Something happened" {
			t.Errorf("Expected message 'Something happened', got '%v'", rpcReq.Params["message"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(KodiRPCResponse{JSONRPC: "2.0", Result: "OK", ID: 1})
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewKodiService().(*KodiService)
	service.hostname = serverURL.Hostname()
	port := 0
	for _, c := range serverURL.Port() {
		port = port*10 + int(c-'0')
	}
	service.port = port

	req := NotificationRequest{
		Title:      "Test Alert",
		Body:       "Something happened",
		NotifyType: NotifyTypeWarning,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := service.Send(ctx, req); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestKodiService_SendRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp := KodiRPCResponse{
			JSONRPC: "2.0",
			Error:   &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32601, Message: "Method not found"},
			ID: 1,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewKodiService().(*KodiService)
	service.hostname = serverURL.Hostname()
	port := 0
	for _, c := range serverURL.Port() {
		port = port*10 + int(c-'0')
	}
	service.port = port

	req := NotificationRequest{Body: "test", NotifyType: NotifyTypeInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for JSON-RPC error response")
	}
	if !strings.Contains(err.Error(), "kodi JSON-RPC error") {
		t.Errorf("Expected kodi JSON-RPC error, got: %v", err)
	}
}

func TestKodiService_SendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	service := NewKodiService().(*KodiService)
	service.hostname = serverURL.Hostname()
	port := 0
	for _, c := range serverURL.Port() {
		port = port*10 + int(c-'0')
	}
	service.port = port

	req := NotificationRequest{Body: "test", NotifyType: NotifyTypeInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected error for HTTP 401")
	}
	if !strings.Contains(err.Error(), "kodi JSON-RPC error") {
		t.Errorf("Expected kodi JSON-RPC error, got: %v", err)
	}
}
