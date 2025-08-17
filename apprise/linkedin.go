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

// LinkedInService implements LinkedIn API v2 notifications
type LinkedInService struct {
	accessToken  string // OAuth 2.0 access token
	clientID     string // LinkedIn Client ID
	clientSecret string // LinkedIn Client Secret
	userID       string // LinkedIn user ID or 'me'
	pageID       string // LinkedIn organization/company page ID (optional)
	webhookURL   string // Webhook proxy URL for secure credential management
	proxyAPIKey  string // API key for webhook authentication
	client       *http.Client
}

// LinkedInMessage represents a LinkedIn post/share
type LinkedInMessage struct {
	Author      string                 `json:"author"`
	Text        LinkedInTextContent    `json:"text"`
	Content     *LinkedInContent       `json:"content,omitempty"`
	Distribution LinkedInDistribution   `json:"distribution"`
	LifecycleState string              `json:"lifecycleState"`
	Visibility   LinkedInVisibility    `json:"visibility"`
}

// LinkedInTextContent represents the text content of a post
type LinkedInTextContent struct {
	Text string `json:"text"`
}

// LinkedInContent represents rich content (links, images, etc.)
type LinkedInContent struct {
	ContentEntities []LinkedInContentEntity `json:"contentEntities,omitempty"`
	Title           string                  `json:"title,omitempty"`
}

// LinkedInContentEntity represents a content entity (link, image)
type LinkedInContentEntity struct {
	Entity      string                     `json:"entity,omitempty"`
	EntityLocation string                  `json:"entityLocation,omitempty"`
	Thumbnails  []LinkedInThumbnail        `json:"thumbnails,omitempty"`
}

// LinkedInThumbnail represents a thumbnail image
type LinkedInThumbnail struct {
	ResolvedURL string `json:"resolvedUrl"`
}

// LinkedInDistribution represents distribution settings
type LinkedInDistribution struct {
	FeedDistribution string `json:"feedDistribution"`
}

// LinkedInVisibility represents post visibility settings
type LinkedInVisibility struct {
	Code string `json:"com.linkedin.ugc.MemberNetworkVisibility"`
}

// LinkedInWebhookPayload represents webhook proxy payload
type LinkedInWebhookPayload struct {
	Service      string          `json:"service"`
	AccessToken  string          `json:"access_token"`
	ClientID     string          `json:"client_id"`
	ClientSecret string          `json:"client_secret"`
	UserID       string          `json:"user_id"`
	PageID       string          `json:"page_id,omitempty"`
	Message      LinkedInMessage `json:"linkedin_message"`
	Timestamp    string          `json:"timestamp"`
	Source       string          `json:"source"`
	Version      string          `json:"version"`
}

// NewLinkedInService creates a new LinkedIn service instance
func NewLinkedInService() Service {
	return &LinkedInService{
		client: GetCloudHTTPClient("linkedin"),
		userID: "me", // Default to current user
	}
}

// GetServiceID returns the service identifier
func (l *LinkedInService) GetServiceID() string {
	return "linkedin"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (l *LinkedInService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a LinkedIn service URL
// Format: linkedin://access_token@api.linkedin.com/v2/shares?user_id=USER_ID&page_id=PAGE_ID
// Format: linkedin://client_id:client_secret:access_token@api.linkedin.com/v2/shares?user_id=USER_ID
// Format: linkedin://proxy-key@webhook.example.com/linkedin?access_token=token&client_id=id&client_secret=secret&user_id=user
func (l *LinkedInService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "linkedin" {
		return fmt.Errorf("invalid scheme: expected 'linkedin', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/linkedin") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		l.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			l.proxyAPIKey = serviceURL.User.Username()
		}

		// Get LinkedIn credentials from query parameters
		l.accessToken = query.Get("access_token")
		if l.accessToken == "" {
			return fmt.Errorf("access_token parameter is required for webhook mode")
		}

		l.clientID = query.Get("client_id")
		l.clientSecret = query.Get("client_secret")

		// User ID for posting context
		if userID := query.Get("user_id"); userID != "" {
			l.userID = userID
		}

		// Optional page ID for organization posts
		l.pageID = query.Get("page_id")
	} else {
		// Direct LinkedIn API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: LinkedIn API credentials must be provided")
		}

		// Parse credentials from user info
		privateKey, hasKey := serviceURL.User.Password()
		
		if hasKey && strings.Contains(privateKey, ":") {
			// Client credentials format (client_id:client_secret:access_token)
			credentials := strings.Split(privateKey, ":")
			if len(credentials) != 2 {
				return fmt.Errorf("LinkedIn private key requires client_secret:access_token format")
			}

			l.clientID = serviceURL.User.Username()
			l.clientSecret = credentials[0]
			l.accessToken = credentials[1]

			if l.clientID == "" || l.clientSecret == "" || l.accessToken == "" {
				return fmt.Errorf("all LinkedIn credentials are required")
			}
		} else if hasKey {
			// Incomplete OAuth credentials (password but not the right format)
			return fmt.Errorf("LinkedIn OAuth requires client_id:client_secret:access_token format")
		} else {
			// Access token only format
			l.accessToken = serviceURL.User.Username()
			if l.accessToken == "" {
				return fmt.Errorf("access token is required")
			}
		}

		// Get user ID from query parameters
		if userID := query.Get("user_id"); userID != "" {
			l.userID = userID
		}

		// Optional page ID for organization posts
		l.pageID = query.Get("page_id")
	}

	return nil
}

// Send sends a LinkedIn notification
func (l *LinkedInService) Send(ctx context.Context, req NotificationRequest) error {
	// Build LinkedIn message
	message := l.buildLinkedInMessage(req)

	if l.webhookURL != "" {
		// Send via webhook proxy
		return l.sendViaWebhook(ctx, message)
	} else {
		// Send directly to LinkedIn API
		return l.sendToLinkedInDirectly(ctx, message)
	}
}

// buildLinkedInMessage creates a LinkedIn message from notification request
func (l *LinkedInService) buildLinkedInMessage(req NotificationRequest) LinkedInMessage {
	// Build post text
	text := l.formatLinkedInText(req)

	// Determine author (user or organization)
	author := fmt.Sprintf("urn:li:person:%s", l.userID)
	if l.pageID != "" {
		author = fmt.Sprintf("urn:li:organization:%s", l.pageID)
	}

	message := LinkedInMessage{
		Author: author,
		Text: LinkedInTextContent{
			Text: text,
		},
		LifecycleState: "PUBLISHED",
		Distribution: LinkedInDistribution{
			FeedDistribution: "MAIN_FEED",
		},
		Visibility: LinkedInVisibility{
			Code: "PUBLIC",
		},
	}

	// Add rich content if URL is provided
	if req.URL != "" {
		message.Content = &LinkedInContent{
			ContentEntities: []LinkedInContentEntity{
				{
					EntityLocation: req.URL,
				},
			},
			Title: req.Title,
		}
	}

	// Add attachment info if present (as content entities)
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		if len(attachments) > 0 {
			if message.Content == nil {
				message.Content = &LinkedInContent{}
			}
			
			// Note: LinkedIn requires pre-uploaded assets for images
			// This would typically involve uploading to LinkedIn's asset API first
			message.Text.Text += fmt.Sprintf("\n\n📎 %d attachment(s) included", len(attachments))
		}
	}

	return message
}

// formatLinkedInText formats notification content for LinkedIn
func (l *LinkedInService) formatLinkedInText(req NotificationRequest) string {
	var text string
	if req.Title != "" && req.Body != "" {
		text = fmt.Sprintf("%s\n\n%s", req.Title, req.Body)
	} else if req.Title != "" {
		text = req.Title
	} else {
		text = req.Body
	}

	// Add professional context based on notification type
	switch req.NotifyType {
	case NotifyTypeError:
		text = "🚨 ALERT: " + text
	case NotifyTypeWarning:
		text = "⚠️ Important: " + text
	case NotifyTypeSuccess:
		text = "✅ Success: " + text
	case NotifyTypeInfo:
		text = "📢 Update: " + text
	}

	// Add hashtags if present (LinkedIn supports hashtags)
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
			text += "\n\n" + strings.Join(hashtags, " ")
		}
	}

	// LinkedIn has a 3000 character limit for posts
	if len(text) > 3000 {
		text = text[:2995] + "..."
	}

	return text
}

// sendViaWebhook sends message via webhook proxy
func (l *LinkedInService) sendViaWebhook(ctx context.Context, message LinkedInMessage) error {
	payload := LinkedInWebhookPayload{
		Service:      "linkedin",
		AccessToken:  l.accessToken,
		ClientID:     l.clientID,
		ClientSecret: l.clientSecret,
		UserID:       l.userID,
		PageID:       l.pageID,
		Message:      message,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Source:       "apprise-go",
		Version:      GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal LinkedIn webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", l.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create LinkedIn webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if l.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", l.proxyAPIKey)
	}

	resp, err := l.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send LinkedIn webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("linkedin webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendToLinkedInDirectly sends message directly to LinkedIn API
func (l *LinkedInService) sendToLinkedInDirectly(ctx context.Context, message LinkedInMessage) error {
	// LinkedIn UGC Posts API endpoint
	apiURL := "https://api.linkedin.com/v2/ugcPosts"

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal LinkedIn message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create LinkedIn API request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.accessToken))
	httpReq.Header.Set("User-Agent", GetUserAgent())
	httpReq.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := l.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send LinkedIn post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("linkedin api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (l *LinkedInService) validateCredentials() error {
	if l.accessToken == "" {
		return fmt.Errorf("access token is required")
	}
	return nil
}

// TestURL validates a LinkedIn service URL
func (l *LinkedInService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return l.ParseURL(parsedURL)
}

// SupportsAttachments returns true (LinkedIn supports rich content)
func (l *LinkedInService) SupportsAttachments() bool {
	return true // LinkedIn supports images and rich content (requires asset upload)
}

// GetMaxBodyLength returns LinkedIn's character limit
func (l *LinkedInService) GetMaxBodyLength() int {
	return 3000 // LinkedIn posts have a 3000 character limit
}

// Example usage and URL formats:
// linkedin://access_token@api.linkedin.com/v2/ugcPosts?user_id=USER_ID
// linkedin://client_id:client_secret:access_token@api.linkedin.com/v2/ugcPosts?user_id=USER_ID&page_id=PAGE_ID
// linkedin://proxy-key@webhook.example.com/linkedin?access_token=token&client_id=id&client_secret=secret&user_id=user&page_id=page