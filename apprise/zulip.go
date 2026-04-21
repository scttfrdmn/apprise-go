package apprise

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ZulipService implements Zulip team messaging notifications
type ZulipService struct {
	domain   string // e.g. myorg.zulipchat.com
	botEmail string // bot email used as HTTP Basic username
	apiKey   string // bot API key used as HTTP Basic password
	stream   string // stream (channel) name
	topic    string // topic within the stream
	client   *http.Client
}

// NewZulipService creates a new Zulip service instance
func NewZulipService() Service {
	return &ZulipService{
		client: GetDefaultHTTPClient(),
	}
}

func (z *ZulipService) GetServiceID() string      { return "zulip" }
func (z *ZulipService) GetDefaultPort() int       { return 443 }
func (z *ZulipService) SupportsAttachments() bool { return false }
func (z *ZulipService) GetMaxBodyLength() int     { return 10000 }

// ParseURL parses a Zulip service URL
// Format: zulip://botname@domain/api_key/stream/topic
// The bot email is constructed as botname@domain (same domain as server)
func (z *ZulipService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "zulip" {
		return fmt.Errorf("invalid scheme: expected 'zulip', got '%s'", serviceURL.Scheme)
	}

	if serviceURL.User == nil || serviceURL.User.Username() == "" {
		return fmt.Errorf("zulip bot name is required (format: zulip://botname@domain/api_key/stream/topic)")
	}

	botName := serviceURL.User.Username()
	z.domain = serviceURL.Hostname()
	if z.domain == "" {
		return fmt.Errorf("zulip domain is required")
	}

	// Bot email is botname@domain
	z.botEmail = fmt.Sprintf("%s@%s", botName, z.domain)

	// Path: /api_key/stream/topic
	parts := strings.SplitN(strings.Trim(serviceURL.Path, "/"), "/", 3)
	if len(parts) < 1 || parts[0] == "" {
		return fmt.Errorf("zulip API key is required in path")
	}
	z.apiKey = parts[0]

	if len(parts) >= 2 && parts[1] != "" {
		z.stream = parts[1]
	} else {
		z.stream = "general"
	}

	if len(parts) >= 3 && parts[2] != "" {
		z.topic = parts[2]
	} else {
		z.topic = "notifications"
	}

	// Allow overrides via query params
	query := serviceURL.Query()
	if v := query.Get("stream"); v != "" {
		z.stream = v
	}
	if v := query.Get("topic"); v != "" {
		z.topic = v
	}

	return nil
}

// Send sends a notification via Zulip REST API
func (z *ZulipService) Send(ctx context.Context, req NotificationRequest) error {
	apiURL := fmt.Sprintf("https://%s/api/v1/messages", z.domain)

	content := req.Body
	if req.Title != "" {
		content = fmt.Sprintf("**%s**\n%s", req.Title, req.Body)
	}

	formData := url.Values{}
	formData.Set("type", "stream")
	formData.Set("to", z.stream)
	formData.Set("topic", z.topic)
	formData.Set("content", content)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.SetBasicAuth(z.botEmail, z.apiKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := z.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Zulip notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zulip API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a Zulip service URL
func (z *ZulipService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return z.ParseURL(parsedURL)
}
