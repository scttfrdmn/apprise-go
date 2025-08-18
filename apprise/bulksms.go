package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// BulkSMSService implements BulkSMS notifications
type BulkSMSService struct {
	username string
	password string
	from     string
	to       []string
	client   *http.Client
}

// BulkSMSRequest represents the API request structure for BulkSMS
type BulkSMSRequest struct {
	From string `json:"from,omitempty"`
	To   string `json:"to"`
	Body string `json:"body"`
}

// NewBulkSMSService creates a new BulkSMS service instance
func NewBulkSMSService() Service {
	return &BulkSMSService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *BulkSMSService) GetServiceID() string {
	return "bulksms"
}

// GetDefaultPort returns the default port
func (s *BulkSMSService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *BulkSMSService) ParseURL(serviceURL *url.URL) error {
	// Extract username and password from URL
	if serviceURL.User == nil {
		return fmt.Errorf("BulkSMS URL must include username and password")
	}

	s.username = serviceURL.User.Username()
	password, hasPassword := serviceURL.User.Password()
	if !hasPassword {
		return fmt.Errorf("BulkSMS URL must include password")
	}
	s.password = password

	// Extract phone numbers from host and path
	phoneNumbers := []string{}
	
	// Host contains the primary phone number
	if serviceURL.Host != "" {
		phoneNumbers = append(phoneNumbers, serviceURL.Host)
	}

	// Path can contain additional phone numbers
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		pathNumbers := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, number := range pathNumbers {
			if number != "" {
				phoneNumbers = append(phoneNumbers, number)
			}
		}
	}

	if len(phoneNumbers) == 0 {
		return fmt.Errorf("BulkSMS URL must specify at least one phone number")
	}

	s.to = phoneNumbers

	// Parse query parameters
	query := serviceURL.Query()
	if from := query.Get("from"); from != "" {
		s.from = from
	}

	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *BulkSMSService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *BulkSMSService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}

	// Send to each recipient
	var lastError error
	successCount := 0

	for _, recipient := range s.to {
		err := s.sendSMS(ctx, recipient, message)
		if err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Return error if all sends failed
	if successCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}

// sendSMS sends an SMS to a single recipient via BulkSMS API
func (s *BulkSMSService) sendSMS(ctx context.Context, to, message string) error {
	// BulkSMS API endpoint
	apiURL := "https://api.bulksms.com/v1/messages"

	// Prepare request payload
	payload := BulkSMSRequest{
		To:   to,
		Body: message,
	}

	// Add from number if specified
	if s.from != "" {
		payload.From = s.from
	}

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal BulkSMS request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create BulkSMS request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.username, s.password)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("BulkSMS API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("BulkSMS API returned status %d", resp.StatusCode)
	}

	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *BulkSMSService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *BulkSMSService) GetMaxBodyLength() int {
	return 160 // Standard SMS length
}