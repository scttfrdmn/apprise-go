package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// MastodonService implements Mastodon notifications via API
type MastodonService struct {
	instanceURL string
	accessToken string
	visibility  string // public, unlisted, private, direct
	client      *http.Client
}

// MastodonStatus represents a Mastodon status (toot) request
type MastodonStatus struct {
	Status     string `json:"status"`
	Visibility string `json:"visibility,omitempty"`
	Language   string `json:"language,omitempty"`
}

// MastodonStatusResponse represents a Mastodon status response
type MastodonStatusResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// NewMastodonService creates a new Mastodon service instance
func NewMastodonService() Service {
	return &MastodonService{
		client:     &http.Client{},
		visibility: "public", // Default visibility
	}
}

// GetServiceID returns the service identifier
func (s *MastodonService) GetServiceID() string {
	return "mastodon"
}

// GetDefaultPort returns the default port
func (s *MastodonService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *MastodonService) ParseURL(serviceURL *url.URL) error {
	// URL format: mastodon://access_token@instance.social?visibility=public
	
	if serviceURL.User == nil {
		return fmt.Errorf("Mastodon URL must include access token")
	}
	
	s.accessToken = serviceURL.User.Username()
	if s.accessToken == "" {
		return fmt.Errorf("Mastodon access token cannot be empty")
	}
	
	// Extract instance URL from host
	if serviceURL.Host == "" {
		return fmt.Errorf("Mastodon URL must specify instance host")
	}
	
	// Build instance URL
	scheme := "https"
	if serviceURL.Scheme == "mastodon+http" {
		scheme = "http"
	}
	
	port := serviceURL.Port()
	if port != "" && port != "443" && port != "80" {
		s.instanceURL = fmt.Sprintf("%s://%s:%s", scheme, serviceURL.Hostname(), port)
	} else {
		s.instanceURL = fmt.Sprintf("%s://%s", scheme, serviceURL.Hostname())
	}
	
	// Parse query parameters
	query := serviceURL.Query()
	if visibility := query.Get("visibility"); visibility != "" {
		// Validate visibility
		switch visibility {
		case "public", "unlisted", "private", "direct":
			s.visibility = visibility
		default:
			return fmt.Errorf("invalid Mastodon visibility: %s (must be public, unlisted, private, or direct)", visibility)
		}
	}
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *MastodonService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *MastodonService) Send(ctx context.Context, req NotificationRequest) error {
	// Build status content
	content := req.Body
	if req.Title != "" {
		content = req.Title + "\n\n" + content
	}
	
	// Post status to Mastodon
	return s.postStatus(ctx, content)
}

// postStatus posts a status (toot) to Mastodon
func (s *MastodonService) postStatus(ctx context.Context, content string) error {
	// Mastodon API endpoint
	apiURL := fmt.Sprintf("%s/api/v1/statuses", s.instanceURL)
	
	// Prepare status data
	status := MastodonStatus{
		Status:     content,
		Visibility: s.visibility,
		Language:   "en", // Default to English
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal Mastodon status: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Mastodon request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	req.Header.Set("User-Agent", "Apprise-Go/1.0")
	
	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Mastodon API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error response
		var errorBody map[string]interface{}
		if json.NewDecoder(resp.Body).Decode(&errorBody) == nil {
			if errorMsg, ok := errorBody["error"].(string); ok {
				return fmt.Errorf("Mastodon API error: %s (status %d)", errorMsg, resp.StatusCode)
			}
		}
		return fmt.Errorf("Mastodon API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// SupportsAttachments returns true if this service supports file attachments
func (s *MastodonService) SupportsAttachments() bool {
	return true // Mastodon supports media attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *MastodonService) GetMaxBodyLength() int {
	return 500 // Mastodon default character limit (configurable per instance)
}