package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const simplePushAPIURL = "https://api.simplepush.io/send"

// SimplePushService implements SimplePush push notification service
type SimplePushService struct {
	apiKey string
	client *http.Client
}

// SimplePushPayload represents the SimplePush API request payload
type SimplePushPayload struct {
	Key   string `json:"key"`
	Title string `json:"title,omitempty"`
	Msg   string `json:"msg"`
}

// NewSimplePushService creates a new SimplePush service instance
func NewSimplePushService() Service {
	return &SimplePushService{
		client: GetWebhookHTTPClient("simplepush"),
	}
}

func (s *SimplePushService) GetServiceID() string      { return "spush" }
func (s *SimplePushService) GetDefaultPort() int       { return 443 }
func (s *SimplePushService) SupportsAttachments() bool { return false }
func (s *SimplePushService) GetMaxBodyLength() int     { return 10000 }

// ParseURL parses a SimplePush service URL
// Format: spush://api_key or simplepush://api_key
func (s *SimplePushService) ParseURL(serviceURL *url.URL) error {
	switch serviceURL.Scheme {
	case "spush", "simplepush":
	default:
		return fmt.Errorf("invalid scheme: expected 'spush' or 'simplepush', got '%s'", serviceURL.Scheme)
	}

	// API key is the hostname
	s.apiKey = serviceURL.Host
	if s.apiKey == "" {
		// Try path if host is empty
		s.apiKey = serviceURL.Path
	}
	if s.apiKey == "" {
		return fmt.Errorf("simplepush API key is required (format: spush://api_key)")
	}

	return nil
}

// Send sends a notification via SimplePush
func (s *SimplePushService) Send(ctx context.Context, req NotificationRequest) error {
	payload := SimplePushPayload{
		Key:   s.apiKey,
		Title: req.Title,
		Msg:   req.Body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SimplePush payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", simplePushAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send SimplePush notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("simplepush API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a SimplePush service URL
func (s *SimplePushService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return s.ParseURL(parsedURL)
}
