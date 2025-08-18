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

// TextMagicService implements TextMagic SMS notifications
type TextMagicService struct {
	username string
	apiKey   string
	from     string
	to       []string
	client   *http.Client
}

// TextMagicSMSRequest represents a TextMagic SMS request
type TextMagicSMSRequest struct {
	Text   string   `json:"text"`
	Phones []string `json:"phones"`
	From   string   `json:"from,omitempty"`
}

// NewTextMagicService creates a new TextMagic service instance
func NewTextMagicService() Service {
	return &TextMagicService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *TextMagicService) GetServiceID() string {
	return "textmagic"
}

// GetDefaultPort returns the default port
func (s *TextMagicService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *TextMagicService) ParseURL(serviceURL *url.URL) error {
	// URL format: textmagic://username:api_key@host/to1/to2?from=sender
	
	if serviceURL.User == nil {
		return fmt.Errorf("TextMagic URL must include username and API key")
	}
	
	s.username = serviceURL.User.Username()
	apiKey, hasKey := serviceURL.User.Password()
	if !hasKey {
		return fmt.Errorf("TextMagic URL must include API key")
	}
	s.apiKey = apiKey
	
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
		return fmt.Errorf("TextMagic URL must specify at least one recipient phone number")
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
func (s *TextMagicService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *TextMagicService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to all recipients in a single request (TextMagic supports bulk)
	return s.sendBulkSMS(ctx, message)
}

// sendBulkSMS sends SMS to all recipients via TextMagic API
func (s *TextMagicService) sendBulkSMS(ctx context.Context, message string) error {
	// TextMagic API endpoint
	apiURL := "https://rest.textmagic.com/api/v2/messages"
	
	// Prepare request payload
	payload := TextMagicSMSRequest{
		Text:   message,
		Phones: s.to,
	}
	
	if s.from != "" {
		payload.From = s.from
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal TextMagic request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create TextMagic request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TM-Username", s.username)
	req.Header.Set("X-TM-Key", s.apiKey)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("TextMagic API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("TextMagic API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *TextMagicService) SupportsAttachments() bool {
	return false // TextMagic SMS doesn't support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *TextMagicService) GetMaxBodyLength() int {
	return 1600 // TextMagic supports long SMS up to 1600 characters
}