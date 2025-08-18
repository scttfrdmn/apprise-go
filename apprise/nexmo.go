package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// NexmoService implements Vonage (formerly Nexmo) SMS notifications
type NexmoService struct {
	apiKey    string
	apiSecret string
	from      string
	to        []string
	client    *http.Client
}

// NewNexmoService creates a new Nexmo/Vonage service instance
func NewNexmoService() Service {
	return &NexmoService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *NexmoService) GetServiceID() string {
	return "nexmo"
}

// GetDefaultPort returns the default port
func (s *NexmoService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *NexmoService) ParseURL(serviceURL *url.URL) error {
	// URL format: nexmo://api_key:api_secret@host/to1/to2?from=sender
	
	if serviceURL.User == nil {
		return fmt.Errorf("Nexmo URL must include API key and secret")
	}
	
	s.apiKey = serviceURL.User.Username()
	apiSecret, hasSecret := serviceURL.User.Password()
	if !hasSecret {
		return fmt.Errorf("Nexmo URL must include API secret")
	}
	s.apiSecret = apiSecret
	
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
		return fmt.Errorf("Nexmo URL must specify at least one recipient phone number")
	}
	s.to = recipients
	
	// Parse query parameters
	query := serviceURL.Query()
	if from := query.Get("from"); from != "" {
		s.from = from
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *NexmoService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *NexmoService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendSMS(ctx, recipient, message); err != nil {
			return fmt.Errorf("failed to send Nexmo SMS to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendSMS sends an SMS to a specific recipient via Nexmo API
func (s *NexmoService) sendSMS(ctx context.Context, to, message string) error {
	// Nexmo REST API endpoint
	apiURL := "https://rest.nexmo.com/sms/json"
	
	// Prepare form data
	formData := url.Values{}
	formData.Set("api_key", s.apiKey)
	formData.Set("api_secret", s.apiSecret)
	formData.Set("to", to)
	formData.Set("text", message)
	
	if s.from != "" {
		formData.Set("from", s.from)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Nexmo request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Nexmo API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Nexmo API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *NexmoService) SupportsAttachments() bool {
	return false // Nexmo SMS doesn't support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *NexmoService) GetMaxBodyLength() int {
	return 1600 // Nexmo supports long SMS up to 1600 characters
}