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

func TestLarkService_GetServiceID(t *testing.T) {
	service := NewLarkService()
	if service.GetServiceID() != "lark" {
		t.Errorf("Expected service ID 'lark', got '%s'", service.GetServiceID())
	}
}

func TestLarkService_GetDefaultPort(t *testing.T) {
	service := NewLarkService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestLarkService_ParseURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		checkFunc   func(*testing.T, *LarkService)
	}{
		{
			name:        "Simple token format",
			url:         "lark://1234567890abcdef1234567890abcdef",
			expectError: false,
			checkFunc: func(t *testing.T, l *LarkService) {
				if l.token != "1234567890abcdef1234567890abcdef" {
					t.Errorf("Expected token '1234567890abcdef1234567890abcdef', got '%s'", l.token)
				}
				if !strings.Contains(l.webhookURL, "open.larksuite.com") {
					t.Errorf("Expected international domain, got '%s'", l.webhookURL)
				}
				if !strings.Contains(l.webhookURL, l.token) {
					t.Error("Webhook URL should contain token")
				}
			},
		},
		{
			name:        "Feishu scheme (China)",
			url:         "feishu://abcdef1234567890abcdef1234567890",
			expectError: false,
			checkFunc: func(t *testing.T, l *LarkService) {
				if !strings.Contains(l.webhookURL, "open.feishu.cn") {
					t.Errorf("Expected China domain for feishu://, got '%s'", l.webhookURL)
				}
			},
		},
		{
			name:        "Full webhook URL format (international)",
			url:         "lark://open.larksuite.com/open-apis/bot/v2/hook/abcd1234efgh5678ijkl9012mnop3456",
			expectError: false,
			checkFunc: func(t *testing.T, l *LarkService) {
				if l.token != "abcd1234efgh5678ijkl9012mnop3456" {
					t.Errorf("Expected token extracted from path, got '%s'", l.token)
				}
				if !strings.HasPrefix(l.webhookURL, "https://open.larksuite.com") {
					t.Errorf("Expected HTTPS URL with correct domain, got '%s'", l.webhookURL)
				}
			},
		},
		{
			name:        "Full webhook URL format (China)",
			url:         "feishu://open.feishu.cn/open-apis/bot/v2/hook/xyz789abc123def456ghi789jkl012mn",
			expectError: false,
			checkFunc: func(t *testing.T, l *LarkService) {
				if !strings.HasPrefix(l.webhookURL, "https://open.feishu.cn") {
					t.Errorf("Expected China HTTPS URL, got '%s'", l.webhookURL)
				}
			},
		},
		{
			name:        "Invalid scheme",
			url:         "slack://token",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "lark://",
			expectError: true,
		},
		{
			name:        "Token too short",
			url:         "lark://short",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLarkService().(*LarkService)
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

func TestLarkService_Send(t *testing.T) {
	tests := []struct {
		name        string
		request     NotificationRequest
		setupServer func(*testing.T) *httptest.Server
		expectError bool
		checkFunc   func(*testing.T, *http.Request, LarkMessagePayload)
	}{
		{
			name: "Info notification",
			request: NotificationRequest{
				Title:      "System Update",
				Body:       "A new version has been deployed",
				NotifyType: NotifyTypeInfo,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 0,
						"msg":  "success",
					})
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload LarkMessagePayload) {
				if payload.MsgType != "text" {
					t.Errorf("Expected msg_type 'text', got '%s'", payload.MsgType)
				}
				if !strings.Contains(payload.Content.Text, "System Update") {
					t.Error("Expected title in message text")
				}
				if !strings.Contains(payload.Content.Text, "A new version has been deployed") {
					t.Error("Expected body in message text")
				}
				if !strings.Contains(payload.Content.Text, "ℹ️") {
					t.Error("Expected info emoji")
				}
			},
		},
		{
			name: "Success notification",
			request: NotificationRequest{
				Title:      "Deploy Complete",
				Body:       "Successfully deployed to production",
				NotifyType: NotifyTypeSuccess,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 0,
						"msg":  "success",
					})
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload LarkMessagePayload) {
				if !strings.Contains(payload.Content.Text, "✅") {
					t.Error("Expected success emoji")
				}
			},
		},
		{
			name: "Warning notification",
			request: NotificationRequest{
				Title:      "High Memory Usage",
				Body:       "Memory usage at 85%",
				NotifyType: NotifyTypeWarning,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 0,
						"msg":  "success",
					})
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload LarkMessagePayload) {
				if !strings.Contains(payload.Content.Text, "⚠️") {
					t.Error("Expected warning emoji")
				}
			},
		},
		{
			name: "Error notification",
			request: NotificationRequest{
				Title:      "Service Down",
				Body:       "API service is not responding",
				NotifyType: NotifyTypeError,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 0,
						"msg":  "success",
					})
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload LarkMessagePayload) {
				if !strings.Contains(payload.Content.Text, "🚨") {
					t.Error("Expected error emoji")
				}
			},
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Alert",
				Body:       "Test with tags",
				NotifyType: NotifyTypeInfo,
				Tags:       []string{"production", "database", "critical"},
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 0,
						"msg":  "success",
					})
				}))
			},
			expectError: false,
			checkFunc: func(t *testing.T, r *http.Request, payload LarkMessagePayload) {
				text := payload.Content.Text
				if !strings.Contains(text, "Tags:") {
					t.Error("Expected tags section")
				}
				if !strings.Contains(text, "production") || !strings.Contains(text, "database") || !strings.Contains(text, "critical") {
					t.Error("Expected all tags in message")
				}
			},
		},
		{
			name: "Lark API error response",
			request: NotificationRequest{
				Title:      "Test",
				Body:       "Test",
				NotifyType: NotifyTypeInfo,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"code": 9499,
						"msg":  "Invalid hook token",
					})
				}))
			},
			expectError: true,
		},
		{
			name: "HTTP error status",
			request: NotificationRequest{
				Title:      "Test",
				Body:       "Test",
				NotifyType: NotifyTypeInfo,
			},
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer(t)
			defer server.Close()

			service := NewLarkService().(*LarkService)
			service.token = "test1234567890abcdef1234567890ab"
			service.webhookURL = server.URL

			// Capture request for validation
			var capturedRequest *http.Request
			var capturedPayload LarkMessagePayload

			// Wrap the test server to capture requests
			if !tt.expectError && tt.checkFunc != nil {
				originalHandler := server.Config.Handler
				server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedRequest = r
					body, _ := io.ReadAll(r.Body)
					json.Unmarshal(body, &capturedPayload)
					r.Body = io.NopCloser(strings.NewReader(string(body)))
					originalHandler.ServeHTTP(w, r)
				})
			}

			// Send notification
			err := service.Send(context.Background(), tt.request)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.checkFunc != nil && capturedRequest != nil {
				tt.checkFunc(t, capturedRequest, capturedPayload)
			}
		})
	}
}

func TestLarkService_MessageTruncation(t *testing.T) {
	service := NewLarkService().(*LarkService)

	// Create a very long message (over 20,000 characters)
	longBody := strings.Repeat("a", 21000)

	req := NotificationRequest{
		Title:      "Test",
		Body:       longBody,
		NotifyType: NotifyTypeInfo,
	}

	messageText := service.buildMessageText(req)

	if len(messageText) > 20000 {
		t.Errorf("Message should be truncated to 20000 chars, got %d", len(messageText))
	}

	if !strings.HasSuffix(messageText, "...") {
		t.Error("Truncated message should end with '...'")
	}
}

func TestLarkService_ContentTypeHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "success",
		})
	}))
	defer server.Close()

	service := NewLarkService().(*LarkService)
	service.token = "test1234567890abcdef1234567890ab"
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

func TestLarkService_TestURL(t *testing.T) {
	service := NewLarkService()

	validURLs := []string{
		"lark://1234567890abcdef1234567890abcdef",
		"feishu://abcdef1234567890abcdef1234567890",
		"lark://open.larksuite.com/open-apis/bot/v2/hook/abcd1234efgh5678ijkl9012mnop3456",
	}

	for _, testURL := range validURLs {
		err := service.TestURL(testURL)
		if err != nil {
			t.Errorf("URL '%s' should be valid, got error: %v", testURL, err)
		}
	}

	invalidURLs := []string{
		"http://example.com",
		"slack://token",
		"lark://short",
		"lark://",
	}

	for _, testURL := range invalidURLs {
		err := service.TestURL(testURL)
		if err == nil {
			t.Errorf("URL '%s' should be invalid", testURL)
		}
	}
}

func TestLarkService_SupportsAttachments(t *testing.T) {
	service := NewLarkService()
	if service.SupportsAttachments() {
		t.Error("Lark service should not support attachments")
	}
}

func TestLarkService_GetMaxBodyLength(t *testing.T) {
	service := NewLarkService()
	if service.GetMaxBodyLength() != 20000 {
		t.Errorf("Expected max body length 20000, got %d", service.GetMaxBodyLength())
	}
}

func TestLarkService_EmojiMapping(t *testing.T) {
	service := NewLarkService().(*LarkService)

	tests := []struct {
		notifyType    NotifyType
		expectedEmoji string
	}{
		{NotifyTypeInfo, "ℹ️"},
		{NotifyTypeSuccess, "✅"},
		{NotifyTypeWarning, "⚠️"},
		{NotifyTypeError, "🚨"},
	}

	for _, tt := range tests {
		emoji := service.getNotificationEmoji(tt.notifyType)
		if emoji != tt.expectedEmoji {
			t.Errorf("For %v, expected emoji '%s', got '%s'", tt.notifyType, tt.expectedEmoji, emoji)
		}
	}
}

func TestLarkService_BuildMessageText(t *testing.T) {
	service := NewLarkService().(*LarkService)

	tests := []struct {
		name     string
		request  NotificationRequest
		contains []string
		notContains []string
	}{
		{
			name: "Title and body",
			request: NotificationRequest{
				Title:      "Test Title",
				Body:       "Test Body",
				NotifyType: NotifyTypeInfo,
			},
			contains: []string{"ℹ️", "Test Title", "Test Body"},
		},
		{
			name: "Body only",
			request: NotificationRequest{
				Body:       "Just the body",
				NotifyType: NotifyTypeSuccess,
			},
			contains: []string{"✅", "Just the body"},
			notContains: []string{"\n\n"}, // No double newline without title
		},
		{
			name: "With tags",
			request: NotificationRequest{
				Title:      "Alert",
				Body:       "Message",
				NotifyType: NotifyTypeWarning,
				Tags:       []string{"tag1", "tag2"},
			},
			contains: []string{"⚠️", "Alert", "Message", "Tags:", "tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := service.buildMessageText(tt.request)

			for _, substr := range tt.contains {
				if !strings.Contains(text, substr) {
					t.Errorf("Expected text to contain '%s', got: %s", substr, text)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(text, substr) {
					t.Errorf("Expected text NOT to contain '%s', got: %s", substr, text)
				}
			}
		})
	}
}
