package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestMatrixService_GetServiceID(t *testing.T) {
	service := NewMatrixService()
	if service.GetServiceID() != "matrix" {
		t.Errorf("Expected service ID 'matrix', got %q", service.GetServiceID())
	}
}

func TestMatrixService_GetDefaultPort(t *testing.T) {
	service := NewMatrixService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestMatrixService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedToken    string
		expectedUsername string
		expectedPassword string
		expectedHomeserver string
		expectedRooms    []string
		expectedMsgType  string
		expectedHtmlFormat bool
	}{
		{
			name:               "Basic token authentication",
			url:                "matrix://access_token@matrix.org/!room:matrix.org",
			expectError:        false,
			expectedToken:      "access_token",
			expectedHomeserver: "https://matrix.org",
			expectedRooms:      []string{"!room:matrix.org"},
			expectedMsgType:    "m.text",
		},
		{
			name:               "Username password authentication",
			url:                "matrix://user:pass@matrix.example.com/general",
			expectError:        false,
			expectedUsername:   "user",
			expectedPassword:   "pass",
			expectedHomeserver: "https://matrix.example.com",
			expectedRooms:      []string{"#general:matrix.example.com"},
			expectedMsgType:    "m.text",
		},
		{
			name:               "Room alias format",
			url:                "matrix://token@matrix.org/#general:matrix.org",
			expectError:        false,
			expectedToken:      "token",
			expectedHomeserver: "https://matrix.org",
			expectedRooms:      []string{"#general:matrix.org"},
		},
		{
			name:               "Multiple rooms",
			url:                "matrix://token@matrix.org/room1/room2/#room3:matrix.org",
			expectError:        false,
			expectedToken:      "token",
			expectedHomeserver: "https://matrix.org",
			expectedRooms:      []string{"#room1:matrix.org", "#room2:matrix.org", "#room3:matrix.org"},
		},
		{
			name:               "Query parameters",
			url:                "matrix://token@matrix.org/general?msgtype=notice&format=html",
			expectError:        false,
			expectedToken:      "token",
			expectedHomeserver: "https://matrix.org",
			expectedRooms:      []string{"#general:matrix.org"},
			expectedMsgType:    "m.notice",
			expectedHtmlFormat: true,
		},
		{
			name:               "Token in query parameter",
			url:                "matrix://user@matrix.org/general?token=access_token",
			expectError:        false,
			expectedToken:      "access_token",
			expectedUsername:   "user",
			expectedHomeserver: "https://matrix.org",
			expectedRooms:      []string{"#general:matrix.org"},
		},
		{
			name:        "Invalid scheme",
			url:         "http://token@matrix.org/room",
			expectError: true,
		},
		{
			name:        "Missing homeserver",
			url:         "matrix://token/room",
			expectError: true,
		},
		{
			name:        "Missing authentication",
			url:         "matrix://@matrix.org/room",
			expectError: true,
		},
		{
			name:        "Missing rooms",
			url:         "matrix://token@matrix.org",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMatrixService().(*MatrixService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				return
			}

			if tt.expectedToken != "" && service.accessToken != tt.expectedToken {
				t.Errorf("Expected token %q, got %q", tt.expectedToken, service.accessToken)
			}

			if tt.expectedUsername != "" && service.username != tt.expectedUsername {
				t.Errorf("Expected username %q, got %q", tt.expectedUsername, service.username)
			}

			if tt.expectedPassword != "" && service.password != tt.expectedPassword {
				t.Errorf("Expected password %q, got %q", tt.expectedPassword, service.password)
			}

			if service.homeserver != tt.expectedHomeserver {
				t.Errorf("Expected homeserver %q, got %q", tt.expectedHomeserver, service.homeserver)
			}

			if !stringSlicesEqual(service.rooms, tt.expectedRooms) {
				t.Errorf("Expected rooms %v, got %v", tt.expectedRooms, service.rooms)
			}

			if tt.expectedMsgType != "" && service.msgType != tt.expectedMsgType {
				t.Errorf("Expected msgType %q, got %q", tt.expectedMsgType, service.msgType)
			}

			if service.htmlFormat != tt.expectedHtmlFormat {
				t.Errorf("Expected htmlFormat %v, got %v", tt.expectedHtmlFormat, service.htmlFormat)
			}
		})
	}
}

func TestMatrixService_NormalizeRoomID(t *testing.T) {
	service := &MatrixService{
		homeserver: "https://matrix.example.com",
	}

	tests := []struct {
		input    string
		expected string
	}{
		// Already normalized
		{"!room:matrix.org", "!room:matrix.org"},
		{"#room:matrix.org", "#room:matrix.org"},
		
		// Simple room name - should become alias
		{"general", "#general:matrix.example.com"},
		{"support", "#support:matrix.example.com"},
		
		// Room with server already specified
		{"room:other.com", "room:other.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.normalizeRoomID(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRoomID(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatrixService_TestURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid matrix://token@matrix.org/room",
			url:         "matrix://access_token@matrix.org/general",
			expectError: false,
		},
		{
			name:        "Valid matrix://user:pass@server/room",
			url:         "matrix://user:password@matrix.example.com/support",
			expectError: false,
		},
		{
			name:        "Valid matrix://token@server/!room_id:server",
			url:         "matrix://token@matrix.org/!AbCdEf:matrix.org",
			expectError: false,
		},
		{
			name:        "Valid matrix://token@server/#room_alias:server",
			url:         "matrix://token@matrix.org/#general:matrix.org",
			expectError: false,
		},
		{
			name:        "Valid with query parameters",
			url:         "matrix://token@matrix.org/room?msgtype=notice&format=html",
			expectError: false,
		},
		{
			name:        "Invalid http://token@server/room",
			url:         "http://token@matrix.org/room",
			expectError: true,
		},
		{
			name:        "Invalid matrix://token/room (no server)",
			url:         "matrix://token/room",
			expectError: true,
		},
		{
			name:        "Invalid matrix://@server/room (no auth)",
			url:         "matrix://@matrix.org/room",
			expectError: true,
		},
		{
			name:        "Invalid matrix://token@server (no room)",
			url:         "matrix://token@matrix.org",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMatrixService()
			err := service.TestURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				}
			}
		})
	}
}

func TestMatrixService_Properties(t *testing.T) {
	service := NewMatrixService()

	if !service.SupportsAttachments() {
		t.Error("Matrix should support attachments")
	}

	expectedMaxLength := 32768
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestMatrixService_FormatMessage(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		body           string
		msgType        string
		htmlFormat     bool
		expectedBody   string
		expectedFormat string
		expectFormatted bool
	}{
		{
			name:         "Title and body - text format",
			title:        "Alert",
			body:         "System error occurred",
			msgType:      "m.text",
			htmlFormat:   false,
			expectedBody: "Alert\nSystem error occurred",
		},
		{
			name:         "Title only - text format",
			title:        "Alert",
			body:         "",
			msgType:      "m.text",
			htmlFormat:   false,
			expectedBody: "Alert",
		},
		{
			name:         "Body only - text format",
			title:        "",
			body:         "System error occurred",
			msgType:      "m.text",
			htmlFormat:   false,
			expectedBody: "System error occurred",
		},
		{
			name:            "Title and body - HTML format",
			title:           "Alert",
			body:            "System error occurred",
			msgType:         "m.text",
			htmlFormat:      true,
			expectedBody:    "Alert\nSystem error occurred",
			expectedFormat:  "org.matrix.custom.html",
			expectFormatted: true,
		},
		{
			name:            "Title only - HTML format",
			title:           "Alert",
			body:            "",
			msgType:         "m.text",
			htmlFormat:      true,
			expectedBody:    "Alert",
			expectedFormat:  "org.matrix.custom.html",
			expectFormatted: true,
		},
		{
			name:         "Notice message type",
			title:        "Notice",
			body:         "System maintenance",
			msgType:      "m.notice",
			htmlFormat:   false,
			expectedBody: "Notice\nSystem maintenance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MatrixService{
				msgType:    tt.msgType,
				htmlFormat: tt.htmlFormat,
			}

			message := service.formatMessage(tt.title, tt.body)

			if message.Body != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, message.Body)
			}

			if message.MsgType != tt.msgType {
				t.Errorf("Expected msgType %q, got %q", tt.msgType, message.MsgType)
			}

			if tt.expectFormatted {
				if message.Format != tt.expectedFormat {
					t.Errorf("Expected format %q, got %q", tt.expectedFormat, message.Format)
				}
				if message.FormattedBody == "" {
					t.Error("Expected formatted body to be set")
				}
			} else {
				if message.Format != "" {
					t.Errorf("Expected no format, got %q", message.Format)
				}
				if message.FormattedBody != "" {
					t.Errorf("Expected no formatted body, got %q", message.FormattedBody)
				}
			}
		})
	}
}

func TestMatrixService_EscapeHTML(t *testing.T) {
	service := &MatrixService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"A & B", "A &amp; B"},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"<tag attr=\"value\">", "&lt;tag attr=&quot;value&quot;&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatrixService_Send_InvalidConfig(t *testing.T) {
	service := NewMatrixService()
	parsedURL, _ := url.Parse("matrix://test_token@matrix.example.com/testroom")
	_ = service.(*MatrixService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real Matrix server
	if err == nil {
		t.Error("Expected network error for invalid Matrix configuration, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "matrix") &&
		!strings.Contains(err.Error(), "Matrix") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "login") {
		t.Errorf("Error should be network-related or login error, got: %v", err)
	}
}

func TestMatrixService_PayloadGeneration(t *testing.T) {
	service := NewMatrixService()
	parsedURL, _ := url.Parse("matrix://test_token@matrix.example.com/general?msgtype=notice&format=html")
	_ = service.(*MatrixService).ParseURL(parsedURL)

	matrixService := service.(*MatrixService)

	// Test that configuration is parsed correctly
	if matrixService.accessToken != "test_token" {
		t.Error("Expected access token to be set from URL")
	}

	if matrixService.homeserver != "https://matrix.example.com" {
		t.Error("Expected homeserver to be parsed correctly")
	}

	if len(matrixService.rooms) != 1 || matrixService.rooms[0] != "#general:matrix.example.com" {
		t.Errorf("Expected room to be normalized correctly, got: %v", matrixService.rooms)
	}

	if matrixService.msgType != "m.notice" {
		t.Error("Expected message type to be set from query parameter")
	}

	if !matrixService.htmlFormat {
		t.Error("Expected HTML format to be enabled from query parameter")
	}
}

