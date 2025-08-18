package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignalService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedNumber     string
		expectedRecipients []string
		expectedServerURL  string
	}{
		{
			name:               "Valid single recipient",
			url:                "signal://+1234567890@localhost:8080/+0987654321",
			expectError:        false,
			expectedNumber:     "+1234567890",
			expectedRecipients: []string{"+0987654321"},
			expectedServerURL:  "http://localhost:8080",
		},
		{
			name:               "Valid multiple recipients",
			url:                "signal://+1234567890@localhost:8080/+0987654321/+1111111111",
			expectError:        false,
			expectedNumber:     "+1234567890",
			expectedRecipients: []string{"+0987654321", "+1111111111"},
			expectedServerURL:  "http://localhost:8080",
		},
		{
			name:        "Missing server host",
			url:         "signal://+1234567890@/+0987654321",
			expectError: true,
		},
		{
			name:        "Missing sender number",
			url:         "signal://localhost:8080/+0987654321",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "signal://+1234567890@localhost:8080",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewSignalService().(*SignalService)
			
			parsedURL, parseErr := url.Parse(tt.url)
			if parseErr != nil && !tt.expectError {
				t.Fatalf("URL parsing failed: %v", parseErr)
			}

			if parseErr != nil {
				return
			}

			err := service.ParseURL(parsedURL)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if service.number != tt.expectedNumber {
				t.Errorf("Expected number to be %s, got %s", tt.expectedNumber, service.number)
			}

			if len(service.to) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.to))
			}

			for i, expected := range tt.expectedRecipients {
				if i < len(service.to) && service.to[i] != expected {
					t.Errorf("Expected recipient %d to be %s, got %s", i, expected, service.to[i])
				}
			}

			if service.serverURL != tt.expectedServerURL {
				t.Errorf("Expected server URL to be %s, got %s", tt.expectedServerURL, service.serverURL)
			}
		})
	}
}

func TestSignalService_GetServiceID(t *testing.T) {
	service := NewSignalService()
	if service.GetServiceID() != "signal" {
		t.Errorf("Expected service ID 'signal', got %s", service.GetServiceID())
	}
}

func TestSignalService_GetDefaultPort(t *testing.T) {
	service := NewSignalService()
	if service.GetDefaultPort() != 8080 {
		t.Errorf("Expected default port 8080, got %d", service.GetDefaultPort())
	}
}

func TestSignalService_SupportsAttachments(t *testing.T) {
	service := NewSignalService()
	if !service.SupportsAttachments() {
		t.Error("Signal should support attachments")
	}
}

func TestSignalService_GetMaxBodyLength(t *testing.T) {
	service := NewSignalService()
	if service.GetMaxBodyLength() != 0 {
		t.Errorf("Expected max body length 0 (unlimited), got %d", service.GetMaxBodyLength())
	}
}

func TestSignalService_Send(t *testing.T) {
	service := NewSignalService().(*SignalService)
	service.number = "+1234567890"
	service.serverURL = "http://localhost:8080"
	service.to = []string{"+0987654321"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to unavailable Signal server
	if err == nil {
		t.Error("Expected error due to unavailable Signal server, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "Signal") {
		t.Errorf("Expected error to mention Signal, got: %v", err)
	}
}