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

// SignalService implements Signal messenger notifications
type SignalService struct {
	serverURL string
	number    string
	to        []string
	client    *http.Client
}

// SignalRequest represents the API request structure for Signal
type SignalRequest struct {
	Message    string   `json:"message"`
	Number     string   `json:"number"`
	Recipients []string `json:"recipients"`
}

// NewSignalService creates a new Signal service instance
func NewSignalService() Service {
	return &SignalService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *SignalService) GetServiceID() string {
	return "signal"
}

// GetDefaultPort returns the default port
func (s *SignalService) GetDefaultPort() int {
	return 8080 // Default Signal REST API port
}

// ParseURL parses the service URL and configures the service
func (s *SignalService) ParseURL(serviceURL *url.URL) error {
	// URL format: signal://number@host:port/recipient1/recipient2?from=sender
	
	if serviceURL.Host == "" {
		return fmt.Errorf("Signal URL must specify server host")
	}
	
	// Extract server URL
	scheme := "http"
	if serviceURL.Port() == "443" || strings.Contains(serviceURL.Host, "https") {
		scheme = "https"
	}
	
	port := serviceURL.Port()
	if port == "" {
		port = "8080"
	}
	
	hostname := serviceURL.Hostname()
	s.serverURL = fmt.Sprintf("%s://%s:%s", scheme, hostname, port)
	
	// Extract sender number from user info
	if serviceURL.User != nil {
		s.number = serviceURL.User.Username()
	}
	
	if s.number == "" {
		return fmt.Errorf("Signal URL must specify sender number")
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
		return fmt.Errorf("Signal URL must specify at least one recipient")
	}
	
	s.to = recipients
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *SignalService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *SignalService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendMessage(ctx, recipient, message); err != nil {
			return fmt.Errorf("failed to send Signal message to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendMessage sends a message to a specific recipient via Signal API
func (s *SignalService) sendMessage(ctx context.Context, recipient, message string) error {
	// Signal REST API endpoint
	apiURL := fmt.Sprintf("%s/v2/send", s.serverURL)
	
	// Prepare request payload
	payload := SignalRequest{
		Message:    message,
		Number:     s.number,
		Recipients: []string{recipient},
	}
	
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Signal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Signal request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Signal API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Signal API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *SignalService) SupportsAttachments() bool {
	return true // Signal supports attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *SignalService) GetMaxBodyLength() int {
	return 0 // Signal has no specific body length limit
}