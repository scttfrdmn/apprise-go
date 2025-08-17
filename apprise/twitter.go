package apprise

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TwitterService implements Twitter API v2 notifications
type TwitterService struct {
	apiKey          string // Twitter API key (consumer key)
	apiKeySecret    string // Twitter API key secret (consumer secret)
	accessToken     string // OAuth access token
	accessSecret    string // OAuth access token secret
	bearerToken     string // Bearer token for application-only auth
	webhookURL      string // Webhook proxy URL for secure credential management
	proxyAPIKey     string // API key for webhook authentication
	client          *http.Client
}

// TwitterMessage represents a Twitter API v2 tweet
type TwitterMessage struct {
	Text        string                 `json:"text"`
	MediaIDs    []string              `json:"media_ids,omitempty"`
	DirectMessage TwitterDirectMessage `json:"direct_message,omitempty"`
}

// TwitterDirectMessage represents a Twitter DM
type TwitterDirectMessage struct {
	Type       string                  `json:"type,omitempty"`
	RecipientID string                 `json:"recipient_id,omitempty"`
	MediaID    string                  `json:"media_id,omitempty"`
	Text       string                  `json:"text,omitempty"`
	Ctas       []TwitterCallToAction   `json:"ctas,omitempty"`
}

// TwitterCallToAction represents Twitter DM call-to-action buttons
type TwitterCallToAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}

// TwitterWebhookPayload represents webhook proxy payload
type TwitterWebhookPayload struct {
	Service       string         `json:"service"`
	APIKey        string         `json:"api_key"`
	APIKeySecret  string         `json:"api_key_secret"`
	AccessToken   string         `json:"access_token"`
	AccessSecret  string         `json:"access_secret"`
	BearerToken   string         `json:"bearer_token,omitempty"`
	Message       TwitterMessage `json:"twitter_message"`
	MessageType   string         `json:"message_type"` // tweet, dm
	Timestamp     string         `json:"timestamp"`
	Source        string         `json:"source"`
	Version       string         `json:"version"`
}

// NewTwitterService creates a new Twitter service instance
func NewTwitterService() Service {
	return &TwitterService{
		client: GetCloudHTTPClient("twitter"),
	}
}

// GetServiceID returns the service identifier
func (t *TwitterService) GetServiceID() string {
	return "twitter"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (t *TwitterService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Twitter service URL
// Format: twitter://api_key:api_secret:access_token:access_secret@api.twitter.com/1.1/statuses/update.json?dm=user_id
// Format: twitter://bearer_token@api.twitter.com/2/tweets
// Format: twitter://proxy-key@webhook.example.com/twitter?api_key=key&api_secret=secret&access_token=token&access_secret=secret
func (t *TwitterService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "twitter" {
		return fmt.Errorf("invalid scheme: expected 'twitter', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/twitter") {
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

		// Get Twitter credentials from query parameters
		t.apiKey = query.Get("api_key")
		if t.apiKey == "" {
			return fmt.Errorf("api_key parameter is required for webhook mode")
		}

		t.apiKeySecret = query.Get("api_secret")
		if t.apiKeySecret == "" {
			return fmt.Errorf("api_secret parameter is required for webhook mode")
		}

		// OAuth 1.0a tokens (optional for app-only auth)
		t.accessToken = query.Get("access_token")
		t.accessSecret = query.Get("access_secret")

		// Bearer token for app-only auth (optional)
		t.bearerToken = query.Get("bearer_token")

		if t.accessToken == "" && t.bearerToken == "" {
			return fmt.Errorf("either access_token or bearer_token must be provided")
		}
	} else {
		// Direct Twitter API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: Twitter API credentials must be provided")
		}

		// Parse credentials from user info
		privateKey, hasKey := serviceURL.User.Password()
		
		// Check for OAuth 1.0a format (api_key:api_secret:access_token:access_secret)
		if hasKey && strings.Contains(privateKey, ":") {
			// Split private key for OAuth 1.0a (api_secret:access_token:access_secret)
			credentials := strings.Split(privateKey, ":")
			if len(credentials) != 3 {
				return fmt.Errorf("OAuth 1.0a private key requires api_secret:access_token:access_secret format")
			}

			t.apiKey = serviceURL.User.Username()
			t.apiKeySecret = credentials[0]
			t.accessToken = credentials[1]
			t.accessSecret = credentials[2]

			if t.apiKey == "" || t.apiKeySecret == "" || t.accessToken == "" || t.accessSecret == "" {
				return fmt.Errorf("all OAuth 1.0a credentials are required")
			}
		} else if hasKey {
			// Incomplete OAuth credentials (password but not the right format)
			return fmt.Errorf("OAuth 1.0a requires api_key:api_secret:access_token:access_secret format")
		} else {
			// Check if this looks like an incomplete OAuth attempt
			username := serviceURL.User.Username()
			if strings.HasSuffix(username, "_key") || strings.HasSuffix(username, "_api") || strings.HasPrefix(username, "api_") {
				// Looks like an incomplete OAuth credential attempt
				return fmt.Errorf("OAuth 1.0a requires api_key:api_secret:access_token:access_secret format")
			}
			
			// Bearer token format
			t.bearerToken = username
			if t.bearerToken == "" {
				return fmt.Errorf("bearer token is required")
			}
		}
	}

	return nil
}

// Send sends a Twitter notification
func (t *TwitterService) Send(ctx context.Context, req NotificationRequest) error {
	// Build Twitter message
	message := t.buildTwitterMessage(req)

	if t.webhookURL != "" {
		// Send via webhook proxy
		return t.sendViaWebhook(ctx, message, "tweet")
	} else {
		// Send directly to Twitter API
		return t.sendToTwitterDirectly(ctx, message)
	}
}

// buildTwitterMessage creates a Twitter message from notification request
func (t *TwitterService) buildTwitterMessage(req NotificationRequest) TwitterMessage {
	// Build tweet text
	text := t.formatTweetText(req)

	message := TwitterMessage{
		Text: text,
	}

	// Add media IDs if attachments are present
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		// For simplicity, we'll include attachment info in text
		// In a full implementation, you'd upload media first and get media IDs
		attachments := req.AttachmentMgr.GetAll()
		if len(attachments) > 0 {
			message.Text += fmt.Sprintf(" [%d attachments]", len(attachments))
		}
	}

	return message
}

// formatTweetText formats notification content for Twitter's character limit
func (t *TwitterService) formatTweetText(req NotificationRequest) string {
	const maxLength = 280 // Twitter character limit

	var text string
	if req.Title != "" && req.Body != "" {
		text = fmt.Sprintf("%s: %s", req.Title, req.Body)
	} else if req.Title != "" {
		text = req.Title
	} else {
		text = req.Body
	}

	// Add notification type indicator
	switch req.NotifyType {
	case NotifyTypeError:
		text = "🚨 " + text
	case NotifyTypeWarning:
		text = "⚠️ " + text
	case NotifyTypeSuccess:
		text = "✅ " + text
	case NotifyTypeInfo:
		text = "ℹ️ " + text
	}

	// Add tags as hashtags
	if len(req.Tags) > 0 {
		hashtags := make([]string, 0, len(req.Tags))
		for _, tag := range req.Tags {
			// Clean tag for hashtag use
			cleanTag := strings.ReplaceAll(tag, " ", "")
			cleanTag = strings.ReplaceAll(cleanTag, "-", "")
			cleanTag = strings.ReplaceAll(cleanTag, ".", "")
			if cleanTag != "" {
				hashtags = append(hashtags, "#"+cleanTag)
			}
		}
		if len(hashtags) > 0 {
			tagText := " " + strings.Join(hashtags, " ")
			if len(text+tagText) <= maxLength {
				text += tagText
			}
		}
	}

	// Add URL if present and space allows
	if req.URL != "" {
		urlText := " " + req.URL
		if len(text+urlText) <= maxLength {
			text += urlText
		}
	}

	// Truncate if necessary
	if len(text) > maxLength {
		text = text[:maxLength-3] + "..."
	}

	return text
}

// sendViaWebhook sends message via webhook proxy
func (t *TwitterService) sendViaWebhook(ctx context.Context, message TwitterMessage, messageType string) error {
	payload := TwitterWebhookPayload{
		Service:      "twitter",
		APIKey:       t.apiKey,
		APIKeySecret: t.apiKeySecret,
		AccessToken:  t.accessToken,
		AccessSecret: t.accessSecret,
		BearerToken:  t.bearerToken,
		Message:      message,
		MessageType:  messageType,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Source:       "apprise-go",
		Version:      GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Twitter webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Twitter webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if t.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", t.proxyAPIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Twitter webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitter webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendToTwitterDirectly sends message directly to Twitter API
func (t *TwitterService) sendToTwitterDirectly(ctx context.Context, message TwitterMessage) error {
	if t.bearerToken != "" {
		// Use Twitter API v2 with Bearer Token
		return t.sendTweetV2(ctx, message)
	} else {
		// Use Twitter API v1.1 with OAuth 1.0a
		return t.sendTweetV1(ctx, message)
	}
}

// sendTweetV2 sends tweet via Twitter API v2
func (t *TwitterService) sendTweetV2(ctx context.Context, message TwitterMessage) error {
	apiURL := "https://api.twitter.com/2/tweets"

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal tweet data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Twitter API request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.bearerToken))
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send tweet: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitter api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendTweetV1 sends tweet via Twitter API v1.1 with OAuth 1.0a
func (t *TwitterService) sendTweetV1(ctx context.Context, message TwitterMessage) error {
	apiURL := "https://api.twitter.com/1.1/statuses/update.json"

	// Prepare form data
	data := url.Values{}
	data.Set("status", message.Text)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Twitter API request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// Add OAuth 1.0a signature
	if err := t.signOAuth1Request(httpReq, data); err != nil {
		return fmt.Errorf("failed to sign OAuth request: %w", err)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send tweet: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitter api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// signOAuth1Request adds OAuth 1.0a signature to HTTP request
func (t *TwitterService) signOAuth1Request(req *http.Request, data url.Values) error {
	// OAuth 1.0a parameters
	oauthParams := map[string]string{
		"oauth_consumer_key":     t.apiKey,
		"oauth_token":            t.accessToken,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconv.FormatInt(time.Now().Unix(), 10),
		"oauth_nonce":            t.generateNonce(),
		"oauth_version":          "1.0",
	}

	// Combine OAuth parameters with request parameters
	allParams := make(map[string]string)
	for key, value := range oauthParams {
		allParams[key] = value
	}
	for key, values := range data {
		if len(values) > 0 {
			allParams[key] = values[0]
		}
	}

	// Create signature base string
	baseString := t.createSignatureBaseString(req.Method, req.URL.String(), allParams)

	// Create signing key
	signingKey := url.QueryEscape(t.apiKeySecret) + "&" + url.QueryEscape(t.accessSecret)

	// Generate signature
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(baseString))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	oauthParams["oauth_signature"] = signature

	// Build Authorization header
	authHeader := "OAuth "
	var authParts []string
	for key, value := range oauthParams {
		authParts = append(authParts, fmt.Sprintf(`%s="%s"`, key, url.QueryEscape(value)))
	}
	authHeader += strings.Join(authParts, ", ")

	req.Header.Set("Authorization", authHeader)
	return nil
}

// createSignatureBaseString creates the OAuth 1.0a signature base string
func (t *TwitterService) createSignatureBaseString(method, rawURL string, params map[string]string) string {
	// Parse URL to get base URL without query parameters
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	
	baseURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)

	// Encode and sort parameters
	var paramPairs []string
	for key, value := range params {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
	}
	sort.Strings(paramPairs)
	
	paramString := strings.Join(paramPairs, "&")

	// Create signature base string
	return fmt.Sprintf("%s&%s&%s",
		strings.ToUpper(method),
		url.QueryEscape(baseURL),
		url.QueryEscape(paramString))
}

// generateNonce generates a random nonce for OAuth
func (t *TwitterService) generateNonce() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Helper methods

func (t *TwitterService) validateCredentials() error {
	if t.bearerToken == "" && (t.apiKey == "" || t.apiKeySecret == "" || t.accessToken == "" || t.accessSecret == "") {
		return fmt.Errorf("either bearer token or complete OAuth 1.0a credentials required")
	}
	return nil
}

// TestURL validates a Twitter service URL
func (t *TwitterService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return t.ParseURL(parsedURL)
}

// SupportsAttachments returns true (Twitter supports media attachments)
func (t *TwitterService) SupportsAttachments() bool {
	return true // Twitter supports media attachments (images, videos, etc.)
}

// GetMaxBodyLength returns Twitter's character limit
func (t *TwitterService) GetMaxBodyLength() int {
	return 280 // Twitter character limit
}

// Example usage and URL formats:
// twitter://api_key:api_secret:access_token:access_secret@api.twitter.com/1.1/statuses/update.json
// twitter://bearer_token@api.twitter.com/2/tweets
// twitter://proxy-key@webhook.example.com/twitter?api_key=key&api_secret=secret&access_token=token&access_secret=secret&bearer_token=bearer