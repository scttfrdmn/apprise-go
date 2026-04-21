package apprise

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const joinAPIBase = "https://joinjoaomgcd.appspot.com/_ah/api/messaging/v1/sendPush"

// JoinService implements Join push notification service by Joaomgcd
type JoinService struct {
	apiKey    string
	deviceIDs []string
	client    *http.Client
}

// NewJoinService creates a new Join service instance
func NewJoinService() Service {
	return &JoinService{
		client: GetDefaultHTTPClient(),
	}
}

func (j *JoinService) GetServiceID() string      { return "join" }
func (j *JoinService) GetDefaultPort() int       { return 443 }
func (j *JoinService) SupportsAttachments() bool { return false }
func (j *JoinService) GetMaxBodyLength() int     { return 0 }

// ParseURL parses a Join service URL
// Format: join://apikey/device_id
// Format: join://apikey/device1/device2/...
func (j *JoinService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "join" {
		return fmt.Errorf("invalid scheme: expected 'join', got '%s'", serviceURL.Scheme)
	}

	j.apiKey = serviceURL.Host
	if j.apiKey == "" {
		return fmt.Errorf("join API key is required (format: join://apikey/device_id)")
	}

	// Reset device IDs before parsing (so ParseURL can be called multiple times)
	j.deviceIDs = nil

	// Device IDs come from path segments
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	for _, part := range pathParts {
		if part != "" {
			j.deviceIDs = append(j.deviceIDs, part)
		}
	}

	if len(j.deviceIDs) == 0 {
		return fmt.Errorf("at least one Join device ID is required (format: join://apikey/device_id)")
	}

	return nil
}

// Send sends a notification via Join
func (j *JoinService) Send(ctx context.Context, req NotificationRequest) error {
	var lastErr error
	for _, deviceID := range j.deviceIDs {
		if err := j.sendToDevice(ctx, req, deviceID); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (j *JoinService) sendToDevice(ctx context.Context, req NotificationRequest, deviceID string) error {
	params := url.Values{}
	params.Set("apikey", j.apiKey)
	params.Set("deviceId", deviceID)
	params.Set("title", req.Title)
	params.Set("text", req.Body)

	requestURL := joinAPIBase + "?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Join notification to %s: %w", deviceID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join API error for device %s (status %d): %s", deviceID, resp.StatusCode, string(body))
	}

	return nil
}

// TestURL validates a Join service URL
func (j *JoinService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return j.ParseURL(parsedURL)
}
