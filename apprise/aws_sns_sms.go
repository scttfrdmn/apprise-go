package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AWSSNSSMSService implements AWS SNS SMS notifications (specifically for SMS)
type AWSSNSSMSService struct {
	accessKey string
	secretKey string
	region    string
	to        []string
	client    *http.Client
}

// NewAWSSNSSMSService creates a new AWS SNS SMS service instance
func NewAWSSNSSMSService() Service {
	return &AWSSNSSMSService{
		client: &http.Client{},
		region: "us-east-1", // Default region
	}
}

// GetServiceID returns the service identifier
func (s *AWSSNSSMSService) GetServiceID() string {
	return "aws-sns-sms"
}

// GetDefaultPort returns the default port
func (s *AWSSNSSMSService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *AWSSNSSMSService) ParseURL(serviceURL *url.URL) error {
	// URL format: aws-sns-sms://access_key:secret_key@region/to1/to2
	
	if serviceURL.User == nil {
		return fmt.Errorf("AWS SNS SMS URL must include access key and secret key")
	}
	
	s.accessKey = serviceURL.User.Username()
	secretKey, hasSecret := serviceURL.User.Password()
	if !hasSecret {
		return fmt.Errorf("AWS SNS SMS URL must include secret key")
	}
	s.secretKey = secretKey
	
	// Extract region from host
	if serviceURL.Host != "" {
		s.region = serviceURL.Host
	}
	
	// Extract recipient numbers from path
	recipients := []string{}
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		pathRecipients := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, recipient := range pathRecipients {
			if recipient != "" {
				recipients = append(recipients, recipient)
			}
		}
	}
	
	if len(recipients) == 0 {
		return fmt.Errorf("AWS SNS SMS URL must specify at least one recipient phone number")
	}
	s.to = recipients
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *AWSSNSSMSService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *AWSSNSSMSService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendSMS(ctx, recipient, message); err != nil {
			return fmt.Errorf("failed to send AWS SNS SMS to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendSMS sends an SMS to a specific recipient via AWS SNS
func (s *AWSSNSSMSService) sendSMS(ctx context.Context, to, message string) error {
	// AWS SNS SMS would require AWS SDK implementation
	// For demonstration, return informative error
	return fmt.Errorf("AWS SNS SMS requires AWS SDK integration - not implemented in basic HTTP client version")
}

// SupportsAttachments returns true if this service supports file attachments
func (s *AWSSNSSMSService) SupportsAttachments() bool {
	return false // AWS SNS SMS doesn't support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *AWSSNSSMSService) GetMaxBodyLength() int {
	return 1600 // AWS SNS SMS supports up to 1600 characters
}