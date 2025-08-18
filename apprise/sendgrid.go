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

// SendGridService implements SendGrid email notifications
type SendGridService struct {
	apiKey    string
	fromEmail string
	fromName  string
	to        []string
	client    *http.Client
}

// SendGridPersonalization represents email personalization settings
type SendGridPersonalization struct {
	To      []SendGridEmail `json:"to"`
	Subject string          `json:"subject,omitempty"`
}

// SendGridEmail represents an email address
type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendGridContent represents email content
type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendGridRequest represents the SendGrid API request structure
type SendGridRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	Subject          string                    `json:"subject,omitempty"`
	Content          []SendGridContent         `json:"content"`
}

// NewSendGridService creates a new SendGrid service instance
func NewSendGridService() Service {
	return &SendGridService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *SendGridService) GetServiceID() string {
	return "sendgrid"
}

// GetDefaultPort returns the default port
func (s *SendGridService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *SendGridService) ParseURL(serviceURL *url.URL) error {
	// URL format: sendgrid://api_key@host/to1/to2?from=sender@example.com&name=sender_name
	
	if serviceURL.User == nil {
		return fmt.Errorf("SendGrid URL must include API key")
	}
	
	s.apiKey = serviceURL.User.Username()
	if s.apiKey == "" {
		return fmt.Errorf("SendGrid API key cannot be empty")
	}
	
	// Extract from email from query parameters, required for SendGrid
	query := serviceURL.Query()
	if from := query.Get("from"); from != "" {
		s.fromEmail = from
	} else {
		return fmt.Errorf("SendGrid URL must specify from email address via 'from' parameter")
	}
	
	// Extract recipient emails from path
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
		return fmt.Errorf("SendGrid URL must specify at least one recipient email address")
	}
	s.to = recipients
	
	// Parse remaining query parameters (from already parsed above)
	if name := query.Get("name"); name != "" {
		s.fromName = name
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *SendGridService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *SendGridService) Send(ctx context.Context, req NotificationRequest) error {
	// Build email content
	subject := "Notification"
	if req.Title != "" {
		subject = req.Title
	}
	
	// Create recipient list
	toEmails := make([]SendGridEmail, len(s.to))
	for i, email := range s.to {
		toEmails[i] = SendGridEmail{Email: email}
	}
	
	// Prepare request payload
	payload := SendGridRequest{
		Personalizations: []SendGridPersonalization{
			{
				To:      toEmails,
				Subject: subject,
			},
		},
		From: SendGridEmail{
			Email: s.fromEmail,
			Name:  s.fromName,
		},
		Content: []SendGridContent{
			{
				Type:  "text/plain",
				Value: req.Body,
			},
		},
	}
	
	// Send the email
	return s.sendEmail(ctx, payload)
}

// sendEmail sends an email via SendGrid API
func (s *SendGridService) sendEmail(ctx context.Context, payload SendGridRequest) error {
	// SendGrid API endpoint
	apiURL := "https://api.sendgrid.com/v3/mail/send"
	
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SendGrid request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create SendGrid request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SendGrid API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errorBody map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errorBody) == nil {
			if errors, ok := errorBody["errors"].([]interface{}); ok && len(errors) > 0 {
				if errorMap, ok := errors[0].(map[string]interface{}); ok {
					if message, ok := errorMap["message"].(string); ok {
						return fmt.Errorf("SendGrid API error: %s (status %d)", message, resp.StatusCode)
					}
				}
			}
		}
		return fmt.Errorf("SendGrid API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *SendGridService) SupportsAttachments() bool {
	return true // SendGrid supports attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *SendGridService) GetMaxBodyLength() int {
	return 0 // SendGrid has no specific body length limit
}