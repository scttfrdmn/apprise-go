package apprise

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestAdvancedDesktopService_ParseURL(t *testing.T) {
	testCases := []struct {
		name            string
		url             string
		expectError     bool
		expectedActions int
		expectedTimeout time.Duration
		expectedUrgent  bool
		expectedReply   bool
	}{
		{
			name:            "Basic advanced desktop URL",
			url:             "desktop-advanced://",
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "With actions",
			url:             "desktop-advanced://?action1=yes:Yes:http://example.com&action2=no:No",
			expectedActions: 2,
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "With timeout and urgency",
			url:             "desktop-advanced://?timeout=30&urgent=true",
			expectedTimeout: 30 * time.Second,
			expectedUrgent:  true,
		},
		{
			name:            "With reply button",
			url:             "desktop-advanced://?reply=true&category=message",
			expectedTimeout: 15 * time.Second,
			expectedReply:   true,
		},
		{
			name:            "Full featured notification",
			url:             "desktop-advanced://?action1=approve:Approve&action2=deny:Deny&timeout=60&urgent=true&reply=true&subtitle=Test&group=test-group",
			expectedActions: 2,
			expectedTimeout: 60 * time.Second,
			expectedUrgent:  true,
			expectedReply:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewAdvancedDesktopService()
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

			if len(service.actions) != tc.expectedActions {
				t.Errorf("Expected %d actions, got %d", tc.expectedActions, len(service.actions))
			}

			if service.timeout != tc.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tc.expectedTimeout, service.timeout)
			}

			if service.urgent != tc.expectedUrgent {
				t.Errorf("Expected urgent %v, got %v", tc.expectedUrgent, service.urgent)
			}

			if service.replyButton != tc.expectedReply {
				t.Errorf("Expected reply button %v, got %v", tc.expectedReply, service.replyButton)
			}
		})
	}
}

func TestAdvancedDesktopService_Actions(t *testing.T) {
	service := NewAdvancedDesktopService()
	
	// Parse URL with actions
	parsedURL, err := url.Parse("desktop-advanced://?action1=approve:Approve:http://example.com/approve&action2=deny:Deny")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	// Verify actions were parsed correctly
	if len(service.actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(service.actions))
	}
	
	// Check first action
	action1 := service.actions[0]
	if action1.ID != "approve" {
		t.Errorf("Expected action1 ID 'approve', got '%s'", action1.ID)
	}
	if action1.Title != "Approve" {
		t.Errorf("Expected action1 title 'Approve', got '%s'", action1.Title)
	}
	if action1.URL != "http://example.com/approve" {
		t.Errorf("Expected action1 URL 'http://example.com/approve', got '%s'", action1.URL)
	}
	
	// Check second action
	action2 := service.actions[1]
	if action2.ID != "deny" {
		t.Errorf("Expected action2 ID 'deny', got '%s'", action2.ID)
	}
	if action2.Title != "Deny" {
		t.Errorf("Expected action2 title 'Deny', got '%s'", action2.Title)
	}
	if action2.URL != "" {
		t.Errorf("Expected action2 URL to be empty, got '%s'", action2.URL)
	}
}

func TestAdvancedDesktopService_Properties(t *testing.T) {
	service := NewAdvancedDesktopService()

	if service.GetServiceID() != "desktop-advanced" {
		t.Errorf("Expected service ID 'desktop-advanced', got '%s'", service.GetServiceID())
	}

	// Test attachment support (advanced desktop supports images)
	if !service.SupportsAttachments() {
		t.Error("Advanced desktop notifications should support attachments")
	}

	// Test increased body length
	if service.GetMaxBodyLength() != 500 {
		t.Errorf("Expected max body length 500, got %d", service.GetMaxBodyLength())
	}
}

func TestAdvancedDesktopService_WindowsToastScript(t *testing.T) {
	service := NewAdvancedDesktopService()
	service.platform = "windows"
	
	// Add test actions
	service.actions = []NotificationAction{
		{ID: "yes", Title: "Yes", URL: "http://example.com/yes"},
		{ID: "no", Title: "No"},
	}
	service.replyButton = true
	service.timeout = 30 * time.Second

	req := NotificationRequest{
		Title: "Test Title",
		Body:  "Test Body",
	}

	script := service.generateWindowsToastScript(req)

	// Verify script contains expected elements
	if !containsStringDesktop(script, "Test Title") {
		t.Error("Toast script should contain title")
	}
	if !containsStringDesktop(script, "Test Body") {
		t.Error("Toast script should contain body")
	}
	if !containsStringDesktop(script, `content="Yes"`) {
		t.Error("Toast script should contain first action")
	}
	if !containsStringDesktop(script, `content="No"`) {
		t.Error("Toast script should contain second action")
	}
	if !containsStringDesktop(script, "Type your reply") {
		t.Error("Toast script should contain reply input")
	}
	if !containsStringDesktop(script, "30") {
		t.Error("Toast script should contain timeout")
	}
}

func TestInteractiveDesktopService(t *testing.T) {
	service := NewInteractiveDesktopService()

	if service.GetServiceID() != "desktop-interactive" {
		t.Errorf("Expected service ID 'desktop-interactive', got '%s'", service.GetServiceID())
	}

	// Test action callback
	var callbackResult *NotificationResult
	service.SetActionCallback(func(result NotificationResult) {
		callbackResult = &result
	})

	// Simulate interaction
	req := NotificationRequest{Title: "Test", Body: "Test"}
	service.handleInteraction("test-action", req)

	// Give callback time to execute
	time.Sleep(100 * time.Millisecond)

	if callbackResult == nil {
		t.Error("Action callback should have been called")
	} else if !callbackResult.Clicked {
		t.Error("Result should indicate notification was clicked")
	}

	// Test result channel
	select {
	case result := <-service.GetResultChannel():
		if !result.Clicked {
			t.Error("Channel result should indicate notification was clicked")
		}
	case <-time.After(time.Second):
		t.Error("Should have received result on channel")
	}
}

func TestPersistentDesktopService(t *testing.T) {
	// Create temporary history file
	tempFile := "/tmp/test-apprise-notifications.json"
	defer os.Remove(tempFile)

	service := NewPersistentDesktopService()
	service.historyFile = tempFile

	// Test notification storage
	req := NotificationRequest{
		Title: "Test Notification",
		Body:  "Test Body",
	}

	// Mock the send operation (avoid actually sending notifications in tests)
	originalPlatform := service.platform
	service.platform = "unsupported" // This will cause fallback to basic service
	
	// The send will fail, but we can still test the persistence logic
	service.notifications["test-id"] = req
	err := service.saveHistory()
	if err != nil {
		t.Errorf("Failed to save history: %v", err)
	}

	// Test loading history
	service.notifications = make(map[string]NotificationRequest) // Clear
	err = service.loadHistory()
	if err != nil {
		t.Errorf("Failed to load history: %v", err)
	}

	history := service.GetNotificationHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 notification in history, got %d", len(history))
	}

	if notification, exists := history["test-id"]; !exists {
		t.Error("Test notification should exist in history")
	} else if notification.Title != "Test Notification" {
		t.Errorf("Expected title 'Test Notification', got '%s'", notification.Title)
	}

	// Restore original platform
	service.platform = originalPlatform
}

func TestNotificationAction_JSON(t *testing.T) {
	action := NotificationAction{
		ID:    "test-action",
		Title: "Test Action",
		URL:   "http://example.com",
	}

	// Test JSON marshaling
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("Failed to marshal action: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledAction NotificationAction
	err = json.Unmarshal(data, &unmarshaledAction)
	if err != nil {
		t.Fatalf("Failed to unmarshal action: %v", err)
	}

	if unmarshaledAction.ID != action.ID {
		t.Errorf("Expected ID '%s', got '%s'", action.ID, unmarshaledAction.ID)
	}
	if unmarshaledAction.Title != action.Title {
		t.Errorf("Expected title '%s', got '%s'", action.Title, unmarshaledAction.Title)
	}
	if unmarshaledAction.URL != action.URL {
		t.Errorf("Expected URL '%s', got '%s'", action.URL, unmarshaledAction.URL)
	}
}

func TestNotificationResult_JSON(t *testing.T) {
	result := NotificationResult{
		ActionID:  "test-action",
		ReplyText: "Test reply",
		Clicked:   true,
		Dismissed: false,
		Timestamp: time.Now(),
		Metadata:  map[string]string{"key": "value"},
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledResult NotificationResult
	err = json.Unmarshal(data, &unmarshaledResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if unmarshaledResult.ActionID != result.ActionID {
		t.Errorf("Expected ActionID '%s', got '%s'", result.ActionID, unmarshaledResult.ActionID)
	}
	if unmarshaledResult.ReplyText != result.ReplyText {
		t.Errorf("Expected ReplyText '%s', got '%s'", result.ReplyText, unmarshaledResult.ReplyText)
	}
	if unmarshaledResult.Clicked != result.Clicked {
		t.Errorf("Expected Clicked %v, got %v", result.Clicked, unmarshaledResult.Clicked)
	}
}

func TestEscapeXML(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"<test>", "&lt;test&gt;"},
		{"A&B", "A&amp;B"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"'apostrophe'", "&apos;apostrophe&apos;"},
		{"<>&\"'", "&lt;&gt;&amp;&quot;&apos;"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeXML(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsStringDesktop(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 && 
		   len(str) >= len(substr) &&
		   findSubstringDesktop(str, substr) != -1
}

func findSubstringDesktop(str, substr string) int {
	if len(substr) > len(str) {
		return -1
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if str[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Integration test for advanced desktop notifications (conditional on environment)
func TestAdvancedDesktopService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("APPRISE_TEST_DESKTOP") != "1" {
		t.Skip("Set APPRISE_TEST_DESKTOP=1 to run desktop notification integration tests")
	}

	service := NewAdvancedDesktopService()
	
	// Configure advanced features
	parsedURL, err := url.Parse("desktop-advanced://?action1=ok:OK&action2=cancel:Cancel&timeout=10&subtitle=Integration Test")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to configure service: %v", err)
	}

	req := NotificationRequest{
		Title:      "Advanced Desktop Test",
		Body:       "This is a test of advanced desktop notifications with actions",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = service.Send(ctx, req)
	if err != nil {
		t.Logf("Advanced desktop notification integration test failed (expected on some systems): %v", err)
	} else {
		t.Log("Advanced desktop notification sent successfully - you should see a rich notification with action buttons")
	}
}

// Test interactive desktop service with real callbacks
func TestInteractiveDesktopService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("APPRISE_TEST_INTERACTIVE") != "1" {
		t.Skip("Set APPRISE_TEST_INTERACTIVE=1 to run interactive desktop notification tests")
	}

	service := NewInteractiveDesktopService()
	
	// Set up callback to track interactions
	interactionReceived := make(chan NotificationResult, 1)
	service.SetActionCallback(func(result NotificationResult) {
		interactionReceived <- result
	})

	// Configure with actions
	parsedURL, err := url.Parse("desktop-interactive://?action1=approve:Approve&action2=deny:Deny&reply=true")
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	
	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to configure service: %v", err)
	}

	req := NotificationRequest{
		Title:      "Interactive Test",
		Body:       "Please interact with this notification to test callback functionality",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = service.Send(ctx, req)
	if err != nil {
		t.Logf("Interactive desktop notification failed (expected on some systems): %v", err)
		return
	}

	t.Log("Interactive notification sent. Waiting for user interaction...")
	
	// Wait for interaction or timeout
	select {
	case result := <-interactionReceived:
		t.Logf("Received interaction: ActionID=%s, Clicked=%v, Dismissed=%v, ReplyText=%s", 
			result.ActionID, result.Clicked, result.Dismissed, result.ReplyText)
	case <-time.After(30 * time.Second):
		t.Log("No interaction received within timeout (this is expected if no user interaction occurs)")
	}
}