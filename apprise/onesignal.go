package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OneSignalService implements OneSignal push notification service
// OneSignal is a leading push notification platform sending 12B+ messages/day
// Used by 1M+ apps including major brands and Fortune 500 companies
type OneSignalService struct {
	appID      string // OneSignal App ID (UUID format)
	restAPIKey string // REST API Key for authentication
	segments   []string // Target segments (default: ["Subscribed Users"])
	client     *http.Client
}

// OneSignalNotification represents a OneSignal notification payload
type OneSignalNotification struct {
	AppID            string                       `json:"app_id"`
	Contents         map[string]string            `json:"contents"`
	Headings         map[string]string            `json:"headings,omitempty"`
	Subtitle         map[string]string            `json:"subtitle,omitempty"`
	IncludedSegments []string                     `json:"included_segments,omitempty"`
	ExcludedSegments []string                     `json:"excluded_segments,omitempty"`
	Data             map[string]interface{}       `json:"data,omitempty"`
	TargetChannel    string                       `json:"target_channel,omitempty"`
	Priority         int                          `json:"priority,omitempty"`
}

// OneSignalResponse represents the API response
type OneSignalResponse struct {
	ID         string   `json:"id"`
	Recipients int      `json:"recipients"`
	Errors     []string `json:"errors,omitempty"`
}

// NewOneSignalService creates a new OneSignal service instance
func NewOneSignalService() Service {
	return &OneSignalService{
		segments: []string{"Subscribed Users"}, // Default segment
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (o *OneSignalService) GetServiceID() string {
	return "onesignal"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (o *OneSignalService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a OneSignal service URL
// Format: onesignal://app_id@rest_api_key
// Format: onesignal://app_id@rest_api_key?segments=segment1,segment2
//
// Parameters:
//   - app_id: Your OneSignal App ID (UUID v4 format)
//   - rest_api_key: Your REST API Key
//
// Query Parameters:
//   - segments=segment1,segment2 - Target segments (default: "Subscribed Users")
//
// Examples:
//   onesignal://5eb5a37e-b458-11e3-ac11-000c2940e62c@os_v2_app_abc123...
//   onesignal://app-id@api-key?segments=Active Users,Premium
func (o *OneSignalService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "onesignal" {
		return fmt.Errorf("invalid scheme: expected 'onesignal', got '%s'", scheme)
	}

	// Extract REST API Key from host (comes after @)
	restAPIKey := serviceURL.Host
	if restAPIKey == "" {
		return fmt.Errorf("missing REST API Key")
	}
	o.restAPIKey = restAPIKey

	// Extract App ID from user info (comes before @)
	if serviceURL.User == nil || serviceURL.User.Username() == "" {
		return fmt.Errorf("missing OneSignal App ID")
	}
	o.appID = serviceURL.User.Username()

	// Parse query parameters
	query := serviceURL.Query()

	// Parse segments
	if segmentsStr := query.Get("segments"); segmentsStr != "" {
		segments := strings.Split(segmentsStr, ",")
		o.segments = make([]string, 0, len(segments))
		for _, seg := range segments {
			if trimmed := strings.TrimSpace(seg); trimmed != "" {
				o.segments = append(o.segments, trimmed)
			}
		}
	}

	return nil
}

// Send sends a push notification via OneSignal
func (o *OneSignalService) Send(ctx context.Context, req NotificationRequest) error {
	// Build notification payload
	notification := o.buildNotification(req)

	// Marshal to JSON
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create HTTP request
	apiURL := "https://onesignal.com/api/v1/notifications"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", o.restAPIKey))

	// Send request
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var osResp OneSignalResponse
	if err := json.NewDecoder(resp.Body).Decode(&osResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(osResp.Errors) > 0 {
			return fmt.Errorf("OneSignal API error: %s (status: %d)", strings.Join(osResp.Errors, ", "), resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check for API errors in successful response
	if len(osResp.Errors) > 0 {
		return fmt.Errorf("OneSignal API errors: %s", strings.Join(osResp.Errors, ", "))
	}

	return nil
}

// buildNotification constructs the OneSignal notification payload
func (o *OneSignalService) buildNotification(req NotificationRequest) OneSignalNotification {
	notification := OneSignalNotification{
		AppID:            o.appID,
		Contents:         make(map[string]string),
		IncludedSegments: o.segments,
		TargetChannel:    "push",
	}

	// Build message content
	var contentBuilder strings.Builder
	if req.Body != "" {
		contentBuilder.WriteString(req.Body)
	} else if req.Title != "" {
		contentBuilder.WriteString(req.Title)
	}

	content := contentBuilder.String()
	if content == "" {
		content = "Notification from apprise-go"
	}
	notification.Contents["en"] = content

	// Add title if present and different from body
	if req.Title != "" && req.Title != req.Body {
		notification.Headings = make(map[string]string)
		notification.Headings["en"] = req.Title
	}

	// Add tags as custom data if present
	if len(req.Tags) > 0 {
		notification.Data = make(map[string]interface{})
		notification.Data["tags"] = req.Tags
		notification.Data["notification_type"] = req.NotifyType.String()
	}

	// Set priority based on notification type
	notification.Priority = o.mapNotifyTypeToPriority(req.NotifyType)

	return notification
}

// mapNotifyTypeToPriority maps notification types to OneSignal priorities
// OneSignal priority: 10 = High (wakes device), 5 = Normal, 1 = Low
func (o *OneSignalService) mapNotifyTypeToPriority(notifyType NotifyType) int {
	switch notifyType {
	case NotifyTypeError:
		return 10 // High priority
	case NotifyTypeWarning:
		return 5 // Normal priority
	case NotifyTypeInfo, NotifyTypeSuccess:
		return 5 // Normal priority
	default:
		return 5
	}
}

// TestURL validates the OneSignal service URL format
func (o *OneSignalService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return o.ParseURL(parsedURL)
}

// SupportsAttachments returns false as basic push notifications don't support attachments
// Note: OneSignal does support images and media, but not through the basic API
func (o *OneSignalService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 2048 (OneSignal's custom data limit)
func (o *OneSignalService) GetMaxBodyLength() int {
	return 2048
}
