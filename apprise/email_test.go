package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestEmailService_GetServiceID(t *testing.T) {
	service := NewEmailService()
	if service.GetServiceID() != "email" {
		t.Errorf("Expected service ID 'email', got '%s'", service.GetServiceID())
	}
}

func TestEmailService_GetDefaultPort(t *testing.T) {
	service := NewEmailService()
	if service.GetDefaultPort() != 587 {
		t.Errorf("Expected default port 587, got %d", service.GetDefaultPort())
	}
}

func TestEmailService_ParseURL(t *testing.T) {
	testCases := []struct {
		name         string
		url          string
		expectError  bool
		expectedTLS  bool
		expectedPort int
	}{
		{
			name:         "Basic mailto",
			url:          "mailto://user:pass@smtp.gmail.com/to@domain.com",
			expectError:  false,
			expectedTLS:  false,
			expectedPort: 587,
		},
		{
			name:         "Secure mailtos",
			url:          "mailtos://user:pass@smtp.gmail.com/to@domain.com",
			expectError:  false,
			expectedTLS:  true,
			expectedPort: 465,
		},
		{
			name:         "With port",
			url:          "mailto://user:pass@smtp.server.com:2525/to@domain.com",
			expectError:  false,
			expectedTLS:  false,
			expectedPort: 2525,
		},
		{
			name:        "Invalid scheme",
			url:         "http://user:pass@server.com/to@domain.com",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "mailto://user:pass@/to@domain.com",
			expectError: true,
		},
		{
			name:        "No recipients",
			url:         "mailto://user:pass@smtp.server.com",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewEmailService().(*EmailService)
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.useTLS != tc.expectedTLS {
				t.Errorf("Expected TLS %v, got %v", tc.expectedTLS, service.useTLS)
			}

			if service.smtpPort != tc.expectedPort {
				t.Errorf("Expected port %d, got %d", tc.expectedPort, service.smtpPort)
			}
		})
	}
}

func TestEmailService_ParseURL_QueryParams(t *testing.T) {
	testURL := "mailto://user:pass@smtp.server.com/to@domain.com?from=sender@domain.com&cc=cc@domain.com&bcc=bcc@domain.com&subject=Test&skip_verify=true&no_tls=true"

	service := NewEmailService().(*EmailService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.fromEmail != "sender@domain.com" {
		t.Errorf("Expected from email 'sender@domain.com', got '%s'", service.fromEmail)
	}

	if len(service.ccEmails) != 1 || service.ccEmails[0] != "cc@domain.com" {
		t.Errorf("Expected CC email 'cc@domain.com', got %v", service.ccEmails)
	}

	if len(service.bccEmails) != 1 || service.bccEmails[0] != "bcc@domain.com" {
		t.Errorf("Expected BCC email 'bcc@domain.com', got %v", service.bccEmails)
	}

	if service.subject != "Test" {
		t.Errorf("Expected subject 'Test', got '%s'", service.subject)
	}

	if !service.skipVerify {
		t.Error("Expected skipVerify to be true")
	}

	if service.useSTARTTLS {
		t.Error("Expected useSTARTTLS to be false")
	}
}

func TestEmailService_TestURL(t *testing.T) {
	service := NewEmailService()

	validURLs := []string{
		"mailto://user:pass@smtp.gmail.com/to@domain.com",
		"mailtos://user:pass@smtp.server.com:465/recipient@example.com",
		"mailto://user:pass@smtp.server.com/to1@domain.com/to2@domain.com",
	}

	for _, testURL := range validURLs {
		t.Run("Valid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", testURL, err)
			}
		})
	}

	invalidURLs := []string{
		"http://user:pass@server.com/to@domain.com",
		"mailto://user:pass@/to@domain.com", // Missing host
	}

	for _, testURL := range invalidURLs {
		t.Run("Invalid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", testURL)
			}
		})
	}
}

func TestEmailService_Properties(t *testing.T) {
	service := NewEmailService()

	if service.SupportsAttachments() {
		t.Error("Email service should not support attachments yet")
	}

	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected unlimited body length, got %d", service.GetMaxBodyLength())
	}
}

func TestEmailService_Send_InvalidConfig(t *testing.T) {
	service := NewEmailService().(*EmailService)

	// Service without proper configuration should fail
	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected Send to fail with invalid configuration")
	}
}

func TestEmailService_isValidEmail(t *testing.T) {
	service := NewEmailService().(*EmailService)

	validEmails := []string{
		"test@example.com",
		"user.name@domain.co.uk",
		"user+tag@example.org",
	}

	for _, email := range validEmails {
		if !service.isValidEmail(email) {
			t.Errorf("Expected '%s' to be valid email", email)
		}
	}

	invalidEmails := []string{
		"invalid",
		"@example.com",
		"user@",
		"",
	}

	for _, email := range invalidEmails {
		if service.isValidEmail(email) {
			t.Errorf("Expected '%s' to be invalid email", email)
		}
	}
}

func TestEmailService_getEmojiForNotifyType(t *testing.T) {
	service := NewEmailService().(*EmailService)

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, "✅"},
		{NotifyTypeWarning, "⚠️"},
		{NotifyTypeError, "❌"},
		{NotifyTypeInfo, "ℹ️"},
	}

	for _, test := range tests {
		result := service.getEmojiForNotifyType(test.notifyType)
		if result != test.expected {
			t.Errorf("Expected emoji '%s' for %v, got '%s'", test.expected, test.notifyType, result)
		}
	}
}

func TestEmailService_formatMessageBody(t *testing.T) {
	service := NewEmailService().(*EmailService)

	// Test plain text format
	plainResult := service.formatMessageBody("Test Title", "Test Body", NotifyTypeInfo, "text")
	if plainResult == "" {
		t.Error("Plain text format should not be empty")
	}

	// Test HTML format
	htmlResult := service.formatMessageBody("Test Title", "Test Body", NotifyTypeInfo, "html")
	if htmlResult == "" {
		t.Error("HTML format should not be empty")
	}

	// HTML should contain HTML tags
	if !containsHTMLTags(htmlResult, "<html>") || !containsHTMLTags(htmlResult, "</html>") {
		t.Error("HTML format should contain HTML tags")
	}
}

func containsHTMLTags(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
