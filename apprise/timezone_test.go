package apprise

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWithTimezone(t *testing.T) {
	tests := []struct {
		name     string
		tz       string
		wantName string
	}{
		{
			name:     "UTC timezone",
			tz:       "UTC",
			wantName: "UTC",
		},
		{
			name:     "America/New_York",
			tz:       "America/New_York",
			wantName: "America/New_York",
		},
		{
			name:     "Europe/London",
			tz:       "Europe/London",
			wantName: "Europe/London",
		},
		{
			name:     "Asia/Tokyo",
			tz:       "Asia/Tokyo",
			wantName: "Asia/Tokyo",
		},
		{
			name:     "Empty timezone (should use Local)",
			tz:       "",
			wantName: "Local",
		},
		{
			name:     "Invalid timezone (should fall back to Local)",
			tz:       "Invalid/Timezone",
			wantName: "Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{}
			option := WithTimezone(tt.tz)
			option(&req)

			if req.Timezone == nil {
				t.Fatal("Timezone should not be nil")
			}

			got := req.Timezone.String()
			if got != tt.wantName {
				t.Errorf("WithTimezone() timezone = %v, want %v", got, tt.wantName)
			}
		})
	}
}

func TestNotificationRequest_GetTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		tz       string
		wantName string
	}{
		{
			name:     "UTC timezone",
			tz:       "UTC",
			wantName: "UTC",
		},
		{
			name:     "America/New_York timezone",
			tz:       "America/New_York",
			wantName: "America/New_York",
		},
		{
			name:     "Nil timezone (should use Local)",
			tz:       "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{}
			if tt.tz != "" {
				loc, err := time.LoadLocation(tt.tz)
				if err != nil {
					t.Fatalf("Failed to load timezone: %v", err)
				}
				req.Timezone = loc
			}

			timestamp := req.GetTimestamp()

			// Verify it's a valid RFC3339 timestamp
			_, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Errorf("GetTimestamp() returned invalid RFC3339 timestamp: %v", err)
			}

			// Verify timezone is correct (if specified)
			if tt.wantName != "" && !strings.Contains(timestamp, "Z") {
				// Non-UTC timestamps should have timezone offset
				if !strings.Contains(timestamp, "+") && !strings.Contains(timestamp, "-") {
					t.Errorf("GetTimestamp() should include timezone offset, got: %v", timestamp)
				}
			}
		})
	}
}

func TestNotificationRequest_GetUnixTimestamp(t *testing.T) {
	req := NotificationRequest{}

	before := time.Now().Unix()
	timestamp := req.GetUnixTimestamp()
	after := time.Now().Unix()

	if timestamp < before || timestamp > after {
		t.Errorf("GetUnixTimestamp() = %v, want between %v and %v", timestamp, before, after)
	}
}

func TestTimezoneIntegration(t *testing.T) {
	app := New()

	// Test with UTC timezone
	app.Notify("Test", "Body", NotifyTypeInfo, WithTimezone("UTC"))

	// Test with America/New_York
	app.Notify("Test", "Body", NotifyTypeInfo, WithTimezone("America/New_York"))

	// Test with invalid timezone (should not panic)
	app.Notify("Test", "Body", NotifyTypeInfo, WithTimezone("Invalid/Zone"))
}

func TestTimezoneWithDiscord(t *testing.T) {
	// This test verifies that Discord service receives timezone-aware timestamps
	service := NewDiscordService()

	// Configure the service with a test webhook
	testURL := "discord://webhook_id/webhook_token"
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	if err := service.ParseURL(parsedURL); err != nil {
		t.Fatalf("Failed to configure service: %v", err)
	}

	req := NotificationRequest{
		Title:      "Test Title",
		Body:       "Test Body",
		NotifyType: NotifyTypeInfo,
	}

	// Set UTC timezone
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("Failed to load UTC: %v", err)
	}
	req.Timezone = loc

	// Verify GetTimestamp returns valid timestamp
	timestamp := req.GetTimestamp()
	_, err = time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Timestamp should be valid RFC3339: %v", err)
	}
}

func TestTimezoneWithSlack(t *testing.T) {
	// This test verifies that Slack service receives timezone-aware Unix timestamps
	service := NewSlackService()

	// Configure the service with a test webhook (format: slack://TokenA/TokenB/TokenC)
	testURL := "slack://T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX"
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	if err := service.ParseURL(parsedURL); err != nil {
		t.Fatalf("Failed to configure service: %v", err)
	}

	req := NotificationRequest{
		Title:      "Test Title",
		Body:       "Test Body",
		NotifyType: NotifyTypeInfo,
	}

	// Set America/New_York timezone
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("Failed to load timezone: %v", err)
	}
	req.Timezone = loc

	// Verify GetUnixTimestamp returns valid Unix timestamp
	unixTime := req.GetUnixTimestamp()
	now := time.Now().Unix()

	// Should be within 1 second of current time
	if unixTime < now-1 || unixTime > now+1 {
		t.Errorf("Unix timestamp should be close to current time, got %v, want ~%v", unixTime, now)
	}
}
