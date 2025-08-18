package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// MailgunService implements Mailgun email notifications
type MailgunService struct {
	apiKey    string
	domain    string
	fromEmail string
	fromName  string
	to        []string
	region    string // us, eu, etc.
	client    *http.Client
}

// NewMailgunService creates a new Mailgun service instance
func NewMailgunService() Service {
	return &MailgunService{
		client: &http.Client{},
		region: "us", // Default to US region
	}
}

// GetServiceID returns the service identifier
func (s *MailgunService) GetServiceID() string {
	return "mailgun"
}

// GetDefaultPort returns the default port
func (s *MailgunService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *MailgunService) ParseURL(serviceURL *url.URL) error {
	// URL format: mailgun://api_key@domain.com/to1/to2?from=sender@domain.com&name=sender_name&region=us
	
	if serviceURL.User == nil {
		return fmt.Errorf("Mailgun URL must include API key")
	}
	
	s.apiKey = serviceURL.User.Username()
	if s.apiKey == "" {
		return fmt.Errorf("Mailgun API key cannot be empty")
	}
	
	// Extract domain from host
	if serviceURL.Host == "" {
		return fmt.Errorf("Mailgun URL must specify domain")
	}
	s.domain = serviceURL.Host
	
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
		return fmt.Errorf("Mailgun URL must specify at least one recipient email address")
	}
	s.to = recipients
	
	// Parse query parameters
	query := serviceURL.Query()
	if from := query.Get("from"); from != "" {
		s.fromEmail = from
	} else {
		// Default from email using domain
		s.fromEmail = "noreply@" + s.domain
	}
	
	if name := query.Get("name"); name != "" {
		s.fromName = name
	}
	
	if region := query.Get("region"); region != "" {
		s.region = region
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *MailgunService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *MailgunService) Send(ctx context.Context, req NotificationRequest) error {
	// Build email content
	subject := "Notification"
	if req.Title != "" {
		subject = req.Title
	}
	
	// Send to each recipient
	for _, recipient := range s.to {
		if err := s.sendEmail(ctx, recipient, subject, req.Body); err != nil {
			return fmt.Errorf("failed to send Mailgun email to %s: %w", recipient, err)
		}
	}
	
	return nil
}

// sendEmail sends an email to a specific recipient via Mailgun API
func (s *MailgunService) sendEmail(ctx context.Context, to, subject, body string) error {
	// Build Mailgun API endpoint based on region
	var baseURL string
	switch s.region {
	case "eu":
		baseURL = "https://api.eu.mailgun.net/v3"
	default:
		baseURL = "https://api.mailgun.net/v3"
	}
	
	apiURL := fmt.Sprintf("%s/%s/messages", baseURL, s.domain)
	
	// Prepare form data
	formData := url.Values{}
	
	// Set from address
	if s.fromName != "" {
		formData.Set("from", fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail))
	} else {
		formData.Set("from", s.fromEmail)
	}
	
	formData.Set("to", to)
	formData.Set("subject", subject)
	formData.Set("text", body)
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Mailgun request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", s.apiKey)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Mailgun API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Mailgun API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *MailgunService) SupportsAttachments() bool {
	return true // Mailgun supports attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *MailgunService) GetMaxBodyLength() int {
	return 0 // Mailgun has no specific body length limit
}