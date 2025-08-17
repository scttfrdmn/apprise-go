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

// MattermostService implements Mattermost team collaboration notifications
type MattermostService struct {
	serverURL string
	token     string
	username  string
	password  string
	channels  []string
	botName   string
	iconURL   string
	iconEmoji string
	client    *http.Client
}

// NewMattermostService creates a new Mattermost service instance
func NewMattermostService() Service {
	return &MattermostService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (m *MattermostService) GetServiceID() string {
	return "mattermost"
}

// GetDefaultPort returns the default port (443 for HTTPS, 80 for HTTP)
func (m *MattermostService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Mattermost service URL
// Format: mattermost://username:password@server[:port]/channel
// Format: mattermost://token@server[:port]/channel
// Format: mmosts://username:password@server[:port]/channel  (HTTPS)
func (m *MattermostService) ParseURL(serviceURL *url.URL) error {
	scheme := strings.ToLower(serviceURL.Scheme)
	if scheme != "mattermost" && scheme != "mmosts" {
		return fmt.Errorf("invalid scheme: expected 'mattermost' or 'mmosts', got '%s'", serviceURL.Scheme)
	}

	// Build server URL
	protocol := "http"
	if scheme == "mmosts" {
		protocol = "https"
	}

	host := serviceURL.Host
	if host == "" {
		return fmt.Errorf("mattermost server host is required")
	}

	// Add default port if not specified
	if !strings.Contains(host, ":") {
		if scheme == "mmosts" {
			host += ":443"
		} else {
			host += ":8065" // Default Mattermost port
		}
	}

	m.serverURL = fmt.Sprintf("%s://%s", protocol, host)

	// Parse authentication
	if serviceURL.User != nil {
		username := serviceURL.User.Username()
		password, hasPassword := serviceURL.User.Password()

		if hasPassword {
			// Username:password authentication
			m.username = username
			m.password = password
		} else {
			// Could be token as username, or username with token in query
			// We'll check query parameters first
			m.username = username // Preserve username for potential query token
		}
	}

	// Extract channels from path
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, part := range pathParts {
			if part != "" {
				// Normalize channel format
				channel := m.normalizeChannelName(part)
				m.channels = append(m.channels, channel)
			}
		}
	}

	// Handle channel in fragment (URLs like mattermost://token@server/#channel)
	if serviceURL.Fragment != "" {
		// Fragment was likely a channel that started with # but URL parsing removed it
		fragment := serviceURL.Fragment
		if !strings.HasPrefix(fragment, "#") && !strings.HasPrefix(fragment, "@") {
			// Add back the # prefix for channel-like fragments
			fragment = "#" + fragment
		}
		channel := m.normalizeChannelName(fragment)
		m.channels = append(m.channels, channel)
	}

	// Parse query parameters
	query := serviceURL.Query()

	if token := query.Get("token"); token != "" {
		m.token = token
		// If we had username but no password, and now have token, keep the username
	} else if m.username != "" && m.password == "" && m.token == "" {
		// Username was provided but no password and no token in query
		// Treat username as token
		m.token = m.username
		m.username = ""
	}

	if botName := query.Get("bot"); botName != "" {
		m.botName = botName
	}

	if iconURL := query.Get("icon_url"); iconURL != "" {
		m.iconURL = iconURL
	}

	if iconEmoji := query.Get("icon_emoji"); iconEmoji != "" {
		m.iconEmoji = iconEmoji
	}

	// Validate we have authentication
	if m.token == "" && (m.username == "" || m.password == "") {
		return fmt.Errorf("mattermost authentication required: either token or username/password")
	}

	// Validate we have at least one channel
	if len(m.channels) == 0 {
		return fmt.Errorf("at least one Mattermost channel is required")
	}

	return nil
}

// normalizeChannelName normalizes channel names
func (m *MattermostService) normalizeChannelName(channel string) string {
	// Remove @ prefix if present
	if strings.HasPrefix(channel, "@") {
		return strings.TrimPrefix(channel, "@")
	}

	// Remove # prefix if present
	if strings.HasPrefix(channel, "#") {
		return strings.TrimPrefix(channel, "#")
	}

	return channel
}

// MattermostAuthResponse represents authentication response
type MattermostAuthResponse struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

// MattermostChannel represents a channel response
type MattermostChannel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// MattermostPost represents a post payload
type MattermostPost struct {
	ChannelID string                 `json:"channel_id"`
	Message   string                 `json:"message"`
	Props     map[string]interface{} `json:"props,omitempty"`
}

// MattermostPostResponse represents post response
type MattermostPostResponse struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

// Send sends a notification to Mattermost
func (m *MattermostService) Send(ctx context.Context, req NotificationRequest) error {
	// Authenticate if using username/password
	if m.token == "" {
		if err := m.authenticate(ctx); err != nil {
			return fmt.Errorf("failed to authenticate with Mattermost: %w", err)
		}
	}

	// Send to each channel
	var lastError error
	successCount := 0

	for _, channel := range m.channels {
		// Get channel ID
		channelID, err := m.getChannelID(ctx, channel)
		if err != nil {
			lastError = err
			continue
		}

		// Send message
		if err := m.sendToChannel(ctx, channelID, req); err != nil {
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

// authenticate logs in to Mattermost and gets an access token
func (m *MattermostService) authenticate(ctx context.Context) error {
	if m.username == "" || m.password == "" {
		return fmt.Errorf("username and password required for authentication")
	}

	loginURL := m.serverURL + "/api/v4/users/login"

	loginData := map[string]string{
		"login_id": m.username,
		"password": m.password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mattermost login failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Extract token from response headers
	token := resp.Header.Get("Token")
	if token == "" {
		return fmt.Errorf("no token received from Mattermost login")
	}

	m.token = token
	return nil
}

// getChannelID resolves a channel name to its ID
func (m *MattermostService) getChannelID(ctx context.Context, channel string) (string, error) {
	// Try by name first
	channelURL := fmt.Sprintf("%s/api/v4/channels/name/%s", m.serverURL, url.PathEscape(channel))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", channelURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create channel request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+m.token)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to get channel info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read channel response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("mattermost channel lookup failed (status %d): %s", resp.StatusCode, string(body))
	}

	var channelInfo MattermostChannel
	if err := json.Unmarshal(body, &channelInfo); err != nil {
		return "", fmt.Errorf("failed to parse channel response: %w", err)
	}

	return channelInfo.ID, nil
}

// sendToChannel sends a message to a specific Mattermost channel
func (m *MattermostService) sendToChannel(ctx context.Context, channelID string, req NotificationRequest) error {
	postURL := m.serverURL + "/api/v4/posts"

	// Format message
	message := m.formatMessage(req.Title, req.Body, req.NotifyType)

	post := MattermostPost{
		ChannelID: channelID,
		Message:   message,
		Props: map[string]interface{}{
			"from_webhook": "true",
		},
	}

	// Add bot customization if specified
	if m.botName != "" {
		post.Props["override_username"] = m.botName
	}
	if m.iconURL != "" {
		post.Props["override_icon_url"] = m.iconURL
	}
	if m.iconEmoji != "" {
		post.Props["override_icon_emoji"] = m.iconEmoji
	}

	jsonData, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("failed to marshal Mattermost post: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create post request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.token)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Mattermost message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mattermost API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// formatMessage formats the title and body into a Mattermost message
func (m *MattermostService) formatMessage(title, body string, notifyType NotifyType) string {
	emoji := m.getEmojiForNotifyType(notifyType)

	var message strings.Builder

	if title != "" {
		message.WriteString(fmt.Sprintf("%s **%s**", emoji, title))
		if body != "" {
			message.WriteString("\n\n")
		}
	}

	if body != "" {
		message.WriteString(body)
	}

	// If neither title nor body, provide a default message
	if title == "" && body == "" {
		message.WriteString(fmt.Sprintf("%s Notification", emoji))
	}

	return message.String()
}

// getEmojiForNotifyType returns an emoji for the notification type
func (m *MattermostService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return ":white_check_mark:"
	case NotifyTypeWarning:
		return ":warning:"
	case NotifyTypeError:
		return ":x:"
	default:
		return ":information_source:"
	}
}

// TestURL validates a Mattermost service URL
func (m *MattermostService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return m.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Mattermost supports file uploads
func (m *MattermostService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Mattermost's message length limit
func (m *MattermostService) GetMaxBodyLength() int {
	return 4000 // Mattermost message limit is 4000 characters
}

// Example usage and URL formats:
// mattermost://username:password@mattermost.example.com/general
// mmosts://token@mattermost.example.com:443/general/alerts
// mattermost://user:pass@mm.company.com/town-square?bot=AlertBot&icon_emoji=:warning:
// mmosts://token@mattermost.example.com/general?icon_url=https://example.com/icon.png
