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

// ClickSendService implements ClickSend SMS notifications
type ClickSendService struct {
	username string
	apiKey   string
	from     string
	to       []string
	client   *http.Client
}

// ClickSendSMSRequest represents a single SMS message in the API request
type ClickSendSMSRequest struct {
	Source string `json:"source,omitempty"`
	To     string `json:"to"`
	Body   string `json:"body"`
}

// ClickSendRequest represents the API request structure for ClickSend
type ClickSendRequest struct {
	Messages []ClickSendSMSRequest `json:"messages"`
}

// NewClickSendService creates a new ClickSend service instance
func NewClickSendService() Service {
	return &ClickSendService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *ClickSendService) GetServiceID() string {
	return "clicksend"
}

// GetDefaultPort returns the default port
func (s *ClickSendService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *ClickSendService) ParseURL(serviceURL *url.URL) error {
	// Extract username and API key from URL
	if serviceURL.User == nil {
		return fmt.Errorf("ClickSend URL must include username and API key")
	}

	s.username = serviceURL.User.Username()
	apiKey, hasAPIKey := serviceURL.User.Password()
	if !hasAPIKey {
		return fmt.Errorf("ClickSend URL must include API key as password")
	}
	s.apiKey = apiKey

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
		return fmt.Errorf("ClickSend URL must specify at least one phone number")
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
func (s *ClickSendService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *ClickSendService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}

	// Prepare messages for all recipients
	messages := make([]ClickSendSMSRequest, len(s.to))
	for i, recipient := range s.to {
		messages[i] = ClickSendSMSRequest{
			To:   recipient,
			Body: message,
		}
		
		// Add from number if specified
		if s.from != "" {
			messages[i].Source = s.from
		}
	}

	// Create request payload
	payload := ClickSendRequest{
		Messages: messages,
	}

	// Send the request
	return s.sendSMS(ctx, payload)
}

// sendSMS sends SMS messages via ClickSend API
func (s *ClickSendService) sendSMS(ctx context.Context, payload ClickSendRequest) error {
	// ClickSend SMS API endpoint
	apiURL := "https://rest.clicksend.com/v3/sms/send"

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal ClickSend request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create ClickSend request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.username, s.apiKey)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("ClickSend API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ClickSend API returned status %d", resp.StatusCode)
	}

	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *ClickSendService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *ClickSendService) GetMaxBodyLength() int {
	return 1600 // ClickSend supports up to 1600 characters
}