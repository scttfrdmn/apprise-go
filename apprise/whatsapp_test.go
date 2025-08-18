package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWhatsAppService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedPhoneID    string
		expectedToken      string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "whatsapp://phone_id_123@access_token_456/+1234567890",
			expectError:        false,
			expectedPhoneID:    "phone_id_123",
			expectedToken:      "access_token_456",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "whatsapp://phone_id_123@access_token_456/+1234567890/+0987654321",
			expectError:        false,
			expectedPhoneID:    "phone_id_123",
			expectedToken:      "access_token_456",
			expectedRecipients: []string{"+1234567890", "+0987654321"},
		},
		{
			name:        "Missing access token",
			url:         "whatsapp://phone_id_123@/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing phone ID",
			url:         "whatsapp://@access_token_456/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "whatsapp://phone_id_123@access_token_456",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewWhatsAppService().(*WhatsAppService)
			
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

			if service.phoneID != tt.expectedPhoneID {
				t.Errorf("Expected phone ID to be %s, got %s", tt.expectedPhoneID, service.phoneID)
			}

			if service.accessToken != tt.expectedToken {
				t.Errorf("Expected access token to be %s, got %s", tt.expectedToken, service.accessToken)
			}

			if len(service.to) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.to))
			}

			for i, expected := range tt.expectedRecipients {
				if i < len(service.to) && service.to[i] != expected {
					t.Errorf("Expected recipient %d to be %s, got %s", i, expected, service.to[i])
				}
			}
		})
	}
}

func TestWhatsAppService_GetServiceID(t *testing.T) {
	service := NewWhatsAppService()
	if service.GetServiceID() != "whatsapp" {
		t.Errorf("Expected service ID 'whatsapp', got %s", service.GetServiceID())
	}
}

func TestWhatsAppService_GetDefaultPort(t *testing.T) {
	service := NewWhatsAppService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestWhatsAppService_SupportsAttachments(t *testing.T) {
	service := NewWhatsAppService()
	if !service.SupportsAttachments() {
		t.Error("WhatsApp should support attachments")
	}
}

func TestWhatsAppService_GetMaxBodyLength(t *testing.T) {
	service := NewWhatsAppService()
	if service.GetMaxBodyLength() != 4096 {
		t.Errorf("Expected max body length 4096, got %d", service.GetMaxBodyLength())
	}
}

func TestWhatsAppService_Send(t *testing.T) {
	service := NewWhatsAppService().(*WhatsAppService)
	service.phoneID = "phone_id_123"
	service.accessToken = "test_access_token"
	service.to = []string{"+1234567890"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to invalid credentials/unreachable API
	if err == nil {
		t.Error("Expected error due to invalid credentials/unreachable API, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "WhatsApp") {
		t.Errorf("Expected error to mention WhatsApp, got: %v", err)
	}
}