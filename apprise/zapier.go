package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ZapierService implements Zapier webhook notifications
type ZapierService struct {
	webhookURL string
	client     *http.Client
}

// ZapierRequest represents a Zapier webhook request
type ZapierRequest struct {
	Title     string `json:"title,omitempty"`
	Message   string `json:"message"`
	Type      string `json:"type,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// NewZapierService creates a new Zapier service instance
func NewZapierService() Service {
	return &ZapierService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *ZapierService) GetServiceID() string {
	return "zapier"
}

// GetDefaultPort returns the default port
func (s *ZapierService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *ZapierService) ParseURL(serviceURL *url.URL) error {
	// URL format: zapier://hooks.zapier.com/hooks/catch/user_id/hook_id/
	// Or: zapier://webhook_id@hooks.zapier.com/hooks/catch/
	
	// Reconstruct the full webhook URL
	if serviceURL.Host == "" {
		return fmt.Errorf("Zapier URL must specify webhook host")
	}
	
	// Build webhook URL
	scheme := "https"
	if serviceURL.Scheme == "zapier+http" {
		scheme = "http"
	}
	
	s.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *ZapierService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *ZapierService) Send(ctx context.Context, req NotificationRequest) error {
	// Trigger Zapier webhook
	return s.triggerWebhook(ctx, req)
}

// triggerWebhook triggers a Zapier webhook
func (s *ZapierService) triggerWebhook(ctx context.Context, req NotificationRequest) error {
	// Prepare payload with notification data
	payload := ZapierRequest{
		Title:   req.Title,
		Message: req.Body,
		Type:    req.NotifyType.String(),
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Zapier request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Zapier request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Zapier webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Zapier webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *ZapierService) SupportsAttachments() bool {
	return false // Zapier webhooks don't support file attachments directly
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *ZapierService) GetMaxBodyLength() int {
	return 0 // No specific limit for Zapier webhook payloads
}