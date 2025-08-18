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

// PlivoService implements Plivo SMS notifications
type PlivoService struct {
	authID    string
	authToken string
	from      string
	to        []string
	client    *http.Client
}

// PlivoSMSRequest represents a Plivo SMS request
type PlivoSMSRequest struct {
	Src  string `json:"src"`
	Dst  string `json:"dst"`
	Text string `json:"text"`
}

// NewPlivoService creates a new Plivo service instance
func NewPlivoService() Service {
	return &PlivoService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *PlivoService) GetServiceID() string {
	return "plivo"
}

// GetDefaultPort returns the default port
func (s *PlivoService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *PlivoService) ParseURL(serviceURL *url.URL) error {
	// URL format: plivo://auth_id:auth_token@host/to1/to2?from=sender
	
	if serviceURL.User == nil {
		return fmt.Errorf("Plivo URL must include Auth ID and Auth Token")
	}
	
	s.authID = serviceURL.User.Username()
	authToken, hasToken := serviceURL.User.Password()
	if !hasToken {
		return fmt.Errorf("Plivo URL must include Auth Token")
	}
	s.authToken = authToken
	
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
		return fmt.Errorf("Plivo URL must specify at least one recipient phone number")
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
func (s *PlivoService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *PlivoService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendSMS(ctx, recipient, message); err != nil {
			return fmt.Errorf("failed to send Plivo SMS to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendSMS sends an SMS to a specific recipient via Plivo API
func (s *PlivoService) sendSMS(ctx context.Context, to, message string) error {
	// Plivo API endpoint
	apiURL := fmt.Sprintf("https://api.plivo.com/v1/Account/%s/Message/", s.authID)
	
	// Prepare request payload
	payload := PlivoSMSRequest{
		Dst:  to,
		Text: message,
	}
	
	if s.from != "" {
		payload.Src = s.from
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Plivo request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Plivo request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.authID, s.authToken)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Plivo API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Plivo API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *PlivoService) SupportsAttachments() bool {
	return false // Plivo SMS doesn't support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *PlivoService) GetMaxBodyLength() int {
	return 1600 // Plivo supports long SMS up to 1600 characters
}