package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// FacebookService implements Facebook notifications via Graph API
type FacebookService struct {
	accessToken string
	pageID      string
	client      *http.Client
}

// FacebookPostRequest represents a Facebook post request
type FacebookPostRequest struct {
	Message     string `json:"message"`
	Link        string `json:"link,omitempty"`
	Published   bool   `json:"published"`
}

// FacebookResponse represents a Facebook API response
type FacebookResponse struct {
	ID    string `json:"id"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

// NewFacebookService creates a new Facebook service instance
func NewFacebookService() Service {
	return &FacebookService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *FacebookService) GetServiceID() string {
	return "facebook"
}

// GetDefaultPort returns the default port
func (s *FacebookService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *FacebookService) ParseURL(serviceURL *url.URL) error {
	// URL format: facebook://access_token@page_id
	
	if serviceURL.User == nil {
		return fmt.Errorf("Facebook URL must include access token")
	}
	
	s.accessToken = serviceURL.User.Username()
	if s.accessToken == "" {
		return fmt.Errorf("Facebook access token cannot be empty")
	}
	
	// Extract page ID from host
	if serviceURL.Host == "" {
		return fmt.Errorf("Facebook URL must specify page ID")
	}
	s.pageID = serviceURL.Host
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *FacebookService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *FacebookService) Send(ctx context.Context, req NotificationRequest) error {
	// Create a Facebook page post
	return s.createPost(ctx, req)
}

// createPost creates a Facebook page post
func (s *FacebookService) createPost(ctx context.Context, req NotificationRequest) error {
	// Facebook Graph API endpoint for page posts
	apiURL := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/feed", s.pageID)
	
	// Build message content
	message := req.Body
	if req.Title != "" {
		message = req.Title + "\n\n" + message
	}
	
	// Prepare post data
	postData := FacebookPostRequest{
		Message:   message,
		Published: true,
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return fmt.Errorf("failed to marshal Facebook post: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Facebook request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.accessToken)
	
	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Facebook API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var fbResp FacebookResponse
		if json.NewDecoder(resp.Body).Decode(&fbResp) == nil && fbResp.Error.Message != "" {
			return fmt.Errorf("Facebook API error: %s (code %d)", fbResp.Error.Message, fbResp.Error.Code)
		}
		return fmt.Errorf("Facebook API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *FacebookService) SupportsAttachments() bool {
	return true // Facebook supports media attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *FacebookService) GetMaxBodyLength() int {
	return 63206 // Facebook post character limit
}