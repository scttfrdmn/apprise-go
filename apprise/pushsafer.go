package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PushsaferService implements Pushsafer push notification service
// Pushsafer is a popular European alternative to Pushover with GDPR compliance
// Supports iOS, Android, and Windows 10 devices
type PushsaferService struct {
	privateKey string   // Private key for authentication
	device     string   // Device ID, device group, or "a" for all devices
	sound      int      // Sound ID (0-60)
	vibration  int      // Vibration pattern (1-3)
	icon       int      // Icon ID (1-176)
	iconColor  string   // Icon color in hex format (#RRGGBB)
	priority   int      // Priority (-2 to 2)
	timeToLive int      // Time to live in minutes (0-43200)
	client     *http.Client
}

// PushsaferRequest represents the API request payload
type PushsaferRequest struct {
	PrivateKey string `json:"k"`           // Private key (required)
	Message    string `json:"m"`           // Message text (required)
	Title      string `json:"t,omitempty"` // Title
	Device     string `json:"d,omitempty"` // Device/group ID or "a" for all
	Icon       int    `json:"i,omitempty"` // Icon ID (1-176)
	IconColor  string `json:"c,omitempty"` // Icon color (#RRGGBB)
	Sound      int    `json:"s,omitempty"` // Sound ID (0-60)
	Vibration  int    `json:"v,omitempty"` // Vibration pattern (1-3)
	URL        string `json:"u,omitempty"` // URL to open
	URLTitle   string `json:"ut,omitempty"` // URL button title
	TimeToLive int    `json:"l,omitempty"` // TTL in minutes
	Priority   int    `json:"pr,omitempty"` // Priority (-2 to 2)
	Retry      int    `json:"re,omitempty"` // Retry interval (60-10800 seconds)
	Expire     int    `json:"ex,omitempty"` // Expire time (60-10800 seconds)
	Answer     int    `json:"a,omitempty"` // Allow answer (0 or 1)
}

// PushsaferResponse represents the API response
type PushsaferResponse struct {
	Status  string   `json:"status"`  // "success" or "error"
	Success string   `json:"success"` // Message on success
	Errors  []string `json:"errors"`  // Error messages if any
}

// NewPushsaferService creates a new Pushsafer service instance
func NewPushsaferService() Service {
	return &PushsaferService{
		device:    "a", // Default: all devices
		sound:     0,   // Default sound
		vibration: 0,   // Default vibration
		icon:      1,   // Default icon
		priority:  0,   // Normal priority
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (p *PushsaferService) GetServiceID() string {
	return "pushsafer"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PushsaferService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Pushsafer service URL
// Format: pushsafer://privatekey
// Format: pushsafer://privatekey/device
// Format: pushsafer://privatekey/device?sound=5&vibration=2&icon=10
//
// Parameters:
//   - privatekey: Your Pushsafer private key (required)
//   - device: Device ID, device group (gs*), or "a" for all devices
//
// Query Parameters:
//   - device=id - Override device from path (device ID, gs* for groups, "a" for all)
//   - sound=0-60 - Sound ID (0=silent, 1-60=various sounds)
//   - vibration=1-3 - Vibration pattern (1=default, 2=once, 3=twice)
//   - icon=1-176 - Icon ID
//   - color=#RRGGBB - Icon color in hex format
//   - priority=-2 to 2 - Priority level (-2=lowest, 2=highest/critical)
//   - ttl=minutes - Time to live (0-43200 minutes)
//
// Examples:
//   pushsafer://a1b2c3d4e5f6
//   pushsafer://a1b2c3d4e5f6/a (all devices)
//   pushsafer://a1b2c3d4e5f6/52 (specific device)
//   pushsafer://a1b2c3d4e5f6/gs100 (device group)
//   pushsafer://key?sound=5&vibration=2&icon=33&color=%23FF0000
func (p *PushsaferService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "pushsafer" && scheme != "psafer" {
		return fmt.Errorf("invalid scheme: expected 'pushsafer' or 'psafer', got '%s'", scheme)
	}

	// Extract private key from host
	privateKey := serviceURL.Host
	if privateKey == "" {
		return fmt.Errorf("missing Pushsafer private key")
	}
	p.privateKey = privateKey

	// Extract device from path
	path := strings.Trim(serviceURL.Path, "/")
	if path != "" {
		p.device = path
	}

	// Parse query parameters
	query := serviceURL.Query()

	// Override device from query param if provided
	if device := query.Get("device"); device != "" {
		p.device = device
	}

	// Parse sound
	if soundStr := query.Get("sound"); soundStr != "" {
		if sound, err := strconv.Atoi(soundStr); err == nil && sound >= 0 && sound <= 60 {
			p.sound = sound
		}
	}

	// Parse vibration
	if vibStr := query.Get("vibration"); vibStr != "" {
		if vibration, err := strconv.Atoi(vibStr); err == nil && vibration >= 1 && vibration <= 3 {
			p.vibration = vibration
		}
	}

	// Parse icon
	if iconStr := query.Get("icon"); iconStr != "" {
		if icon, err := strconv.Atoi(iconStr); err == nil && icon >= 1 && icon <= 176 {
			p.icon = icon
		}
	}

	// Parse icon color
	if color := query.Get("color"); color != "" {
		// Ensure it starts with #
		if !strings.HasPrefix(color, "#") {
			color = "#" + color
		}
		p.iconColor = color
	}

	// Parse priority
	if priorityStr := query.Get("priority"); priorityStr != "" {
		if priority, err := strconv.Atoi(priorityStr); err == nil && priority >= -2 && priority <= 2 {
			p.priority = priority
		}
	}

	// Parse TTL
	if ttlStr := query.Get("ttl"); ttlStr != "" {
		if ttl, err := strconv.Atoi(ttlStr); err == nil && ttl >= 0 && ttl <= 43200 {
			p.timeToLive = ttl
		}
	}

	return nil
}

// Send sends a push notification via Pushsafer
func (p *PushsaferService) Send(ctx context.Context, req NotificationRequest) error {
	// Build request payload
	payload := p.buildRequest(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	apiURL := "https://www.pushsafer.com/api"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var psResp PushsaferResponse
	if err := json.NewDecoder(resp.Body).Decode(&psResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if psResp.Status != "success" {
		if len(psResp.Errors) > 0 {
			return fmt.Errorf("pushsafer error: %s", strings.Join(psResp.Errors, ", "))
		}
		return fmt.Errorf("pushsafer request failed with status: %s", psResp.Status)
	}

	return nil
}

// buildRequest constructs the Pushsafer API request
func (p *PushsaferService) buildRequest(req NotificationRequest) PushsaferRequest {
	payload := PushsaferRequest{
		PrivateKey: p.privateKey,
		Device:     p.device,
		Icon:       p.icon,
		IconColor:  p.iconColor,
		Sound:      p.sound,
		Vibration:  p.vibration,
		Priority:   p.mapNotifyTypeToPriority(req.NotifyType),
	}

	// Set title
	if req.Title != "" {
		payload.Title = req.Title
	}

	// Set message (required)
	if req.Body != "" {
		payload.Message = req.Body
	} else if req.Title != "" {
		payload.Message = req.Title
	} else {
		payload.Message = "Notification from apprise-go"
	}

	// Truncate message if too long (Pushsafer limit is 10000 characters)
	if len(payload.Message) > 10000 {
		payload.Message = payload.Message[:9997] + "..."
	}

	// Set TTL if configured
	if p.timeToLive > 0 {
		payload.TimeToLive = p.timeToLive
	}

	// Override priority if configured
	if p.priority != 0 {
		payload.Priority = p.priority
	}

	return payload
}

// mapNotifyTypeToPriority maps notification types to Pushsafer priorities
// Priority: -2=very low, -1=moderate, 0=normal, 1=high, 2=critical (ignores DND)
func (p *PushsaferService) mapNotifyTypeToPriority(notifyType NotifyType) int {
	switch notifyType {
	case NotifyTypeError:
		return 2 // Critical priority (ignores Do Not Disturb)
	case NotifyTypeWarning:
		return 1 // High priority
	case NotifyTypeSuccess:
		return 0 // Normal priority
	case NotifyTypeInfo:
		return 0 // Normal priority
	default:
		return 0
	}
}

// TestURL validates the Pushsafer service URL format
func (p *PushsaferService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns true as Pushsafer supports up to 3 images
func (p *PushsaferService) SupportsAttachments() bool {
	return true
}

// GetMaxBodyLength returns 10000 (Pushsafer's message length limit)
func (p *PushsaferService) GetMaxBodyLength() int {
	return 10000
}
