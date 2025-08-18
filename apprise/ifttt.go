package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// IFTTTService implements IFTTT webhook notifications
type IFTTTService struct {
	webhookKey string
	event      string
	client     *http.Client
}

// IFTTTRequest represents an IFTTT webhook request
type IFTTTRequest struct {
	Value1 string `json:"value1,omitempty"`
	Value2 string `json:"value2,omitempty"`
	Value3 string `json:"value3,omitempty"`
}

// NewIFTTTService creates a new IFTTT service instance
func NewIFTTTService() Service {
	return &IFTTTService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *IFTTTService) GetServiceID() string {
	return "ifttt"
}

// GetDefaultPort returns the default port
func (s *IFTTTService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *IFTTTService) ParseURL(serviceURL *url.URL) error {
	// URL format: ifttt://webhook_key@event_name
	
	if serviceURL.User == nil {
		return fmt.Errorf("IFTTT URL must include webhook key")
	}
	
	s.webhookKey = serviceURL.User.Username()
	if s.webhookKey == "" {
		return fmt.Errorf("IFTTT webhook key cannot be empty")
	}
	
	// Extract event name from host
	if serviceURL.Host == "" {
		return fmt.Errorf("IFTTT URL must specify event name")
	}
	s.event = serviceURL.Host
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *IFTTTService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *IFTTTService) Send(ctx context.Context, req NotificationRequest) error {
	// Trigger IFTTT webhook
	return s.triggerWebhook(ctx, req)
}

// triggerWebhook triggers an IFTTT webhook
func (s *IFTTTService) triggerWebhook(ctx context.Context, req NotificationRequest) error {
	// IFTTT Webhook URL
	webhookURL := fmt.Sprintf("https://maker.ifttt.com/trigger/%s/with/key/%s", s.event, s.webhookKey)
	
	// Prepare payload with notification data
	payload := IFTTTRequest{
		Value1: req.Title,
		Value2: req.Body,
		Value3: req.NotifyType.String(),
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal IFTTT request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create IFTTT request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("IFTTT webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("IFTTT webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *IFTTTService) SupportsAttachments() bool {
	return false // IFTTT webhooks don't support file attachments directly
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *IFTTTService) GetMaxBodyLength() int {
	return 0 // No specific limit for IFTTT webhook values
}