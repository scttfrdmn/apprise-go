package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SynologyService implements Synology Chat incoming webhook notifications
type SynologyService struct {
	hostname string
	port     int
	token    string
	secure   bool // synologys:// for HTTPS
	client   *http.Client
}

// NewSynologyService creates a new Synology Chat service instance
func NewSynologyService() Service {
	return &SynologyService{
		client: GetWebhookHTTPClient("synology"),
		port:   -1,
	}
}

func (s *SynologyService) GetServiceID() string      { return "synology" }
func (s *SynologyService) GetDefaultPort() int       { return 443 }
func (s *SynologyService) SupportsAttachments() bool { return false }
func (s *SynologyService) GetMaxBodyLength() int     { return 0 }

// ParseURL parses a Synology Chat service URL
// Format: synology://hostname/token
// Format: synology://hostname:port/token
// Format: synologys://hostname/token  (HTTPS)
func (s *SynologyService) ParseURL(serviceURL *url.URL) error {
	switch serviceURL.Scheme {
	case "synologys":
		s.secure = true
	case "synology":
		s.secure = false
	default:
		return fmt.Errorf("invalid scheme: expected 'synology' or 'synologys', got '%s'", serviceURL.Scheme)
	}

	s.hostname = serviceURL.Hostname()
	if s.hostname == "" {
		return fmt.Errorf("synology hostname is required")
	}

	if portStr := serviceURL.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		s.port = port
	}

	s.token = strings.Trim(serviceURL.Path, "/")
	if s.token == "" {
		return fmt.Errorf("synology chat incoming webhook token is required")
	}

	return nil
}

// Send sends a notification via Synology Chat webhook
func (s *SynologyService) Send(ctx context.Context, req NotificationRequest) error {
	scheme := "http"
	if s.secure {
		scheme = "https"
	}

	var baseURL string
	if s.port > 0 {
		baseURL = fmt.Sprintf("%s://%s:%d", scheme, s.hostname, s.port)
	} else {
		baseURL = fmt.Sprintf("%s://%s", scheme, s.hostname)
	}

	message := req.Body
	if req.Title != "" {
		message = fmt.Sprintf("%s\n%s", req.Title, req.Body)
	}

	// Synology Chat expects a JSON payload URL-encoded as 'payload' form field
	chatPayload := map[string]string{"text": message}
	chatPayloadJSON, err := json.Marshal(chatPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal Synology Chat payload: %w", err)
	}

	params := url.Values{}
	params.Set("api", "SYNO.Chat.External")
	params.Set("method", "incoming")
	params.Set("version", "2")
	params.Set("token", s.token)

	endpointURL := fmt.Sprintf("%s/webapi/entry.cgi?%s", baseURL, params.Encode())

	formData := url.Values{}
	formData.Set("payload", string(chatPayloadJSON))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpointURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Synology Chat notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("synology chat API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Check Synology API response for success field
	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err == nil && !result.Success {
		return fmt.Errorf("synology chat API error code %d", result.Error.Code)
	}

	return nil
}

// TestURL validates a Synology Chat service URL
func (s *SynologyService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return s.ParseURL(parsedURL)
}
