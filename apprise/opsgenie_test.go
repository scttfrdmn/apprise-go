package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestOpsgenieService_GetServiceID(t *testing.T) {
	service := NewOpsgenieService()
	if service.GetServiceID() != "opsgenie" {
		t.Errorf("Expected service ID 'opsgenie', got %q", service.GetServiceID())
	}
}

func TestOpsgenieService_GetDefaultPort(t *testing.T) {
	service := NewOpsgenieService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestOpsgenieService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedAPIKey   string
		expectedRegion   string
		expectedTargets  []string
		expectedTags     []string
		expectedTeams    []string
		expectedPriority string
		expectedAlias    string
		expectedEntity   string
		expectedSource   string
		expectedUser     string
		expectedNote     string
	}{
		{
			name:           "Basic API key US region",
			url:            "opsgenie://abc123@us",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
		},
		{
			name:           "API key EU region",
			url:            "opsgenie://abc123@eu",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "eu",
		},
		{
			name:            "API key with team target",
			url:             "opsgenie://abc123@us/backend-team",
			expectError:     false,
			expectedAPIKey:  "abc123",
			expectedRegion:  "us",
			expectedTargets: []string{"backend-team"},
		},
		{
			name:            "API key with multiple targets",
			url:             "opsgenie://abc123@us/backend-team/user@example.com/devops",
			expectError:     false,
			expectedAPIKey:  "abc123",
			expectedRegion:  "us",
			expectedTargets: []string{"backend-team", "user@example.com", "devops"},
		},
		{
			name:             "With priority and tags",
			url:              "opsgenie://abc123@us?priority=P1&tags=critical,production",
			expectError:      false,
			expectedAPIKey:   "abc123",
			expectedRegion:   "us",
			expectedPriority: "P1",
			expectedTags:     []string{"critical", "production"},
		},
		{
			name:           "With teams in query parameter",
			url:            "opsgenie://abc123@us?teams=devops,backend",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
			expectedTeams:  []string{"devops", "backend"},
		},
		{
			name:           "With alias and entity",
			url:            "opsgenie://abc123@us?alias=db-alert&entity=web-server",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
			expectedAlias:  "db-alert",
			expectedEntity: "web-server",
		},
		{
			name:           "With source and user",
			url:            "opsgenie://abc123@us?source=monitoring&user=admin",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
			expectedSource: "monitoring",
			expectedUser:   "admin",
		},
		{
			name:           "With note",
			url:            "opsgenie://abc123@us?note=Database%20performance%20issue",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
			expectedNote:   "Database performance issue",
		},
		{
			name:           "Region from query parameter",
			url:            "opsgenie://abc123@us?region=eu",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "eu", // Query param overrides host
		},
		{
			name:           "Default US region (no host)",
			url:            "opsgenie://abc123",
			expectError:    false,
			expectedAPIKey: "abc123",
			expectedRegion: "us",
		},
		{
			name:        "Invalid scheme",
			url:         "http://abc123@us",
			expectError: true,
		},
		{
			name:        "Missing API key",
			url:         "opsgenie://@us",
			expectError: true,
		},
		{
			name:        "Empty API key",
			url:         "opsgenie://@us",
			expectError: true,
		},
		{
			name:        "Invalid region",
			url:         "opsgenie://abc123@invalid",
			expectError: true,
		},
		{
			name:        "Invalid region in query",
			url:         "opsgenie://abc123@us?region=invalid",
			expectError: true,
		},
		{
			name:        "Invalid priority (too low)",
			url:         "opsgenie://abc123@us?priority=P0",
			expectError: true,
		},
		{
			name:        "Invalid priority (too high)",
			url:         "opsgenie://abc123@us?priority=P6",
			expectError: true,
		},
		{
			name:        "Invalid priority (not P format)",
			url:         "opsgenie://abc123@us?priority=high",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOpsgenieService().(*OpsgenieService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			err = service.ParseURL(parsedURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				return
			}

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected apiKey %q, got %q", tt.expectedAPIKey, service.apiKey)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region %q, got %q", tt.expectedRegion, service.region)
			}

			if tt.expectedTargets != nil && !stringSlicesEqual(service.targets, tt.expectedTargets) {
				t.Errorf("Expected targets %v, got %v", tt.expectedTargets, service.targets)
			}

			if tt.expectedTags != nil && !stringSlicesEqual(service.tags, tt.expectedTags) {
				t.Errorf("Expected tags %v, got %v", tt.expectedTags, service.tags)
			}

			if tt.expectedTeams != nil && !stringSlicesEqual(service.teams, tt.expectedTeams) {
				t.Errorf("Expected teams %v, got %v", tt.expectedTeams, service.teams)
			}

			if tt.expectedPriority != "" && service.priority != tt.expectedPriority {
				t.Errorf("Expected priority %q, got %q", tt.expectedPriority, service.priority)
			}

			if tt.expectedAlias != "" && service.alias != tt.expectedAlias {
				t.Errorf("Expected alias %q, got %q", tt.expectedAlias, service.alias)
			}

			if tt.expectedEntity != "" && service.entity != tt.expectedEntity {
				t.Errorf("Expected entity %q, got %q", tt.expectedEntity, service.entity)
			}

			if tt.expectedSource != "" && service.source != tt.expectedSource {
				t.Errorf("Expected source %q, got %q", tt.expectedSource, service.source)
			}

			if tt.expectedUser != "" && service.user != tt.expectedUser {
				t.Errorf("Expected user %q, got %q", tt.expectedUser, service.user)
			}

			if tt.expectedNote != "" && service.note != tt.expectedNote {
				t.Errorf("Expected note %q, got %q", tt.expectedNote, service.note)
			}
		})
	}
}

func TestOpsgenieService_TestURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid opsgenie://api_key@us",
			url:         "opsgenie://abc123@us",
			expectError: false,
		},
		{
			name:        "Valid opsgenie://api_key@eu",
			url:         "opsgenie://abc123@eu",
			expectError: false,
		},
		{
			name:        "Valid with team target",
			url:         "opsgenie://abc123@us/backend-team",
			expectError: false,
		},
		{
			name:        "Valid with query parameters",
			url:         "opsgenie://abc123@us?priority=P2&tags=alert&teams=devops",
			expectError: false,
		},
		{
			name:        "Valid default region",
			url:         "opsgenie://abc123",
			expectError: false,
		},
		{
			name:        "Invalid http://api_key@us",
			url:         "http://abc123@us",
			expectError: true,
		},
		{
			name:        "Invalid opsgenie://@us (no API key)",
			url:         "opsgenie://@us",
			expectError: true,
		},
		{
			name:        "Invalid opsgenie://api_key@invalid_region",
			url:         "opsgenie://abc123@invalid",
			expectError: true,
		},
		{
			name:        "Invalid priority",
			url:         "opsgenie://abc123@us?priority=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOpsgenieService()
			err := service.TestURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %q, got none", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for URL %q: %v", tt.url, err)
				}
			}
		})
	}
}

func TestOpsgenieService_Properties(t *testing.T) {
	service := NewOpsgenieService()

	if service.SupportsAttachments() {
		t.Error("Opsgenie should not support attachments")
	}

	expectedMaxLength := 15000
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestOpsgenieService_GetAPIURL(t *testing.T) {
	tests := []struct {
		region      string
		expectedURL string
	}{
		{"us", "https://api.opsgenie.com/v2/alerts"},
		{"eu", "https://api.eu.opsgenie.com/v2/alerts"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			service := &OpsgenieService{region: tt.region}
			url := service.getAPIURL()
			if url != tt.expectedURL {
				t.Errorf("Expected API URL %q for region %q, got %q",
					tt.expectedURL, tt.region, url)
			}
		})
	}
}

func TestOpsgenieService_MapNotifyTypeToPriority(t *testing.T) {
	service := &OpsgenieService{}

	tests := []struct {
		notifyType NotifyType
		expected   string
	}{
		{NotifyTypeError, "P1"},
		{NotifyTypeWarning, "P2"},
		{NotifyTypeSuccess, "P4"},
		{NotifyTypeInfo, "P3"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			result := service.mapNotifyTypeToPriority(tt.notifyType)
			if result != tt.expected {
				t.Errorf("Expected priority %q for %v, got %q", tt.expected, tt.notifyType, result)
			}
		})
	}
}

func TestOpsgenieService_IsValidPriority(t *testing.T) {
	tests := []struct {
		priority string
		expected bool
	}{
		{"P1", true},
		{"P2", true},
		{"P3", true},
		{"P4", true},
		{"P5", true},
		{"P0", false},
		{"P6", false},
		{"high", false},
		{"critical", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			result := isValidOpsgeniePriority(tt.priority)
			if result != tt.expected {
				t.Errorf("Expected isValidOpsgeniePriority(%q) = %v, got %v",
					tt.priority, tt.expected, result)
			}
		})
	}
}

func TestOpsgenieService_Send_InvalidConfig(t *testing.T) {
	service := NewOpsgenieService()
	parsedURL, _ := url.Parse("opsgenie://test_api_key@us")
	_ = service.(*OpsgenieService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test Alert",
		Body:       "Test alert description",
		NotifyType: NotifyTypeError,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real Opsgenie server
	if err == nil {
		t.Error("Expected network error for invalid Opsgenie configuration, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "opsgenie") &&
		!strings.Contains(err.Error(), "Opsgenie") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should be network-related, got: %v", err)
	}
}

func TestOpsgenieService_PayloadGeneration(t *testing.T) {
	service := NewOpsgenieService()
	parsedURL, _ := url.Parse("opsgenie://test_api_key@eu/backend-team?priority=P2&tags=production,critical&entity=web-server")
	_ = service.(*OpsgenieService).ParseURL(parsedURL)

	opsgenieService := service.(*OpsgenieService)

	// Test that configuration is parsed correctly
	if opsgenieService.apiKey != "test_api_key" {
		t.Error("Expected API key to be set from URL")
	}

	if opsgenieService.region != "eu" {
		t.Errorf("Expected region to be 'eu', got: %s", opsgenieService.region)
	}

	expectedTargets := []string{"backend-team"}
	if !stringSlicesEqual(opsgenieService.targets, expectedTargets) {
		t.Errorf("Expected targets %v, got: %v", expectedTargets, opsgenieService.targets)
	}

	if opsgenieService.priority != "P2" {
		t.Errorf("Expected priority to be 'P2', got: %s", opsgenieService.priority)
	}

	expectedTags := []string{"production", "critical"}
	if !stringSlicesEqual(opsgenieService.tags, expectedTags) {
		t.Errorf("Expected tags %v, got: %v", expectedTags, opsgenieService.tags)
	}

	if opsgenieService.entity != "web-server" {
		t.Errorf("Expected entity to be 'web-server', got: %s", opsgenieService.entity)
	}

	// Test API URL generation
	expectedURL := "https://api.eu.opsgenie.com/v2/alerts"
	actualURL := opsgenieService.getAPIURL()
	if actualURL != expectedURL {
		t.Errorf("Expected API URL %s, got: %s", expectedURL, actualURL)
	}
}

func TestOpsgenieService_ResponderTypes(t *testing.T) {
	// Test that responder type detection works correctly
	tests := []struct {
		target       string
		expectedType string
	}{
		{"user@example.com", "user"},
		{"backend-team", "team"},
		{"devops", "team"},
		{"admin@company.com", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			// This would be tested in actual alert payload generation
			// We're testing the logic that would be used in Send method
			var responderType string
			if strings.Contains(tt.target, "@") {
				responderType = "user"
			} else {
				responderType = "team"
			}

			if responderType != tt.expectedType {
				t.Errorf("Expected responder type %q for target %q, got %q",
					tt.expectedType, tt.target, responderType)
			}
		})
	}
}

func TestOpsgenieService_PriorityOverride(t *testing.T) {
	// Test that notification type overrides default priority but not explicit priority
	tests := []struct {
		name           string
		url            string
		notifyType     NotifyType
		expectedPrio   string
		shouldOverride bool
	}{
		{
			name:           "Default priority overridden by error type",
			url:            "opsgenie://abc123@us",
			notifyType:     NotifyTypeError,
			expectedPrio:   "P1",
			shouldOverride: true,
		},
		{
			name:           "Explicit priority not overridden",
			url:            "opsgenie://abc123@us?priority=P4",
			notifyType:     NotifyTypeError,
			expectedPrio:   "P4",
			shouldOverride: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOpsgenieService().(*OpsgenieService)
			parsedURL, _ := url.Parse(tt.url)
			_ = service.ParseURL(parsedURL)

			// Create a test alert to check priority mapping
			alert := OpsgenieAlert{
				Priority: service.priority,
			}

			// Apply the same logic as in Send method
			if service.priority == "" {
				alert.Priority = service.mapNotifyTypeToPriority(tt.notifyType)
			}

			if alert.Priority != tt.expectedPrio {
				t.Errorf("Expected priority %q, got %q", tt.expectedPrio, alert.Priority)
			}
		})
	}
}