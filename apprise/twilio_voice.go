package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TwilioVoiceService implements voice call notifications via Twilio
type TwilioVoiceService struct {
	accountSID    string // Twilio Account SID
	authToken     string // Twilio Auth Token
	fromNumber    string // From phone number (Twilio verified number)
	toNumbers     []string // Destination phone numbers
	webhookURL    string // Webhook proxy URL for secure credential management
	proxyAPIKey   string // API key for webhook authentication
	voiceLanguage string // Voice language (e.g., en-US, es-ES)
	voiceGender   string // Voice gender (male, female)
	client        *http.Client
}

// TwilioVoiceCall represents a Twilio voice call request
type TwilioVoiceCall struct {
	From string `json:"From"`
	To   string `json:"To"`
	Url  string `json:"Url,omitempty"`  // TwiML URL for custom voice content
	Twiml string `json:"Twiml,omitempty"` // Inline TwiML content
}

// TwilioVoiceWebhookPayload represents webhook proxy payload
type TwilioVoiceWebhookPayload struct {
	Service     string             `json:"service"`
	AccountSID  string             `json:"account_sid"`
	Calls       []TwilioVoiceCall  `json:"calls"`
	TwiML       string             `json:"twiml"`
	Language    string             `json:"language"`
	Gender      string             `json:"gender"`
	Timestamp   string             `json:"timestamp"`
	Source      string             `json:"source"`
	Version     string             `json:"version"`
}

// TwilioVoiceResponse represents Twilio API response
type TwilioVoiceResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewTwilioVoiceService creates a new Twilio Voice service instance
func NewTwilioVoiceService() Service {
	return &TwilioVoiceService{
		client:        GetCloudHTTPClient("twilio-voice"),
		voiceLanguage: "en-US", // Default language
		voiceGender:   "female", // Default gender
	}
}

// GetServiceID returns the service identifier
func (t *TwilioVoiceService) GetServiceID() string {
	return "twilio-voice"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (t *TwilioVoiceService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Twilio Voice service URL
// Format: twilio-voice://account_sid:auth_token@api.twilio.com/+1234567890/+1987654321?language=en-US&gender=female
// Format: twilio-voice://proxy-key@webhook.example.com/twilio-voice?account_sid=sid&auth_token=token&from=+1234567890&to=+1987654321
func (t *TwilioVoiceService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "twilio-voice" {
		return fmt.Errorf("invalid scheme: expected 'twilio-voice', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/twilio-voice") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		t.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			t.proxyAPIKey = serviceURL.User.Username()
		}

		// Get Twilio credentials from query parameters
		t.accountSID = query.Get("account_sid")
		if t.accountSID == "" {
			return fmt.Errorf("account_sid parameter is required for webhook mode")
		}

		t.authToken = query.Get("auth_token")
		if t.authToken == "" {
			return fmt.Errorf("auth_token parameter is required for webhook mode")
		}

		// Get from number
		t.fromNumber = strings.TrimSpace(query.Get("from"))
		if t.fromNumber == "" {
			return fmt.Errorf("from parameter is required for webhook mode")
		}
		// Add + prefix if missing
		if !strings.HasPrefix(t.fromNumber, "+") {
			t.fromNumber = "+" + t.fromNumber
		}

		// Get to numbers
		if toNumbers := query.Get("to"); toNumbers != "" {
			numberList := strings.Split(toNumbers, ",")
			for _, number := range numberList {
				number = strings.TrimSpace(number)
				if number != "" {
					// Add + prefix if missing
					if !strings.HasPrefix(number, "+") {
						number = "+" + number
					}
					t.toNumbers = append(t.toNumbers, number)
				}
			}
		}
	} else {
		// Direct Twilio API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: account_sid and auth_token must be provided")
		}

		t.accountSID = serviceURL.User.Username()
		if t.accountSID == "" {
			return fmt.Errorf("twilio account SID is required")
		}

		if token, hasToken := serviceURL.User.Password(); hasToken {
			t.authToken = token
		}
		if t.authToken == "" {
			return fmt.Errorf("twilio auth token is required")
		}

		// Parse phone numbers from path
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		if len(pathParts) < 1 || pathParts[0] == "" {
			return fmt.Errorf("from phone number is required in path")
		}

		t.fromNumber = pathParts[0]
		if !t.isValidPhoneNumber(t.fromNumber) {
			return fmt.Errorf("invalid from phone number format: %s", t.fromNumber)
		}

		// Additional destination numbers in path
		for i := 1; i < len(pathParts); i++ {
			if pathParts[i] != "" {
				if !t.isValidPhoneNumber(pathParts[i]) {
					return fmt.Errorf("invalid to phone number format: %s", pathParts[i])
				}
				t.toNumbers = append(t.toNumbers, pathParts[i])
			}
		}

		// Additional destination numbers in query
		if toNumbers := query.Get("to"); toNumbers != "" {
			additionalNumbers := strings.Split(toNumbers, ",")
			for _, number := range additionalNumbers {
				number = strings.TrimSpace(number)
				if number != "" {
					// Add + prefix if missing
					if !strings.HasPrefix(number, "+") {
						number = "+" + number
					}
					if !t.isValidPhoneNumber(number) {
						return fmt.Errorf("invalid to phone number format: %s", number)
					}
					t.toNumbers = append(t.toNumbers, number)
				}
			}
		}
	}

	// Parse optional voice parameters
	if language := query.Get("language"); language != "" {
		if t.isValidLanguage(language) {
			t.voiceLanguage = language
		} else {
			return fmt.Errorf("invalid voice language: %s", language)
		}
	}

	if gender := query.Get("gender"); gender != "" {
		if t.isValidGender(gender) {
			t.voiceGender = gender
		} else {
			return fmt.Errorf("invalid voice gender: %s (valid: male, female)", gender)
		}
	}

	if len(t.toNumbers) == 0 {
		return fmt.Errorf("at least one destination phone number is required")
	}

	return nil
}

// Send sends a voice notification via Twilio
func (t *TwilioVoiceService) Send(ctx context.Context, req NotificationRequest) error {
	// Generate TwiML content for the voice message
	twiml := t.generateTwiML(req)
	
	calls := make([]TwilioVoiceCall, 0, len(t.toNumbers))
	for _, toNumber := range t.toNumbers {
		call := TwilioVoiceCall{
			From:  t.fromNumber,
			To:    toNumber,
			Twiml: twiml,
		}
		calls = append(calls, call)
	}

	if t.webhookURL != "" {
		// Send via webhook proxy
		return t.sendViaWebhook(ctx, calls, twiml)
	} else {
		// Send directly to Twilio API
		return t.sendCallsDirectly(ctx, calls)
	}
}

// generateTwiML creates TwiML (Twilio Markup Language) for voice synthesis
func (t *TwilioVoiceService) generateTwiML(req NotificationRequest) string {
	// Create voice message content
	message := req.Title
	if req.Body != "" {
		if message != "" {
			message += ". " + req.Body
		} else {
			message = req.Body
		}
	}

	// Add notification type context
	switch req.NotifyType {
	case NotifyTypeError:
		message = "Alert: " + message
	case NotifyTypeWarning:
		message = "Warning: " + message
	case NotifyTypeSuccess:
		message = "Success: " + message
	}

	// Clean message for voice synthesis
	message = t.cleanMessageForVoice(message)

	// Build TwiML with voice settings
	twiml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say voice="%s" language="%s">%s</Say>
</Response>`, t.getVoiceName(), t.voiceLanguage, message)

	return twiml
}

// sendViaWebhook sends calls via webhook proxy
func (t *TwilioVoiceService) sendViaWebhook(ctx context.Context, calls []TwilioVoiceCall, twiml string) error {
	payload := TwilioVoiceWebhookPayload{
		Service:    "twilio-voice",
		AccountSID: t.accountSID,
		Calls:      calls,
		TwiML:      twiml,
		Language:   t.voiceLanguage,
		Gender:     t.voiceGender,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Source:     "apprise-go",
		Version:    GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Twilio Voice webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Twilio Voice webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if t.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", t.proxyAPIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Twilio Voice webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio voice webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendCallsDirectly sends calls directly to Twilio API
func (t *TwilioVoiceService) sendCallsDirectly(ctx context.Context, calls []TwilioVoiceCall) error {
	callsURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", t.accountSID)

	for _, call := range calls {
		if err := t.sendSingleCall(ctx, callsURL, call); err != nil {
			return err
		}
	}

	return nil
}

// sendSingleCall sends a single voice call
func (t *TwilioVoiceService) sendSingleCall(ctx context.Context, callsURL string, call TwilioVoiceCall) error {
	// Twilio expects form-encoded data for calls
	formData := url.Values{}
	formData.Set("From", call.From)
	formData.Set("To", call.To)
	formData.Set("Twiml", call.Twiml)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", callsURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create call request: %w", err)
	}

	t.setAuthHeaders(httpReq)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send call to %s: %w", call.To, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio API error for %s (status %d): %s", call.To, resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (t *TwilioVoiceService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("Accept", "application/json")
	
	// Use basic authentication
	req.SetBasicAuth(t.accountSID, t.authToken)
}

func (t *TwilioVoiceService) isValidPhoneNumber(number string) bool {
	// Basic E.164 format validation (starts with +, followed by digits)
	if len(number) < 7 || len(number) > 15 {
		return false
	}
	
	if !strings.HasPrefix(number, "+") {
		return false
	}
	
	// Check that everything after + is digits
	for _, char := range number[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

func (t *TwilioVoiceService) isValidLanguage(language string) bool {
	// Common Twilio supported languages
	validLanguages := []string{
		"en-US", "en-GB", "es-ES", "es-MX", "fr-FR", "de-DE",
		"it-IT", "ja-JP", "ko-KR", "pt-BR", "ru-RU", "zh-CN",
		"zh-TW", "nl-NL", "sv-SE", "da-DK", "nb-NO", "pl-PL",
	}
	
	for _, valid := range validLanguages {
		if strings.EqualFold(language, valid) {
			return true
		}
	}
	return false
}

func (t *TwilioVoiceService) isValidGender(gender string) bool {
	return strings.EqualFold(gender, "male") || strings.EqualFold(gender, "female")
}

func (t *TwilioVoiceService) getVoiceName() string {
	// Map language and gender to Twilio voice names
	switch t.voiceLanguage {
	case "en-US":
		if strings.EqualFold(t.voiceGender, "male") {
			return "man"
		}
		return "woman"
	case "en-GB":
		if strings.EqualFold(t.voiceGender, "male") {
			return "man"
		}
		return "woman"
	default:
		if strings.EqualFold(t.voiceGender, "male") {
			return "man"
		}
		return "woman"
	}
}

func (t *TwilioVoiceService) cleanMessageForVoice(message string) string {
	// Remove or replace problematic characters for voice synthesis
	message = strings.ReplaceAll(message, "&", " and ")
	message = strings.ReplaceAll(message, "<", " less than ")
	message = strings.ReplaceAll(message, ">", " greater than ")
	message = strings.ReplaceAll(message, "\"", " quote ")
	message = strings.ReplaceAll(message, "'", "")
	
	// Replace multiple spaces with single space
	for strings.Contains(message, "  ") {
		message = strings.ReplaceAll(message, "  ", " ")
	}
	
	return strings.TrimSpace(message)
}

// TestURL validates a Twilio Voice service URL
func (t *TwilioVoiceService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return t.ParseURL(parsedURL)
}

// SupportsAttachments returns false (voice calls don't support file attachments)
func (t *TwilioVoiceService) SupportsAttachments() bool {
	return false // Voice calls cannot include file attachments
}

// GetMaxBodyLength returns Twilio's content length limit for voice synthesis
func (t *TwilioVoiceService) GetMaxBodyLength() int {
	return 4096 // Reasonable limit for voice synthesis (about 4-5 minutes of speech)
}

// Example usage and URL formats:
// twilio-voice://account_sid:auth_token@api.twilio.com/+1234567890/+1987654321?language=en-US&gender=female
// twilio-voice://account_sid:auth_token@api.twilio.com/+1234567890?to=+1987654321,+1456789012&language=es-ES
// twilio-voice://proxy-key@webhook.example.com/twilio-voice?account_sid=sid&auth_token=token&from=+1234567890&to=+1987654321