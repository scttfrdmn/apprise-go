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

func TestRocketChatService_GetServiceID(t *testing.T) {
	service := NewRocketChatService()
	if service.GetServiceID() != "rocketchat" {
		t.Errorf("Expected service ID 'rocketchat', got '%s'", service.GetServiceID())
	}
}

func TestRocketChatService_GetDefaultPort(t *testing.T) {
	service := NewRocketChatService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestRocketChatService_SupportsAttachments(t *testing.T) {
	service := NewRocketChatService()
	if !service.SupportsAttachments() {
		t.Error("Rocket.Chat should support attachments")
	}
}

func TestRocketChatService_GetMaxBodyLength(t *testing.T) {
	service := NewRocketChatService()
	expected := 1000
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestRocketChatService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedServer   string
		expectedChannel  string
		expectedUsername string
		expectedPassword string
		expectedUserID   string
		expectedToken    string
		expectedWebhook  string
		expectedBotName  string
	}{
		{
			name:             "Username password authentication - HTTP",
			url:              "rocket://user:pass@chat.example.com/general",
			expectError:      false,
			expectedServer:   "http://chat.example.com:3000",
			expectedChannel:  "#general",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "Username password authentication - HTTPS",
			url:              "rockets://user:pass@chat.example.com/general",
			expectError:      false,
			expectedServer:   "https://chat.example.com",
			expectedChannel:  "#general",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:            "User ID and token authentication",
			url:             "rockets://userid123:tokenABC@chat.example.com/alerts",
			expectError:     false,
			expectedServer:  "https://chat.example.com",
			expectedChannel: "#alerts",
			expectedUserID:  "userid123",
			expectedToken:   "tokenABC",
		},
		{
			name:             "Channel with @ prefix (direct message)",
			url:              "rockets://user:pass@chat.example.com/@admin",
			expectError:      false,
			expectedServer:   "https://chat.example.com",
			expectedChannel:  "@admin",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "Channel with # prefix",
			url:              "rockets://user:pass@chat.example.com/#support",
			expectError:      false,
			expectedServer:   "https://chat.example.com",
			expectedChannel:  "#support",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "Channel in fragment",
			url:              "rockets://user:pass@chat.example.com#development",
			expectError:      false,
			expectedServer:   "https://chat.example.com",
			expectedChannel:  "#development",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "Query parameters for authentication",
			url:              "rockets://chat.example.com/general?username=bot&password=secret&bot_name=AlertBot",
			expectError:      false,
			expectedServer:   "https://chat.example.com",
			expectedChannel:  "#general",
			expectedUsername: "bot",
			expectedPassword: "secret",
			expectedBotName:  "AlertBot",
		},
		{
			name:             "Custom port",
			url:              "rockets://user:pass@chat.example.com:8080/team",
			expectError:      false,
			expectedServer:   "https://chat.example.com:8080",
			expectedChannel:  "#team",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:            "Webhook URL",
			url:             "rocket://webhook@chat.example.com/hooks/abc123/def456",
			expectError:     false,
			expectedServer:  "http://chat.example.com:3000",
			expectedWebhook: "http://chat.example.com:3000/hooks/abc123/def456",
			expectedChannel: "#webhook", // From user part, normalized
		},
		{
			name:            "Webhook URL with channel query",
			url:             "rockets://chat.example.com/hooks/webhook_id/token?channel=alerts",
			expectError:     false,
			expectedServer:  "https://chat.example.com",
			expectedWebhook: "https://chat.example.com/hooks/webhook_id/token",
			expectedChannel: "alerts", // Explicit from query, not normalized
		},
		{
			name:        "Invalid scheme",
			url:         "http://user:pass@chat.example.com/general",
			expectError: true,
		},
		{
			name:        "Missing authentication for REST API",
			url:         "rockets://chat.example.com/general",
			expectError: true,
		},
		{
			name:        "Missing channel",
			url:         "rockets://user:pass@chat.example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRocketChatService().(*RocketChatService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse URL: %v", err)
				}
				return
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.server != tt.expectedServer {
				t.Errorf("Expected server '%s', got '%s'", tt.expectedServer, service.server)
			}

			if service.channel != tt.expectedChannel {
				t.Errorf("Expected channel '%s', got '%s'", tt.expectedChannel, service.channel)
			}

			if tt.expectedUsername != "" && service.username != tt.expectedUsername {
				t.Errorf("Expected username '%s', got '%s'", tt.expectedUsername, service.username)
			}

			if tt.expectedPassword != "" && service.password != tt.expectedPassword {
				t.Errorf("Expected password '%s', got '%s'", tt.expectedPassword, service.password)
			}

			if tt.expectedUserID != "" && service.userID != tt.expectedUserID {
				t.Errorf("Expected user ID '%s', got '%s'", tt.expectedUserID, service.userID)
			}

			if tt.expectedToken != "" && service.authToken != tt.expectedToken {
				t.Errorf("Expected auth token '%s', got '%s'", tt.expectedToken, service.authToken)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedBotName != "" && service.botName != tt.expectedBotName {
				t.Errorf("Expected bot name '%s', got '%s'", tt.expectedBotName, service.botName)
			}
		})
	}
}

func TestRocketChatService_NormalizeChannel(t *testing.T) {
	service := &RocketChatService{}

	tests := []struct {
		input    string
		expected string
	}{
		// Already normalized
		{"@user", "@user"},
		{"#channel", "#channel"},
		{"room_id_123abc", "room_id_123abc"},

		// Simple channel name - should get # prefix
		{"general", "#general"},
		{"support", "#support"},
		{"team-dev", "#team-dev"},

		// Whitespace handling
		{" general ", "#general"},
		{" @user ", "@user"},
		{" #channel ", "#channel"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.normalizeChannel(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeChannel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRocketChatService_TestURL(t *testing.T) {
	service := NewRocketChatService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Rocket.Chat URL with username/password",
			url:         "rockets://user:pass@chat.example.com/general",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "rockets://chat.example.com/hooks/abc/def?channel=alerts",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://user:pass@chat.example.com/general",
			expectError: true,
		},
		{
			name:        "Valid webhook URL without authentication",
			url:         "rockets://chat.example.com/hooks/webhook_id/token",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRocketChatService_SendWebhook(t *testing.T) {
	// Create mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse and verify request body
		var message RocketChatMessage
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify message structure
		if message.Channel != "#test" {
			t.Errorf("Expected channel '#test', got '%s'", message.Channel)
		}

		if message.Username != "TestBot" {
			t.Errorf("Expected username 'TestBot', got '%s'", message.Username)
		}

		if len(message.Attachments) == 0 {
			t.Error("Expected attachments to be present")
		}

		if len(message.Attachments) > 0 {
			attachment := message.Attachments[0]
			if attachment.Title == "" {
				t.Error("Expected attachment title to be set")
			}
			if attachment.Color == "" {
				t.Error("Expected attachment color to be set")
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewRocketChatService().(*RocketChatService)
	service.webhookURL = server.URL
	service.channel = "#test"
	service.botName = "TestBot"

	// Test different notification types
	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
	}{
		{
			name:       "Info notification",
			title:      "Test Info",
			body:       "This is an info notification",
			notifyType: NotifyTypeInfo,
		},
		{
			name:       "Success notification",
			title:      "Test Success",
			body:       "This is a success notification",
			notifyType: NotifyTypeSuccess,
		},
		{
			name:       "Warning notification",
			title:      "Test Warning",
			body:       "This is a warning notification",
			notifyType: NotifyTypeWarning,
		},
		{
			name:       "Error notification",
			title:      "Test Error",
			body:       "This is an error notification",
			notifyType: NotifyTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{
				Title:      tt.title,
				Body:       tt.body,
				NotifyType: tt.notifyType,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := service.Send(ctx, req)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}
		})
	}
}

func TestRocketChatService_SendAPI(t *testing.T) {
	// Create mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/login") {
			// Mock login endpoint
			if r.Method != "POST" {
				t.Errorf("Expected POST for login, got %s", r.Method)
			}

			var loginReq RocketChatLoginRequest
			if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
				t.Fatalf("Failed to decode login request: %v", err)
			}

			if loginReq.User != "testuser" || loginReq.Password != "testpass" {
				t.Error("Invalid login credentials")
			}

			response := RocketChatLoginResponse{
				Status: "success",
				Data: RocketChatLoginData{
					UserID:    "test_user_id",
					AuthToken: "test_auth_token",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)

		} else if strings.HasSuffix(r.URL.Path, "/chat.postMessage") {
			// Mock message posting endpoint
			if r.Method != "POST" {
				t.Errorf("Expected POST for message, got %s", r.Method)
			}

			// Verify authentication headers
			if r.Header.Get("X-User-Id") != "test_user_id" {
				t.Errorf("Expected X-User-Id 'test_user_id', got '%s'", r.Header.Get("X-User-Id"))
			}

			if r.Header.Get("X-Auth-Token") != "test_auth_token" {
				t.Errorf("Expected X-Auth-Token 'test_auth_token', got '%s'", r.Header.Get("X-Auth-Token"))
			}

			var messageReq RocketChatPostMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&messageReq); err != nil {
				t.Fatalf("Failed to decode message request: %v", err)
			}

			// Verify message structure
			if messageReq.Channel != "#test" {
				t.Errorf("Expected channel '#test', got '%s'", messageReq.Channel)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success": true}`))

		} else {
			t.Errorf("Unexpected API endpoint: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	// Configure service for API
	service := NewRocketChatService().(*RocketChatService)
	service.server = server.URL
	service.channel = "#test"
	service.username = "testuser"
	service.password = "testpass"

	req := NotificationRequest{
		Title:      "API Test",
		Body:       "Testing API authentication and message sending",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("API Send failed: %v", err)
	}
}

func TestRocketChatService_CreateAttachment(t *testing.T) {
	service := &RocketChatService{
		botName: "TestBot",
	}

	req := NotificationRequest{
		Title:      "Test Message",
		Body:       "Test body content",
		NotifyType: NotifyTypeWarning,
		Tags:       []string{"test", "automation"},
	}

	attachment := service.createAttachment(req)

	// Check basic fields
	if attachment.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, attachment.Title)
	}

	if attachment.Text != req.Body {
		t.Errorf("Expected text '%s', got '%s'", req.Body, attachment.Text)
	}

	if attachment.Color != "warning" {
		t.Errorf("Expected warning color, got '%s'", attachment.Color)
	}

	// Check author
	if attachment.AuthorName != "Apprise-Go" {
		t.Errorf("Expected author name 'Apprise-Go', got '%s'", attachment.AuthorName)
	}

	// Check fields
	if len(attachment.Fields) < 2 {
		t.Errorf("Expected at least 2 fields, got %d", len(attachment.Fields))
	}

	// Find type field
	typeFieldFound := false
	for _, field := range attachment.Fields {
		if field.Title == "Type" && field.Value == "WARNING" {
			typeFieldFound = true
			break
		}
	}
	if !typeFieldFound {
		t.Error("Expected to find Type field with value WARNING")
	}

	// Find tags field
	tagsFieldFound := false
	for _, field := range attachment.Fields {
		if field.Title == "Tags" && field.Value == "test, automation" {
			tagsFieldFound = true
			break
		}
	}
	if !tagsFieldFound {
		t.Error("Expected to find Tags field with correct value")
	}
}

func TestRocketChatService_HelperMethods(t *testing.T) {
	service := &RocketChatService{}

	// Test color mapping
	tests := []struct {
		notifyType    NotifyType
		expectedColor string
		expectedEmoji string
	}{
		{NotifyTypeInfo, "#439FE0", ":information_source:"},
		{NotifyTypeSuccess, "good", ":white_check_mark:"},
		{NotifyTypeWarning, "warning", ":warning:"},
		{NotifyTypeError, "danger", ":exclamation:"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if color := service.getColorForNotifyType(tt.notifyType); color != tt.expectedColor {
				t.Errorf("Expected color '%s', got '%s'", tt.expectedColor, color)
			}

			if emoji := service.getEmojiForNotifyType(tt.notifyType); emoji != tt.expectedEmoji {
				t.Errorf("Expected emoji '%s', got '%s'", tt.expectedEmoji, emoji)
			}

			// Test that icon URL is returned (just check it's not empty)
			if icon := service.getIconForNotifyType(tt.notifyType); icon == "" {
				t.Error("Expected icon URL to be non-empty")
			}
		})
	}
}

func TestRocketChatService_WithAttachments(t *testing.T) {
	service := &RocketChatService{
		botName: "TestBot",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test image data"), "test.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This message has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	attachment := service.createAttachment(req)

	// Should have attachment information in fields
	attachmentFieldFound := false
	for _, field := range attachment.Fields {
		if field.Title == "Attachment" && strings.Contains(field.Value, "test.jpg") {
			attachmentFieldFound = true
			break
		}
	}

	if !attachmentFieldFound {
		t.Error("Expected to find attachment field with file information")
	}
}
