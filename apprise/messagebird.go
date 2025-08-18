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

// MessageBirdService implements MessageBird SMS notifications
type MessageBirdService struct {
	apiKey     string
	originator string
	recipients []string
	client     *http.Client
}

// MessageBirdRequest represents the API request structure for MessageBird
type MessageBirdRequest struct {
	Recipients []string `json:"recipients"`
	Originator string   `json:"originator,omitempty"`
	Body       string   `json:"body"`
	Type       string   `json:"type,omitempty"`
	DataCoding string   `json:"datacoding,omitempty"`
}

// NewMessageBirdService creates a new MessageBird service instance
func NewMessageBirdService() Service {
	return &MessageBirdService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *MessageBirdService) GetServiceID() string {
	return "messagebird"
}

// GetDefaultPort returns the default port
func (s *MessageBirdService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *MessageBirdService) ParseURL(serviceURL *url.URL) error {
	// Extract API key from user info
	if serviceURL.User == nil {
		return fmt.Errorf("MessageBird URL must include API key")
	}

	s.apiKey = serviceURL.User.Username()
	if s.apiKey == "" {
		return fmt.Errorf("MessageBird API key cannot be empty")
	}

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
		return fmt.Errorf("MessageBird URL must specify at least one phone number")
	}

	s.recipients = phoneNumbers

	// Parse query parameters
	query := serviceURL.Query()
	if from := query.Get("from"); from != "" {
		s.originator = from
	} else if originator := query.Get("originator"); originator != "" {
		s.originator = originator
	}

	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *MessageBirdService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *MessageBirdService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}

	// Prepare request payload
	payload := MessageBirdRequest{
		Recipients: s.recipients,
		Body:       message,
		Type:       "sms",
	}

	// Add originator if specified
	if s.originator != "" {
		payload.Originator = s.originator
	}

	// Send the message
	return s.sendSMS(ctx, payload)
}

// sendSMS sends an SMS via MessageBird API
func (s *MessageBirdService) sendSMS(ctx context.Context, payload MessageBirdRequest) error {
	// MessageBird API endpoint
	apiURL := "https://rest.messagebird.com/messages"

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal MessageBird request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create MessageBird request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "AccessKey "+s.apiKey)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("MessageBird API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errorBody map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errorBody) == nil {
			if errors, ok := errorBody["errors"].([]interface{}); ok && len(errors) > 0 {
				if errorMap, ok := errors[0].(map[string]interface{}); ok {
					if description, ok := errorMap["description"].(string); ok {
						return fmt.Errorf("MessageBird API error: %s (status %d)", description, resp.StatusCode)
					}
				}
			}
		}
		return fmt.Errorf("MessageBird API returned status %d", resp.StatusCode)
	}

	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *MessageBirdService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *MessageBirdService) GetMaxBodyLength() int {
	return 1600 // MessageBird supports up to 1600 characters
}