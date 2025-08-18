package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// RedditService implements Reddit notifications via API
type RedditService struct {
	clientID     string
	clientSecret string
	username     string
	password     string
	subreddit    string
	recipient    string // For direct messages
	userAgent    string
	client       *http.Client
	accessToken  string
}

// RedditAuthResponse represents Reddit OAuth response
type RedditAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// RedditSubmitRequest represents a Reddit submission request
type RedditSubmitRequest struct {
	APIType    string `json:"api_type"`
	Kind       string `json:"kind"`
	Subreddit  string `json:"sr"`
	Title      string `json:"title"`
	Text       string `json:"text"`
	SendReplies bool  `json:"sendreplies"`
}

// RedditMessageRequest represents a Reddit direct message request
type RedditMessageRequest struct {
	APIType string `json:"api_type"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

// NewRedditService creates a new Reddit service instance
func NewRedditService() Service {
	return &RedditService{
		client:    &http.Client{},
		userAgent: "Apprise-Go/1.0",
	}
}

// GetServiceID returns the service identifier
func (s *RedditService) GetServiceID() string {
	return "reddit"
}

// GetDefaultPort returns the default port
func (s *RedditService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *RedditService) ParseURL(serviceURL *url.URL) error {
	// URL format: reddit://client_id:client_secret@host/subreddit_or_user?username=user&password=pass&mode=post|message
	
	if serviceURL.User == nil {
		return fmt.Errorf("Reddit URL must include client credentials")
	}
	
	// Parse client credentials from user info
	s.clientID = serviceURL.User.Username()
	clientSecret, hasSecret := serviceURL.User.Password()
	if !hasSecret {
		return fmt.Errorf("Reddit URL must include client secret")
	}
	s.clientSecret = clientSecret
	
	// Parse query parameters for user credentials
	query := serviceURL.Query()
	s.username = query.Get("username")
	s.password = query.Get("password")
	
	if s.username == "" || s.password == "" {
		return fmt.Errorf("Reddit URL must include username and password query parameters")
	}
	
	// Extract target from path
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		target := strings.Trim(serviceURL.Path, "/")
		
		// Check mode to determine if it's a subreddit post or direct message
		query := serviceURL.Query()
		mode := query.Get("mode")
		
		if mode == "message" || mode == "dm" {
			s.recipient = target
		} else {
			s.subreddit = target
		}
	}
	
	if s.subreddit == "" && s.recipient == "" {
		return fmt.Errorf("Reddit URL must specify either subreddit for posts or recipient for messages")
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *RedditService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *RedditService) Send(ctx context.Context, req NotificationRequest) error {
	// Authenticate first
	if err := s.authenticate(ctx); err != nil {
		return fmt.Errorf("Reddit authentication failed: %w", err)
	}
	
	// Send notification based on configuration
	if s.subreddit != "" {
		return s.postToSubreddit(ctx, req)
	} else if s.recipient != "" {
		return s.sendDirectMessage(ctx, req)
	}
	
	return fmt.Errorf("no valid Reddit target configured")
}

// authenticate obtains an access token from Reddit
func (s *RedditService) authenticate(ctx context.Context) error {
	authURL := "https://www.reddit.com/api/v1/access_token"
	
	// Prepare form data
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("username", s.username)
	formData.Set("password", s.password)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", s.userAgent)
	req.SetBasicAuth(s.clientID, s.clientSecret)
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("Reddit auth returned status %d", resp.StatusCode)
	}
	
	// Parse response
	var authResp RedditAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to parse auth response: %w", err)
	}
	
	s.accessToken = authResp.AccessToken
	return nil
}

// postToSubreddit posts a text submission to a subreddit
func (s *RedditService) postToSubreddit(ctx context.Context, req NotificationRequest) error {
	submitURL := "https://oauth.reddit.com/api/submit"
	
	// Prepare submission data
	title := req.Title
	if title == "" {
		title = "Notification"
	}
	
	formData := url.Values{}
	formData.Set("api_type", "json")
	formData.Set("kind", "self")
	formData.Set("sr", s.subreddit)
	formData.Set("title", title)
	formData.Set("text", req.Body)
	formData.Set("sendreplies", "false")
	
	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", submitURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create submit request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", s.userAgent)
	httpReq.Header.Set("Authorization", "Bearer "+s.accessToken)
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Reddit submit request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Reddit submit returned status %d", resp.StatusCode)
	}
	
	return nil
}

// sendDirectMessage sends a direct message to a Reddit user
func (s *RedditService) sendDirectMessage(ctx context.Context, req NotificationRequest) error {
	messageURL := "https://oauth.reddit.com/api/compose"
	
	// Prepare message data
	subject := req.Title
	if subject == "" {
		subject = "Notification"
	}
	
	formData := url.Values{}
	formData.Set("api_type", "json")
	formData.Set("to", s.recipient)
	formData.Set("subject", subject)
	formData.Set("text", req.Body)
	
	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", messageURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create message request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", s.userAgent)
	httpReq.Header.Set("Authorization", "Bearer "+s.accessToken)
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Reddit message request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Reddit message returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *RedditService) SupportsAttachments() bool {
	return false // Reddit API doesn't support direct file uploads in text posts/messages
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *RedditService) GetMaxBodyLength() int {
	return 40000 // Reddit has a practical limit of ~40,000 characters
}