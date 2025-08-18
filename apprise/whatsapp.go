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

// WhatsAppService implements WhatsApp Business API notifications
type WhatsAppService struct {
	accessToken string
	phoneID     string
	to          []string
	client      *http.Client
}

// WhatsAppMessage represents a WhatsApp message
type WhatsAppMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

// WhatsAppRequest represents the API request structure for WhatsApp Business API
type WhatsAppRequest struct {
	MessagingProduct string            `json:"messaging_product"`
	To               string            `json:"to"`
	Type             string            `json:"type"`
	Text             map[string]string `json:"text"`
}

// NewWhatsAppService creates a new WhatsApp Business API service instance
func NewWhatsAppService() Service {
	return &WhatsAppService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *WhatsAppService) GetServiceID() string {
	return "whatsapp"
}

// GetDefaultPort returns the default port
func (s *WhatsAppService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *WhatsAppService) ParseURL(serviceURL *url.URL) error {
	// URL format: whatsapp://phone_id@access_token/recipient1/recipient2
	
	if serviceURL.Host == "" {
		return fmt.Errorf("WhatsApp URL must specify access token as host")
	}
	
	// Extract access token from host
	s.accessToken = serviceURL.Host
	
	// Extract phone ID from user info
	if serviceURL.User != nil {
		s.phoneID = serviceURL.User.Username()
	}
	
	if s.phoneID == "" {
		return fmt.Errorf("WhatsApp URL must specify Phone Number ID")
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
		return fmt.Errorf("WhatsApp URL must specify at least one recipient")
	}
	
	s.to = recipients
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *WhatsAppService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *WhatsAppService) Send(ctx context.Context, req NotificationRequest) error {
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n" + message
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendMessage(ctx, recipient, message); err != nil {
			return fmt.Errorf("failed to send WhatsApp message to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendMessage sends a message to a specific recipient via WhatsApp Business API
func (s *WhatsAppService) sendMessage(ctx context.Context, recipient, message string) error {
	// WhatsApp Business API endpoint
	apiURL := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", s.phoneID)
	
	// Prepare request payload
	payload := WhatsAppRequest{
		MessagingProduct: "whatsapp",
		To:               recipient,
		Type:             "text",
		Text: map[string]string{
			"body": message,
		},
	}
	
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("WhatsApp API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errorBody map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errorBody) == nil {
			if errorData, ok := errorBody["error"].(map[string]interface{}); ok {
				if message, ok := errorData["message"].(string); ok {
					return fmt.Errorf("WhatsApp API error: %s (status %d)", message, resp.StatusCode)
				}
			}
		}
		return fmt.Errorf("WhatsApp API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *WhatsAppService) SupportsAttachments() bool {
	return true // WhatsApp Business API supports attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *WhatsAppService) GetMaxBodyLength() int {
	return 4096 // WhatsApp has a practical limit of ~4096 characters
}