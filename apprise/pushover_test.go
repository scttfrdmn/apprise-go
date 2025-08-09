package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestPushoverService_GetServiceID(t *testing.T) {
	service := NewPushoverService()
	if service.GetServiceID() != "pushover" {
		t.Errorf("Expected service ID 'pushover', got '%s'", service.GetServiceID())
	}
}

func TestPushoverService_GetDefaultPort(t *testing.T) {
	service := NewPushoverService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestPushoverService_ParseURL(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectError bool
		expectedToken string
		expectedUserKey string
	}{
		{
			name:            "Basic format",
			url:             "pushover://token@userkey",
			expectError:     false,
			expectedToken:   "token",
			expectedUserKey: "userkey",
		},
		{
			name:            "With devices",
			url:             "pushover://token@userkey/device1/device2",
			expectError:     false,
			expectedToken:   "token",
			expectedUserKey: "userkey",
		},
		{
			name:            "Pover alias",
			url:             "pover://token@userkey",
			expectError:     false,
			expectedToken:   "token",
			expectedUserKey: "userkey",
		},
		{
			name:        "Invalid scheme",
			url:         "http://token@userkey",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "pushover://@userkey",
			expectError: true,
		},
		{
			name:        "Missing userkey",
			url:         "pushover://token@",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewPushoverService().(*PushoverService)
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

			if service.token != tc.expectedToken {
				t.Errorf("Expected token '%s', got '%s'", tc.expectedToken, service.token)
			}

			if service.userKey != tc.expectedUserKey {
				t.Errorf("Expected userKey '%s', got '%s'", tc.expectedUserKey, service.userKey)
			}
		})
	}
}

func TestPushoverService_ParseURL_QueryParams(t *testing.T) {
	testURL := "pushover://token@userkey?priority=2&sound=cosmic&retry=60&expire=3600"
	
	service := NewPushoverService().(*PushoverService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.priority != 2 {
		t.Errorf("Expected priority 2, got %d", service.priority)
	}

	if service.sound != "cosmic" {
		t.Errorf("Expected sound 'cosmic', got '%s'", service.sound)
	}

	if service.retry != 60 {
		t.Errorf("Expected retry 60, got %d", service.retry)
	}

	if service.expire != 3600 {
		t.Errorf("Expected expire 3600, got %d", service.expire)
	}
}

func TestPushoverService_ParseURL_Devices(t *testing.T) {
	testURL := "pushover://token@userkey/device1/device2"
	
	service := NewPushoverService().(*PushoverService)
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
}

func TestPushoverService_TestURL(t *testing.T) {
	service := NewPushoverService()

	validURLs := []string{
		"pushover://token@userkey",
		"pover://token@userkey/device",
		"pushover://token@userkey?priority=1&sound=cosmic",
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
		"http://token@userkey",
		"pushover://@userkey", // Missing token
		"pushover://token@",   // Missing userkey
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

func TestPushoverService_Properties(t *testing.T) {
	service := NewPushoverService()

	if !service.SupportsAttachments() {
		t.Error("Pushover service should support attachments")
	}

	if service.GetMaxBodyLength() != 1024 {
		t.Errorf("Expected max body length 1024, got %d", service.GetMaxBodyLength())
	}
}

func TestPushoverService_Send_InvalidConfig(t *testing.T) {
	service := NewPushoverService().(*PushoverService)
	
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

func TestPushoverService_PriorityValidation(t *testing.T) {
	testCases := []struct {
		name           string
		priority       string
		expectedPriority int
	}{
		{"Valid priority -2", "-2", -2},
		{"Valid priority -1", "-1", -1},
		{"Valid priority 0", "0", 0},
		{"Valid priority 1", "1", 1},
		{"Valid priority 2", "2", 2},
		{"Invalid priority 3", "3", 0}, // Should default to 0
		{"Invalid priority -3", "-3", 0}, // Should default to 0
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testURL := "pushover://token@userkey?priority=" + tc.priority
			
			service := NewPushoverService().(*PushoverService)
			parsedURL, err := url.Parse(testURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			if service.priority != tc.expectedPriority {
				t.Errorf("Expected priority %d, got %d", tc.expectedPriority, service.priority)
			}
		})
	}
}

func TestPushoverService_EmergencyPriorityDefaults(t *testing.T) {
	// Test emergency priority (2) with default retry/expire
	testURL := "pushover://token@userkey?priority=2"
	
	service := NewPushoverService().(*PushoverService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.priority != 2 {
		t.Errorf("Expected priority 2, got %d", service.priority)
	}

	if service.retry != 60 {
		t.Errorf("Expected default retry 60 for emergency priority, got %d", service.retry)
	}

	if service.expire != 3600 {
		t.Errorf("Expected default expire 3600 for emergency priority, got %d", service.expire)
	}
}

func TestPushoverService_RetryExpireOnlyForEmergency(t *testing.T) {
	// Test that retry/expire are only set for emergency priority (2)
	testURL := "pushover://token@userkey?priority=1&retry=30&expire=1800"
	
	service := NewPushoverService().(*PushoverService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	// For non-emergency priority, retry and expire should remain 0
	if service.retry != 0 {
		t.Errorf("Expected retry 0 for non-emergency priority, got %d", service.retry)
	}

	if service.expire != 0 {
		t.Errorf("Expected expire 0 for non-emergency priority, got %d", service.expire)
	}
}