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

// MatrixService implements Matrix messaging notifications
type MatrixService struct {
	homeserver  string
	accessToken string
	username    string
	password    string
	rooms       []string
	msgType     string // "m.text" or "m.notice"
	htmlFormat  bool
	client      *http.Client
}

// NewMatrixService creates a new Matrix service instance
func NewMatrixService() Service {
	return &MatrixService{
		client:     &http.Client{},
		msgType:    "m.text", // Default to text message
		htmlFormat: false,
	}
}

// GetServiceID returns the service identifier
func (m *MatrixService) GetServiceID() string {
	return "matrix"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (m *MatrixService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Matrix service URL
// Format: matrix://token@homeserver/room1/room2
// Format: matrix://user:password@homeserver/room1/room2
// Format: matrix://user@homeserver/room1?token=access_token
func (m *MatrixService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "matrix" {
		return fmt.Errorf("invalid scheme: expected 'matrix', got '%s'", serviceURL.Scheme)
	}

	// Extract homeserver
	m.homeserver = serviceURL.Host
	if m.homeserver == "" {
		return fmt.Errorf("matrix homeserver is required")
	}

	// Ensure homeserver has protocol
	if !strings.HasPrefix(m.homeserver, "http://") && !strings.HasPrefix(m.homeserver, "https://") {
		m.homeserver = "https://" + m.homeserver
	}

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

	// Extract rooms from path and fragment
	if serviceURL.Path != "" {
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, part := range pathParts {
			if part != "" {
				// Normalize room format
				room := m.normalizeRoomID(part)
				m.rooms = append(m.rooms, room)
			}
		}
	}

	// Handle room in fragment (URLs like matrix://token@server/#room)
	if serviceURL.Fragment != "" {
		// Fragment was likely a room alias that started with # but URL parsing removed it
		fragment := serviceURL.Fragment
		if !strings.HasPrefix(fragment, "#") && !strings.HasPrefix(fragment, "!") && strings.Contains(fragment, ":") {
			// This looks like a room alias that lost its # prefix during URL parsing
			fragment = "#" + fragment
		}
		room := m.normalizeRoomID(fragment)
		m.rooms = append(m.rooms, room)
	}

	// Parse query parameters
	query := serviceURL.Query()

	if token := query.Get("token"); token != "" {
		m.accessToken = token
		// If we had username but no password, and now have token, keep the username
	} else if m.username != "" && m.password == "" && m.accessToken == "" {
		// Username was provided but no password and no token in query
		// Treat username as access token
		m.accessToken = m.username
		m.username = ""
	}

	if msgType := query.Get("msgtype"); msgType != "" {
		switch strings.ToLower(msgType) {
		case "text", "m.text":
			m.msgType = "m.text"
		case "notice", "m.notice":
			m.msgType = "m.notice"
		default:
			m.msgType = "m.text"
		}
	}

	if format := query.Get("format"); format != "" {
		m.htmlFormat = strings.ToLower(format) == "html"
	}

	// Validate we have authentication
	if m.accessToken == "" && (m.username == "" || m.password == "") {
		return fmt.Errorf("matrix authentication required: either access token or username/password")
	}

	// Validate we have at least one room
	if len(m.rooms) == 0 {
		return fmt.Errorf("at least one Matrix room is required")
	}

	return nil
}

// normalizeRoomID normalizes room identifiers
func (m *MatrixService) normalizeRoomID(room string) string {
	// If it's already a room ID (!room:server) or alias (#room:server), keep as is
	if strings.HasPrefix(room, "!") || strings.HasPrefix(room, "#") {
		return room
	}

	// If it's just a room name, convert to alias format
	if !strings.Contains(room, ":") {
		// Extract domain from homeserver for room alias
		homeserverURL, err := url.Parse(m.homeserver)
		if err == nil && homeserverURL.Host != "" {
			return fmt.Sprintf("#%s:%s", room, homeserverURL.Host)
		}
	}

	return room
}

// MatrixLoginRequest represents login request payload
type MatrixLoginRequest struct {
	Type                     string                `json:"type"`
	User                     string                `json:"user,omitempty"`
	Password                 string                `json:"password,omitempty"`
	Identifier               *MatrixUserIdentifier `json:"identifier,omitempty"`
	InitialDeviceDisplayName string                `json:"initial_device_display_name,omitempty"`
}

// MatrixUserIdentifier represents user identifier in login
type MatrixUserIdentifier struct {
	Type string `json:"type"`
	User string `json:"user"`
}

// MatrixLoginResponse represents login response
type MatrixLoginResponse struct {
	AccessToken string `json:"access_token"`
	DeviceID    string `json:"device_id"`
	HomeServer  string `json:"home_server"`
	UserID      string `json:"user_id"`
}

// MatrixMessage represents a Matrix message payload
type MatrixMessage struct {
	MsgType       string `json:"msgtype"`
	Body          string `json:"body"`
	Format        string `json:"format,omitempty"`
	FormattedBody string `json:"formatted_body,omitempty"`
}

// MatrixSendResponse represents message send response
type MatrixSendResponse struct {
	EventID string `json:"event_id"`
}

// Send sends a notification to Matrix
func (m *MatrixService) Send(ctx context.Context, req NotificationRequest) error {
	// Ensure we have an access token
	if m.accessToken == "" {
		if err := m.login(ctx); err != nil {
			return fmt.Errorf("failed to login to Matrix: %w", err)
		}
	}

	// Send to each room
	var lastError error
	successCount := 0

	for _, room := range m.rooms {
		if err := m.sendToRoom(ctx, room, req); err != nil {
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

// login authenticates with Matrix server
func (m *MatrixService) login(ctx context.Context) error {
	if m.username == "" || m.password == "" {
		return fmt.Errorf("username and password required for login")
	}

	loginURL := m.homeserver + "/_matrix/client/v3/login"

	loginReq := MatrixLoginRequest{
		Type: "m.login.password",
		Identifier: &MatrixUserIdentifier{
			Type: "m.id.user",
			User: m.username,
		},
		Password:                 m.password,
		InitialDeviceDisplayName: "Apprise-Go",
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

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("matrix login failed (status %d): %s", resp.StatusCode, string(body))
	}

	var loginResp MatrixLoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	m.accessToken = loginResp.AccessToken
	return nil
}

// sendToRoom sends a message to a specific Matrix room
func (m *MatrixService) sendToRoom(ctx context.Context, room string, req NotificationRequest) error {
	// Generate transaction ID (simple timestamp-based)
	txnID := fmt.Sprintf("apprise_%d", time.Now().UnixNano())

	sendURL := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		m.homeserver, url.PathEscape(room), txnID)

	message := m.formatMessage(req.Title, req.Body)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Matrix message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", sendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.accessToken)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Matrix message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("matrix API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result MatrixSendResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Matrix response: %w", err)
	}

	return nil
}

// formatMessage formats the title and body into a Matrix message
func (m *MatrixService) formatMessage(title, body string) MatrixMessage {
	message := MatrixMessage{
		MsgType: m.msgType,
	}

	// Format text content
	if title != "" && body != "" {
		message.Body = fmt.Sprintf("%s\n%s", title, body)
	} else if title != "" {
		message.Body = title
	} else {
		message.Body = body
	}

	// Add HTML formatting if enabled
	if m.htmlFormat && title != "" {
		if body != "" {
			message.Format = "org.matrix.custom.html"
			message.FormattedBody = fmt.Sprintf("<h4>%s</h4><p>%s</p>",
				m.escapeHTML(title), m.escapeHTML(body))
		} else {
			message.Format = "org.matrix.custom.html"
			message.FormattedBody = fmt.Sprintf("<h4>%s</h4>", m.escapeHTML(title))
		}
	}

	return message
}

// escapeHTML escapes HTML characters
func (m *MatrixService) escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	return text
}

// TestURL validates a Matrix service URL
func (m *MatrixService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return m.ParseURL(parsedURL)
}

// SupportsAttachments returns true since Matrix supports file attachments
func (m *MatrixService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns Matrix's message length limit
func (m *MatrixService) GetMaxBodyLength() int {
	return 32768 // Matrix supports large messages (~32KB)
}

// Example usage and URL formats:
// matrix://access_token@matrix.org/!room_id:matrix.org
// matrix://access_token@matrix.org/#room_alias:matrix.org
// matrix://username:password@matrix.example.com/general
// matrix://access_token@matrix.org/room1/room2?msgtype=notice&format=html
