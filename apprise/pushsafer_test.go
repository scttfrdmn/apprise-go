package apprise

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPushsaferService_GetServiceID(t *testing.T) {
	service := NewPushsaferService()
	if service.GetServiceID() != "pushsafer" {
		t.Errorf("Expected service ID 'pushsafer', got '%s'", service.GetServiceID())
	}
}

func TestPushsaferService_GetDefaultPort(t *testing.T) {
	service := NewPushsaferService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected port 443, got %d", service.GetDefaultPort())
	}
}

func TestPushsaferService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *PushsaferService)
	}{
		{
			name:        "Basic Pushsafer URL",
			url:         "pushsafer://a1b2c3d4e5f6",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.privateKey != "a1b2c3d4e5f6" {
					t.Errorf("Expected private key 'a1b2c3d4e5f6', got '%s'", p.privateKey)
				}
				if p.device != "a" {
					t.Errorf("Expected default device 'a', got '%s'", p.device)
				}
			},
		},
		{
			name:        "With specific device",
			url:         "pushsafer://key123/52",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.device != "52" {
					t.Errorf("Expected device '52', got '%s'", p.device)
				}
			},
		},
		{
			name:        "With device group",
			url:         "pushsafer://key123/gs100",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.device != "gs100" {
					t.Errorf("Expected device group 'gs100', got '%s'", p.device)
				}
			},
		},
		{
			name:        "All devices explicit",
			url:         "pushsafer://key123/a",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.device != "a" {
					t.Errorf("Expected device 'a', got '%s'", p.device)
				}
			},
		},
		{
			name:        "With sound parameter",
			url:         "pushsafer://key123?sound=5",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.sound != 5 {
					t.Errorf("Expected sound 5, got %d", p.sound)
				}
			},
		},
		{
			name:        "With vibration parameter",
			url:         "pushsafer://key123?vibration=2",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.vibration != 2 {
					t.Errorf("Expected vibration 2, got %d", p.vibration)
				}
			},
		},
		{
			name:        "With icon parameter",
			url:         "pushsafer://key123?icon=33",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.icon != 33 {
					t.Errorf("Expected icon 33, got %d", p.icon)
				}
			},
		},
		{
			name:        "With color parameter",
			url:         "pushsafer://key123?color=%23FF0000",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.iconColor != "#FF0000" {
					t.Errorf("Expected color '#FF0000', got '%s'", p.iconColor)
				}
			},
		},
		{
			name:        "With color without hash",
			url:         "pushsafer://key123?color=FF0000",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.iconColor != "#FF0000" {
					t.Errorf("Expected color '#FF0000', got '%s'", p.iconColor)
				}
			},
		},
		{
			name:        "With priority parameter",
			url:         "pushsafer://key123?priority=2",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.priority != 2 {
					t.Errorf("Expected priority 2, got %d", p.priority)
				}
			},
		},
		{
			name:        "With TTL parameter",
			url:         "pushsafer://key123?ttl=60",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.timeToLive != 60 {
					t.Errorf("Expected TTL 60, got %d", p.timeToLive)
				}
			},
		},
		{
			name:        "With all parameters",
			url:         "pushsafer://key123/52?sound=5&vibration=2&icon=33&color=%23FF0000&priority=1&ttl=120",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.device != "52" {
					t.Errorf("Expected device '52', got '%s'", p.device)
				}
				if p.sound != 5 {
					t.Errorf("Expected sound 5, got %d", p.sound)
				}
				if p.vibration != 2 {
					t.Errorf("Expected vibration 2, got %d", p.vibration)
				}
				if p.icon != 33 {
					t.Errorf("Expected icon 33, got %d", p.icon)
				}
				if p.iconColor != "#FF0000" {
					t.Errorf("Expected color '#FF0000', got '%s'", p.iconColor)
				}
				if p.priority != 1 {
					t.Errorf("Expected priority 1, got %d", p.priority)
				}
				if p.timeToLive != 120 {
					t.Errorf("Expected TTL 120, got %d", p.timeToLive)
				}
			},
		},
		{
			name:        "Device override from query param",
			url:         "pushsafer://key123/52?device=a",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.device != "a" {
					t.Errorf("Expected device 'a' (from query param), got '%s'", p.device)
				}
			},
		},
		{
			name:        "Psafer alias scheme",
			url:         "psafer://key123",
			expectError: false,
			checkFunc: func(t *testing.T, p *PushsaferService) {
				if p.privateKey != "key123" {
					t.Errorf("Expected private key 'key123', got '%s'", p.privateKey)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "pushover://key123",
			expectError: true,
		},
		{
			name:        "Missing private key",
			url:         "pushsafer://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPushsaferService().(*PushsaferService)
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

func TestPushsaferService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		checkFunc   func(*testing.T, PushsaferRequest)
		expectError bool
	}{
		{
			name: "Basic notification",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test message body",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Title != "Test Alert" {
					t.Errorf("Expected title 'Test Alert', got '%s'", req.Title)
				}
				if req.Message != "Test message body" {
					t.Errorf("Expected message, got '%s'", req.Message)
				}
				if req.Priority != 0 {
					t.Errorf("Expected priority 0, got %d", req.Priority)
				}
			},
		},
		{
			name: "Error notification (critical priority)",
			request: NotificationRequest{
				Title:      "Critical Error",
				Body:       "Database connection failed",
				NotifyType: NotifyTypeError,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Priority != 2 {
					t.Errorf("Expected priority 2 for error, got %d", req.Priority)
				}
			},
		},
		{
			name: "Warning notification (high priority)",
			request: NotificationRequest{
				Title:      "Warning",
				Body:       "High memory usage",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Priority != 1 {
					t.Errorf("Expected priority 1 for warning, got %d", req.Priority)
				}
			},
		},
		{
			name: "Success notification",
			request: NotificationRequest{
				Title:      "Success",
				Body:       "Task completed",
				NotifyType: NotifyTypeSuccess,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Priority != 0 {
					t.Errorf("Expected priority 0 for success, got %d", req.Priority)
				}
			},
		},
		{
			name: "Body only (no title)",
			request: NotificationRequest{
				Body:       "Simple message",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Message != "Simple message" {
					t.Errorf("Expected message 'Simple message', got '%s'", req.Message)
				}
				if req.Title != "" {
					t.Error("Expected no title for body-only notification")
				}
			},
		},
		{
			name: "Title only (no body)",
			request: NotificationRequest{
				Title:      "Title Only",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Message != "Title Only" {
					t.Errorf("Expected message 'Title Only', got '%s'", req.Message)
				}
			},
		},
		{
			name: "Empty notification (default message)",
			request: NotificationRequest{
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, req PushsaferRequest) {
				if req.Message != "Notification from apprise-go" {
					t.Errorf("Expected default message, got '%s'", req.Message)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest PushsaferRequest

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got '%s'", r.Method)
				}

				// Check headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", r.Header.Get("Content-Type"))
				}

				// Parse body
				body, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(body, &receivedRequest); err != nil {
					t.Errorf("Failed to parse request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return success response
				response := PushsaferResponse{
					Status:  "success",
					Success: "Notification sent",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create service
			service := NewPushsaferService().(*PushsaferService)
			service.privateKey = "test-key"
			service.client = &http.Client{}

			// Override API URL for testing
			// (We'll build the request directly for testing)
			req := service.buildRequest(tt.request)

			if tt.checkFunc != nil {
				tt.checkFunc(t, req)
			}

			// Verify common fields
			if req.PrivateKey != "test-key" {
				t.Errorf("Expected private key 'test-key', got '%s'", req.PrivateKey)
			}
			if req.Device != "a" {
				t.Errorf("Expected default device 'a', got '%s'", req.Device)
			}
			if req.Message == "" {
				t.Error("Expected message to be set")
			}
		})
	}
}

func TestPushsaferService_MapNotifyTypeToPriority(t *testing.T) {
	service := NewPushsaferService().(*PushsaferService)

	tests := []struct {
		notifyType       NotifyType
		expectedPriority int
	}{
		{NotifyTypeError, 2},   // Critical
		{NotifyTypeWarning, 1}, // High
		{NotifyTypeInfo, 0},    // Normal
		{NotifyTypeSuccess, 0}, // Normal
	}

	for _, tt := range tests {
		priority := service.mapNotifyTypeToPriority(tt.notifyType)
		if priority != tt.expectedPriority {
			t.Errorf("For %v, expected priority %d, got %d", tt.notifyType, tt.expectedPriority, priority)
		}
	}
}

func TestPushsaferService_TestURL(t *testing.T) {
	service := NewPushsaferService()

	validURLs := []string{
		"pushsafer://a1b2c3d4e5f6",
		"pushsafer://key123/52",
		"pushsafer://key123/gs100",
		"pushsafer://key123?sound=5&vibration=2",
		"psafer://key123",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"pushover://key123",
		"pushsafer://",
		"http://pushsafer.com",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestPushsaferService_SupportsAttachments(t *testing.T) {
	service := NewPushsaferService()
	if !service.SupportsAttachments() {
		t.Error("Pushsafer service should support attachments (up to 3 images)")
	}
}

func TestPushsaferService_GetMaxBodyLength(t *testing.T) {
	service := NewPushsaferService()
	if service.GetMaxBodyLength() != 10000 {
		t.Errorf("Expected max body length 10000, got %d", service.GetMaxBodyLength())
	}
}

func TestPushsaferService_MessageTruncation(t *testing.T) {
	service := NewPushsaferService().(*PushsaferService)
	service.privateKey = "test-key"

	// Create a message longer than 10000 characters
	longMessage := string(make([]byte, 10500))
	for i := range longMessage {
		longMessage = string(append([]byte(longMessage[:i]), 'A'))
	}

	req := NotificationRequest{
		Title:      "Test",
		Body:       longMessage,
		NotifyType: NotifyTypeInfo,
	}

	payload := service.buildRequest(req)

	if len(payload.Message) > 10000 {
		t.Errorf("Expected message to be truncated to 10000 chars, got %d", len(payload.Message))
	}

	if len(payload.Message) != 10000 {
		t.Errorf("Expected truncated message to be exactly 10000 chars, got %d", len(payload.Message))
	}

	// Should end with "..."
	if payload.Message[len(payload.Message)-3:] != "..." {
		t.Error("Expected truncated message to end with '...'")
	}
}

func TestPushsaferService_BuildRequest(t *testing.T) {
	service := NewPushsaferService().(*PushsaferService)
	service.privateKey = "test-key-123"
	service.device = "gs100"
	service.sound = 5
	service.vibration = 2
	service.icon = 33
	service.iconColor = "#FF0000"
	service.timeToLive = 120

	req := NotificationRequest{
		Title:      "Test Title",
		Body:       "Test Body",
		NotifyType: NotifyTypeWarning,
	}

	payload := service.buildRequest(req)

	if payload.PrivateKey != "test-key-123" {
		t.Errorf("Expected private key 'test-key-123', got '%s'", payload.PrivateKey)
	}
	if payload.Device != "gs100" {
		t.Errorf("Expected device 'gs100', got '%s'", payload.Device)
	}
	if payload.Sound != 5 {
		t.Errorf("Expected sound 5, got %d", payload.Sound)
	}
	if payload.Vibration != 2 {
		t.Errorf("Expected vibration 2, got %d", payload.Vibration)
	}
	if payload.Icon != 33 {
		t.Errorf("Expected icon 33, got %d", payload.Icon)
	}
	if payload.IconColor != "#FF0000" {
		t.Errorf("Expected color '#FF0000', got '%s'", payload.IconColor)
	}
	if payload.TimeToLive != 120 {
		t.Errorf("Expected TTL 120, got %d", payload.TimeToLive)
	}
	if payload.Priority != 1 {
		t.Errorf("Expected priority 1 for warning, got %d", payload.Priority)
	}
	if payload.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", payload.Title)
	}
	if payload.Message != "Test Body" {
		t.Errorf("Expected message 'Test Body', got '%s'", payload.Message)
	}
}
