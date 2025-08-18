package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// TikTokService implements TikTok notifications
// Note: TikTok API has very limited posting capabilities and requires special approval
type TikTokService struct {
	accessToken string
	userID      string
	client      *http.Client
}

// NewTikTokService creates a new TikTok service instance
func NewTikTokService() Service {
	return &TikTokService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *TikTokService) GetServiceID() string {
	return "tiktok"
}

// GetDefaultPort returns the default port
func (s *TikTokService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *TikTokService) ParseURL(serviceURL *url.URL) error {
	// URL format: tiktok://access_token@user_id
	
	if serviceURL.User == nil {
		return fmt.Errorf("TikTok URL must include access token")
	}
	
	s.accessToken = serviceURL.User.Username()
	if s.accessToken == "" {
		return fmt.Errorf("TikTok access token cannot be empty")
	}
	
	// Extract user ID from host
	if serviceURL.Host == "" {
		return fmt.Errorf("TikTok URL must specify user ID")
	}
	s.userID = serviceURL.Host
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *TikTokService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *TikTokService) Send(ctx context.Context, req NotificationRequest) error {
	// TikTok API does not support text-only posts or general notifications
	// Video uploads require special permissions and are complex
	// For now, return an informative error
	return fmt.Errorf("TikTok API does not support general text notifications - requires video content and special API approval")
}

// SupportsAttachments returns true if this service supports file attachments
func (s *TikTokService) SupportsAttachments() bool {
	return true // TikTok supports video attachments (with special permissions)
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *TikTokService) GetMaxBodyLength() int {
	return 2200 // TikTok caption character limit
}