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

// RocketChatService implements Rocket.Chat notifications
type RocketChatService struct {
	server      string            // Rocket.Chat server URL
	userID      string            // User ID for authentication
	authToken   string            // Authentication token
	username    string            // Username for login authentication
	password    string            // Password for login authentication
	channel     string            // Channel or user to send to (#channel, @user, or room ID)
	botName     string            // Override bot name
	botAvatar   string            // Override bot avatar URL
	webhookURL  string            // Webhook URL (alternative to REST API)
	client      *http.Client
}

// RocketChatLoginRequest represents login request payload
type RocketChatLoginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// RocketChatLoginResponse represents login response
type RocketChatLoginResponse struct {
	Status string                     `json:"status"`
	Data   RocketChatLoginData        `json:"data"`
}

// RocketChatLoginData represents login data
type RocketChatLoginData struct {
	UserID    string `json:"userId"`
	AuthToken string `json:"authToken"`
}

// RocketChatMessage represents a Rocket.Chat message
type RocketChatMessage struct {
	Channel     string                    `json:"channel,omitempty"`
	Text        string                    `json:"text,omitempty"`
	Username    string                    `json:"username,omitempty"`
	IconURL     string                    `json:"icon_url,omitempty"`
	IconEmoji   string                    `json:"icon_emoji,omitempty"`
	Attachments []RocketChatAttachment    `json:"attachments,omitempty"`
}

// RocketChatAttachment represents a rich message attachment
type RocketChatAttachment struct {
	Title      string                    `json:"title,omitempty"`
	TitleLink  string                    `json:"title_link,omitempty"`
	Text       string                    `json:"text,omitempty"`
	Color      string                    `json:"color,omitempty"`
	ImageURL   string                    `json:"image_url,omitempty"`
	ThumbURL   string                    `json:"thumb_url,omitempty"`
	AuthorName string                    `json:"author_name,omitempty"`
	AuthorIcon string                    `json:"author_icon,omitempty"`
	AuthorLink string                    `json:"author_link,omitempty"`
	Fields     []RocketChatAttachmentField `json:"fields,omitempty"`
	Timestamp  string                    `json:"ts,omitempty"`
}

// RocketChatAttachmentField represents an attachment field
type RocketChatAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// RocketChatPostMessageRequest represents REST API message request
type RocketChatPostMessageRequest struct {
	Channel string                    `json:"channel"`
	Text    string                    `json:"text,omitempty"`
	Alias   string                    `json:"alias,omitempty"`
	Avatar  string                    `json:"avatar,omitempty"`
	Emoji   string                    `json:"emoji,omitempty"`
	Attachments []RocketChatAttachment `json:"attachments,omitempty"`
}

// RocketChatResponse represents a generic Rocket.Chat API response
type RocketChatResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewRocketChatService creates a new Rocket.Chat service instance
func NewRocketChatService() Service {
	return &RocketChatService{
		client:    &http.Client{Timeout: 30 * time.Second},
		botName:   "Apprise",
		botAvatar: "",
	}
}

// GetServiceID returns the service identifier
func (r *RocketChatService) GetServiceID() string {
	return "rocketchat"
}

// GetDefaultPort returns the default port (443 for HTTPS, 3000 for HTTP)
func (r *RocketChatService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Rocket.Chat service URL
// Format: rocket://username:password@server.com/channel
// Format: rocket://userid:token@server.com/#channel
// Format: rocket://webhook@server.com/hooks/webhook_id/webhook_token
// Format: rocket://server.com/channel?username=user&password=pass&bot_name=MyBot
func (r *RocketChatService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "rocket" && serviceURL.Scheme != "rockets" {
		return fmt.Errorf("invalid scheme: expected 'rocket' or 'rockets', got '%s'", serviceURL.Scheme)
	}

	// Determine server URL
	scheme := "https"
	if serviceURL.Scheme == "rocket" {
		scheme = "http"
	}
	
	port := serviceURL.Port()
	if port == "" {
		if serviceURL.Scheme == "rocket" {
			port = "3000" // Default Rocket.Chat HTTP port
		}
	}

	if port != "" && port != "80" && port != "443" {
		r.server = fmt.Sprintf("%s://%s:%s", scheme, serviceURL.Hostname(), port)
	} else {
		r.server = fmt.Sprintf("%s://%s", scheme, serviceURL.Hostname())
	}

	// Check if this is a webhook URL
	if strings.Contains(serviceURL.Path, "/hooks/") {
		// Webhook mode
		r.webhookURL = r.server + serviceURL.Path
		
		// Extract channel from query or user info
		query := serviceURL.Query()
		if channel := query.Get("channel"); channel != "" {
			r.channel = channel
		} else if serviceURL.User != nil {
			r.channel = serviceURL.User.Username()
		}
		
		if r.channel == "" {
			r.channel = "#general" // Default channel
		}
	} else {
		// REST API mode
		// Parse authentication
		if serviceURL.User != nil {
			username := serviceURL.User.Username()
			password, hasPassword := serviceURL.User.Password()

			if hasPassword {
				// Username:password or userId:authToken
				// If username looks like a user ID (contains numbers/letters but no spaces) and is longer, treat as token auth
				if len(username) > 8 && strings.ContainsAny(username, "0123456789") && !strings.Contains(username, " ") && !strings.HasPrefix(username, "@") {
					// Looks like userId:authToken
					r.userID = username
					r.authToken = password
				} else {
					// Looks like username:password
					r.username = strings.TrimPrefix(username, "@")
					r.password = password
				}
			}
		}

		// Parse channel from path
		if serviceURL.Path != "" {
			pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				r.channel = pathParts[0]
			}
		}

		// Handle fragment as channel (URLs like rocket://server/#channel)
		if serviceURL.Fragment != "" {
			r.channel = "#" + serviceURL.Fragment
		}

		// Parse query parameters
		query := serviceURL.Query()
		
		if username := query.Get("username"); username != "" {
			r.username = username
		}
		
		if password := query.Get("password"); password != "" {
			r.password = password
		}
		
		if userID := query.Get("user_id"); userID != "" {
			r.userID = userID
		}
		
		if authToken := query.Get("auth_token"); authToken != "" {
			r.authToken = authToken
		}
		
		if channel := query.Get("channel"); channel != "" {
			r.channel = channel
		}
		
		if botName := query.Get("bot_name"); botName != "" {
			r.botName = botName
		}
		
		if botAvatar := query.Get("bot_avatar"); botAvatar != "" {
			r.botAvatar = botAvatar
		}

		// Validate authentication for REST API mode
		hasTokenAuth := r.userID != "" && r.authToken != ""
		hasPasswordAuth := r.username != "" && r.password != ""
		
		if !hasTokenAuth && !hasPasswordAuth {
			return fmt.Errorf("authentication required: either user_id+auth_token or username+password")
		}
	}

	// Validate channel
	if r.channel == "" {
		return fmt.Errorf("channel is required")
	}

	// Normalize channel format (but not for webhook mode if it was explicitly set in query)
	if r.webhookURL == "" || serviceURL.Query().Get("channel") == "" {
		r.channel = r.normalizeChannel(r.channel)
	}

	return nil
}

// normalizeChannel normalizes channel identifiers
func (r *RocketChatService) normalizeChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	
	// If it starts with @ or # or is a room ID, keep as is
	if strings.HasPrefix(channel, "@") || strings.HasPrefix(channel, "#") {
		return channel
	}
	
	// If it looks like a room ID (contains random characters), keep as is
	if len(channel) > 10 && strings.ContainsAny(channel, "0123456789abcdef") {
		return channel
	}
	
	// Otherwise, assume it's a channel name and add #
	return "#" + channel
}

// Send sends a notification to Rocket.Chat
func (r *RocketChatService) Send(ctx context.Context, req NotificationRequest) error {
	if r.webhookURL != "" {
		// Send via webhook
		return r.sendViaWebhook(ctx, req)
	} else {
		// Send via REST API
		return r.sendViaAPI(ctx, req)
	}
}

// sendViaWebhook sends message via Rocket.Chat webhook
func (r *RocketChatService) sendViaWebhook(ctx context.Context, req NotificationRequest) error {
	message := r.createWebhookMessage(req)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Rocket.Chat webhook message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Rocket.Chat webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Rocket.Chat webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendViaAPI sends message via Rocket.Chat REST API
func (r *RocketChatService) sendViaAPI(ctx context.Context, req NotificationRequest) error {
	// Ensure we have authentication
	if r.authToken == "" {
		if err := r.login(ctx); err != nil {
			return fmt.Errorf("failed to login to Rocket.Chat: %w", err)
		}
	}

	// Create message
	apiMessage := r.createAPIMessage(req)

	jsonData, err := json.Marshal(apiMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal Rocket.Chat API message: %w", err)
	}

	// Send message via REST API
	apiURL := r.server + "/api/v1/chat.postMessage"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create API request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-User-Id", r.userID)
	httpReq.Header.Set("X-Auth-Token", r.authToken)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Rocket.Chat API request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Rocket.Chat API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp RocketChatResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("Rocket.Chat API error: %s", apiResp.Error)
	}

	return nil
}

// login authenticates with Rocket.Chat server using username/password
func (r *RocketChatService) login(ctx context.Context) error {
	if r.username == "" || r.password == "" {
		return fmt.Errorf("username and password required for login")
	}

	loginURL := r.server + "/api/v1/login"
	loginReq := RocketChatLoginRequest{
		User:     r.username,
		Password: r.password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Rocket.Chat login failed (status %d): %s", resp.StatusCode, string(body))
	}

	var loginResp RocketChatLoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResp.Status != "success" {
		return fmt.Errorf("Rocket.Chat login failed: %s", loginResp.Status)
	}

	r.userID = loginResp.Data.UserID
	r.authToken = loginResp.Data.AuthToken

	return nil
}

// createWebhookMessage creates a webhook message payload
func (r *RocketChatService) createWebhookMessage(req NotificationRequest) RocketChatMessage {
	message := RocketChatMessage{
		Channel:  r.channel,
		Username: r.botName,
	}

	if r.botAvatar != "" {
		message.IconURL = r.botAvatar
	} else {
		// Set emoji based on notification type
		message.IconEmoji = r.getEmojiForNotifyType(req.NotifyType)
	}

	// Create rich attachment
	attachment := r.createAttachment(req)
	message.Attachments = []RocketChatAttachment{attachment}

	return message
}

// createAPIMessage creates an API message payload
func (r *RocketChatService) createAPIMessage(req NotificationRequest) RocketChatPostMessageRequest {
	message := RocketChatPostMessageRequest{
		Channel: r.channel,
		Alias:   r.botName,
	}

	if r.botAvatar != "" {
		message.Avatar = r.botAvatar
	} else {
		message.Emoji = r.getEmojiForNotifyType(req.NotifyType)
	}

	// Create rich attachment
	attachment := r.createAttachment(req)
	message.Attachments = []RocketChatAttachment{attachment}

	return message
}

// createAttachment creates a rich message attachment
func (r *RocketChatService) createAttachment(req NotificationRequest) RocketChatAttachment {
	attachment := RocketChatAttachment{
		Title:     req.Title,
		Text:      req.Body,
		Color:     r.getColorForNotifyType(req.NotifyType),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add author info
	attachment.AuthorName = "Apprise-Go"
	attachment.AuthorIcon = r.getIconForNotifyType(req.NotifyType)

	// Add fields for metadata
	var fields []RocketChatAttachmentField

	// Add notification type field
	fields = append(fields, RocketChatAttachmentField{
		Title: "Type",
		Value: strings.ToTitle(req.NotifyType.String()),
		Short: true,
	})

	// Add timestamp field
	fields = append(fields, RocketChatAttachmentField{
		Title: "Time",
		Value: time.Now().Format("15:04:05 MST"),
		Short: true,
	})

	// Add tags if present
	if len(req.Tags) > 0 {
		fields = append(fields, RocketChatAttachmentField{
			Title: "Tags",
			Value: strings.Join(req.Tags, ", "),
			Short: false,
		})
	}

	attachment.Fields = fields

	// Handle attachments (images)
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		for _, att := range attachments {
			mimeType := att.GetMimeType()
			if strings.HasPrefix(mimeType, "image/") {
				// For webhook mode, we'd need to upload the image first
				// For now, just indicate there's an image attachment
				attachment.Fields = append(attachment.Fields, RocketChatAttachmentField{
					Title: "Attachment",
					Value: fmt.Sprintf("Image: %s (%s)", att.GetName(), mimeType),
					Short: false,
				})
			}
		}
	}

	return attachment
}

// Helper methods for styling

func (r *RocketChatService) getColorForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "good"      // Green
	case NotifyTypeWarning:
		return "warning"   // Yellow
	case NotifyTypeError:
		return "danger"    // Red
	case NotifyTypeInfo:
		fallthrough
	default:
		return "#439FE0"   // Blue
	}
}

func (r *RocketChatService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return ":white_check_mark:"
	case NotifyTypeWarning:
		return ":warning:"
	case NotifyTypeError:
		return ":exclamation:"
	case NotifyTypeInfo:
		fallthrough
	default:
		return ":information_source:"
	}
}

func (r *RocketChatService) getIconForNotifyType(notifyType NotifyType) string {
	// URL to notification type icons (these would need to be hosted somewhere)
	baseURL := "https://raw.githubusercontent.com/simple-icons/simple-icons/develop/icons/"
	
	switch notifyType {
	case NotifyTypeSuccess:
		return baseURL + "checkmarx.svg"
	case NotifyTypeWarning:
		return baseURL + "alertmanager.svg"
	case NotifyTypeError:
		return baseURL + "bugsnag.svg"
	case NotifyTypeInfo:
		fallthrough
	default:
		return baseURL + "information.svg"
	}
}

// TestURL validates a Rocket.Chat service URL
func (r *RocketChatService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return r.ParseURL(parsedURL)
}

// SupportsAttachments returns true for Rocket.Chat (supports file uploads and rich content)
func (r *RocketChatService) SupportsAttachments() bool {
	return true // Rocket.Chat supports file uploads and rich attachments
}

// GetMaxBodyLength returns Rocket.Chat message length limit
func (r *RocketChatService) GetMaxBodyLength() int {
	return 1000 // Reasonable limit for rich attachments
}

// Example usage and URL formats:
// rocket://username:password@rocketchat.company.com/general
// rockets://userid:token@rocketchat.company.com/#alerts  
// rocket://webhook@rocketchat.company.com/hooks/webhook_id/webhook_token
// rocket://rocketchat.company.com/support?username=bot&password=secret&bot_name=AlertBot
// rockets://rocketchat.company.com:443/team-dev?user_id=abc123&auth_token=xyz789