package apprise

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// YouTubeService implements YouTube notifications via API
type YouTubeService struct {
	apiKey    string
	channelID string
	client    *http.Client
}

// YouTubeCommentRequest represents a YouTube comment request
type YouTubeCommentRequest struct {
	Snippet struct {
		VideoID   string `json:"videoId,omitempty"`
		TopLevel  struct {
			Snippet struct {
				TextOriginal string `json:"textOriginal"`
			} `json:"snippet"`
		} `json:"topLevelComment"`
	} `json:"snippet"`
}

// NewYouTubeService creates a new YouTube service instance
func NewYouTubeService() Service {
	return &YouTubeService{
		client: &http.Client{},
	}
}

// GetServiceID returns the service identifier
func (s *YouTubeService) GetServiceID() string {
	return "youtube"
}

// GetDefaultPort returns the default port
func (s *YouTubeService) GetDefaultPort() int {
	return 443 // HTTPS
}

// ParseURL parses the service URL and configures the service
func (s *YouTubeService) ParseURL(serviceURL *url.URL) error {
	// URL format: youtube://api_key@channel_id?video=video_id
	
	if serviceURL.User == nil {
		return fmt.Errorf("YouTube URL must include API key")
	}
	
	s.apiKey = serviceURL.User.Username()
	if s.apiKey == "" {
		return fmt.Errorf("YouTube API key cannot be empty")
	}
	
	// Extract channel ID from host
	if serviceURL.Host == "" {
		return fmt.Errorf("YouTube URL must specify channel ID")
	}
	s.channelID = serviceURL.Host
	
	return nil
}

// TestURL validates that a service URL is properly formatted
func (s *YouTubeService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return s.ParseURL(parsedURL)
}

// Send sends a notification and returns the result
func (s *YouTubeService) Send(ctx context.Context, req NotificationRequest) error {
	// Note: YouTube API has limited posting capabilities
	// Community posts require special permissions and are not available via standard API
	// Comments require a specific video ID
	
	// For demonstration, we'll return an informative error
	return fmt.Errorf("YouTube notifications require specific video ID for comments or special permissions for community posts - not implemented for general notifications")
}

// SupportsAttachments returns true if this service supports file attachments
func (s *YouTubeService) SupportsAttachments() bool {
	return false // YouTube comments don't support attachments
}

// GetMaxBodyLength returns max body length (0 = unlimited)
func (s *YouTubeService) GetMaxBodyLength() int {
	return 10000 // YouTube comment length limit
}