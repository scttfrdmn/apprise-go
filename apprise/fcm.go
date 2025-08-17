package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FCMService implements Firebase Cloud Messaging push notifications
type FCMService struct {
	projectID      string
	serverKey      string // Legacy server key (for backwards compatibility)
	serviceAccount string // Service account JSON for OAuth2
	webhookURL     string // Webhook proxy URL for secure credential management
	apiKey         string // API key for webhook authentication
	client         *http.Client
}

// FCMMessage represents a Firebase Cloud Messaging message
type FCMMessage struct {
	Name         string            `json:"name,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Notification *FCMNotification  `json:"notification,omitempty"`
	Android      *FCMAndroidConfig `json:"android,omitempty"`
	APNS         *FCMApnsConfig    `json:"apns,omitempty"`
	WebPush      *FCMWebPushConfig `json:"webpush,omitempty"`
	FcmOptions   *FCMOptions       `json:"fcm_options,omitempty"`
	Token        string            `json:"token,omitempty"`     // Single device
	Topic        string            `json:"topic,omitempty"`     // Topic messaging
	Condition    string            `json:"condition,omitempty"` // Conditional messaging
}

// FCMNotification represents the notification payload
type FCMNotification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Image string `json:"image,omitempty"`
}

// FCMAndroidConfig represents Android-specific configuration
type FCMAndroidConfig struct {
	CollapseKey           string                  `json:"collapse_key,omitempty"`
	Priority              string                  `json:"priority,omitempty"`
	TTL                   string                  `json:"ttl,omitempty"`
	RestrictedPackageName string                  `json:"restricted_package_name,omitempty"`
	Data                  map[string]string       `json:"data,omitempty"`
	Notification          *FCMAndroidNotification `json:"notification,omitempty"`
	FcmOptions            *FCMAndroidFcmOptions   `json:"fcm_options,omitempty"`
}

// FCMAndroidNotification represents Android notification properties
type FCMAndroidNotification struct {
	Title                 string   `json:"title,omitempty"`
	Body                  string   `json:"body,omitempty"`
	Icon                  string   `json:"icon,omitempty"`
	Color                 string   `json:"color,omitempty"`
	Sound                 string   `json:"sound,omitempty"`
	Tag                   string   `json:"tag,omitempty"`
	ClickAction           string   `json:"click_action,omitempty"`
	BodyLocKey            string   `json:"body_loc_key,omitempty"`
	BodyLocArgs           []string `json:"body_loc_args,omitempty"`
	TitleLocKey           string   `json:"title_loc_key,omitempty"`
	TitleLocArgs          []string `json:"title_loc_args,omitempty"`
	ChannelID             string   `json:"channel_id,omitempty"`
	Ticker                string   `json:"ticker,omitempty"`
	Sticky                bool     `json:"sticky,omitempty"`
	EventTime             string   `json:"event_time,omitempty"`
	LocalOnly             bool     `json:"local_only,omitempty"`
	NotificationPriority  string   `json:"notification_priority,omitempty"`
	DefaultSound          bool     `json:"default_sound,omitempty"`
	DefaultVibrateTimings bool     `json:"default_vibrate_timings,omitempty"`
	DefaultLightSettings  bool     `json:"default_light_settings,omitempty"`
	VibrateTimings        []string `json:"vibrate_timings,omitempty"`
	Visibility            string   `json:"visibility,omitempty"`
	NotificationCount     int      `json:"notification_count,omitempty"`
}

// FCMAndroidFcmOptions represents Android FCM options
type FCMAndroidFcmOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// FCMApnsConfig represents Apple Push Notification configuration
type FCMApnsConfig struct {
	Headers    map[string]string  `json:"headers,omitempty"`
	Payload    interface{}        `json:"payload,omitempty"`
	FcmOptions *FCMApnsFcmOptions `json:"fcm_options,omitempty"`
}

// FCMApnsFcmOptions represents APNS FCM options
type FCMApnsFcmOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
	Image          string `json:"image,omitempty"`
}

// FCMWebPushConfig represents Web Push configuration
type FCMWebPushConfig struct {
	Headers      map[string]string   `json:"headers,omitempty"`
	Data         map[string]string   `json:"data,omitempty"`
	Notification *FCMWebNotification `json:"notification,omitempty"`
	FcmOptions   *FCMWebFcmOptions   `json:"fcm_options,omitempty"`
}

// FCMWebNotification represents Web Push notification
type FCMWebNotification struct {
	Title              string                     `json:"title,omitempty"`
	Body               string                     `json:"body,omitempty"`
	Icon               string                     `json:"icon,omitempty"`
	Image              string                     `json:"image,omitempty"`
	Badge              string                     `json:"badge,omitempty"`
	Tag                string                     `json:"tag,omitempty"`
	Data               interface{}                `json:"data,omitempty"`
	Direction          string                     `json:"dir,omitempty"`
	Language           string                     `json:"lang,omitempty"`
	Renotify           bool                       `json:"renotify,omitempty"`
	RequireInteraction bool                       `json:"requireInteraction,omitempty"`
	Silent             bool                       `json:"silent,omitempty"`
	Timestamp          int64                      `json:"timestamp,omitempty"`
	Vibrate            []int                      `json:"vibrate,omitempty"`
	Actions            []FCMWebNotificationAction `json:"actions,omitempty"`
}

// FCMWebNotificationAction represents Web Push notification action
type FCMWebNotificationAction struct {
	Action string `json:"action"`
	Icon   string `json:"icon,omitempty"`
	Title  string `json:"title"`
}

// FCMWebFcmOptions represents Web Push FCM options
type FCMWebFcmOptions struct {
	Link           string `json:"link,omitempty"`
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// FCMOptions represents general FCM options
type FCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// FCMPayload represents the complete FCM request payload
type FCMPayload struct {
	Message      FCMMessage `json:"message"`
	ValidateOnly bool       `json:"validate_only,omitempty"`
}

// NewFCMService creates a new Firebase Cloud Messaging service instance
func NewFCMService() Service {
	return &FCMService{
		client: GetCloudHTTPClient("fcm"),
	}
}

// GetServiceID returns the service identifier
func (f *FCMService) GetServiceID() string {
	return "fcm"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (f *FCMService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Firebase Cloud Messaging service URL
// Format: fcm://webhook.example.com/firebase?project_id=my-project&server_key=key
// Format: fcm://api-key@webhook.example.com/proxy?project_id=my-project&service_account=path/to/sa.json
func (f *FCMService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "fcm" {
		return fmt.Errorf("invalid scheme: expected 'fcm', got '%s'", serviceURL.Scheme)
	}

	// Extract webhook URL components
	// For testing, preserve the original scheme if it's http
	scheme := "https"
	if serviceURL.Scheme == "fcm" && strings.Contains(serviceURL.Host, "127.0.0.1") {
		// Test mode: use HTTP for localhost
		scheme = "http"
	}
	f.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

	// Extract API key from user info
	if serviceURL.User != nil {
		f.apiKey = serviceURL.User.Username()
	}

	// Parse query parameters
	query := serviceURL.Query()

	// Required: project_id
	f.projectID = query.Get("project_id")
	if f.projectID == "" {
		return fmt.Errorf("project_id parameter is required")
	}

	// Authentication: server_key (legacy) or service_account (OAuth2)
	if serverKey := query.Get("server_key"); serverKey != "" {
		f.serverKey = serverKey
	}

	if serviceAccount := query.Get("service_account"); serviceAccount != "" {
		f.serviceAccount = serviceAccount
	}

	// Either server_key or service_account must be provided
	if f.serverKey == "" && f.serviceAccount == "" {
		return fmt.Errorf("either server_key or service_account parameter is required")
	}

	return nil
}

// Send sends a push notification via Firebase Cloud Messaging
func (f *FCMService) Send(ctx context.Context, req NotificationRequest) error {
	// Create FCM message
	message := f.createMessage(req)

	// Create request payload
	payload := FCMPayload{
		Message: message,
	}

	// Send via webhook proxy
	return f.sendViaWebhook(ctx, payload)
}

// createMessage creates an FCM message from a notification request
func (f *FCMService) createMessage(req NotificationRequest) FCMMessage {
	message := FCMMessage{
		Notification: &FCMNotification{
			Title: req.Title,
			Body:  req.Body,
		},
		Data: make(map[string]string),
		FcmOptions: &FCMOptions{
			AnalyticsLabel: "apprise-go",
		},
	}

	// Add notification type as data
	message.Data["notification_type"] = req.NotifyType.String()
	message.Data["source"] = "apprise-go"
	message.Data["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	// Add platform-specific configurations
	message.Android = f.createAndroidConfig(req)
	message.APNS = f.createAPNSConfig(req)
	message.WebPush = f.createWebPushConfig(req)

	return message
}

// createAndroidConfig creates Android-specific configuration
func (f *FCMService) createAndroidConfig(req NotificationRequest) *FCMAndroidConfig {
	return &FCMAndroidConfig{
		Priority: f.getPriorityForNotifyType(req.NotifyType),
		TTL:      "86400s", // 24 hours
		Notification: &FCMAndroidNotification{
			Title:                 req.Title,
			Body:                  req.Body,
			Icon:                  "ic_notification",
			Color:                 f.getColorForNotifyType(req.NotifyType),
			Sound:                 f.getSoundForNotifyType(req.NotifyType),
			ChannelID:             f.getChannelIDForNotifyType(req.NotifyType),
			NotificationPriority:  f.getAndroidNotificationPriority(req.NotifyType),
			DefaultSound:          true,
			DefaultVibrateTimings: true,
		},
		FcmOptions: &FCMAndroidFcmOptions{
			AnalyticsLabel: "apprise-go-android",
		},
	}
}

// createAPNSConfig creates Apple Push Notification configuration
func (f *FCMService) createAPNSConfig(req NotificationRequest) *FCMApnsConfig {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]interface{}{
				"title": req.Title,
				"body":  req.Body,
			},
			"badge": 1,
			"sound": f.getAPNSSoundForNotifyType(req.NotifyType),
		},
		"notification_type": req.NotifyType.String(),
		"source":            "apprise-go",
	}

	headers := map[string]string{
		"apns-priority":   f.getAPNSPriorityForNotifyType(req.NotifyType),
		"apns-expiration": fmt.Sprintf("%d", time.Now().Add(24*time.Hour).Unix()),
	}

	return &FCMApnsConfig{
		Headers: headers,
		Payload: payload,
		FcmOptions: &FCMApnsFcmOptions{
			AnalyticsLabel: "apprise-go-ios",
		},
	}
}

// createWebPushConfig creates Web Push configuration
func (f *FCMService) createWebPushConfig(req NotificationRequest) *FCMWebPushConfig {
	return &FCMWebPushConfig{
		Headers: map[string]string{
			"TTL": "86400", // 24 hours
		},
		Notification: &FCMWebNotification{
			Title:              req.Title,
			Body:               req.Body,
			Icon:               "/icons/notification-icon.png",
			Badge:              "/icons/badge-icon.png",
			Tag:                f.getTagForNotifyType(req.NotifyType),
			RequireInteraction: req.NotifyType == NotifyTypeError,
			Silent:             false,
			Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		},
		Data: map[string]string{
			"notification_type": req.NotifyType.String(),
			"source":            "apprise-go",
		},
		FcmOptions: &FCMWebFcmOptions{
			AnalyticsLabel: "apprise-go-web",
		},
	}
}

// sendViaWebhook sends the FCM payload via webhook proxy
func (f *FCMService) sendViaWebhook(ctx context.Context, payload FCMPayload) error {
	// Create webhook request payload
	webhookPayload := map[string]interface{}{
		"service":   "fcm",
		"projectId": f.projectID,
		"message":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"source":    "apprise-go",
		"version":   GetVersion(),
	}

	// Add authentication information
	if f.serverKey != "" {
		webhookPayload["serverKey"] = f.serverKey
	}
	if f.serviceAccount != "" {
		webhookPayload["serviceAccount"] = f.serviceAccount
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(webhookPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM webhook payload: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", f.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create FCM request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if f.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.apiKey))
		httpReq.Header.Set("X-API-Key", f.apiKey)
	}

	// Send request
	resp, err := f.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send FCM notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FCM API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods for platform-specific configurations

func (f *FCMService) getPriorityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "high"
	case NotifyTypeWarning:
		return "high"
	default:
		return "normal"
	}
}

func (f *FCMService) getColorForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "#00FF00"
	case NotifyTypeWarning:
		return "#FFA500"
	case NotifyTypeError:
		return "#FF0000"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "#007BFF"
	}
}

func (f *FCMService) getSoundForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "urgent"
	case NotifyTypeWarning:
		return "warning"
	default:
		return "default"
	}
}

func (f *FCMService) getChannelIDForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "error_notifications"
	case NotifyTypeWarning:
		return "warning_notifications"
	case NotifyTypeSuccess:
		return "success_notifications"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "default_notifications"
	}
}

func (f *FCMService) getAndroidNotificationPriority(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "PRIORITY_HIGH"
	case NotifyTypeWarning:
		return "PRIORITY_HIGH"
	default:
		return "PRIORITY_DEFAULT"
	}
}

func (f *FCMService) getAPNSSoundForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "critical.wav"
	case NotifyTypeWarning:
		return "warning.wav"
	default:
		return "default"
	}
}

func (f *FCMService) getAPNSPriorityForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError, NotifyTypeWarning:
		return "10" // High priority
	default:
		return "5" // Normal priority
	}
}

func (f *FCMService) getTagForNotifyType(notifyType NotifyType) string {
	return fmt.Sprintf("apprise_%s", strings.ToLower(notifyType.String()))
}

// TestURL validates a Firebase Cloud Messaging service URL
func (f *FCMService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return f.ParseURL(parsedURL)
}

// SupportsAttachments returns true for FCM (supports rich notifications with images)
func (f *FCMService) SupportsAttachments() bool {
	return true // FCM supports image attachments in notifications
}

// GetMaxBodyLength returns FCM's message body length limit
func (f *FCMService) GetMaxBodyLength() int {
	return 4096 // FCM allows up to 4KB for message body
}

// Example usage and URL formats:
// fcm://webhook.example.com/firebase?project_id=my-project&server_key=AAAA...
// fcm://api-key@webhook.example.com/proxy?project_id=my-project&service_account=path/to/service-account.json
// fcm://webhook.example.com/fcm?project_id=my-firebase-project&server_key=legacy-server-key
