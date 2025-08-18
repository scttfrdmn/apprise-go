package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAWSSNSSMSService_ParseURL(t *testing.T) {

	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedAccessKey  string
		expectedSecretKey  string
		expectedRegion     string
		expectedRecipients []string
	}{
		{
			name:               "Valid single recipient",
			url:                "aws-sns-sms://access_key:secret_key@us-east-1/+1234567890",
			expectError:        false,
			expectedAccessKey:  "access_key",
			expectedSecretKey:  "secret_key",
			expectedRegion:     "us-east-1",
			expectedRecipients: []string{"+1234567890"},
		},
		{
			name:               "Valid multiple recipients",
			url:                "aws-sns-sms://access_key:secret_key@us-west-2/+1234567890/+0987654321",
			expectError:        false,
			expectedAccessKey:  "access_key",
			expectedSecretKey:  "secret_key",
			expectedRegion:     "us-west-2",
			expectedRecipients: []string{"+1234567890", "+0987654321"},
		},
		{
			name:        "Missing access key",
			url:         "aws-sns-sms://@us-east-1/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing secret key",
			url:         "aws-sns-sms://access_key@us-east-1/+1234567890",
			expectError: true,
		},
		{
			name:        "Missing recipients",
			url:         "aws-sns-sms://access_key:secret_key@us-east-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAWSSNSSMSService().(*AWSSNSSMSService)
			
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

			if service.accessKey != tt.expectedAccessKey {
				t.Errorf("Expected access key to be %s, got %s", tt.expectedAccessKey, service.accessKey)
			}

			if service.secretKey != tt.expectedSecretKey {
				t.Errorf("Expected secret key to be %s, got %s", tt.expectedSecretKey, service.secretKey)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region to be %s, got %s", tt.expectedRegion, service.region)
			}

			if len(service.to) != len(tt.expectedRecipients) {
				t.Errorf("Expected %d recipients, got %d", len(tt.expectedRecipients), len(service.to))
			}
		})
	}
}

func TestAWSSNSSMSService_GetServiceID(t *testing.T) {
	service := NewAWSSNSSMSService()
	if service.GetServiceID() != "aws-sns-sms" {
		t.Errorf("Expected service ID 'aws-sns-sms', got %s", service.GetServiceID())
	}
}

func TestAWSSNSSMSService_GetDefaultPort(t *testing.T) {
	service := NewAWSSNSSMSService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestAWSSNSSMSService_SupportsAttachments(t *testing.T) {
	service := NewAWSSNSSMSService()
	if service.SupportsAttachments() {
		t.Error("AWS SNS SMS should not support attachments")
	}
}

func TestAWSSNSSMSService_GetMaxBodyLength(t *testing.T) {
	service := NewAWSSNSSMSService()
	if service.GetMaxBodyLength() != 1600 {
		t.Errorf("Expected max body length 1600, got %d", service.GetMaxBodyLength())
	}
}

func TestAWSSNSSMSService_Send(t *testing.T) {
	service := NewAWSSNSSMSService().(*AWSSNSSMSService)
	service.accessKey = "test_access_key"
	service.secretKey = "test_secret_key"
	service.region = "us-east-1"
	service.to = []string{"+1234567890"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := NotificationRequest{
		Title: "Test Title",
		Body:  "Test message body",
	}

	err := service.Send(ctx, notification)

	// Expect error due to SDK requirement
	if err == nil {
		t.Error("Expected error due to SDK requirement, but got none")
	}

	if err != nil && !strings.Contains(err.Error(), "AWS SDK") {
		t.Errorf("Expected error to mention AWS SDK, got: %v", err)
	}
}