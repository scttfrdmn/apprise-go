package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestTwilioService_GetServiceID(t *testing.T) {
	service := NewTwilioService()
	if service.GetServiceID() != "twilio" {
		t.Errorf("Expected service ID 'twilio', got '%s'", service.GetServiceID())
	}
}

func TestTwilioService_GetDefaultPort(t *testing.T) {
	service := NewTwilioService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestTwilioService_ParseURL(t *testing.T) {
	testCases := []struct {
		name               string
		url                string
		expectError        bool
		expectedAccountSID string
		expectedAuthToken  string
		expectedFromPhone  string
		expectedToPhones   []string
	}{
		{
			name:               "Basic format",
			url:                "twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543",
			expectError:        false,
			expectedAccountSID: "ACCOUNT_SID",
			expectedAuthToken:  "AUTH_TOKEN",
			expectedFromPhone:  "+15551234567",
			expectedToPhones:   []string{"+15559876543"},
		},
		{
			name:               "Multiple recipients",
			url:                "twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543/+15551111111",
			expectError:        false,
			expectedAccountSID: "ACCOUNT_SID",
			expectedAuthToken:  "AUTH_TOKEN",
			expectedFromPhone:  "+15551234567",
			expectedToPhones:   []string{"+15559876543", "+15551111111"},
		},
		{
			name:               "US numbers without + prefix",
			url:                "twilio://ACCOUNT_SID:AUTH_TOKEN@15551234567/15559876543",
			expectError:        false,
			expectedAccountSID: "ACCOUNT_SID",
			expectedAuthToken:  "AUTH_TOKEN",
			expectedFromPhone:  "+15551234567", // Should auto-add +
			expectedToPhones:   []string{"+15559876543"},
		},
		{
			name:        "Invalid scheme",
			url:         "http://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "twilio://+15551234567/+15559876543",
			expectError: true,
		},
		{
			name:        "Missing from phone",
			url:         "twilio://ACCOUNT_SID:AUTH_TOKEN@/+15559876543",
			expectError: true,
		},
		{
			name:        "No recipients",
			url:         "twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewTwilioService().(*TwilioService)
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

			if service.accountSID != tc.expectedAccountSID {
				t.Errorf("Expected accountSID '%s', got '%s'", tc.expectedAccountSID, service.accountSID)
			}

			if service.authToken != tc.expectedAuthToken {
				t.Errorf("Expected authToken '%s', got '%s'", tc.expectedAuthToken, service.authToken)
			}

			if service.fromPhone != tc.expectedFromPhone {
				t.Errorf("Expected fromPhone '%s', got '%s'", tc.expectedFromPhone, service.fromPhone)
			}

			if len(service.toPhones) != len(tc.expectedToPhones) {
				t.Errorf("Expected %d to phones, got %d", len(tc.expectedToPhones), len(service.toPhones))
			}

			for i, expected := range tc.expectedToPhones {
				if i < len(service.toPhones) && service.toPhones[i] != expected {
					t.Errorf("Expected toPhone[%d] '%s', got '%s'", i, expected, service.toPhones[i])
				}
			}
		})
	}
}

func TestTwilioService_ParseURL_QueryParams(t *testing.T) {
	testURL := "twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543?apikey=test_key"

	service := NewTwilioService().(*TwilioService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.apiKey != "test_key" {
		t.Errorf("Expected apiKey 'test_key', got '%s'", service.apiKey)
	}
}

func TestTwilioService_TestURL(t *testing.T) {
	service := NewTwilioService()

	validURLs := []string{
		"twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543",
		"twilio://SID:TOKEN@15551234567/15559876543",                             // US numbers without +
		"twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543/+15551111111", // Multiple recipients
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
		"http://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543",
		"twilio://+15551234567/+15559876543",            // Missing credentials
		"twilio://ACCOUNT_SID:AUTH_TOKEN@/+15559876543", // Missing from phone
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

func TestTwilioService_Properties(t *testing.T) {
	service := NewTwilioService()

	if service.SupportsAttachments() {
		t.Error("Twilio service should not support attachments")
	}

	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestTwilioService_NormalizePhoneNumber(t *testing.T) {
	service := NewTwilioService().(*TwilioService)

	testCases := []struct {
		input    string
		expected string
	}{
		{"+15551234567", "+15551234567"}, // Already normalized
		{"15551234567", "+15551234567"},  // US number without +
		{"5551234567", "+15551234567"},   // US number without country code
		{"+44123456789", "+44123456789"}, // Non-US international
		{"123", "+123"},                  // Short number gets + prefix
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := service.normalizePhoneNumber(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestTwilioService_Send_InvalidConfig(t *testing.T) {
	service := NewTwilioService().(*TwilioService)

	// Service with minimal but invalid configuration should fail
	service.accountSID = "test_sid"
	service.authToken = "test_token"
	service.fromPhone = "+15551234567"
	service.toPhones = []string{"+15559876543"}

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected Send to fail with invalid credentials")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}
