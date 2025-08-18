package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// InstagramService implements Instagram notifications via Basic Display API
type InstagramService struct {
	accessToken string
	userID      string
	client      *http.Client
}

// InstagramMediaRequest represents an Instagram media upload request
type InstagramMediaRequest struct {
	ImageURL string `json:"image_url"`
	Caption  string `json:"caption"`
}

// InstagramPublishRequest represents an Instagram media publish request
type InstagramPublishRequest struct {
	CreationID string `json:"creation_id"`
}

// NewInstagramService creates a new Instagram service instance
func NewInstagramService() Service {
	return &InstagramService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *InstagramService) GetServiceID() string {
	return "instagram"
}

// GetDefaultPort returns the default port
func (s *InstagramService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *InstagramService) ParseURL(serviceURL *url.URL) error {
	// URL format: instagram://access_token@user_id
	
	if serviceURL.User == nil {
		return fmt.Errorf("Instagram URL must include access token")
	}
	
	s.accessToken = serviceURL.User.Username()
	if s.accessToken == "" {
		return fmt.Errorf("Instagram access token cannot be empty")
	}
	
	// Extract user ID from host
	if serviceURL.Host == "" {
		return fmt.Errorf("Instagram URL must specify user ID")
	}
	s.userID = serviceURL.Host
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *InstagramService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *InstagramService) Send(ctx context.Context, req NotificationRequest) error {
	// Instagram Basic Display API is read-only, so we'll create a story instead
	// Note: This requires Instagram Business API for actual posting
	return s.createTextPost(ctx, req)
}

// createTextPost creates a text-based Instagram post
func (s *InstagramService) createTextPost(ctx context.Context, req NotificationRequest) error {
	// Note: Instagram requires either image_url or video_url for posts
	// For text-only notifications, we would need to use Instagram Stories API
	// which requires additional permissions
	
	// For now, return an error indicating limitation
	return fmt.Errorf("Instagram text-only posts not supported via Basic Display API - requires Instagram Business API with media content")
}

// SupportsAttachments returns true if this service supports file attachments
func (s *InstagramService) SupportsAttachments() bool {
	return true // Instagram supports media attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *InstagramService) GetMaxBodyLength() int {
	return 2200 // Instagram caption limit
}