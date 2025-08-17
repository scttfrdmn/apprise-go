package apprise

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGitLabService_GetServiceID(t *testing.T) {
	service := NewGitLabService()
	if service.GetServiceID() != "gitlab" {
		t.Errorf("Expected service ID 'gitlab', got '%s'", service.GetServiceID())
	}
}

func TestGitLabService_GetDefaultPort(t *testing.T) {
	service := NewGitLabService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestGitLabService_SupportsAttachments(t *testing.T) {
	service := NewGitLabService()
	if !service.SupportsAttachments() {
		t.Error("GitLab should support attachments (metadata)")
	}
}

func TestGitLabService_GetMaxBodyLength(t *testing.T) {
	service := NewGitLabService()
	expected := 16384
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestGitLabService_ParseURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedToken     string
		expectedProjectID string
		expectedServerURL string
		expectedEvents    []string
		expectedBranches  []string
		expectedLabels    []string
		expectedWebhook   string
		expectedProxyKey  string
	}{
		{
			name:              "Basic token with project ID",
			url:               "gitlab://token123@gitlab.com/456",
			expectError:       false,
			expectedToken:     "token123",
			expectedProjectID: "456",
			expectedServerURL: "https://gitlab.com",
			expectedEvents:    []string{"push", "merge_request", "issue", "pipeline"}, // defaults
		},
		{
			name:              "Self-hosted with custom events",
			url:               "gitlab://token456@gitlab.example.com/789?events=push,mr,pipeline",
			expectError:       false,
			expectedToken:     "token456",
			expectedProjectID: "789",
			expectedServerURL: "https://gitlab.example.com",
			expectedEvents:    []string{"push", "merge_request", "pipeline"}, // mr -> merge_request
		},
		{
			name:              "With branch and label filters",
			url:               "gitlab://token789@gitlab.com/123?events=all&branches=main,develop&labels=bug,feature",
			expectError:       false,
			expectedToken:     "token789",
			expectedProjectID: "123",
			expectedServerURL: "https://gitlab.com",
			expectedEvents:    []string{"push", "tag_push", "merge_request", "issue", "pipeline", "job", "release", "wiki", "deployment"},
			expectedBranches:  []string{"main", "develop"},
			expectedLabels:    []string{"bug", "feature"},
		},
		{
			name:              "Webhook proxy mode",
			url:               "gitlab://proxy-key@webhook.example.com/gitlab?token=gl_token&project_id=555&events=push,issue",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/gitlab",
			expectedProxyKey:  "proxy-key",
			expectedToken:     "gl_token",
			expectedProjectID: "555",
			expectedServerURL: "https://gitlab.com", // default
			expectedEvents:    []string{"push", "issue"},
		},
		{
			name:              "Webhook with custom server URL",
			url:               "gitlab://proxy@webhook.example.com/gitlab?token=token&project_id=777&server_url=https://gitlab.corp.com",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/gitlab",
			expectedProxyKey:  "proxy",
			expectedToken:     "token",
			expectedProjectID: "777",
			expectedServerURL: "https://gitlab.corp.com",
		},
		{
			name:              "Event aliases (ci, pr)",
			url:               "gitlab://token@gitlab.com/123?events=ci,pr",
			expectError:       false,
			expectedToken:     "token",
			expectedProjectID: "123",
			expectedServerURL: "https://gitlab.com",
			expectedEvents:    []string{"pipeline", "merge_request"}, // aliases resolved
		},
		{
			name:        "Invalid scheme",
			url:         "http://token@gitlab.com/123",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "gitlab://@gitlab.com/123",
			expectError: true,
		},
		{
			name:        "Missing project ID",
			url:         "gitlab://token@gitlab.com/",
			expectError: true,
		},
		{
			name:        "Webhook missing token",
			url:         "gitlab://proxy@webhook.example.com/gitlab?project_id=123",
			expectError: true,
		},
		{
			name:        "Webhook missing project ID",
			url:         "gitlab://proxy@webhook.example.com/gitlab?token=token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewGitLabService().(*GitLabService)
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Failed to parse URL: %v", err)
				}
				return
			}

			err = service.ParseURL(parsedURL)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if service.token != tt.expectedToken {
				t.Errorf("Expected token '%s', got '%s'", tt.expectedToken, service.token)
			}

			if service.projectID != tt.expectedProjectID {
				t.Errorf("Expected project ID '%s', got '%s'", tt.expectedProjectID, service.projectID)
			}

			if service.serverURL != tt.expectedServerURL {
				t.Errorf("Expected server URL '%s', got '%s'", tt.expectedServerURL, service.serverURL)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedProxyKey != "" && service.proxyAPIKey != tt.expectedProxyKey {
				t.Errorf("Expected proxy key '%s', got '%s'", tt.expectedProxyKey, service.proxyAPIKey)
			}

			if len(tt.expectedEvents) > 0 {
				if len(service.eventTypes) != len(tt.expectedEvents) {
					t.Errorf("Expected %d events, got %d", len(tt.expectedEvents), len(service.eventTypes))
				}
				for i, expectedEvent := range tt.expectedEvents {
					if i < len(service.eventTypes) && service.eventTypes[i] != expectedEvent {
						t.Errorf("Expected event[%d] '%s', got '%s'", i, expectedEvent, service.eventTypes[i])
					}
				}
			}

			if len(tt.expectedBranches) > 0 {
				if len(service.branches) != len(tt.expectedBranches) {
					t.Errorf("Expected %d branches, got %d", len(tt.expectedBranches), len(service.branches))
				}
				for i, expectedBranch := range tt.expectedBranches {
					if i < len(service.branches) && service.branches[i] != expectedBranch {
						t.Errorf("Expected branch[%d] '%s', got '%s'", i, expectedBranch, service.branches[i])
					}
				}
			}

			if len(tt.expectedLabels) > 0 {
				if len(service.labels) != len(tt.expectedLabels) {
					t.Errorf("Expected %d labels, got %d", len(tt.expectedLabels), len(service.labels))
				}
				for i, expectedLabel := range tt.expectedLabels {
					if i < len(service.labels) && service.labels[i] != expectedLabel {
						t.Errorf("Expected label[%d] '%s', got '%s'", i, expectedLabel, service.labels[i])
					}
				}
			}
		})
	}
}

func TestGitLabService_TestURL(t *testing.T) {
	service := NewGitLabService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid GitLab URL",
			url:         "gitlab://token@gitlab.com/123?events=push,mr",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "gitlab://proxy@webhook.example.com/gitlab?token=token&project_id=456",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://token@gitlab.com/123",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "gitlab://@gitlab.com/123",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestURL(tt.url)
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGitLabService_SendWebhook(t *testing.T) {
	// Create mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if !strings.Contains(r.Header.Get("User-Agent"), "Apprise-Go") {
			t.Errorf("Expected User-Agent to contain Apprise-Go, got %s", r.Header.Get("User-Agent"))
		}

		// Verify authentication
		if r.Header.Get("X-API-Key") != "test-proxy-key" {
			t.Errorf("Expected X-API-Key 'test-proxy-key', got '%s'", r.Header.Get("X-API-Key"))
		}

		// Parse and verify request body
		var payload GitLabWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "gitlab" {
			t.Errorf("Expected service 'gitlab', got '%s'", payload.Service)
		}

		if payload.ProjectID != "123" {
			t.Errorf("Expected project ID '123', got '%s'", payload.ProjectID)
		}

		if payload.Event == nil {
			t.Error("Expected event to be present")
		} else {
			if payload.Event.Title == "" {
				t.Error("Expected event title to be present")
			}

			if payload.Event.EventType != "notification" {
				t.Errorf("Expected event type 'notification', got '%s'", payload.Event.EventType)
			}

			if payload.Event.ObjectKind != "note" {
				t.Errorf("Expected object kind 'note', got '%s'", payload.Event.ObjectKind)
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "ok"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewGitLabService().(*GitLabService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.token = "gl-token"
	service.projectID = "123"

	// Test different notification types
	tests := []struct {
		name       string
		title      string
		body       string
		notifyType NotifyType
	}{
		{
			name:       "Info notification",
			title:      "Test GitLab Notification",
			body:       "This is a test notification for GitLab",
			notifyType: NotifyTypeInfo,
		},
		{
			name:       "Success notification",
			title:      "Pipeline Succeeded",
			body:       "CI/CD pipeline completed successfully",
			notifyType: NotifyTypeSuccess,
		},
		{
			name:       "Warning notification",
			title:      "Merge Request Review",
			body:       "Please review the pending merge request",
			notifyType: NotifyTypeWarning,
		},
		{
			name:       "Error notification",
			title:      "Pipeline Failed",
			body:       "CI/CD pipeline failed with errors",
			notifyType: NotifyTypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NotificationRequest{
				Title:      tt.title,
				Body:       tt.body,
				NotifyType: tt.notifyType,
				Tags:       []string{"ci", "pipeline"},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := service.Send(ctx, req)
			if err != nil {
				t.Fatalf("Send failed: %v", err)
			}
		})
	}
}

func TestGitLabService_SendAPI(t *testing.T) {
	// Create mock GitLab API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify authentication
		if r.Header.Get("Private-Token") != "test-token" {
			t.Errorf("Expected Private-Token 'test-token', got '%s'", r.Header.Get("Private-Token"))
		}

		// Verify URL path
		expectedPath := "/api/v4/projects/123/notes"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		// Parse and verify request body
		var noteData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&noteData); err != nil {
			t.Fatalf("Failed to decode note data: %v", err)
		}

		// Verify note body contains expected content
		body, ok := noteData["body"].(string)
		if !ok {
			t.Error("Expected note body to be a string")
		} else {
			if !strings.Contains(body, "API Test") {
				t.Errorf("Expected note body to contain 'API Test', got '%s'", body)
			}
			if !strings.Contains(body, "Direct API integration test") {
				t.Errorf("Expected note body to contain test description, got '%s'", body)
			}
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 1, "body": "Note created"}`))
	}))
	defer server.Close()

	// Configure service for direct API mode
	service := NewGitLabService().(*GitLabService)
	service.serverURL = server.URL
	service.token = "test-token"
	service.projectID = "123"

	req := NotificationRequest{
		Title:      "API Test",
		Body:       "Direct API integration test",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"api", "test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("API Send failed: %v", err)
	}
}

func TestGitLabService_CreateEvent(t *testing.T) {
	service := &GitLabService{
		projectID: "456",
	}

	req := NotificationRequest{
		Title:      "Test Event",
		Body:       "Test event body",
		NotifyType: NotifyTypeWarning,
		Tags:       []string{"test", "warning"},
		URL:        "https://example.com/webhook",
	}

	event := service.createEvent(req)

	if event.EventType != "notification" {
		t.Errorf("Expected event type 'notification', got '%s'", event.EventType)
	}

	if event.ObjectKind != "note" {
		t.Errorf("Expected object kind 'note', got '%s'", event.ObjectKind)
	}

	if event.Title != req.Title {
		t.Errorf("Expected title '%s', got '%s'", req.Title, event.Title)
	}

	if event.Description != req.Body {
		t.Errorf("Expected description '%s', got '%s'", req.Body, event.Description)
	}

	if event.State != "warning" {
		t.Errorf("Expected state 'warning', got '%s'", event.State)
	}

	if event.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", event.Action)
	}

	if event.ProjectID != 456 {
		t.Errorf("Expected project ID 456, got %d", event.ProjectID)
	}

	if event.URL != req.URL {
		t.Errorf("Expected URL '%s', got '%s'", req.URL, event.URL)
	}

	// Check labels
	if len(event.Labels) != len(req.Tags) {
		t.Errorf("Expected %d labels, got %d", len(req.Tags), len(event.Labels))
	}
}

func TestGitLabService_HelperMethods(t *testing.T) {
	service := &GitLabService{}

	// Test state mapping
	tests := []struct {
		notifyType    NotifyType
		expectedState string
	}{
		{NotifyTypeInfo, "pending"},
		{NotifyTypeSuccess, "success"},
		{NotifyTypeWarning, "warning"},
		{NotifyTypeError, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.notifyType.String(), func(t *testing.T) {
			if state := service.getStateForNotifyType(tt.notifyType); state != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, state)
			}
		})
	}
}

func TestGitLabService_FilterMethods(t *testing.T) {
	service := &GitLabService{
		eventTypes: []string{"push", "merge_request"},
		branches:   []string{"main", "develop"},
		labels:     []string{"bug", "feature"},
	}

	// Test event filter
	if !service.matchesEventFilter("push") {
		t.Error("Expected 'push' to match event filter")
	}
	if service.matchesEventFilter("issue") {
		t.Error("Expected 'issue' to not match event filter")
	}

	// Test branch filter
	if !service.matchesBranchFilter("main") {
		t.Error("Expected 'main' to match branch filter")
	}
	if service.matchesBranchFilter("feature-branch") {
		t.Error("Expected 'feature-branch' to not match branch filter")
	}

	// Test label filter
	if !service.matchesLabelFilter([]string{"bug", "urgent"}) {
		t.Error("Expected labels with 'bug' to match label filter")
	}
	if service.matchesLabelFilter([]string{"docs", "refactor"}) {
		t.Error("Expected labels without required labels to not match")
	}

	// Test empty filters (should match all)
	emptyService := &GitLabService{}
	if !emptyService.matchesEventFilter("anything") {
		t.Error("Expected empty event filter to match all")
	}
	if !emptyService.matchesBranchFilter("anything") {
		t.Error("Expected empty branch filter to match all")
	}
	if !emptyService.matchesLabelFilter([]string{"anything"}) {
		t.Error("Expected empty label filter to match all")
	}
}

func TestGitLabService_WithAttachments(t *testing.T) {
	service := &GitLabService{
		projectID: "123",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test data"), "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	event := service.createEvent(req)

	// Should have attachment info in labels
	found := false
	for _, label := range event.Labels {
		if strings.Contains(label, "attachments:1") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find attachment count in labels")
	}
}