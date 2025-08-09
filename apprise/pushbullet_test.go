package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestPushbulletService_GetServiceID(t *testing.T) {
	service := NewPushbulletService()
	if service.GetServiceID() != "pushbullet" {
		t.Errorf("Expected service ID 'pushbullet', got '%s'", service.GetServiceID())
	}
}

func TestPushbulletService_GetDefaultPort(t *testing.T) {
	service := NewPushbulletService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestPushbulletService_ParseURL(t *testing.T) {
	testCases := []struct {
		name             string
		url              string
		expectError      bool
		expectedToken    string
		expectedDevices  []string
		expectedEmails   []string
		expectedChannels []string
	}{
		{
			name:          "Basic access token",
			url:           "pushbullet://access_token",
			expectError:   false,
			expectedToken: "access_token",
		},
		{
			name:            "Token with device",
			url:             "pushbullet://access_token/device_id",
			expectError:     false,
			expectedToken:   "access_token",
			expectedDevices: []string{"device_id"},
		},
		{
			name:           "Token with email",
			url:            "pushbullet://access_token/user@email.com",
			expectError:    false,
			expectedToken:  "access_token",
			expectedEmails: []string{"user@email.com"},
		},
		{
			name:             "Token with channel",
			url:              "pushbullet://access_token/#channel_name",
			expectError:      false,
			expectedToken:    "access_token",
			expectedChannels: []string{"channel_name"},
		},
		{
			name:          "Pball alias",
			url:           "pball://access_token",
			expectError:   false,
			expectedToken: "access_token",
		},
		{
			name:            "Multiple targets",
			url:             "pushbullet://token/device1/device2/user@email.com",
			expectError:     false,
			expectedToken:   "token",
			expectedDevices: []string{"device1", "device2"},
			expectedEmails:  []string{"user@email.com"},
		},
		{
			name:        "Invalid scheme",
			url:         "http://access_token",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "pushbullet://",
			expectError: true,
		},
		{
			name:        "Empty token",
			url:         "pushbullet:///device",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewPushbulletService().(*PushbulletService)
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

			if service.accessToken != tc.expectedToken {
				t.Errorf("Expected access token '%s', got '%s'", tc.expectedToken, service.accessToken)
			}

			if len(service.devices) != len(tc.expectedDevices) {
				t.Errorf("Expected %d devices, got %d", len(tc.expectedDevices), len(service.devices))
			}
			for i, expected := range tc.expectedDevices {
				if i < len(service.devices) && service.devices[i] != expected {
					t.Errorf("Expected device[%d] '%s', got '%s'", i, expected, service.devices[i])
				}
			}

			if len(service.emails) != len(tc.expectedEmails) {
				t.Errorf("Expected %d emails, got %d", len(tc.expectedEmails), len(service.emails))
			}
			for i, expected := range tc.expectedEmails {
				if i < len(service.emails) && service.emails[i] != expected {
					t.Errorf("Expected email[%d] '%s', got '%s'", i, expected, service.emails[i])
				}
			}

			if len(service.channels) != len(tc.expectedChannels) {
				t.Errorf("Expected %d channels, got %d", len(tc.expectedChannels), len(service.channels))
			}
			for i, expected := range tc.expectedChannels {
				if i < len(service.channels) && service.channels[i] != expected {
					t.Errorf("Expected channel[%d] '%s', got '%s'", i, expected, service.channels[i])
				}
			}
		})
	}
}

func TestPushbulletService_ParseURL_QueryParams(t *testing.T) {
	testURL := "pushbullet://access_token?device=device1,device2&email=user1@example.com,user2@example.com&channel=channel1,channel2"

	service := NewPushbulletService().(*PushbulletService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	expectedDevices := []string{"device1", "device2"}
	if len(service.devices) != len(expectedDevices) {
		t.Errorf("Expected %d devices, got %d", len(expectedDevices), len(service.devices))
	}
	for i, expected := range expectedDevices {
		if i < len(service.devices) && service.devices[i] != expected {
			t.Errorf("Expected device[%d] '%s', got '%s'", i, expected, service.devices[i])
		}
	}

	expectedEmails := []string{"user1@example.com", "user2@example.com"}
	if len(service.emails) != len(expectedEmails) {
		t.Errorf("Expected %d emails, got %d", len(expectedEmails), len(service.emails))
	}

	expectedChannels := []string{"channel1", "channel2"}
	if len(service.channels) != len(expectedChannels) {
		t.Errorf("Expected %d channels, got %d", len(expectedChannels), len(service.channels))
	}
}

func TestPushbulletService_TestURL(t *testing.T) {
	service := NewPushbulletService()

	validURLs := []string{
		"pushbullet://access_token",
		"pball://access_token/device_id",
		"pushbullet://access_token/user@email.com",
		"pushbullet://access_token/#channel_name",
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
		"http://access_token",
		"pushbullet://", // Missing token
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

func TestPushbulletService_Properties(t *testing.T) {
	service := NewPushbulletService()

	if !service.SupportsAttachments() {
		t.Error("Pushbullet service should support attachments")
	}

	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected unlimited body length, got %d", service.GetMaxBodyLength())
	}
}

func TestPushbulletService_Send_InvalidConfig(t *testing.T) {
	service := NewPushbulletService().(*PushbulletService)

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

func TestPushbulletService_ParseTargetPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		devices  []string
		emails   []string
		channels []string
	}{
		{
			name:    "Device ID",
			path:    "/device123",
			devices: []string{"device123"},
		},
		{
			name:   "Email",
			path:   "/user@example.com",
			emails: []string{"user@example.com"},
		},
		{
			name:     "Channel",
			path:     "/#mychannel",
			channels: []string{"mychannel"},
		},
		{
			name:     "Mixed targets",
			path:     "/device1/user@email.com/#channel1/device2",
			devices:  []string{"device1", "device2"},
			emails:   []string{"user@email.com"},
			channels: []string{"channel1"},
		},
		{
			name: "Empty path",
			path: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewPushbulletService().(*PushbulletService)
			service.accessToken = "test_token" // Set required token

			// Create a URL with the test path
			fullURL := "pushbullet://test_token" + tc.path
			parsedURL, err := url.Parse(fullURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			// Check devices
			if len(service.devices) != len(tc.devices) {
				t.Errorf("Expected %d devices, got %d", len(tc.devices), len(service.devices))
			}
			for i, expected := range tc.devices {
				if i < len(service.devices) && service.devices[i] != expected {
					t.Errorf("Expected device[%d] '%s', got '%s'", i, expected, service.devices[i])
				}
			}

			// Check emails
			if len(service.emails) != len(tc.emails) {
				t.Errorf("Expected %d emails, got %d", len(tc.emails), len(service.emails))
			}
			for i, expected := range tc.emails {
				if i < len(service.emails) && service.emails[i] != expected {
					t.Errorf("Expected email[%d] '%s', got '%s'", i, expected, service.emails[i])
				}
			}

			// Check channels
			if len(service.channels) != len(tc.channels) {
				t.Errorf("Expected %d channels, got %d", len(tc.channels), len(service.channels))
			}
			for i, expected := range tc.channels {
				if i < len(service.channels) && service.channels[i] != expected {
					t.Errorf("Expected channel[%d] '%s', got '%s'", i, expected, service.channels[i])
				}
			}
		})
	}
}
