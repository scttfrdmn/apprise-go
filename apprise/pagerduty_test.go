package apprise

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestPagerDutyService_GetServiceID(t *testing.T) {
	service := NewPagerDutyService()
	if service.GetServiceID() != "pagerduty" {
		t.Errorf("Expected service ID 'pagerduty', got %q", service.GetServiceID())
	}
}

func TestPagerDutyService_GetDefaultPort(t *testing.T) {
	service := NewPagerDutyService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestPagerDutyService_ParseURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedKey       string
		expectedRegion    string
		expectedSource    string
		expectedComponent string
	}{
		{
			name:           "Basic integration key",
			url:            "pagerduty://abc123def456",
			expectError:    false,
			expectedKey:    "abc123def456",
			expectedRegion: "us",
		},
		{
			name:           "Integration key with US region",
			url:            "pagerduty://abc123def456@us",
			expectError:    false,
			expectedKey:    "abc123def456",
			expectedRegion: "us",
		},
		{
			name:           "Integration key with EU region",
			url:            "pagerduty://abc123def456@eu",
			expectError:    false,
			expectedKey:    "abc123def456",
			expectedRegion: "eu",
		},
		{
			name:              "Integration key with query parameters",
			url:               "pagerduty://abc123def456?region=eu&source=monitoring&component=api",
			expectError:       false,
			expectedKey:       "abc123def456",
			expectedRegion:    "eu",
			expectedSource:    "monitoring",
			expectedComponent: "api",
		},
		{
			name:              "Integration key with source and component",
			url:               "pagerduty://abc123def456?source=server-01&component=database",
			expectError:       false,
			expectedKey:       "abc123def456",
			expectedRegion:    "us",
			expectedSource:    "server-01",
			expectedComponent: "database",
		},
		{
			name:        "Invalid scheme",
			url:         "http://abc123def456",
			expectError: true,
		},
		{
			name:        "Missing integration key",
			url:         "pagerduty://",
			expectError: true,
		},
		{
			name:        "Invalid region",
			url:         "pagerduty://abc123def456?region=invalid",
			expectError: true,
		},
		{
			name:        "Empty integration key",
			url:         "pagerduty://@us",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPagerDutyService().(*PagerDutyService)
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

			if service.integrationKey != tt.expectedKey {
				t.Errorf("Expected integration key %q, got %q", tt.expectedKey, service.integrationKey)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region %q, got %q", tt.expectedRegion, service.region)
			}

			if tt.expectedSource != "" && service.source != tt.expectedSource {
				t.Errorf("Expected source %q, got %q", tt.expectedSource, service.source)
			}

			if tt.expectedComponent != "" && service.component != tt.expectedComponent {
				t.Errorf("Expected component %q, got %q", tt.expectedComponent, service.component)
			}
		})
	}
}

func TestPagerDutyService_TestURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid pagerduty://integration_key",
			url:         "pagerduty://abc123def456",
			expectError: false,
		},
		{
			name:        "Valid pagerduty://integration_key@us",
			url:         "pagerduty://abc123def456@us",
			expectError: false,
		},
		{
			name:        "Valid pagerduty://integration_key@eu",
			url:         "pagerduty://abc123def456@eu",
			expectError: false,
		},
		{
			name:        "Valid pagerduty://integration_key?region=eu&source=monitoring",
			url:         "pagerduty://abc123def456?region=eu&source=monitoring&component=api",
			expectError: false,
		},
		{
			name:        "Invalid http://integration_key",
			url:         "http://abc123def456",
			expectError: true,
		},
		{
			name:        "Invalid pagerduty://",
			url:         "pagerduty://",
			expectError: true,
		},
		{
			name:        "Invalid pagerduty://key?region=invalid",
			url:         "pagerduty://abc123def456?region=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPagerDutyService()
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

func TestPagerDutyService_Properties(t *testing.T) {
	service := NewPagerDutyService()

	if service.SupportsAttachments() {
		t.Error("PagerDuty should not support attachments")
	}

	expectedMaxLength := 1024
	if service.GetMaxBodyLength() != expectedMaxLength {
		t.Errorf("Expected max body length %d, got %d",
			expectedMaxLength, service.GetMaxBodyLength())
	}
}

func TestPagerDutyService_GetAPIURL(t *testing.T) {
	tests := []struct {
		region      string
		expectedURL string
	}{
		{
			region:      "us",
			expectedURL: "https://events.pagerduty.com/v2/enqueue",
		},
		{
			region:      "eu",
			expectedURL: "https://events.eu.pagerduty.com/v2/enqueue",
		},
		{
			region:      "", // Should default to US
			expectedURL: "https://events.pagerduty.com/v2/enqueue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			service := &PagerDutyService{region: tt.region}
			url := service.getAPIURL()

			if url != tt.expectedURL {
				t.Errorf("Expected URL %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

func TestPagerDutyService_MapSeverity(t *testing.T) {
	service := &PagerDutyService{}

	tests := []struct {
		notifyType       NotifyType
		expectedSeverity string
	}{
		{NotifyTypeSuccess, "info"},
		{NotifyTypeInfo, "info"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			severity := service.mapSeverity(tt.notifyType)
			if severity != tt.expectedSeverity {
				t.Errorf("Expected severity %q for %s, got %q",
					tt.expectedSeverity, tt.notifyType.String(), severity)
			}
		})
	}
}

func TestPagerDutyService_FormatSummary(t *testing.T) {
	service := &PagerDutyService{}

	tests := []struct {
		name            string
		title           string
		body            string
		expectedSummary string
	}{
		{
			name:            "Title and body",
			title:           "Alert Title",
			body:            "Alert body content",
			expectedSummary: "Alert Title",
		},
		{
			name:            "Body only",
			title:           "",
			body:            "Alert body content",
			expectedSummary: "Alert body content",
		},
		{
			name:            "Long body truncation",
			title:           "",
			body:            strings.Repeat("a", 1030),
			expectedSummary: strings.Repeat("a", 1021) + "...",
		},
		{
			name:            "Empty title and body",
			title:           "",
			body:            "",
			expectedSummary: "Alert from Apprise-Go",
		},
		{
			name:            "Title only",
			title:           "Just a title",
			body:            "",
			expectedSummary: "Just a title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := service.formatSummary(tt.title, tt.body)
			if summary != tt.expectedSummary {
				t.Errorf("Expected summary %q, got %q", tt.expectedSummary, summary)
			}
		})
	}
}

func TestPagerDutyService_GetSource(t *testing.T) {
	tests := []struct {
		name             string
		configuredSource string
		expectedSource   string
	}{
		{
			name:             "Custom source",
			configuredSource: "monitoring-server",
			expectedSource:   "monitoring-server",
		},
		{
			name:             "Default source",
			configuredSource: "",
			expectedSource:   "apprise-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &PagerDutyService{source: tt.configuredSource}
			source := service.getSource()

			if source != tt.expectedSource {
				t.Errorf("Expected source %q, got %q", tt.expectedSource, source)
			}
		})
	}
}

func TestPagerDutyService_Send_InvalidConfig(t *testing.T) {
	service := NewPagerDutyService()
	parsedURL, _ := url.Parse("pagerduty://test_integration_key")
	_ = service.(*PagerDutyService).ParseURL(parsedURL)

	req := NotificationRequest{
		Title:      "Test",
		Body:       "Test message",
		NotifyType: NotifyTypeError,
	}

	// Test that Send method exists and can be called
	// (It will fail with network error, but should not panic)
	err := service.Send(context.Background(), req)

	// We expect a network error since we're not hitting a real PagerDuty endpoint
	if err == nil {
		t.Error("Expected network error for invalid PagerDuty integration key, got none")
	}

	// Check that error message makes sense (network or API error)
	if !strings.Contains(err.Error(), "pagerduty") &&
		!strings.Contains(err.Error(), "PagerDuty") &&
		!strings.Contains(err.Error(), "connect") &&
		!strings.Contains(err.Error(), "no such host") &&
		!strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should be network-related, got: %v", err)
	}
}

func TestPagerDutyService_PayloadGeneration(t *testing.T) {
	service := NewPagerDutyService()
	parsedURL, _ := url.Parse("pagerduty://test_key?source=test-source&component=test-component&group=test-group")
	_ = service.(*PagerDutyService).ParseURL(parsedURL)

	pagerDutyService := service.(*PagerDutyService)

	// Test that payload generation doesn't panic
	if pagerDutyService.integrationKey != "test_key" {
		t.Error("Expected integration key to be set from URL")
	}

	if pagerDutyService.source != "test-source" {
		t.Error("Expected source to be parsed correctly")
	}

	if pagerDutyService.component != "test-component" {
		t.Error("Expected component to be parsed correctly")
	}

	if pagerDutyService.group != "test-group" {
		t.Error("Expected group to be parsed correctly")
	}
}
