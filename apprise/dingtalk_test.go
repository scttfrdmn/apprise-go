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

func TestDingTalkService_GetServiceID(t *testing.T) {
	service := NewDingTalkService()
	if service.GetServiceID() != "dingtalk" {
		t.Errorf("Expected service ID 'dingtalk', got '%s'", service.GetServiceID())
	}
}

func TestDingTalkService_GetDefaultPort(t *testing.T) {
	service := NewDingTalkService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected port 443, got %d", service.GetDefaultPort())
	}
}

func TestDingTalkService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *DingTalkService)
	}{
		{
			name:        "Basic DingTalk URL (token only)",
			url:         "dingtalk://a1b2c3d4e5f6",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if !strings.Contains(d.webhookURL, "a1b2c3d4e5f6") {
					t.Errorf("Expected webhook URL to contain token, got '%s'", d.webhookURL)
				}
				if !strings.Contains(d.webhookURL, "oapi.dingtalk.com") {
					t.Errorf("Expected webhook URL to contain oapi.dingtalk.com, got '%s'", d.webhookURL)
				}
			},
		},
		{
			name:        "With secret parameter",
			url:         "dingtalk://token123?secret=SEC456abc",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if d.secret != "SEC456abc" {
					t.Errorf("Expected secret 'SEC456abc', got '%s'", d.secret)
				}
			},
		},
		{
			name:        "With @all parameter",
			url:         "dingtalk://token123?atall=true",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if !d.atAll {
					t.Error("Expected atAll to be true")
				}
			},
		},
		{
			name:        "With @mobile parameter",
			url:         "dingtalk://token123?atmobile=13800138000,13900139000",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if len(d.atMobiles) != 2 {
					t.Errorf("Expected 2 mobile numbers, got %d", len(d.atMobiles))
				}
				if d.atMobiles[0] != "13800138000" {
					t.Errorf("Expected first mobile '13800138000', got '%s'", d.atMobiles[0])
				}
			},
		},
		{
			name:        "Full webhook URL format",
			url:         "dingtalk://oapi.dingtalk.com/robot/send?access_token=abc123&secret=SEC456",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if !strings.Contains(d.webhookURL, "abc123") {
					t.Errorf("Expected webhook URL to contain token, got '%s'", d.webhookURL)
				}
				if d.secret != "SEC456" {
					t.Errorf("Expected secret 'SEC456', got '%s'", d.secret)
				}
			},
		},
		{
			name:        "With all parameters",
			url:         "dingtalk://token?secret=SEC123&atall=true&atmobile=13800138000",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if d.secret != "SEC123" {
					t.Errorf("Expected secret, got '%s'", d.secret)
				}
				if !d.atAll {
					t.Error("Expected atAll to be true")
				}
				if len(d.atMobiles) != 1 {
					t.Errorf("Expected 1 mobile, got %d", len(d.atMobiles))
				}
			},
		},
		{
			name:        "Dingding alias scheme",
			url:         "dingding://token123",
			expectError: false,
			checkFunc: func(t *testing.T, d *DingTalkService) {
				if !strings.Contains(d.webhookURL, "token123") {
					t.Errorf("Expected token in URL, got '%s'", d.webhookURL)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "slack://token123",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "dingtalk://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewDingTalkService().(*DingTalkService)
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

func TestDingTalkService_Send(t *testing.T) {
	tests := []struct {
		name      string
		request   NotificationRequest
		checkFunc func(*testing.T, DingTalkRequest)
	}{
		{
			name: "Basic notification",
			request: NotificationRequest{
				Title:      "Test Alert",
				Body:       "Test message body",
				NotifyType: NotifyTypeInfo,
			},
			checkFunc: func(t *testing.T, req DingTalkRequest) {
				if req.MsgType != "markdown" {
					t.Errorf("Expected msgtype 'markdown', got '%s'", req.MsgType)
				}
				if req.Markdown == nil {
					t.Fatal("Expected markdown content")
				}
				if !strings.Contains(req.Markdown.Text, "Test Alert") {
					t.Error("Expected title in markdown text")
				}
				if !strings.Contains(req.Markdown.Text, "Test message body") {
					t.Error("Expected body in markdown text")
				}
			},
		},
		{
			name: "Error notification with emoji",
			request: NotificationRequest{
				Title:      "Critical Error",
				Body:       "Database connection failed",
				NotifyType: NotifyTypeError,
			},
			checkFunc: func(t *testing.T, req DingTalkRequest) {
				if !strings.Contains(req.Markdown.Text, "🔴") {
					t.Error("Expected error emoji (🔴) in content")
				}
			},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "Warning",
				Body:       "High memory usage",
				NotifyType: NotifyTypeWarning,
			},
			checkFunc: func(t *testing.T, req DingTalkRequest) {
				if !strings.Contains(req.Markdown.Text, "⚠️") {
					t.Error("Expected warning emoji (⚠️) in content")
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
			checkFunc: func(t *testing.T, req DingTalkRequest) {
				if !strings.Contains(req.Markdown.Text, "✅") {
					t.Error("Expected success emoji (✅) in content")
				}
			},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Tagged Alert",
				Body:       "Alert with tags",
				NotifyType: NotifyTypeInfo,
				Tags:       []string{"production", "database"},
			},
			checkFunc: func(t *testing.T, req DingTalkRequest) {
				if !strings.Contains(req.Markdown.Text, "production") {
					t.Error("Expected 'production' tag in content")
				}
				if !strings.Contains(req.Markdown.Text, "database") {
					t.Error("Expected 'database' tag in content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest DingTalkRequest

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got '%s'", r.Method)
				}

				// Parse body
				body, _ := io.ReadAll(r.Body)
				if err := json.Unmarshal(body, &receivedRequest); err != nil {
					t.Errorf("Failed to parse request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return success response
				response := DingTalkResponse{
					ErrCode: 0,
					ErrMsg:  "ok",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create service
			service := NewDingTalkService().(*DingTalkService)
			service.webhookURL = server.URL

			// Build request for testing
			payload := service.buildRequest(tt.request)

			if tt.checkFunc != nil {
				tt.checkFunc(t, payload)
			}

			// Verify common fields
			if payload.MsgType != "markdown" {
				t.Errorf("Expected msgtype 'markdown', got '%s'", payload.MsgType)
			}
			if payload.Markdown == nil {
				t.Error("Expected markdown content to be set")
			}
		})
	}
}

func TestDingTalkService_GetEmojiForNotifyType(t *testing.T) {
	service := NewDingTalkService().(*DingTalkService)

	tests := []struct {
		notifyType    NotifyType
		expectedEmoji string
	}{
		{NotifyTypeError, "🔴"},
		{NotifyTypeWarning, "⚠️"},
		{NotifyTypeInfo, "ℹ️"},
		{NotifyTypeSuccess, "✅"},
	}

	for _, tt := range tests {
		emoji := service.getEmojiForNotifyType(tt.notifyType)
		if emoji != tt.expectedEmoji {
			t.Errorf("For %v, expected emoji '%s', got '%s'", tt.notifyType, tt.expectedEmoji, emoji)
		}
	}
}

func TestDingTalkService_TestURL(t *testing.T) {
	service := NewDingTalkService()

	validURLs := []string{
		"dingtalk://a1b2c3d4e5f6",
		"dingtalk://token?secret=SEC123",
		"dingtalk://token?atall=true",
		"dingding://token123",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"slack://token",
		"dingtalk://",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestDingTalkService_SupportsAttachments(t *testing.T) {
	service := NewDingTalkService()
	if service.SupportsAttachments() {
		t.Error("DingTalk service should not support direct attachments")
	}
}

func TestDingTalkService_GetMaxBodyLength(t *testing.T) {
	service := NewDingTalkService()
	if service.GetMaxBodyLength() != 20000 {
		t.Errorf("Expected max body length 20000, got %d", service.GetMaxBodyLength())
	}
}

func TestDingTalkService_GenerateSignature(t *testing.T) {
	service := NewDingTalkService().(*DingTalkService)

	timestamp := int64(1609459200000)
	secret := "SEC123abc"

	signature := service.generateSignature(timestamp, secret)

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	// Signature should be URL-encoded base64
	if !strings.Contains(signature, "%") || len(signature) < 20 {
		t.Errorf("Expected URL-encoded base64 signature, got '%s'", signature)
	}
}

func TestDingTalkService_BuildRequest(t *testing.T) {
	service := NewDingTalkService().(*DingTalkService)
	service.atAll = true
	service.atMobiles = []string{"13800138000"}

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Body",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"tag1"},
	}

	payload := service.buildRequest(req)

	if payload.MsgType != "markdown" {
		t.Errorf("Expected msgtype 'markdown', got '%s'", payload.MsgType)
	}
	if payload.Markdown == nil {
		t.Fatal("Expected markdown content")
	}
	if payload.At == nil {
		t.Fatal("Expected At configuration")
	}
	if !payload.At.IsAtAll {
		t.Error("Expected IsAtAll to be true")
	}
	if len(payload.At.AtMobiles) != 1 {
		t.Errorf("Expected 1 mobile, got %d", len(payload.At.AtMobiles))
	}
}

func TestDingTalkService_SendWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := DingTalkResponse{
			ErrCode: 0,
			ErrMsg:  "ok",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	service := NewDingTalkService().(*DingTalkService)
	service.webhookURL = server.URL

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test",
		NotifyType: NotifyTypeInfo,
	}

	err := service.Send(context.Background(), req)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
