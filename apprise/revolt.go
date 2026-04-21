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
)

const revoltWebhookBase = "https://rvlt.gg/webhooks/%s/%s"

// RevoltService implements Revolt webhook notifications
type RevoltService struct {
	webhookID    string
	webhookToken string
	client       *http.Client
}

// RevoltPayload represents the Revolt webhook request payload
type RevoltPayload struct {
	Content string `json:"content"`
}

// NewRevoltService creates a new Revolt service instance
func NewRevoltService() Service {
	return &RevoltService{
		client: GetWebhookHTTPClient("revolt"),
	}
}

func (r *RevoltService) GetServiceID() string      { return "revolt" }
func (r *RevoltService) GetDefaultPort() int       { return 443 }
func (r *RevoltService) SupportsAttachments() bool { return false }
func (r *RevoltService) GetMaxBodyLength() int     { return 2000 }

// ParseURL parses a Revolt webhook URL
// Format: revolt://webhook_id/webhook_token
func (r *RevoltService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "revolt" {
		return fmt.Errorf("invalid scheme: expected 'revolt', got '%s'", serviceURL.Scheme)
	}

	r.webhookID = serviceURL.Host
	if r.webhookID == "" {
		return fmt.Errorf("revolt webhook ID is required")
	}

	r.webhookToken = strings.Trim(serviceURL.Path, "/")
	if r.webhookToken == "" {
		return fmt.Errorf("revolt webhook token is required")
	}

	return nil
}

// Send sends a notification via Revolt webhook
func (r *RevoltService) Send(ctx context.Context, req NotificationRequest) error {
	webhookURL := fmt.Sprintf(revoltWebhookBase, r.webhookID, r.webhookToken)

	var content string
	if req.Title != "" {
		content = fmt.Sprintf("**%s**\n%s", req.Title, req.Body)
	} else {
		content = req.Body
	}

	payload := RevoltPayload{Content: content}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Revolt payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Revolt notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revolt API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a Revolt service URL
func (r *RevoltService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return r.ParseURL(parsedURL)
}
