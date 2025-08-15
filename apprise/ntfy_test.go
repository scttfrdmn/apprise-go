package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestNtfyService_GetServiceID(t *testing.T) {
	service := NewNtfyService()
	if service.GetServiceID() != "ntfy" {
		t.Errorf("Expected service ID 'ntfy', got %q", service.GetServiceID())
	}
}

func TestNtfyService_GetDefaultPort(t *testing.T) {
	service := NewNtfyService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestNtfyService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedBaseURL  string
		expectedTopic    string
		expectedUsername string
		expectedPassword string
		expectedToken    string
		expectedPriority int
		expectedTags     []string
		expectedDelay    string
		expectedActions  []string
		expectedAttach   string
		expectedFilename string
		expectedClick    string
		expectedEmail    string
	}{
		{
			name:             "Basic HTTPS ntfy.sh",
			url:              "ntfys://ntfy.sh/my-topic",
			expectError:      false,
			expectedBaseURL:  "https://ntfy.sh:443",
			expectedTopic:    "my-topic",
			expectedPriority: 3,
		},
		{
			name:             "Basic HTTP self-hosted",
			url:              "ntfy://ntfy.example.com:8080/alerts",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.example.com:8080",
			expectedTopic:    "alerts",
			expectedPriority: 3,
		},
		{
			name:             "Username/password authentication",
			url:              "ntfy://user:pass@ntfy.example.com/notifications",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.example.com:80",
			expectedTopic:    "notifications",
			expectedUsername: "user",
			expectedPassword: "pass",
			expectedPriority: 3,
		},
		{
			name:             "Token authentication",
			url:              "ntfys://token123@ntfy.sh/alerts",
			expectError:      false,
			expectedBaseURL:  "https://ntfy.sh:443",
			expectedTopic:    "alerts",
			expectedToken:    "token123",
			expectedPriority: 3,
		},
		{
			name:             "With priority and tags",
			url:              "ntfy://ntfy.sh/alerts?priority=5&tags=urgent,production",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.sh:80",
			expectedTopic:    "alerts",
			expectedPriority: 5,
			expectedTags:     []string{"urgent", "production"},
		},
		{
			name:             "With delay and email",
			url:              "ntfy://ntfy.sh/alerts?delay=30min&email=admin@example.com",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.sh:80",
			expectedTopic:    "alerts",
			expectedPriority: 3,
			expectedDelay:    "30min",
			expectedEmail:    "admin@example.com",
		},
		{
			name:             "With attachment and filename",
			url:              "ntfy://ntfy.sh/alerts?attach=https://example.com/file.pdf&filename=report.pdf",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.sh:80",
			expectedTopic:    "alerts",
			expectedPriority: 3,
			expectedAttach:   "https://example.com/file.pdf",
			expectedFilename: "report.pdf",
		},
		{
			name:             "With click URL and actions",
			url:              "ntfy://ntfy.sh/alerts?click=https://dashboard.example.com&actions=view,View Dashboard,https://dashboard.example.com",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.sh:80",
			expectedTopic:    "alerts",
			expectedPriority: 3,
			expectedClick:    "https://dashboard.example.com",
			expectedActions:  []string{"view", "View Dashboard", "https://dashboard.example.com"},
		},
		{
			name:             "Token in query parameter",
			url:              "ntfy://ntfy.sh/alerts?token=mytoken123",
			expectError:      false,
			expectedBaseURL:  "http://ntfy.sh:80",
			expectedTopic:    "alerts",
			expectedToken:    "mytoken123",
			expectedPriority: 3,
		},
		{
			name:        "Invalid scheme",
			url:         "http://ntfy.sh/topic",
			expectError: true,
		},
		{
			name:        "Missing host",
			url:         "ntfy:///topic",
			expectError: true,
		},
		{
			name:        "Missing topic",
			url:         "ntfy://ntfy.sh",
			expectError: true,
		},
		{
			name:        "Empty topic",
			url:         "ntfy://ntfy.sh/",
			expectError: true,
		},
		{
			name:        "Invalid priority (too low)",
			url:         "ntfy://ntfy.sh/topic?priority=0",
			expectError: true,
		},
		{
			name:        "Invalid priority (too high)",
			url:         "ntfy://ntfy.sh/topic?priority=6",
			expectError: true,
		},
		{
			name:        "Invalid priority (not number)",
			url:         "ntfy://ntfy.sh/topic?priority=high",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNtfyService().(*NtfyService)
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

			if service.baseURL != tt.expectedBaseURL {
				t.Errorf("Expected baseURL %q, got %q", tt.expectedBaseURL, service.baseURL)
			}

			if service.topic != tt.expectedTopic {
				t.Errorf("Expected topic %q, got %q", tt.expectedTopic, service.topic)
			}

			if tt.expectedUsername != "" && service.username != tt.expectedUsername {
				t.Errorf("Expected username %q, got %q", tt.expectedUsername, service.username)
			}

			if tt.expectedPassword != "" && service.password != tt.expectedPassword {
				t.Errorf("Expected password %q, got %q", tt.expectedPassword, service.password)
			}

			if tt.expectedToken != "" && service.token != tt.expectedToken {
				t.Errorf("Expected token %q, got %q", tt.expectedToken, service.token)
			}

			if service.priority != tt.expectedPriority {
				t.Errorf("Expected priority %d, got %d", tt.expectedPriority, service.priority)
			}

			if tt.expectedTags != nil && !stringSlicesEqual(service.tags, tt.expectedTags) {
				t.Errorf("Expected tags %v, got %v", tt.expectedTags, service.tags)
			}

			if tt.expectedDelay != "" && service.delay != tt.expectedDelay {
				t.Errorf("Expected delay %q, got %q", tt.expectedDelay, service.delay)
			}

			if tt.expectedActions != nil && !stringSlicesEqual(service.actions, tt.expectedActions) {
				t.Errorf("Expected actions %v, got %v", tt.expectedActions, service.actions)
			}

			if tt.expectedAttach != "" && service.attach != tt.expectedAttach {
				t.Errorf("Expected attach %q, got %q", tt.expectedAttach, service.attach)
			}

			if tt.expectedFilename != "" && service.filename != tt.expectedFilename {
				t.Errorf("Expected filename %q, got %q", tt.expectedFilename, service.filename)
			}

			if tt.expectedClick != "" && service.click != tt.expectedClick {
				t.Errorf("Expected click %q, got %q", tt.expectedClick, service.click)
			}

			if tt.expectedEmail != "" && service.email != tt.expectedEmail {
				t.Errorf("Expected email %q, got %q", tt.expectedEmail, service.email)
			}
		})
	}
}

func TestNtfyService_TestURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid ntfy://ntfy.sh/topic",
			url:         "ntfy://ntfy.sh/my-topic",
			expectError: false,
		},
		{
			name:        "Valid ntfys://ntfy.sh/topic",
			url:         "ntfys://ntfy.sh/my-topic",
			expectError: false,
		},
		{
			name:        "Valid with authentication",
			url:         "ntfy://user:pass@ntfy.example.com/alerts",
			expectError: false,
		},
		{
			name:        "Valid with token",
			url:         "ntfys://token@ntfy.sh/notifications",
			expectError: false,
		},
		{
			name:        "Valid with query parameters",
			url:         "ntfy://ntfy.sh/topic?priority=4&tags=alert&delay=5min",
			expectError: false,
		},
		{
			name:        "Invalid http://ntfy.sh/topic",
			url:         "http://ntfy.sh/topic",
			expectError: true,
		},
		{
			name:        "Invalid ntfy://host (no topic)",
			url:         "ntfy://ntfy.sh",
			expectError: true,
		},
		{
			name:        "Invalid ntfy:///topic (no host)",
			url:         "ntfy:///topic",
			expectError: true,
		},
		{
			name:        "Invalid priority",
			url:         "ntfy://ntfy.sh/topic?priority=10",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNtfyService()
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

func TestNtfyService_Properties(t *testing.T) {
	service := NewNtfyService()

	if !service.SupportsAttachments() {
		t.Error("Ntfy should support attachments")
	}

	expectedMaxLength := 4096
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestNtfyService_GetEmojiForNotifyType(t *testing.T) {
	service := &NtfyService{}

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, "white_check_mark"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "rotating_light"},
		{NotifyTypeInfo, "information_source"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			result := service.getEmojiForNotifyType(tt.notifyType)
			if result != tt.expected {
				t.Errorf("Expected emoji %q for %v, got %q", tt.expected, tt.notifyType, result)
			}
		})
	}
}

func TestNtfyService_MapNotifyTypeToPriority(t *testing.T) {
	service := &NtfyService{}

	tests := []struct {
		notifyType NotifyType
		expected   int
	}{
		{NotifyTypeSuccess, 3},
		{NotifyTypeWarning, 4},
		{NotifyTypeError, 5},
		{NotifyTypeInfo, 3},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			result := service.mapNotifyTypeToPriority(tt.notifyType)
			if result != tt.expected {
				t.Errorf("Expected priority %d for %v, got %d", tt.expected, tt.notifyType, result)
			}
		})
	}
}

func TestNtfyService_Send_InvalidConfig(t *testing.T) {
	service := NewNtfyService()
	parsedURL, _ := url.Parse("ntfy://ntfy.example.com/test-topic")
	_ = service.(*NtfyService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real Ntfy server
	if err == nil {
		t.Error("Expected network error for invalid Ntfy configuration, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "ntfy") &&
		!strings.Contains(err.Error(), "Ntfy") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should be network-related, got: %v", err)
	}
}

func TestNtfyService_PayloadGeneration(t *testing.T) {
	service := NewNtfyService()
	parsedURL, _ := url.Parse("ntfy://token@ntfy.example.com/alerts?priority=4&tags=production,urgent&delay=5min")
	_ = service.(*NtfyService).ParseURL(parsedURL)

	ntfyService := service.(*NtfyService)

	// Test that configuration is parsed correctly
	if ntfyService.token != "token" {
		t.Error("Expected token to be set from URL")
	}

	if ntfyService.baseURL != "http://ntfy.example.com:80" {
		t.Errorf("Expected baseURL to be parsed correctly, got: %s", ntfyService.baseURL)
	}

	if ntfyService.topic != "alerts" {
		t.Errorf("Expected topic to be 'alerts', got: %s", ntfyService.topic)
	}

	if ntfyService.priority != 4 {
		t.Errorf("Expected priority to be 4, got: %d", ntfyService.priority)
	}

	expectedTags := []string{"production", "urgent"}
	if !stringSlicesEqual(ntfyService.tags, expectedTags) {
		t.Errorf("Expected tags %v, got: %v", expectedTags, ntfyService.tags)
	}

	if ntfyService.delay != "5min" {
		t.Errorf("Expected delay to be '5min', got: %s", ntfyService.delay)
	}
}

func TestNtfyService_PriorityOverride(t *testing.T) {
	// Test that notification type overrides default priority but not explicit priority
	tests := []struct {
		name           string
		url            string
		notifyType     NotifyType
		expectedPrio   int
		shouldOverride bool
	}{
		{
			name:           "Default priority overridden by error type",
			url:            "ntfy://ntfy.sh/topic",
			notifyType:     NotifyTypeError,
			expectedPrio:   5,
			shouldOverride: true,
		},
		{
			name:           "Explicit priority not overridden",
			url:            "ntfy://ntfy.sh/topic?priority=2",
			notifyType:     NotifyTypeError,
			expectedPrio:   2,
			shouldOverride: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNtfyService().(*NtfyService)
			parsedURL, _ := url.Parse(tt.url)
			_ = service.ParseURL(parsedURL)

			// Create a test message to check priority mapping
			message := NtfyMessage{
				Topic:    service.topic,
				Priority: service.priority,
			}

			// Apply the same logic as in Send method
			if service.priority == 3 {
				message.Priority = service.mapNotifyTypeToPriority(tt.notifyType)
			}

			if message.Priority != tt.expectedPrio {
				t.Errorf("Expected priority %d, got %d", tt.expectedPrio, message.Priority)
			}
		})
	}
}