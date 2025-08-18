package apprise

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"
)

func TestRichMobilePushService_ParseURL(t *testing.T) {
	testCases := []struct {
		name               string
		url                string
		expectError        bool
		expectedPlatform   string
		expectedTokenCount int
		expectedSound      string
		expectedBadge      int
		expectedPriority   string
		expectedActions    int
		expectedReply      bool
	}{
		{
			name:               "Basic iOS URL",
			url:                "rich-mobile-push://ios@token1",
			expectedPlatform:   "ios",
			expectedTokenCount: 1,
			expectedPriority:   "normal",
		},
		{
			name:               "Android with multiple tokens",
			url:                "rich-mobile-push://android@token1,token2,token3",
			expectedPlatform:   "android",
			expectedTokenCount: 3,
			expectedPriority:   "normal",
		},
		{
			name:               "Both platforms with rich features",
			url:                "rich-mobile-push://both@token1?sound=default&badge=5&priority=high&reply=true",
			expectedPlatform:   "both",
			expectedTokenCount: 1,
			expectedSound:      "default",
			expectedBadge:      5,
			expectedPriority:   "high",
			expectedReply:      true,
		},
		{
			name:               "iOS with actions",
			url:                "rich-mobile-push://ios@token1?action1=yes:Yes:check:app://approve&action2=no:No:close",
			expectedPlatform:   "ios",
			expectedTokenCount: 1,
			expectedPriority:   "normal",
			expectedActions:    2,
		},
		{
			name:               "Full featured notification",
			url:                "rich-mobile-push://both@token1,token2?sound=alert&badge=10&priority=high&reply=true&tracking=track123",
			expectedPlatform:   "both",
			expectedTokenCount: 2,
			expectedSound:      "alert",
			expectedBadge:      10,
			expectedPriority:   "high",
			expectedReply:      true,
		},
		{
			name:        "Missing platform",
			url:         "rich-mobile-push://token1",
			expectError: true,
		},
		{
			name:        "Invalid platform",
			url:         "rich-mobile-push://windows@token1",
			expectError: true,
		},
		{
			name:        "Missing tokens",
			url:         "rich-mobile-push://ios@",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewRichMobilePushService()
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.platform != tc.expectedPlatform {
				t.Errorf("Expected platform %s, got %s", tc.expectedPlatform, service.platform)
			}

			if len(service.deviceTokens) != tc.expectedTokenCount {
				t.Errorf("Expected %d tokens, got %d", tc.expectedTokenCount, len(service.deviceTokens))
			}

			if tc.expectedSound != "" && service.sound != tc.expectedSound {
				t.Errorf("Expected sound %s, got %s", tc.expectedSound, service.sound)
			}

			if tc.expectedBadge > 0 && service.badge != tc.expectedBadge {
				t.Errorf("Expected badge %d, got %d", tc.expectedBadge, service.badge)
			}

			if service.priority != tc.expectedPriority {
				t.Errorf("Expected priority %s, got %s", tc.expectedPriority, service.priority)
			}

			if tc.expectedActions > 0 && len(service.actions) != tc.expectedActions {
				t.Errorf("Expected %d actions, got %d", tc.expectedActions, len(service.actions))
			}

			if service.replyAction != tc.expectedReply {
				t.Errorf("Expected reply action %v, got %v", tc.expectedReply, service.replyAction)
			}
		})
	}
}

func TestRichMobilePushService_Actions(t *testing.T) {
	service := NewRichMobilePushService()
	
	// Parse URL with complex actions
	parsedURL, err := url.Parse("rich-mobile-push://ios@token1?action1=approve:Approve:check:app://approve&action2=deny:Deny:close:https://example.com/deny&action3=later:Later::app://later")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	// Verify actions were parsed correctly
	if len(service.actions) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(service.actions))
	}
	
	// Create a map to find actions by ID (since order may vary)
	actionMap := make(map[string]MobilePushAction)
	for _, action := range service.actions {
		actionMap[action.ID] = action
	}
	
	// Check approve action
	if approveAction, exists := actionMap["approve"]; exists {
		if approveAction.Title != "Approve" {
			t.Errorf("Expected approve action title 'Approve', got '%s'", approveAction.Title)
		}
		if approveAction.Icon != "check" {
			t.Errorf("Expected approve action icon 'check', got '%s'", approveAction.Icon)
		}
		if approveAction.DeepLink != "app://approve" {
			t.Errorf("Expected approve action deep link 'app://approve', got '%s'", approveAction.DeepLink)
		}
	} else {
		t.Error("Approve action not found")
	}
	
	// Check deny action (with URL)
	if denyAction, exists := actionMap["deny"]; exists {
		if denyAction.Title != "Deny" {
			t.Errorf("Expected deny action title 'Deny', got '%s'", denyAction.Title)
		}
		if denyAction.URL != "https://example.com/deny" {
			t.Errorf("Expected deny action URL 'https://example.com/deny', got '%s'", denyAction.URL)
		}
	} else {
		t.Error("Deny action not found")
	}
	
	// Check later action (no icon)
	if laterAction, exists := actionMap["later"]; exists {
		if laterAction.Title != "Later" {
			t.Errorf("Expected later action title 'Later', got '%s'", laterAction.Title)
		}
		if laterAction.Icon != "" {
			t.Errorf("Expected later action icon to be empty, got '%s'", laterAction.Icon)
		}
		if laterAction.DeepLink != "app://later" {
			t.Errorf("Expected later action deep link 'app://later', got '%s'", laterAction.DeepLink)
		}
	} else {
		t.Error("Later action not found")
	}
}

func TestRichMobilePushService_Properties(t *testing.T) {
	service := NewRichMobilePushService()

	if service.GetServiceID() != "rich-mobile-push" {
		t.Errorf("Expected service ID 'rich-mobile-push', got '%s'", service.GetServiceID())
	}

	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}

	// Test attachment support
	if !service.SupportsAttachments() {
		t.Error("Rich mobile push should support attachments")
	}

	// Test increased body length
	if service.GetMaxBodyLength() != 4000 {
		t.Errorf("Expected max body length 4000, got %d", service.GetMaxBodyLength())
	}
}

func TestRichMobilePushService_PayloadCreation(t *testing.T) {
	service := NewRichMobilePushService()
	service.platform = "both"
	service.deviceTokens = []string{"token1"}
	service.sound = "alert"
	service.badge = 5
	service.priority = "high"
	service.trackingID = "track123"
	service.campaignID = "camp456"
	service.deepLink = "app://message"
	service.images = []string{"https://example.com/image.jpg"}
	
	// Add custom data
	service.AddCustomData("user_id", "12345")
	service.AddCustomData("message_type", "promotional")
	
	// Add actions
	service.AddAction(MobilePushAction{
		ID:       "view",
		Title:    "View",
		Icon:     "eye",
		DeepLink: "app://view",
	})

	req := NotificationRequest{
		Title:      "Test Notification",
		Body:       "This is a test notification",
		NotifyType: NotifyTypeInfo,
	}

	payload := service.createRichPayload(req)

	// Verify common payload fields
	if payload.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, payload.Title)
	}
	if payload.Body != req.Body {
		t.Errorf("Expected body '%s', got '%s'", req.Body, payload.Body)
	}
	if payload.Sound != "alert" {
		t.Errorf("Expected sound 'alert', got '%s'", payload.Sound)
	}
	if payload.Badge != 5 {
		t.Errorf("Expected badge 5, got %d", payload.Badge)
	}
	if payload.Priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", payload.Priority)
	}
	if payload.TrackingID != "track123" {
		t.Errorf("Expected tracking ID 'track123', got '%s'", payload.TrackingID)
	}
	if payload.CampaignID != "camp456" {
		t.Errorf("Expected campaign ID 'camp456', got '%s'", payload.CampaignID)
	}
	if payload.DeepLink != "app://message" {
		t.Errorf("Expected deep link 'app://message', got '%s'", payload.DeepLink)
	}
	if payload.Image != "https://example.com/image.jpg" {
		t.Errorf("Expected image 'https://example.com/image.jpg', got '%s'", payload.Image)
	}

	// Verify actions
	if len(payload.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(payload.Actions))
	}
	if payload.Actions[0].ID != "view" {
		t.Errorf("Expected action ID 'view', got '%s'", payload.Actions[0].ID)
	}

	// Verify custom data
	if payload.CustomData["user_id"] != "12345" {
		t.Errorf("Expected custom data user_id '12345', got %v", payload.CustomData["user_id"])
	}

	// Verify platform-specific payloads exist
	if payload.IOSPayload == nil {
		t.Error("iOS payload should be present for 'both' platform")
	}
	if payload.AndroidPayload == nil {
		t.Error("Android payload should be present for 'both' platform")
	}
}

func TestRichMobilePushService_IOSPayload(t *testing.T) {
	service := NewRichMobilePushService()
	service.platform = "ios"
	service.sound = "default"
	service.badge = 3
	service.priority = "high"
	service.category = "message"
	service.trackingID = "ios_track"
	service.deepLink = "myapp://chat"
	service.images = []string{"https://example.com/ios.jpg"}

	service.AddAction(MobilePushAction{
		ID:    "reply",
		Title: "Reply",
		Icon:  "reply",
	})

	req := NotificationRequest{
		Title: "iOS Test",
		Body:  "iOS notification body",
	}

	iosPayload := service.createIOSPayload(req)

	// Verify APS payload
	if iosPayload.APS == nil {
		t.Fatal("APS payload should not be nil")
	}

	alert, ok := iosPayload.APS.Alert.(*APNSAlert)
	if !ok {
		t.Fatal("APS alert should be APNSAlert type")
	}

	if alert.Title != "iOS Test" {
		t.Errorf("Expected alert title 'iOS Test', got '%s'", alert.Title)
	}
	if alert.Body != "iOS notification body" {
		t.Errorf("Expected alert body 'iOS notification body', got '%s'", alert.Body)
	}

	if iosPayload.APS.Sound != "default" {
		t.Errorf("Expected sound 'default', got '%v'", iosPayload.APS.Sound)
	}
	if iosPayload.APS.Badge != 3 {
		t.Errorf("Expected badge 3, got '%v'", iosPayload.APS.Badge)
	}
	if iosPayload.APS.Category != "message" {
		t.Errorf("Expected category 'message', got '%s'", iosPayload.APS.Category)
	}
	if iosPayload.APS.InterruptionLevel != "time-sensitive" {
		t.Errorf("Expected interruption level 'time-sensitive', got '%s'", iosPayload.APS.InterruptionLevel)
	}
	if iosPayload.APS.MutableContent != 1 {
		t.Errorf("Expected mutable content 1, got %d", iosPayload.APS.MutableContent)
	}

	// Verify custom data
	if iosPayload.Data["tracking_id"] != "ios_track" {
		t.Errorf("Expected tracking_id 'ios_track', got %v", iosPayload.Data["tracking_id"])
	}
	if iosPayload.Data["deep_link"] != "myapp://chat" {
		t.Errorf("Expected deep_link 'myapp://chat', got %v", iosPayload.Data["deep_link"])
	}

	// Verify images
	images, ok := iosPayload.Data["images"].([]string)
	if !ok || len(images) != 1 || images[0] != "https://example.com/ios.jpg" {
		t.Errorf("Expected images array with 'https://example.com/ios.jpg', got %v", iosPayload.Data["images"])
	}

	// Verify actions
	actions, ok := iosPayload.Data["actions"].([]MobilePushAction)
	if !ok || len(actions) != 1 || actions[0].ID != "reply" {
		t.Errorf("Expected actions array with reply action, got %v", iosPayload.Data["actions"])
	}
}

func TestRichMobilePushService_AndroidPayload(t *testing.T) {
	service := NewRichMobilePushService()
	service.platform = "android"
	service.icon = "notification_icon"
	service.color = "#FF0000"
	service.sound = "notification_sound"
	service.priority = "high"
	service.channelID = "alerts"
	service.collapseKey = "message_update"
	service.timeToLive = 2 * time.Hour
	service.trackingID = "android_track"
	service.deepLink = "myapp://message"
	service.images = []string{"https://example.com/android.jpg"}

	req := NotificationRequest{
		Title: "Android Test",
		Body:  "Android notification body",
	}

	androidPayload := service.createAndroidPayload(req)

	// Extract notification section
	notification, ok := androidPayload["notification"].(map[string]interface{})
	if !ok {
		t.Fatal("Notification section should be present")
	}

	if notification["title"] != "Android Test" {
		t.Errorf("Expected title 'Android Test', got %v", notification["title"])
	}
	if notification["body"] != "Android notification body" {
		t.Errorf("Expected body 'Android notification body', got %v", notification["body"])
	}
	if notification["icon"] != "notification_icon" {
		t.Errorf("Expected icon 'notification_icon', got %v", notification["icon"])
	}
	if notification["color"] != "#FF0000" {
		t.Errorf("Expected color '#FF0000', got %v", notification["color"])
	}
	if notification["sound"] != "notification_sound" {
		t.Errorf("Expected sound 'notification_sound', got %v", notification["sound"])
	}
	if notification["image"] != "https://example.com/android.jpg" {
		t.Errorf("Expected image 'https://example.com/android.jpg', got %v", notification["image"])
	}

	// Extract android section
	android, ok := androidPayload["android"].(map[string]interface{})
	if !ok {
		t.Fatal("Android section should be present")
	}

	if android["priority"] != "high" {
		t.Errorf("Expected priority 'high', got %v", android["priority"])
	}
	if android["ttl"] != "7200s" {
		t.Errorf("Expected TTL '7200s', got %v", android["ttl"])
	}
	if android["collapse_key"] != "message_update" {
		t.Errorf("Expected collapse_key 'message_update', got %v", android["collapse_key"])
	}

	// Verify notification config in android section
	androidNotification, ok := android["notification"].(map[string]interface{})
	if !ok {
		t.Fatal("Android notification config should be present")
	}
	if androidNotification["channel_id"] != "alerts" {
		t.Errorf("Expected channel_id 'alerts', got %v", androidNotification["channel_id"])
	}

	// Extract data section
	data, ok := androidPayload["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Data section should be present")
	}

	if data["tracking_id"] != "android_track" {
		t.Errorf("Expected tracking_id 'android_track', got %v", data["tracking_id"])
	}
	if data["deep_link"] != "myapp://message" {
		t.Errorf("Expected deep_link 'myapp://message', got %v", data["deep_link"])
	}
}

func TestRichMobilePushService_Send(t *testing.T) {
	service := NewRichMobilePushService()
	service.platform = "both"
	service.deviceTokens = []string{"ios_token", "android_token"}

	req := NotificationRequest{
		Title:      "Test Send",
		Body:       "Testing send functionality",
		NotifyType: NotifyTypeInfo,
	}

	ctx := context.Background()
	
	// This should not fail since we're using mock webhook requests
	err := service.Send(ctx, req)
	if err != nil {
		t.Errorf("Send should not fail with mock implementation: %v", err)
	}
}

func TestRichMobilePushService_LocalizedContent(t *testing.T) {
	service := NewRichMobilePushService()
	
	// Add localized content
	service.AddLocalizedContent("en", "Hello", "Welcome to our app")
	service.AddLocalizedContent("es", "Hola", "Bienvenido a nuestra aplicación")
	service.AddLocalizedContent("fr", "Bonjour", "Bienvenue dans notre application")

	if len(service.localizedTitle) != 3 {
		t.Errorf("Expected 3 localized titles, got %d", len(service.localizedTitle))
	}
	if len(service.localizedBody) != 3 {
		t.Errorf("Expected 3 localized bodies, got %d", len(service.localizedBody))
	}

	if service.localizedTitle["es"] != "Hola" {
		t.Errorf("Expected Spanish title 'Hola', got '%s'", service.localizedTitle["es"])
	}
	if service.localizedBody["fr"] != "Bienvenue dans notre application" {
		t.Errorf("Expected French body 'Bienvenue dans notre application', got '%s'", service.localizedBody["fr"])
	}
}

func TestRichMobilePushService_AdvancedFeatures(t *testing.T) {
	service := NewRichMobilePushService()
	
	// Test custom data
	service.AddCustomData("feature_flag", "premium")
	service.AddCustomData("experiment_id", 12345)
	
	if service.customData["feature_flag"] != "premium" {
		t.Errorf("Expected custom data feature_flag 'premium', got %v", service.customData["feature_flag"])
	}
	if service.customData["experiment_id"] != 12345 {
		t.Errorf("Expected custom data experiment_id 12345, got %v", service.customData["experiment_id"])
	}

	// Test user data
	service.AddUserData("user_segment", "vip")
	service.AddUserData("subscription_tier", "gold")
	
	if service.userData["user_segment"] != "vip" {
		t.Errorf("Expected user data user_segment 'vip', got '%s'", service.userData["user_segment"])
	}

	// Test scheduling
	scheduleTime := time.Now().Add(2 * time.Hour)
	service.SetSchedule(scheduleTime, "UTC")
	
	if !service.scheduleTime.Equal(scheduleTime) {
		t.Errorf("Expected schedule time %v, got %v", scheduleTime, service.scheduleTime)
	}
	if service.timeZone != "UTC" {
		t.Errorf("Expected timezone 'UTC', got '%s'", service.timeZone)
	}

	// Test adding actions
	action := MobilePushAction{
		ID:          "share",
		Title:       "Share",
		Icon:        "share_icon",
		URL:         "https://example.com/share",
		InputType:   "text",
		InputPrompt: "Add your comment",
	}
	service.AddAction(action)
	
	if len(service.actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(service.actions))
	}
	if service.actions[0].InputType != "text" {
		t.Errorf("Expected input type 'text', got '%s'", service.actions[0].InputType)
	}

	// Test adding images
	service.AddImage("https://example.com/hero.jpg")
	service.AddImage("https://example.com/thumbnail.jpg")
	
	if len(service.images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(service.images))
	}
	if service.images[1] != "https://example.com/thumbnail.jpg" {
		t.Errorf("Expected second image 'https://example.com/thumbnail.jpg', got '%s'", service.images[1])
	}
}

func TestBatchMobilePushService(t *testing.T) {
	service := NewBatchMobilePushService()

	if service.GetServiceID() != "batch-mobile-push" {
		t.Errorf("Expected service ID 'batch-mobile-push', got '%s'", service.GetServiceID())
	}

	if service.batchSize != 100 {
		t.Errorf("Expected batch size 100, got %d", service.batchSize)
	}
	if service.retryAttempts != 3 {
		t.Errorf("Expected retry attempts 3, got %d", service.retryAttempts)
	}

	// Test batch processing with mock tokens
	deviceTokens := make([]string, 250) // Create 250 tokens to test batching
	for i := 0; i < 250; i++ {
		deviceTokens[i] = fmt.Sprintf("token_%03d", i)
	}

	req := NotificationRequest{
		Title: "Batch Test",
		Body:  "Testing batch delivery",
	}

	ctx := context.Background()
	service.platform = "ios" // Set platform to avoid errors
	
	// This should process in 3 batches (100, 100, 50)
	err := service.SendBatch(ctx, req, deviceTokens)
	if err != nil {
		t.Errorf("Batch send should not fail with mock implementation: %v", err)
	}

	// Check delivery stats
	stats := service.GetDeliveryStats()
	if stats["successful_batches"] != 3 {
		t.Errorf("Expected 3 successful batches, got %d", stats["successful_batches"])
	}
	if stats["delivered_tokens"] != 250 {
		t.Errorf("Expected 250 delivered tokens, got %d", stats["delivered_tokens"])
	}
}

func TestRichMobilePushService_TestURL(t *testing.T) {
	service := NewRichMobilePushService()

	validURLs := []string{
		"rich-mobile-push://ios@token123",
		"rich-mobile-push://android@token1,token2",
		"rich-mobile-push://both@token123?sound=default&priority=high",
		"rich-mobile-push://ios@token123?action1=yes:Yes:check&reply=true",
	}

	for _, url := range validURLs {
		t.Run("Valid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", url, err)
			}
		})
	}

	invalidURLs := []string{
		"rich-mobile-push://invalid_platform@token123",
		"rich-mobile-push://ios@", // No tokens
		"http://example.com",       // Wrong scheme
	}

	for _, url := range invalidURLs {
		t.Run("Invalid_"+url, func(t *testing.T) {
			err := service.TestURL(url)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", url)
			}
		})
	}
}