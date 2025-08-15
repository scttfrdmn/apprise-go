package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// APNSService implements Apple Push Notification Service for iOS notifications
type APNSService struct {
	keyID           string            // Key ID for JWT authentication (.p8 key)
	teamID          string            // Team ID (App Store Connect)
	bundleID        string            // App bundle identifier
	keyPath         string            // Path to .p8 private key file
	certificatePath string            // Path to .p12 certificate (alternative auth)
	certificatePass string            // Password for .p12 certificate
	environment     string            // "production" or "sandbox"
	webhookURL      string            // Webhook proxy URL for secure credential management
	apiKey          string            // API key for webhook authentication
	client          *http.Client
}

// APNSPayload represents the complete APNS request payload
type APNSPayload struct {
	APS  *APSPayload            `json:"aps"`
	Data map[string]interface{} `json:",inline"`
}

// APSPayload represents the Apple-specific payload
type APSPayload struct {
	Alert            interface{}       `json:"alert,omitempty"`
	Badge            interface{}       `json:"badge,omitempty"`
	Sound            interface{}       `json:"sound,omitempty"`
	ThreadID         string            `json:"thread-id,omitempty"`
	Category         string            `json:"category,omitempty"`
	ContentAvailable int               `json:"content-available,omitempty"`
	MutableContent   int               `json:"mutable-content,omitempty"`
	TargetContentID  string            `json:"target-content-id,omitempty"`
	InterruptionLevel string           `json:"interruption-level,omitempty"`
	RelevanceScore   float64           `json:"relevance-score,omitempty"`
	FilterCriteria   string            `json:"filter-criteria,omitempty"`
	Timestamp        int64             `json:"timestamp,omitempty"`
	Event            string            `json:"event,omitempty"`
	DismissalDate    int64             `json:"dismissal-date,omitempty"`
	Stale            bool              `json:"stale,omitempty"`
	ContentState     interface{}       `json:"content-state,omitempty"`
	URLArgs          []string          `json:"url-args,omitempty"`
}

// APNSAlert represents the alert portion of the APS payload
type APNSAlert struct {
	Title        string   `json:"title,omitempty"`
	Subtitle     string   `json:"subtitle,omitempty"`
	Body         string   `json:"body,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
	TitleLocArgs []string `json:"title-loc-args,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
	SummaryArg   string   `json:"summary-arg,omitempty"`
	SummaryArgCount int   `json:"summary-arg-count,omitempty"`
}

// APNSSound represents sound configuration
type APNSSound struct {
	Critical int     `json:"critical,omitempty"`
	Name     string  `json:"name,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
}

// APNSRequest represents the complete APNS webhook request
type APNSRequest struct {
	DeviceToken   string                 `json:"device_token"`
	Payload       APNSPayload            `json:"payload"`
	Headers       map[string]string      `json:"headers"`
	Authentication APNSAuthentication    `json:"authentication"`
	Environment   string                 `json:"environment"`
	Timestamp     string                 `json:"timestamp"`
	Source        string                 `json:"source"`
	Version       string                 `json:"version"`
}

// APNSAuthentication contains authentication information
type APNSAuthentication struct {
	Method          string `json:"method"`           // "jwt" or "certificate"
	KeyID           string `json:"key_id,omitempty"`
	TeamID          string `json:"team_id,omitempty"`
	BundleID        string `json:"bundle_id,omitempty"`
	KeyPath         string `json:"key_path,omitempty"`
	CertificatePath string `json:"certificate_path,omitempty"`
	CertificatePass string `json:"certificate_pass,omitempty"`
}

// NewAPNSService creates a new Apple Push Notification service instance
func NewAPNSService() Service {
	return &APNSService{
		client:      GetCloudHTTPClient("apns"),
		environment: "production", // Default to production
	}
}

// GetServiceID returns the service identifier
func (a *APNSService) GetServiceID() string {
	return "apns"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (a *APNSService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an Apple Push Notification service URL
// Format: apns://webhook.example.com/apns?key_id=KEY&team_id=TEAM&bundle_id=com.app&key_path=path/to/key.p8
// Format: apns://api-key@webhook.example.com/proxy?team_id=TEAM&bundle_id=com.app&cert_path=cert.p12&cert_pass=pass
// Format: apns://webhook.example.com/apns?bundle_id=com.app&environment=sandbox&key_id=KEY&team_id=TEAM
func (a *APNSService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "apns" {
		return fmt.Errorf("invalid scheme: expected 'apns', got '%s'", serviceURL.Scheme)
	}

	// Extract webhook URL components
	scheme := "https"
	if serviceURL.Scheme == "apns" && strings.Contains(serviceURL.Host, "127.0.0.1") {
		// Test mode: use HTTP for localhost
		scheme = "http"
	}
	a.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

	// Extract API key from user info
	if serviceURL.User != nil {
		a.apiKey = serviceURL.User.Username()
	}

	// Parse query parameters
	query := serviceURL.Query()
	
	// Required: bundle_id
	a.bundleID = query.Get("bundle_id")
	if a.bundleID == "" {
		return fmt.Errorf("bundle_id parameter is required")
	}

	// Environment (optional, defaults to production)
	if env := query.Get("environment"); env != "" {
		if env != "production" && env != "sandbox" {
			return fmt.Errorf("environment must be 'production' or 'sandbox', got '%s'", env)
		}
		a.environment = env
	}

	// Authentication: JWT (.p8) or Certificate (.p12)
	a.keyID = query.Get("key_id")
	a.teamID = query.Get("team_id")
	a.keyPath = query.Get("key_path")
	a.certificatePath = query.Get("cert_path")
	a.certificatePass = query.Get("cert_pass")

	// Validate authentication parameters
	hasJWT := a.keyID != "" && a.teamID != ""
	hasCert := a.certificatePath != ""

	if !hasJWT && !hasCert {
		return fmt.Errorf("either JWT authentication (key_id, team_id) or certificate authentication (cert_path) is required")
	}

	if hasJWT && a.keyPath == "" {
		return fmt.Errorf("key_path is required when using JWT authentication")
	}

	return nil
}

// Send sends a push notification via Apple Push Notification Service
func (a *APNSService) Send(ctx context.Context, req NotificationRequest) error {
	// APNS requires device tokens to be specified in the notification request
	// Since we use webhook proxy, we'll send the complete configuration
	// The webhook service will handle device token management
	
	// Create APNS payload
	payload := a.createPayload(req)
	
	// Create authentication config
	auth := a.createAuthentication()
	
	// Create APNS headers
	headers := a.createHeaders(req)
	
	// Create webhook request
	apnsReq := APNSRequest{
		DeviceToken:    "webhook-managed", // Webhook will handle device tokens
		Payload:        payload,
		Headers:        headers,
		Authentication: auth,
		Environment:    a.environment,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Source:         "apprise-go",
		Version:        GetVersion(),
	}

	// Send via webhook proxy
	return a.sendViaWebhook(ctx, apnsReq)
}

// createPayload creates an APNS payload from a notification request
func (a *APNSService) createPayload(req NotificationRequest) APNSPayload {
	// Create alert
	alert := APNSAlert{
		Title: req.Title,
		Body:  req.Body,
	}

	// Create APS payload
	aps := &APSPayload{
		Alert:             alert,
		Badge:             1, // Default badge increment
		Sound:             a.getSoundForNotifyType(req.NotifyType),
		Category:          a.getCategoryForNotifyType(req.NotifyType),
		InterruptionLevel: a.getInterruptionLevelForNotifyType(req.NotifyType),
		RelevanceScore:    a.getRelevanceScoreForNotifyType(req.NotifyType),
		Timestamp:         time.Now().Unix(),
	}

	// Add content-available for background updates
	if req.NotifyType == NotifyTypeInfo {
		aps.ContentAvailable = 1
	}

	// Add mutable-content for rich notifications
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		aps.MutableContent = 1
	}

	// Create custom data
	data := map[string]interface{}{
		"notification_type": req.NotifyType.String(),
		"source":           "apprise-go",
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	// Add body format if specified
	if req.BodyFormat != "" {
		data["body_format"] = req.BodyFormat
	}

	// Add attachment information
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		attachmentInfo := make([]map[string]string, len(attachments))
		
		for i, attachment := range attachments {
			attachmentInfo[i] = map[string]string{
				"name":      attachment.GetName(),
				"mime_type": attachment.GetMimeType(),
			}
		}
		
		data["attachments"] = attachmentInfo
	}

	return APNSPayload{
		APS:  aps,
		Data: data,
	}
}

// createAuthentication creates authentication configuration
func (a *APNSService) createAuthentication() APNSAuthentication {
	if a.keyID != "" && a.teamID != "" {
		// JWT authentication
		return APNSAuthentication{
			Method:   "jwt",
			KeyID:    a.keyID,
			TeamID:   a.teamID,
			BundleID: a.bundleID,
			KeyPath:  a.keyPath,
		}
	} else {
		// Certificate authentication
		return APNSAuthentication{
			Method:          "certificate",
			BundleID:        a.bundleID,
			CertificatePath: a.certificatePath,
			CertificatePass: a.certificatePass,
		}
	}
}

// createHeaders creates APNS HTTP headers
func (a *APNSService) createHeaders(req NotificationRequest) map[string]string {
	headers := map[string]string{
		"apns-topic":      a.bundleID,
		"apns-priority":   a.getPriorityForNotifyType(req.NotifyType),
		"apns-expiration": strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10),
		"apns-push-type":  "alert",
	}

	// Add collapse ID for notification grouping
	if req.NotifyType != NotifyTypeInfo {
		headers["apns-collapse-id"] = fmt.Sprintf("apprise-%s", req.NotifyType.String())
	}

	// Add thread ID for notification threading
	headers["apns-thread-id"] = fmt.Sprintf("apprise-%s", a.bundleID)

	return headers
}

// sendViaWebhook sends the APNS request via webhook proxy
func (a *APNSService) sendViaWebhook(ctx context.Context, apnsReq APNSRequest) error {
	// Marshal to JSON
	jsonData, err := json.Marshal(apnsReq)
	if err != nil {
		return fmt.Errorf("failed to marshal APNS webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create APNS request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())
	
	if a.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.apiKey))
		httpReq.Header.Set("X-API-Key", a.apiKey)
	}

	// Send request
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send APNS notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("APNS API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods for APNS configuration

func (a *APNSService) getSoundForNotifyType(notifyType NotifyType) interface{} {
	switch notifyType {
	case NotifyTypeError:
		return APNSSound{
			Critical: 1,
			Name:     "critical.wav",
			Volume:   1.0,
		}
	case NotifyTypeWarning:
		return APNSSound{
			Name:   "warning.wav",
			Volume: 0.8,
		}
	case NotifyTypeSuccess:
		return "success.wav"
	case NotifyTypeInfo:
		return "default"
	default:
		return "default"
	}
}

func (a *APNSService) getCategoryForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "ERROR_CATEGORY"
	case NotifyTypeWarning:
		return "WARNING_CATEGORY"
	case NotifyTypeSuccess:
		return "SUCCESS_CATEGORY"
	case NotifyTypeInfo:
		return "INFO_CATEGORY"
	default:
		return "DEFAULT_CATEGORY"
	}
}

func (a *APNSService) getInterruptionLevelForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "critical"
	case NotifyTypeWarning:
		return "time-sensitive"
	case NotifyTypeSuccess:
		return "active"
	case NotifyTypeInfo:
		return "passive"
	default:
		return "active"
	}
}

func (a *APNSService) getRelevanceScoreForNotifyType(notifyType NotifyType) float64 {
	switch notifyType {
	case NotifyTypeError:
		return 1.0
	case NotifyTypeWarning:
		return 0.8
	case NotifyTypeSuccess:
		return 0.6
	case NotifyTypeInfo:
		return 0.4
	default:
		return 0.5
	}
}

func (a *APNSService) getPriorityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError, NotifyTypeWarning:
		return "10" // High priority
	default:
		return "5" // Normal priority
	}
}

// TestURL validates an Apple Push Notification service URL
func (a *APNSService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return a.ParseURL(parsedURL)
}

// SupportsAttachments returns true for APNS (supports rich notifications with images)
func (a *APNSService) SupportsAttachments() bool {
	return true // APNS supports rich notifications with images and media
}

// GetMaxBodyLength returns APNS message body length limit
func (a *APNSService) GetMaxBodyLength() int {
	return 4096 // APNS allows up to 4KB for the entire payload
}

// Example usage and URL formats:
// apns://webhook.example.com/apns?key_id=ABC123&team_id=DEF456&bundle_id=com.example.app&key_path=path/to/AuthKey.p8
// apns://api-key@webhook.example.com/proxy?bundle_id=com.example.app&cert_path=cert.p12&cert_pass=password
// apns://webhook.example.com/apns?bundle_id=com.example.app&environment=sandbox&key_id=KEY&team_id=TEAM&key_path=path/to/key.p8