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

const webexAPIMessages = "https://webexapis.com/v1/messages"

// WebexService implements Cisco Webex Teams notifications
type WebexService struct {
	token  string // Bearer token
	roomID string // Webex room/space ID
	client *http.Client
}

// WebexPayload represents the Webex messages API request payload
type WebexPayload struct {
	RoomID   string `json:"roomId"`
	Text     string `json:"text,omitempty"`
	Markdown string `json:"markdown,omitempty"`
}

// NewWebexService creates a new Webex service instance
func NewWebexService() Service {
	return &WebexService{
		client: GetCloudHTTPClient("webex"),
	}
}

func (w *WebexService) GetServiceID() string      { return "webex" }
func (w *WebexService) GetDefaultPort() int       { return 443 }
func (w *WebexService) SupportsAttachments() bool { return false }
func (w *WebexService) GetMaxBodyLength() int     { return 7439 }

// ParseURL parses a Webex service URL
// Format: webex://token@room_id
// Format: wxteams://token@room_id
func (w *WebexService) ParseURL(serviceURL *url.URL) error {
	switch serviceURL.Scheme {
	case "webex", "wxteams":
	default:
		return fmt.Errorf("invalid scheme: expected 'webex' or 'wxteams', got '%s'", serviceURL.Scheme)
	}

	if serviceURL.User == nil || serviceURL.User.Username() == "" {
		return fmt.Errorf("webex bearer token is required (format: webex://token@room_id)")
	}
	w.token = serviceURL.User.Username()

	w.roomID = serviceURL.Host
	if w.roomID == "" {
		return fmt.Errorf("webex room ID is required (format: webex://token@room_id)")
	}

	return nil
}

// Send sends a notification via Cisco Webex Teams API
func (w *WebexService) Send(ctx context.Context, req NotificationRequest) error {
	payload := WebexPayload{
		RoomID: w.roomID,
	}

	// Use markdown if body format indicates it, otherwise plain text
	if req.BodyFormat == "markdown" {
		body := req.Body
		if req.Title != "" {
			body = fmt.Sprintf("**%s**\n%s", req.Title, req.Body)
		}
		payload.Markdown = body
	} else {
		body := req.Body
		if req.Title != "" {
			body = fmt.Sprintf("%s\n%s", req.Title, req.Body)
		}
		payload.Text = body
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Webex payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", webexAPIMessages, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+w.token)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := w.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Webex notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webex API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a Webex service URL
func (w *WebexService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return w.ParseURL(parsedURL)
}
