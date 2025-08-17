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

func TestAWSSESService_GetServiceID(t *testing.T) {
	service := NewAWSSESService()
	if service.GetServiceID() != "ses" {
		t.Errorf("Expected service ID 'ses', got '%s'", service.GetServiceID())
	}
}

func TestAWSSESService_GetDefaultPort(t *testing.T) {
	service := NewAWSSESService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAWSSESService_SupportsAttachments(t *testing.T) {
	service := NewAWSSESService()
	if !service.SupportsAttachments() {
		t.Error("AWS SES should support attachments")
	}
}

func TestAWSSESService_GetMaxBodyLength(t *testing.T) {
	service := NewAWSSESService()
	expected := 10 * 1024 * 1024 // 10MB
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestAWSSESService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedFrom     string
		expectedTo       []string
		expectedCC       []string
		expectedBCC      []string
		expectedAPIKey   string
		expectedRegion   string
		expectedTemplate string
	}{
		{
			name:         "Valid basic SES URL",
			url:          "ses://api.example.com/ses?from=sender@example.com&to=recipient@example.com",
			expectError:  false,
			expectedFrom: "sender@example.com",
			expectedTo:   []string{"recipient@example.com"},
		},
		{
			name:         "Valid SES URL with multiple recipients",
			url:          "ses://webhook.example.com/ses?from=alerts@company.com&to=admin@company.com,team@company.com&cc=manager@company.com",
			expectError:  false,
			expectedFrom: "alerts@company.com",
			expectedTo:   []string{"admin@company.com", "team@company.com"},
			expectedCC:   []string{"manager@company.com"},
		},
		{
			name:             "Valid SES URL with API key and all options",
			url:              "ses://api-key@api.gateway.com/prod/ses?from=noreply@company.com&to=alerts@company.com&cc=team@company.com&bcc=audit@company.com&region=eu-west-1&template=alert-template",
			expectError:      false,
			expectedAPIKey:   "api-key",
			expectedFrom:     "noreply@company.com",
			expectedTo:       []string{"alerts@company.com"},
			expectedCC:       []string{"team@company.com"},
			expectedBCC:      []string{"audit@company.com"},
			expectedRegion:   "eu-west-1",
			expectedTemplate: "alert-template",
		},
		{
			name:        "Invalid scheme",
			url:         "http://api.example.com/ses?from=test@example.com&to=test@example.com",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "ses:///webhook?from=test@example.com&to=test@example.com",
			expectError: true,
		},
		{
			name:        "Missing from parameter",
			url:         "ses://api.example.com/ses?to=recipient@example.com",
			expectError: true,
		},
		{
			name:        "Missing to parameter",
			url:         "ses://api.example.com/ses?from=sender@example.com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAWSSESService().(*AWSSESService)
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

			if tt.expectedFrom != "" && service.fromEmail != tt.expectedFrom {
				t.Errorf("Expected from email '%s', got '%s'", tt.expectedFrom, service.fromEmail)
			}

			if len(tt.expectedTo) > 0 {
				if len(service.toEmails) != len(tt.expectedTo) {
					t.Errorf("Expected %d to emails, got %d", len(tt.expectedTo), len(service.toEmails))
				}
				for i, expected := range tt.expectedTo {
					if i < len(service.toEmails) && service.toEmails[i] != expected {
						t.Errorf("Expected to email %d '%s', got '%s'", i, expected, service.toEmails[i])
					}
				}
			}

			if len(tt.expectedCC) > 0 {
				if len(service.ccEmails) != len(tt.expectedCC) {
					t.Errorf("Expected %d CC emails, got %d", len(tt.expectedCC), len(service.ccEmails))
				}
			}

			if len(tt.expectedBCC) > 0 {
				if len(service.bccEmails) != len(tt.expectedBCC) {
					t.Errorf("Expected %d BCC emails, got %d", len(tt.expectedBCC), len(service.bccEmails))
				}
			}

			if tt.expectedAPIKey != "" && service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}

			if tt.expectedRegion != "" && service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
			}

			if tt.expectedTemplate != "" && service.template != tt.expectedTemplate {
				t.Errorf("Expected template '%s', got '%s'", tt.expectedTemplate, service.template)
			}
		})
	}
}

func TestAWSSESService_TestURL(t *testing.T) {
	service := NewAWSSESService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid SES URL",
			url:         "ses://api.example.com/webhook?from=test@example.com&to=recipient@example.com",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://api.example.com/webhook?from=test@example.com&to=recipient@example.com",
			expectError: true,
		},
		{
			name:        "Missing required parameters",
			url:         "ses://api.example.com/webhook",
			expectError: true,
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

func TestAWSSESService_Send(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse and verify request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload["source"] == "" {
			t.Error("Expected source (from) in payload")
		}

		if payload["destination"] == nil {
			t.Error("Expected destination in payload")
		}

		if payload["message"] == nil {
			t.Error("Expected message in payload")
		}

		// Verify destination structure
		destination, ok := payload["destination"].(map[string]interface{})
		if !ok {
			t.Error("Expected destination to be an object")
		} else {
			if destination["toAddresses"] == nil {
				t.Error("Expected toAddresses in destination")
			}
		}

		// Verify message structure
		message, ok := payload["message"].(map[string]interface{})
		if !ok {
			t.Error("Expected message to be an object")
		} else {
			if message["subject"] == nil {
				t.Error("Expected subject in message")
			}
			if message["body"] == nil {
				t.Error("Expected body in message")
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MessageId":"12345-67890-abcdef"}`))
	}))
	defer server.Close()

	// Parse server URL and create SES URL with test_mode=true for HTTP
	serverURL, _ := url.Parse(server.URL)
	sesURL := "ses://" + serverURL.Host + "/webhook?from=test@example.com&to=recipient@example.com&test_mode=true"

	service := NewAWSSESService().(*AWSSESService)
	parsedURL, _ := url.Parse(sesURL)
	_ = service.ParseURL(parsedURL)

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

func TestAWSSESService_SendWithAPIKey(t *testing.T) {
	// Create mock server that checks for API key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		authHeader := r.Header.Get("Authorization")

		if apiKey != "test-api-key" {
			t.Errorf("Expected X-API-Key 'test-api-key', got '%s'", apiKey)
		}

		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization 'Bearer test-api-key', got '%s'", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"MessageId":"test-message-id"}`))
	}))
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	sesURL := "ses://test-api-key@" + serverURL.Host + "/webhook?from=test@example.com&to=recipient@example.com&test_mode=true"

	service := NewAWSSESService().(*AWSSESService)
	parsedURL, _ := url.Parse(sesURL)
	_ = service.ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "API Key Test",
		Body:       "Testing API key authentication",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send with API key failed: %v", err)
	}
}

func TestAWSSESService_FormatMessageBody(t *testing.T) {
	service := &AWSSESService{}

	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
		checkHTML  bool
		checkText  bool
	}{
		{
			name:       "Info message",
			title:      "Test Title",
			body:       "Test Body",
			notifyType: NotifyTypeInfo,
			checkHTML:  true,
			checkText:  true,
		},
		{
			name:       "Error message with HTML characters",
			title:      "Error <critical>",
			body:       "Failed to process: <error> & \"warning\"",
			notifyType: NotifyTypeError,
			checkHTML:  true,
			checkText:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, text := service.formatMessageBody(tt.title, tt.body, tt.notifyType)

			if tt.checkHTML {
				if !strings.Contains(html, "<!DOCTYPE html>") {
					t.Error("HTML message should contain DOCTYPE declaration")
				}
				if !strings.Contains(html, service.getEmojiForNotifyType(tt.notifyType)) {
					t.Error("HTML message should contain appropriate emoji")
				}
				// Check that HTML characters are escaped
				if tt.title == "Error <critical>" && strings.Contains(html, "<critical>") {
					t.Error("HTML message should escape HTML characters")
				}
			}

			if tt.checkText {
				if !strings.Contains(text, service.getEmojiForNotifyType(tt.notifyType)) {
					t.Error("Text message should contain appropriate emoji")
				}
				if !strings.Contains(text, "Apprise-Go") {
					t.Error("Text message should contain signature")
				}
			}
		})
	}
}

func TestAWSSESService_BuildFromEmail(t *testing.T) {
	tests := []struct {
		name     string
		service  *AWSSESService
		expected string
	}{
		{
			name:     "Email without name",
			service:  &AWSSESService{fromEmail: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "Email with name",
			service:  &AWSSESService{fromEmail: "test@example.com", fromName: "Test Sender"},
			expected: "Test Sender <test@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.service.buildFromEmail()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAWSSESService_BuildDestination(t *testing.T) {
	service := &AWSSESService{
		toEmails:  []string{"to1@example.com", "to2@example.com"},
		ccEmails:  []string{"cc@example.com"},
		bccEmails: []string{"bcc@example.com"},
	}

	destination := service.buildDestination()

	// Check TO addresses
	toAddresses, ok := destination["toAddresses"].([]string)
	if !ok {
		t.Error("Expected toAddresses to be []string")
	} else if len(toAddresses) != 2 {
		t.Errorf("Expected 2 toAddresses, got %d", len(toAddresses))
	}

	// Check CC addresses
	ccAddresses, ok := destination["ccAddresses"].([]string)
	if !ok {
		t.Error("Expected ccAddresses to be []string")
	} else if len(ccAddresses) != 1 {
		t.Errorf("Expected 1 ccAddresses, got %d", len(ccAddresses))
	}

	// Check BCC addresses
	bccAddresses, ok := destination["bccAddresses"].([]string)
	if !ok {
		t.Error("Expected bccAddresses to be []string")
	} else if len(bccAddresses) != 1 {
		t.Errorf("Expected 1 bccAddresses, got %d", len(bccAddresses))
	}
}

func TestAWSSESService_ParseEmailList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single email",
			input:    "test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "Multiple emails",
			input:    "test1@example.com,test2@example.com,test3@example.com",
			expected: []string{"test1@example.com", "test2@example.com", "test3@example.com"},
		},
		{
			name:     "Emails with spaces",
			input:    "test1@example.com, test2@example.com , test3@example.com",
			expected: []string{"test1@example.com", "test2@example.com", "test3@example.com"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Comma only",
			input:    ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEmailList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d emails, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected email %d '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}
