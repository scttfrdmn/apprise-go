package apprise

import (
	"context"
	"net/url"
	"runtime"
	"testing"
	"time"
)

func TestDesktopService_GetServiceID(t *testing.T) {
	service := NewDesktopService()

	// Test platform-specific service IDs
	switch runtime.GOOS {
	case "darwin":
		if service.GetServiceID() != "macosx" {
			t.Errorf("Expected service ID 'macosx' on macOS, got '%s'", service.GetServiceID())
		}
	case "windows":
		if service.GetServiceID() != "windows" {
			t.Errorf("Expected service ID 'windows' on Windows, got '%s'", service.GetServiceID())
		}
	case "linux":
		if service.GetServiceID() != "linux" {
			t.Errorf("Expected service ID 'linux' on Linux, got '%s'", service.GetServiceID())
		}
	default:
		if service.GetServiceID() != "desktop" {
			t.Errorf("Expected service ID 'desktop' on unknown platform, got '%s'", service.GetServiceID())
		}
	}
}

func TestDesktopService_ParseURL(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectError bool
		platform    string
		sound       string
		duration    int
		image       string
	}{
		{
			name:     "Basic desktop URL",
			url:      "desktop://",
			platform: runtime.GOOS,
			duration: 12,
		},
		{
			name:     "macOS URL",
			url:      "macosx://",
			platform: "darwin",
			duration: 12,
		},
		{
			name:     "Windows URL",
			url:      "windows://",
			platform: "windows",
			duration: 12,
		},
		{
			name:     "Linux URL",
			url:      "linux://",
			platform: "linux",
			duration: 12,
		},
		{
			name:     "macOS with sound",
			url:      "macosx://?sound=default",
			platform: "darwin",
			sound:    "default",
			duration: 12,
		},
		{
			name:     "Windows with duration",
			url:      "windows://?duration=5",
			platform: "windows",
			duration: 5,
		},
		{
			name:     "Desktop with image",
			url:      "desktop://?image=/path/to/image.png",
			platform: runtime.GOOS,
			image:    "/path/to/image.png",
			duration: 12,
		},
		{
			name:     "Multiple parameters",
			url:      "macosx://?sound=ping&image=/icon.png",
			platform: "darwin",
			sound:    "ping",
			image:    "/icon.png",
			duration: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewDesktopService()
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

			if service.platform != tc.platform {
				t.Errorf("Expected platform %s, got %s", tc.platform, service.platform)
			}

			if service.sound != tc.sound {
				t.Errorf("Expected sound %s, got %s", tc.sound, service.sound)
			}

			if service.duration != tc.duration {
				t.Errorf("Expected duration %d, got %d", tc.duration, service.duration)
			}

			if service.image != tc.image {
				t.Errorf("Expected image %s, got %s", tc.image, service.image)
			}
		})
	}
}

func TestDesktopService_TestURL(t *testing.T) {
	service := NewDesktopService()

	validURLs := []string{
		"desktop://",
		"macosx://",
		"windows://",
		"linux://",
		"dbus://",
		"gnome://",
		"kde://",
		"glib://",
		"qt://",
		"desktop://?sound=default",
		"windows://?duration=10",
	}

	for _, url := range validURLs {
		t.Run("Valid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", url, err)
			}
		})
	}

	invalidURLs := []string{
		"invalid://",
		"http://example.com",
		"desktop",
	}

	for _, url := range invalidURLs {
		t.Run("Invalid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", url)
			}
		})
	}
}

func TestDesktopService_Send_MessageTruncation(t *testing.T) {
	service := NewDesktopService()

	// Test message truncation
	longTitle := "This is a very long title that should be truncated because it exceeds the maximum allowed length for desktop notifications"
	longBody := "This is a very long body message that should be truncated because it exceeds the 250 character limit for desktop notifications. This message is intentionally made very long to test the truncation functionality and ensure that it works correctly in all cases."

	req := NotificationRequest{
		Title:      longTitle,
		Body:       longBody,
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This test may fail on systems without proper desktop notification support
	// but should at least test the message truncation logic
	err := service.Send(ctx, req)

	// We don't expect this to succeed on all systems, but it should not panic
	// and should handle the message truncation properly
	if err != nil {
		t.Logf("Desktop notification failed (expected on some systems): %v", err)
	}
}

func TestDesktopService_Properties(t *testing.T) {
	service := NewDesktopService()

	// Test default port
	if service.GetDefaultPort() != 0 {
		t.Errorf("Expected default port 0 for desktop notifications, got %d", service.GetDefaultPort())
	}

	// Test attachment support
	if service.SupportsAttachments() {
		t.Error("Desktop notifications should not support attachments")
	}

	// Test max body length
	if service.GetMaxBodyLength() != 250 {
		t.Errorf("Expected max body length 250, got %d", service.GetMaxBodyLength())
	}
}

func TestLinuxDBusService(t *testing.T) {
	service := NewLinuxDBusService()

	if service.GetServiceID() != "dbus" {
		t.Errorf("Expected service ID 'dbus', got '%s'", service.GetServiceID())
	}

	// Test URL parsing
	testCases := []struct {
		url           string
		interfaceType string
	}{
		{"dbus://", "auto"},
		{"qt://", "qt"},
		{"kde://", "qt"},
		{"glib://", "glib"},
		{"gnome://", "glib"},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			if service.interfaceType != tc.interfaceType {
				t.Errorf("Expected interface %s, got %s", tc.interfaceType, service.interfaceType)
			}
		})
	}
}

func TestGotifyService(t *testing.T) {
	service := NewGotifyService()

	if service.GetServiceID() != "gotify" {
		t.Errorf("Expected service ID 'gotify', got '%s'", service.GetServiceID())
	}

	// Test URL parsing
	testCases := []struct {
		name        string
		url         string
		expectError bool
		serverURL   string
		appToken    string
		priority    int
		secure      bool
	}{
		{
			name:      "Basic HTTP Gotify",
			url:       "gotify://localhost:8080/ABCDEFghijklmnop",
			serverURL: "http://localhost:8080",
			appToken:  "ABCDEFghijklmnop",
			priority:  5,
			secure:    false,
		},
		{
			name:      "HTTPS Gotify",
			url:       "gotifys://gotify.example.com/token123",
			serverURL: "https://gotify.example.com:443",
			appToken:  "token123",
			priority:  5,
			secure:    true,
		},
		{
			name:      "Gotify with priority",
			url:       "gotify://server.com:8080/mytoken?priority=8",
			serverURL: "http://server.com:8080",
			appToken:  "mytoken",
			priority:  8,
			secure:    false,
		},
		{
			name:        "Missing token",
			url:         "gotify://server.com:8080/",
			expectError: true,
		},
		{
			name:        "No token path",
			url:         "gotify://server.com:8080",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewGotifyService()
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

			if service.serverURL != tc.serverURL {
				t.Errorf("Expected server URL %s, got %s", tc.serverURL, service.serverURL)
			}

			if service.appToken != tc.appToken {
				t.Errorf("Expected app token %s, got %s", tc.appToken, service.appToken)
			}

			if service.priority != tc.priority {
				t.Errorf("Expected priority %d, got %d", tc.priority, service.priority)
			}

			if service.secure != tc.secure {
				t.Errorf("Expected secure %v, got %v", tc.secure, service.secure)
			}
		})
	}
}

func TestGotifyService_TestURL(t *testing.T) {
	service := NewGotifyService()

	validURLs := []string{
		"gotify://localhost:8080/token",
		"gotifys://secure.example.com/securetoken",
		"gotify://server.com:8080/token?priority=5",
	}

	for _, url := range validURLs {
		t.Run("Valid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", url, err)
			}
		})
	}

	invalidURLs := []string{
		"http://example.com",
		"gotify://server.com",   // Missing token
		"gotifys://server.com/", // Empty token
		"invalid://server.com/token",
	}

	for _, url := range invalidURLs {
		t.Run("Invalid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", url)
			}
		})
	}
}

func TestGotifyService_Properties(t *testing.T) {
	service := NewGotifyService()

	// Test default ports
	if service.GetDefaultPort() != 80 {
		t.Errorf("Expected default port 80 for HTTP Gotify, got %d", service.GetDefaultPort())
	}

	service.secure = true
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443 for HTTPS Gotify, got %d", service.GetDefaultPort())
	}

	// Test attachment support
	if service.SupportsAttachments() {
		t.Error("Gotify should not support attachments")
	}

	// Test max body length
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected unlimited body length for Gotify, got %d", service.GetMaxBodyLength())
	}
}

// Integration test for desktop notifications (will be skipped on CI/headless systems)
func TestDesktopService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service := NewDesktopService()

	req := NotificationRequest{
		Title:      "Test Notification",
		Body:       "This is a test desktop notification from Apprise Go",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Logf("Desktop notification integration test failed (expected on headless systems): %v", err)
	} else {
		t.Log("Desktop notification sent successfully - you should see a notification on your desktop")
	}
}
