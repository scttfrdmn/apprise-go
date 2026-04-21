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

const signl4APIBase = "https://connect.signl4.com/webhook/%s"

// SIGNL4Service implements SIGNL4 on-call alerting via webhook
type SIGNL4Service struct {
	teamSecret string
	client     *http.Client
}

// SIGNL4Payload represents the SIGNL4 webhook request payload
type SIGNL4Payload struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	S4Status   string `json:"X-S4-Status"`
	S4SourceSystem string `json:"X-S4-SourceSystem,omitempty"`
}

// NewSIGNL4Service creates a new SIGNL4 service instance
func NewSIGNL4Service() Service {
	return &SIGNL4Service{
		client: GetWebhookHTTPClient("signl4"),
	}
}

func (s *SIGNL4Service) GetServiceID() string      { return "signl4" }
func (s *SIGNL4Service) GetDefaultPort() int       { return 443 }
func (s *SIGNL4Service) SupportsAttachments() bool { return false }
func (s *SIGNL4Service) GetMaxBodyLength() int     { return 0 }

// ParseURL parses a SIGNL4 service URL
// Format: signl4://team_secret
func (s *SIGNL4Service) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "signl4" {
		return fmt.Errorf("invalid scheme: expected 'signl4', got '%s'", serviceURL.Scheme)
	}

	s.teamSecret = serviceURL.Host
	if s.teamSecret == "" {
		return fmt.Errorf("signl4 team secret is required (format: signl4://team_secret)")
	}

	return nil
}

// Send sends a notification via SIGNL4
func (s *SIGNL4Service) Send(ctx context.Context, req NotificationRequest) error {
	webhookURL := fmt.Sprintf(signl4APIBase, s.teamSecret)

	payload := SIGNL4Payload{
		Title:    req.Title,
		Body:     req.Body,
		S4Status: "new",
		S4SourceSystem: "Apprise-Go",
	}
	if payload.Title == "" {
		payload.Title = req.Body
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SIGNL4 payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send SIGNL4 notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signl4 API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a SIGNL4 service URL
func (s *SIGNL4Service) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return s.ParseURL(parsedURL)
}
