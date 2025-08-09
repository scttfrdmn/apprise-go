package apprise

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioService implements Twilio SMS/MMS notifications
type TwilioService struct {
	accountSID  string
	authToken   string
	apiKey      string // Optional API key
	fromPhone   string
	toPhones    []string
	client      *http.Client
	rateLimiter *time.Ticker // Rate limiting to 0.2 requests/sec
}

// NewTwilioService creates a new Twilio service instance
func NewTwilioService() Service {
	return &TwilioService{
		client:      &http.Client{},
		rateLimiter: time.NewTicker(5 * time.Second), // 0.2 requests per second
	}
}

// GetServiceID returns the service identifier
func (t *TwilioService) GetServiceID() string {
	return "twilio"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (t *TwilioService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Twilio service URL
// Format: twilio://ACCOUNT_SID:AUTH_TOKEN@FROM_PHONE/TO_PHONE[/TO_PHONE2/...]
// Format: twilio://ACCOUNT_SID:AUTH_TOKEN@FROM_PHONE/TO_PHONE?apikey=KEY
func (t *TwilioService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "twilio" {
		return fmt.Errorf("invalid scheme: expected 'twilio', got '%s'", serviceURL.Scheme)
	}

	// Extract Account SID and Auth Token from user info
	if serviceURL.User == nil {
		return fmt.Errorf("Twilio Account SID and Auth Token are required")
	}

	t.accountSID = serviceURL.User.Username()
	if password, hasPassword := serviceURL.User.Password(); hasPassword {
		t.authToken = password
	}

	if t.accountSID == "" || t.authToken == "" {
		return fmt.Errorf("both Twilio Account SID and Auth Token are required")
	}

	// Extract from phone number from host
	t.fromPhone = t.normalizePhoneNumber(serviceURL.Host)
	if t.fromPhone == "" {
		return fmt.Errorf("Twilio from phone number is required")
	}

	// Extract destination phone numbers from path
	if serviceURL.Path != "" {
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, part := range pathParts {
			if part != "" {
				normalizedPhone := t.normalizePhoneNumber(part)
				if normalizedPhone != "" {
					t.toPhones = append(t.toPhones, normalizedPhone)
				}
			}
		}
	}

	if len(t.toPhones) == 0 {
		return fmt.Errorf("at least one destination phone number is required")
	}

	// Parse query parameters
	query := serviceURL.Query()
	if apiKey := query.Get("apikey"); apiKey != "" {
		t.apiKey = apiKey
	}

	return nil
}

// normalizePhoneNumber normalizes phone numbers to E.164 format
func (t *TwilioService) normalizePhoneNumber(phone string) string {
	// Remove common phone number separators
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.ReplaceAll(phone, ".", "")

	// Ensure it starts with + for E.164 format
	if !strings.HasPrefix(phone, "+") && len(phone) > 0 {
		// Assume US/Canada if 10 digits
		if len(phone) == 10 {
			phone = "+1" + phone
		} else if len(phone) == 11 && strings.HasPrefix(phone, "1") {
			phone = "+" + phone
		} else {
			// For other formats, add + prefix
			phone = "+" + phone
		}
	}

	return phone
}

// Send sends an SMS notification via Twilio
func (t *TwilioService) Send(ctx context.Context, req NotificationRequest) error {
	// Combine title and body for SMS
	message := t.formatSMSMessage(req.Title, req.Body)

	// Send to each phone number with rate limiting
	var lastError error
	successCount := 0

	for _, toPhone := range t.toPhones {
		// Rate limit requests
		select {
		case <-t.rateLimiter.C:
			// Proceed with request
		case <-ctx.Done():
			return ctx.Err()
		}

		if err := t.sendToPhone(ctx, toPhone, message); err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	// Return error only if all sends failed
	if successCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}

// sendToPhone sends an SMS to a specific phone number
func (t *TwilioService) sendToPhone(ctx context.Context, toPhone, message string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

	// Prepare form data
	data := url.Values{
		"From": {t.fromPhone},
		"To":   {toPhone},
		"Body": {message},
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Twilio request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Set authentication
	auth := t.accountSID + ":" + t.authToken
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	httpReq.Header.Set("Authorization", "Basic "+encodedAuth)

	// Send request
	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Twilio SMS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Twilio response: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Twilio API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// formatSMSMessage formats title and body for SMS
func (t *TwilioService) formatSMSMessage(title, body string) string {
	var message strings.Builder

	if title != "" {
		message.WriteString(title)
		if body != "" {
			message.WriteString(": ")
		}
	}

	if body != "" {
		message.WriteString(body)
	}

	return message.String()
}

// TestURL validates a Twilio service URL
func (t *TwilioService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return t.ParseURL(parsedURL)
}

// SupportsAttachments returns false for SMS (basic implementation)
func (t *TwilioService) SupportsAttachments() bool {
	return false // Can be extended to support MMS
}

// GetMaxBodyLength returns SMS message length limit
func (t *TwilioService) GetMaxBodyLength() int {
	return 1600 // SMS limit (160 chars per segment, ~10 segments max)
}

// Example usage and URL formats:
// twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543
// twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543/+15551111111
// twilio://ACCOUNT_SID:AUTH_TOKEN@15551234567/15559876543 (US numbers)
// twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543?apikey=KEY
