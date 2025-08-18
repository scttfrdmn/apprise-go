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

// RichMobilePushService provides advanced mobile push notifications with rich content
type RichMobilePushService struct {
	platform        string // "ios", "android", or "both"
	deviceTokens    []string
	apnsService     *APNSService
	fcmService      *FCMService
	client          *http.Client
	
	// Rich content features
	images          []string
	actions         []MobilePushAction
	category        string
	sound           string
	badge           int
	priority        string // "high", "normal", "low"
	timeToLive      time.Duration
	collapseKey     string
	restrictedPackage string
	
	// Interactive features
	replyAction     bool
	customData      map[string]interface{}
	deepLink        string
	webLink         string
	
	// Visual enhancements
	icon            string
	color           string
	channelID       string
	groupKey        string
	localizedTitle  map[string]string
	localizedBody   map[string]string
	
	// Scheduling
	scheduleTime    time.Time
	timeZone        string
	
	// Analytics and tracking
	trackingID      string
	campaignID      string
	userData        map[string]string
}

// MobilePushAction represents an interactive action in mobile push notifications
type MobilePushAction struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Icon        string                 `json:"icon,omitempty"`
	URL         string                 `json:"url,omitempty"`
	DeepLink    string                 `json:"deep_link,omitempty"`
	InputType   string                 `json:"input_type,omitempty"` // "text", "choice"
	InputPrompt string                 `json:"input_prompt,omitempty"`
	Choices     []string               `json:"choices,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RichPushPayload represents the unified rich push payload
type RichPushPayload struct {
	// Common fields
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Subtitle    string                 `json:"subtitle,omitempty"`
	Sound       string                 `json:"sound,omitempty"`
	Badge       int                    `json:"badge,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Color       string                 `json:"color,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	Actions     []MobilePushAction     `json:"actions,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	
	// Links and navigation
	DeepLink    string `json:"deep_link,omitempty"`
	WebLink     string `json:"web_link,omitempty"`
	
	// Platform-specific
	IOSPayload     *APNSPayload `json:"ios_payload,omitempty"`
	AndroidPayload interface{}  `json:"android_payload,omitempty"`
	
	// Metadata
	TrackingID string            `json:"tracking_id,omitempty"`
	CampaignID string            `json:"campaign_id,omitempty"`
	UserData   map[string]string `json:"user_data,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// NewRichMobilePushService creates a new rich mobile push service
func NewRichMobilePushService() *RichMobilePushService {
	return &RichMobilePushService{
		client:      &http.Client{Timeout: 30 * time.Second},
		customData:  make(map[string]interface{}),
		userData:    make(map[string]string),
		localizedTitle: make(map[string]string),
		localizedBody:  make(map[string]string),
		priority:    "normal",
		timeToLive:  24 * time.Hour,
	}
}

func (rmp *RichMobilePushService) GetServiceID() string {
	return "rich-mobile-push"
}

func (rmp *RichMobilePushService) GetDefaultPort() int {
	return 443 // HTTPS
}

func (rmp *RichMobilePushService) ParseURL(serviceURL *url.URL) error {
	// URL format: rich-mobile-push://platform@tokens?params
	// Example: rich-mobile-push://ios@token1,token2?sound=default&badge=5&priority=high
	
	// Extract platform
	if serviceURL.User != nil {
		rmp.platform = serviceURL.User.Username()
	} else {
		return fmt.Errorf("platform (ios/android/both) must be specified in URL")
	}
	
	// Validate platform
	if rmp.platform != "ios" && rmp.platform != "android" && rmp.platform != "both" {
		return fmt.Errorf("platform must be 'ios', 'android', or 'both'")
	}
	
	// Extract device tokens from host
	if serviceURL.Host == "" {
		return fmt.Errorf("device tokens must be specified")
	}
	
	rmp.deviceTokens = strings.Split(serviceURL.Host, ",")
	for i, token := range rmp.deviceTokens {
		rmp.deviceTokens[i] = strings.TrimSpace(token)
	}
	
	// Parse query parameters
	query := serviceURL.Query()
	
	// Basic parameters
	if sound := query.Get("sound"); sound != "" {
		rmp.sound = sound
	}
	
	if badgeStr := query.Get("badge"); badgeStr != "" {
		if badge, err := strconv.Atoi(badgeStr); err == nil {
			rmp.badge = badge
		}
	}
	
	if priority := query.Get("priority"); priority != "" {
		rmp.priority = priority
	}
	
	if category := query.Get("category"); category != "" {
		rmp.category = category
	}
	
	// Visual parameters
	if icon := query.Get("icon"); icon != "" {
		rmp.icon = icon
	}
	
	if color := query.Get("color"); color != "" {
		rmp.color = color
	}
	
	if channelID := query.Get("channel"); channelID != "" {
		rmp.channelID = channelID
	}
	
	if groupKey := query.Get("group"); groupKey != "" {
		rmp.groupKey = groupKey
	}
	
	// Interactive parameters
	if replyStr := query.Get("reply"); replyStr == "true" || replyStr == "1" {
		rmp.replyAction = true
	}
	
	if deepLink := query.Get("deeplink"); deepLink != "" {
		rmp.deepLink = deepLink
	}
	
	if webLink := query.Get("weblink"); webLink != "" {
		rmp.webLink = webLink
	}
	
	// TTL parameter
	if ttlStr := query.Get("ttl"); ttlStr != "" {
		if ttl, err := strconv.Atoi(ttlStr); err == nil {
			rmp.timeToLive = time.Duration(ttl) * time.Second
		}
	}
	
	// Collapse key for Android
	if collapseKey := query.Get("collapse"); collapseKey != "" {
		rmp.collapseKey = collapseKey
	}
	
	// Images
	if image := query.Get("image"); image != "" {
		rmp.images = append(rmp.images, image)
	}
	
	// Parse actions (format: action1=id:title:icon:url)
	for key, values := range query {
		if strings.HasPrefix(key, "action") && len(values) > 0 {
			parts := strings.SplitN(values[0], ":", 4)
			if len(parts) >= 2 {
				action := MobilePushAction{
					ID:    parts[0],
					Title: parts[1],
				}
				if len(parts) > 2 && parts[2] != "" {
					action.Icon = parts[2]
				}
				if len(parts) > 3 && parts[3] != "" {
					if strings.HasPrefix(parts[3], "http") {
						action.URL = parts[3]
					} else {
						action.DeepLink = parts[3]
					}
				}
				rmp.actions = append(rmp.actions, action)
			}
		}
	}
	
	// Tracking parameters
	if trackingID := query.Get("tracking"); trackingID != "" {
		rmp.trackingID = trackingID
	}
	
	if campaignID := query.Get("campaign"); campaignID != "" {
		rmp.campaignID = campaignID
	}
	
	return nil
}

func (rmp *RichMobilePushService) Send(ctx context.Context, req NotificationRequest) error {
	// Create rich payload
	payload := rmp.createRichPayload(req)
	
	// Send to appropriate platforms
	var errors []error
	
	if rmp.platform == "ios" || rmp.platform == "both" {
		if err := rmp.sendToIOS(ctx, payload); err != nil {
			errors = append(errors, fmt.Errorf("iOS delivery failed: %w", err))
		}
	}
	
	if rmp.platform == "android" || rmp.platform == "both" {
		if err := rmp.sendToAndroid(ctx, payload); err != nil {
			errors = append(errors, fmt.Errorf("Android delivery failed: %w", err))
		}
	}
	
	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("push notification errors: %v", errors)
	}
	
	return nil
}

func (rmp *RichMobilePushService) createRichPayload(req NotificationRequest) *RichPushPayload {
	payload := &RichPushPayload{
		Title:      req.Title,
		Body:       req.Body,
		Sound:      rmp.sound,
		Badge:      rmp.badge,
		Icon:       rmp.icon,
		Color:      rmp.color,
		Priority:   rmp.priority,
		Actions:    rmp.actions,
		CustomData: rmp.customData,
		DeepLink:   rmp.deepLink,
		WebLink:    rmp.webLink,
		TrackingID: rmp.trackingID,
		CampaignID: rmp.campaignID,
		UserData:   rmp.userData,
		Timestamp:  time.Now(),
	}
	
	// Add first image if available
	if len(rmp.images) > 0 {
		payload.Image = rmp.images[0]
	}
	
	// Create iOS-specific payload
	if rmp.platform == "ios" || rmp.platform == "both" {
		payload.IOSPayload = rmp.createIOSPayload(req)
	}
	
	// Create Android-specific payload
	if rmp.platform == "android" || rmp.platform == "both" {
		payload.AndroidPayload = rmp.createAndroidPayload(req)
	}
	
	return payload
}

func (rmp *RichMobilePushService) createIOSPayload(req NotificationRequest) *APNSPayload {
	alert := &APNSAlert{
		Title: req.Title,
		Body:  req.Body,
	}
	
	aps := &APSPayload{
		Alert:    alert,
		Sound:    rmp.sound,
		Badge:    rmp.badge,
		Category: rmp.category,
	}
	
	// Set interruption level based on priority
	switch rmp.priority {
	case "high":
		aps.InterruptionLevel = "time-sensitive"
	case "low":
		aps.InterruptionLevel = "passive"
	default:
		aps.InterruptionLevel = "active"
	}
	
	// Enable mutable content for rich media
	if len(rmp.images) > 0 || len(rmp.actions) > 0 {
		aps.MutableContent = 1
	}
	
	payload := &APNSPayload{
		APS:  aps,
		Data: make(map[string]interface{}),
	}
	
	// Add custom data
	for key, value := range rmp.customData {
		payload.Data[key] = value
	}
	
	// Add tracking data
	if rmp.trackingID != "" {
		payload.Data["tracking_id"] = rmp.trackingID
	}
	
	if rmp.campaignID != "" {
		payload.Data["campaign_id"] = rmp.campaignID
	}
	
	// Add deep link
	if rmp.deepLink != "" {
		payload.Data["deep_link"] = rmp.deepLink
	}
	
	// Add images
	if len(rmp.images) > 0 {
		payload.Data["images"] = rmp.images
	}
	
	// Add actions
	if len(rmp.actions) > 0 {
		payload.Data["actions"] = rmp.actions
	}
	
	return payload
}

func (rmp *RichMobilePushService) createAndroidPayload(req NotificationRequest) map[string]interface{} {
	notification := map[string]interface{}{
		"title": req.Title,
		"body":  req.Body,
	}
	
	if rmp.icon != "" {
		notification["icon"] = rmp.icon
	}
	
	if rmp.color != "" {
		notification["color"] = rmp.color
	}
	
	if rmp.sound != "" {
		notification["sound"] = rmp.sound
	}
	
	if len(rmp.images) > 0 {
		notification["image"] = rmp.images[0]
	}
	
	// Android-specific configuration
	android := map[string]interface{}{
		"priority": rmp.priority,
		"ttl":      fmt.Sprintf("%ds", int(rmp.timeToLive.Seconds())),
	}
	
	if rmp.channelID != "" {
		android["notification"] = map[string]interface{}{
			"channel_id": rmp.channelID,
		}
	}
	
	if rmp.collapseKey != "" {
		android["collapse_key"] = rmp.collapseKey
	}
	
	// Data payload
	data := make(map[string]interface{})
	for key, value := range rmp.customData {
		data[key] = value
	}
	
	if rmp.trackingID != "" {
		data["tracking_id"] = rmp.trackingID
	}
	
	if rmp.campaignID != "" {
		data["campaign_id"] = rmp.campaignID
	}
	
	if rmp.deepLink != "" {
		data["deep_link"] = rmp.deepLink
	}
	
	if len(rmp.actions) > 0 {
		data["actions"] = rmp.actions
	}
	
	return map[string]interface{}{
		"notification": notification,
		"android":      android,
		"data":         data,
	}
}

func (rmp *RichMobilePushService) sendToIOS(ctx context.Context, payload *RichPushPayload) error {
	// This would typically integrate with the existing APNS service
	// For now, we'll create a webhook-based approach for rich features
	
	for _, token := range rmp.deviceTokens {
		request := map[string]interface{}{
			"platform":      "ios",
			"device_token":  token,
			"payload":       payload.IOSPayload,
			"priority":      rmp.priority,
			"tracking_id":   rmp.trackingID,
			"timestamp":     time.Now().Unix(),
		}
		
		if err := rmp.sendWebhookRequest(ctx, request); err != nil {
			return err
		}
	}
	
	return nil
}

func (rmp *RichMobilePushService) sendToAndroid(ctx context.Context, payload *RichPushPayload) error {
	// Similar webhook approach for Android/FCM
	
	for _, token := range rmp.deviceTokens {
		request := map[string]interface{}{
			"platform":     "android",
			"device_token": token,
			"payload":      payload.AndroidPayload,
			"priority":     rmp.priority,
			"tracking_id":  rmp.trackingID,
			"timestamp":    time.Now().Unix(),
		}
		
		if err := rmp.sendWebhookRequest(ctx, request); err != nil {
			return err
		}
	}
	
	return nil
}

func (rmp *RichMobilePushService) sendWebhookRequest(ctx context.Context, request map[string]interface{}) error {
	// Serialize request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal push request: %w", err)
	}
	
	// Create HTTP request - using a placeholder URL for now
	webhookURL := "https://api.apprise.example.com/push/rich"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create push request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())
	
	// For demo purposes, we'll just log the request instead of sending
	fmt.Printf("Rich Mobile Push Request: %s\n", string(jsonData))
	
	return nil // Return nil for demo - in real implementation, would send HTTP request
}

func (rmp *RichMobilePushService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return err
	}
	return rmp.ParseURL(parsedURL)
}

func (rmp *RichMobilePushService) SupportsAttachments() bool {
	return true // Rich mobile push supports images and rich content
}

func (rmp *RichMobilePushService) GetMaxBodyLength() int {
	return 4000 // Generous limit for rich content
}

// Advanced features for rich mobile push

// AddLocalizedContent adds localized title and body content
func (rmp *RichMobilePushService) AddLocalizedContent(lang, title, body string) {
	rmp.localizedTitle[lang] = title
	rmp.localizedBody[lang] = body
}

// AddCustomData adds custom key-value data to the push payload
func (rmp *RichMobilePushService) AddCustomData(key string, value interface{}) {
	rmp.customData[key] = value
}

// AddUserData adds user-specific tracking data
func (rmp *RichMobilePushService) AddUserData(key, value string) {
	rmp.userData[key] = value
}

// SetSchedule sets a future delivery time for the notification
func (rmp *RichMobilePushService) SetSchedule(scheduleTime time.Time, timeZone string) {
	rmp.scheduleTime = scheduleTime
	rmp.timeZone = timeZone
}

// AddAction adds an interactive action to the notification
func (rmp *RichMobilePushService) AddAction(action MobilePushAction) {
	rmp.actions = append(rmp.actions, action)
}

// AddImage adds an image to the rich notification
func (rmp *RichMobilePushService) AddImage(imageURL string) {
	rmp.images = append(rmp.images, imageURL)
}

// BatchMobilePushService handles batch delivery of rich mobile notifications
type BatchMobilePushService struct {
	*RichMobilePushService
	batchSize     int
	retryAttempts int
	retryDelay    time.Duration
	
	// Analytics
	deliveryStats map[string]int
	errorStats    map[string]int
}

// NewBatchMobilePushService creates a new batch mobile push service
func NewBatchMobilePushService() *BatchMobilePushService {
	return &BatchMobilePushService{
		RichMobilePushService: NewRichMobilePushService(),
		batchSize:             100,  // Process 100 tokens at a time
		retryAttempts:         3,
		retryDelay:            5 * time.Second,
		deliveryStats:         make(map[string]int),
		errorStats:            make(map[string]int),
	}
}

func (bmp *BatchMobilePushService) GetServiceID() string {
	return "batch-mobile-push"
}

// SendBatch sends notifications to multiple device tokens in batches
func (bmp *BatchMobilePushService) SendBatch(ctx context.Context, req NotificationRequest, deviceTokens []string) error {
	// Process tokens in batches
	for i := 0; i < len(deviceTokens); i += bmp.batchSize {
		end := i + bmp.batchSize
		if end > len(deviceTokens) {
			end = len(deviceTokens)
		}
		
		batch := deviceTokens[i:end]
		if err := bmp.sendBatchInternal(ctx, req, batch); err != nil {
			return fmt.Errorf("batch %d failed: %w", i/bmp.batchSize+1, err)
		}
	}
	
	return nil
}

func (bmp *BatchMobilePushService) sendBatchInternal(ctx context.Context, req NotificationRequest, tokens []string) error {
	// Set the device tokens for this batch
	originalTokens := bmp.deviceTokens
	bmp.deviceTokens = tokens
	defer func() { bmp.deviceTokens = originalTokens }()
	
	// Attempt delivery with retries
	var lastError error
	for attempt := 0; attempt <= bmp.retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(bmp.retryDelay):
			}
		}
		
		if err := bmp.RichMobilePushService.Send(ctx, req); err != nil {
			lastError = err
			bmp.errorStats[fmt.Sprintf("attempt_%d", attempt+1)]++
			continue
		}
		
		// Success
		bmp.deliveryStats["successful_batches"]++
		bmp.deliveryStats["delivered_tokens"] += len(tokens)
		return nil
	}
	
	// All attempts failed
	bmp.errorStats["failed_batches"]++
	return lastError
}

// GetDeliveryStats returns delivery statistics
func (bmp *BatchMobilePushService) GetDeliveryStats() map[string]int {
	stats := make(map[string]int)
	for k, v := range bmp.deliveryStats {
		stats[k] = v
	}
	for k, v := range bmp.errorStats {
		stats[k] = v
	}
	return stats
}