package apprise

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestMSTeamsService_GetServiceID(t *testing.T) {
	service := NewMSTeamsService()
	if service.GetServiceID() != "msteams" {
		t.Errorf("Expected service ID 'msteams', got '%s'", service.GetServiceID())
	}
}

func TestMSTeamsService_GetDefaultPort(t *testing.T) {
	service := NewMSTeamsService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestMSTeamsService_ParseURL(t *testing.T) {
	testCases := []struct {
		name            string
		url             string
		expectError     bool
		expectedVersion int
		expectedTeam    string
	}{
		{
			name:            "Modern format (version 2)",
			url:             "msteams://team_name/token_a/token_b/token_c",
			expectError:     false,
			expectedVersion: 2,
			expectedTeam:    "team_name",
		},
		{
			name:            "Version 3 format",
			url:             "msteams://team_name/token_a/token_b/token_c/token_d",
			expectError:     false,
			expectedVersion: 3,
			expectedTeam:    "team_name",
		},
		{
			name:            "Legacy format (version 1)",
			url:             "msteams:///token_a/token_b/token_c",
			expectError:     false,
			expectedVersion: 1,
		},
		{
			name:        "Invalid scheme",
			url:         "http://team_name/token_a/token_b/token_c",
			expectError: true,
		},
		{
			name:        "Insufficient tokens modern",
			url:         "msteams://team_name/token_a/token_b",
			expectError: true,
		},
		{
			name:        "Insufficient tokens legacy",
			url:         "msteams:///token_a/token_b",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewMSTeamsService().(*MSTeamsService)
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

			if service.version != tc.expectedVersion {
				t.Errorf("Expected version %d, got %d", tc.expectedVersion, service.version)
			}

			if tc.expectedTeam != "" && service.teamName != tc.expectedTeam {
				t.Errorf("Expected team name '%s', got '%s'", tc.expectedTeam, service.teamName)
			}
		})
	}
}

func TestMSTeamsService_ParseURL_QueryParams(t *testing.T) {
	testURL := "msteams://team_name/token_a/token_b/token_c?image=no"
	
	service := NewMSTeamsService().(*MSTeamsService)
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	err = service.ParseURL(parsedURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}

	if service.includeImage {
		t.Error("Expected includeImage to be false when image=no")
	}
}

func TestMSTeamsService_TestURL(t *testing.T) {
	service := NewMSTeamsService()

	validURLs := []string{
		"msteams://team_name/token_a/token_b/token_c",
		"msteams://team_name/token_a/token_b/token_c/token_d",
		"msteams:///token_a/token_b/token_c",
	}

	for _, testURL := range validURLs {
		t.Run("Valid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err != nil {
				t.Errorf("Expected valid URL %s to pass, got error: %v", testURL, err)
			}
		})
	}

	invalidURLs := []string{
		"http://team_name/token_a/token_b/token_c",
		"msteams://team_name/token_a/token_b", // Insufficient tokens
	}

	for _, testURL := range invalidURLs {
		t.Run("Invalid_"+testURL, func(t *testing.T) {
			err := service.TestURL(testURL)
			if err == nil {
				t.Errorf("Expected invalid URL %s to fail", testURL)
			}
		})
	}
}

func TestMSTeamsService_Properties(t *testing.T) {
	service := NewMSTeamsService()

	if service.SupportsAttachments() {
		t.Error("MSTeams service should not support attachments")
	}

	if service.GetMaxBodyLength() != 28000 {
		t.Errorf("Expected max body length 28000, got %d", service.GetMaxBodyLength())
	}
}

func TestMSTeamsService_BuildWebhookURL(t *testing.T) {
	testCases := []struct {
		name        string
		version     int
		teamName    string
		tokenA      string
		tokenB      string
		tokenC      string
		tokenD      string
		expectedURL string
	}{
		{
			name:        "Version 1",
			version:     1,
			tokenA:      "token_a",
			tokenB:      "token_b",
			tokenC:      "token_c",
			expectedURL: "https://outlook.office.com/webhook/token_a/IncomingWebhook/token_b/token_c",
		},
		{
			name:        "Version 2",
			version:     2,
			teamName:    "team_name",
			tokenA:      "token_a",
			tokenB:      "token_b",
			tokenC:      "token_c",
			expectedURL: "https://team_name.webhook.office.com/webhookb2/token_a/IncomingWebhook/token_b/token_c",
		},
		{
			name:        "Version 3",
			version:     3,
			teamName:    "team_name",
			tokenA:      "token_a",
			tokenB:      "token_b",
			tokenC:      "token_c",
			tokenD:      "token_d",
			expectedURL: "https://team_name.webhook.office.com/webhookb2/token_a/IncomingWebhook/token_b/token_c/token_d",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewMSTeamsService().(*MSTeamsService)
			service.version = tc.version
			service.teamName = tc.teamName
			service.tokenA = tc.tokenA
			service.tokenB = tc.tokenB
			service.tokenC = tc.tokenC
			service.tokenD = tc.tokenD

			actualURL := service.buildWebhookURL()
			if actualURL != tc.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tc.expectedURL, actualURL)
			}
		})
	}
}

func TestMSTeamsService_GetColorForNotifyType(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeSuccess, "00FF00"},
		{NotifyTypeWarning, "FFFF00"},
		{NotifyTypeError, "FF0000"},
		{NotifyTypeInfo, "0078D4"},
	}

	for _, test := range tests {
		result := service.getColorForNotifyType(test.notifyType)
		if result != test.expected {
			t.Errorf("Expected color '%s' for %v, got '%s'", test.expected, test.notifyType, result)
		}
	}
}

func TestMSTeamsService_GetImageForNotifyType(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)

	tests := []struct {
		notifyType NotifyType
		contains   string
	}{
		{NotifyTypeSuccess, "check_mark_button_3d.png"},
		{NotifyTypeWarning, "warning_3d.png"},
		{NotifyTypeError, "cross_mark_3d.png"},
		{NotifyTypeInfo, "information_3d.png"},
	}

	for _, test := range tests {
		result := service.getImageForNotifyType(test.notifyType)
		if result == "" || !containsString(result, test.contains) {
			t.Errorf("Expected image URL to contain '%s' for %v, got '%s'", test.contains, test.notifyType, result)
		}
	}
}

func TestMSTeamsService_CreateSummary(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)

	// Test with title
	summary := service.createSummary("Test Title", "Test Body")
	if summary != "Test Title" {
		t.Errorf("Expected summary 'Test Title', got '%s'", summary)
	}

	// Test without title (should use body)
	summary = service.createSummary("", "Short body")
	if summary != "Short body" {
		t.Errorf("Expected summary 'Short body', got '%s'", summary)
	}

	// Test long body truncation
	longBody := "This is a very long body message that exceeds the 100 character limit and should be truncated with ellipsis"
	summary = service.createSummary("", longBody)
	if len(summary) > 103 { // 100 + "..."
		t.Errorf("Expected summary to be truncated, got length %d: '%s'", len(summary), summary)
	}
	if !containsString(summary, "...") {
		t.Error("Expected truncated summary to contain '...'")
	}
}

func TestMSTeamsService_Send_InvalidConfig(t *testing.T) {
	service := NewMSTeamsService().(*MSTeamsService)
	
	// Service without proper configuration should fail
	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeInfo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err == nil {
		t.Error("Expected Send to fail with invalid configuration")
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}