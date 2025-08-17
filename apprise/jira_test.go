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

func TestJiraService_GetServiceID(t *testing.T) {
	service := NewJiraService()
	if service.GetServiceID() != "jira" {
		t.Errorf("Expected service ID 'jira', got '%s'", service.GetServiceID())
	}
}

func TestJiraService_GetDefaultPort(t *testing.T) {
	service := NewJiraService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestJiraService_SupportsAttachments(t *testing.T) {
	service := NewJiraService()
	if !service.SupportsAttachments() {
		t.Error("Jira should support attachments (metadata)")
	}
}

func TestJiraService_GetMaxBodyLength(t *testing.T) {
	service := NewJiraService()
	expected := 32768
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestJiraService_ParseURL(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		expectError        bool
		expectedUsername   string
		expectedAPIToken   string
		expectedServerURL  string
		expectedProjectKey string
		expectedIssueType  string
		expectedPriority   string
		expectedLabels     []string
		expectedComponents []string
		expectedWebhook    string
		expectedProxyKey   string
	}{
		{
			name:               "Basic Jira Cloud URL",
			url:                "jira://user@example.com:token123@company.atlassian.net/PROJ",
			expectError:        false,
			expectedUsername:   "user@example.com",
			expectedAPIToken:   "token123",
			expectedServerURL:  "https://company.atlassian.net",
			expectedProjectKey: "PROJ",
			expectedIssueType:  "Task", // default
			expectedPriority:   "Medium", // default
		},
		{
			name:               "Self-hosted with custom parameters",
			url:                "jira://admin:secret@jira.company.com/PROJECT?issue_type=Bug&priority=High&labels=urgent,api",
			expectError:        false,
			expectedUsername:   "admin",
			expectedAPIToken:   "secret",
			expectedServerURL:  "https://jira.company.com",
			expectedProjectKey: "PROJECT",
			expectedIssueType:  "Bug",
			expectedPriority:   "High",
			expectedLabels:     []string{"urgent", "api"},
		},
		{
			name:               "With components and labels",
			url:                "jira://user:token@company.atlassian.net/PROJ?issue_type=Story&priority=Low&labels=feature,backend&components=api,database",
			expectError:        false,
			expectedUsername:   "user",
			expectedAPIToken:   "token",
			expectedServerURL:  "https://company.atlassian.net",
			expectedProjectKey: "PROJ",
			expectedIssueType:  "Story",
			expectedPriority:   "Low",
			expectedLabels:     []string{"feature", "backend"},
			expectedComponents: []string{"api", "database"},
		},
		{
			name:               "Webhook proxy mode",
			url:                "jira://proxy-key@webhook.example.com/jira?username=user@company.com&token=jira_token&server_url=https://company.atlassian.net&project_key=PROJ",
			expectError:        false,
			expectedWebhook:    "https://webhook.example.com/jira",
			expectedProxyKey:   "proxy-key",
			expectedUsername:   "user@company.com",
			expectedAPIToken:   "jira_token",
			expectedServerURL:  "https://company.atlassian.net",
			expectedProjectKey: "PROJ",
			expectedIssueType:  "Task", // default
			expectedPriority:   "Medium", // default
		},
		{
			name:        "Invalid scheme",
			url:         "http://user:token@company.atlassian.net/PROJ",
			expectError: true,
		},
		{
			name:        "Missing username",
			url:         "jira://:token@company.atlassian.net/PROJ",
			expectError: true,
		},
		{
			name:        "Missing token",
			url:         "jira://user@company.atlassian.net/PROJ",
			expectError: true,
		},
		{
			name:        "Invalid priority",
			url:         "jira://user:token@company.atlassian.net/PROJ?priority=Invalid",
			expectError: true,
		},
		{
			name:        "Webhook missing username",
			url:         "jira://proxy@webhook.example.com/jira?token=token&server_url=https://company.atlassian.net",
			expectError: true,
		},
		{
			name:        "Webhook missing token",
			url:         "jira://proxy@webhook.example.com/jira?username=user&server_url=https://company.atlassian.net",
			expectError: true,
		},
		{
			name:        "Webhook missing server_url",
			url:         "jira://proxy@webhook.example.com/jira?username=user&token=token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewJiraService().(*JiraService)
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

			if service.username != tt.expectedUsername {
				t.Errorf("Expected username '%s', got '%s'", tt.expectedUsername, service.username)
			}

			if service.apiToken != tt.expectedAPIToken {
				t.Errorf("Expected API token '%s', got '%s'", tt.expectedAPIToken, service.apiToken)
			}

			if service.serverURL != tt.expectedServerURL {
				t.Errorf("Expected server URL '%s', got '%s'", tt.expectedServerURL, service.serverURL)
			}

			if service.projectKey != tt.expectedProjectKey {
				t.Errorf("Expected project key '%s', got '%s'", tt.expectedProjectKey, service.projectKey)
			}

			if service.issueType != tt.expectedIssueType {
				t.Errorf("Expected issue type '%s', got '%s'", tt.expectedIssueType, service.issueType)
			}

			if service.priority != tt.expectedPriority {
				t.Errorf("Expected priority '%s', got '%s'", tt.expectedPriority, service.priority)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedProxyKey != "" && service.proxyAPIKey != tt.expectedProxyKey {
				t.Errorf("Expected proxy key '%s', got '%s'", tt.expectedProxyKey, service.proxyAPIKey)
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

			if len(tt.expectedComponents) > 0 {
				if len(service.components) != len(tt.expectedComponents) {
					t.Errorf("Expected %d components, got %d", len(tt.expectedComponents), len(service.components))
				}
				for i, expectedComponent := range tt.expectedComponents {
					if i < len(service.components) && service.components[i] != expectedComponent {
						t.Errorf("Expected component[%d] '%s', got '%s'", i, expectedComponent, service.components[i])
					}
				}
			}
		})
	}
}

func TestJiraService_TestURL(t *testing.T) {
	service := NewJiraService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Jira Cloud URL",
			url:         "jira://user:token@company.atlassian.net/PROJ?issue_type=Bug",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "jira://proxy@webhook.example.com/jira?username=user&token=token&server_url=https://company.atlassian.net",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://user:token@company.atlassian.net/PROJ",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "jira://company.atlassian.net/PROJ",
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

func TestJiraService_SendWebhookCreateIssue(t *testing.T) {
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
		var payload JiraWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "jira" {
			t.Errorf("Expected service 'jira', got '%s'", payload.Service)
		}

		if payload.Action != "create_issue" {
			t.Errorf("Expected action 'create_issue', got '%s'", payload.Action)
		}

		if payload.Issue == nil {
			t.Error("Expected issue to be present")
		} else {
			if payload.Issue.Fields["summary"] != "Test Jira Issue" {
				t.Errorf("Expected summary 'Test Jira Issue', got '%v'", payload.Issue.Fields["summary"])
			}

			if payload.Issue.Fields["description"] != "This is a test issue for Jira integration" {
				t.Errorf("Expected description to match, got '%v'", payload.Issue.Fields["description"])
			}

			// Check issue type
			if issueType, ok := payload.Issue.Fields["issuetype"].(map[string]interface{}); ok {
				if issueType["name"] != "Task" {
					t.Errorf("Expected issue type 'Task', got '%v'", issueType["name"])
				}
			} else {
				t.Error("Expected issuetype to be present")
			}

			// Check project
			if project, ok := payload.Issue.Fields["project"].(map[string]interface{}); ok {
				if project["key"] != "TEST" {
					t.Errorf("Expected project key 'TEST', got '%v'", project["key"])
				}
			} else {
				t.Error("Expected project to be present")
			}
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "12345", "key": "TEST-123", "self": "https://company.atlassian.net/rest/api/3/issue/12345"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewJiraService().(*JiraService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.username = "test@example.com"
	service.apiToken = "test-token"
	service.serverURL = "https://company.atlassian.net"
	service.projectKey = "TEST"

	req := NotificationRequest{
		Title:      "Test Jira Issue",
		Body:       "This is a test issue for Jira integration",
		NotifyType: NotifyTypeError,
		Tags:       []string{"urgent", "bug"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestJiraService_SendWebhookAddComment(t *testing.T) {
	// Create mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse and verify request body
		var payload JiraWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify this is a comment action
		if payload.Action != "add_comment" {
			t.Errorf("Expected action 'add_comment', got '%s'", payload.Action)
		}

		if payload.Comment == nil {
			t.Error("Expected comment to be present")
		} else {
			if !strings.Contains(payload.Comment.Body, "Test Comment") {
				t.Errorf("Expected comment body to contain 'Test Comment', got '%s'", payload.Comment.Body)
			}
			if !strings.Contains(payload.Comment.Body, "This is a comment for existing issue") {
				t.Errorf("Expected comment body to contain test description, got '%s'", payload.Comment.Body)
			}
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "67890", "body": "Comment created"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewJiraService().(*JiraService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"

	// Use URL that indicates this is a comment (contains browse)
	req := NotificationRequest{
		Title:      "Test Comment",
		Body:       "This is a comment for existing issue",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"comment", "update"},
		URL:        "https://company.atlassian.net/browse/TEST-123",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestJiraService_SendAPICreateIssue(t *testing.T) {
	// Create mock Jira API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be present")
		}
		if username != "test@example.com" {
			t.Errorf("Expected username 'test@example.com', got '%s'", username)
		}
		if password != "test-api-token" {
			t.Errorf("Expected password 'test-api-token', got '%s'", password)
		}

		// Verify URL path
		expectedPath := "/rest/api/3/issue"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		// Parse and verify request body
		var issue JiraIssue
		if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
			t.Fatalf("Failed to decode issue data: %v", err)
		}

		// Verify issue fields
		if issue.Fields["summary"] != "API Test Issue" {
			t.Errorf("Expected summary 'API Test Issue', got '%v'", issue.Fields["summary"])
		}

		if issue.Fields["description"] != "Direct API integration test" {
			t.Errorf("Expected description 'Direct API integration test', got '%v'", issue.Fields["description"])
		}

		// Check labels contain apprise-go
		labelsRaw, exists := issue.Fields["labels"]
		if !exists {
			t.Error("Expected labels field to exist")
		} else {
			foundAppriseGo := false
			switch labels := labelsRaw.(type) {
			case []string:
				for _, label := range labels {
					if label == "apprise-go" {
						foundAppriseGo = true
						break
					}
				}
			case []interface{}:
				for _, label := range labels {
					if str, ok := label.(string); ok && str == "apprise-go" {
						foundAppriseGo = true
						break
					}
				}
			default:
				t.Errorf("Expected labels to be string array or interface array, got %T", labelsRaw)
			}
			if !foundAppriseGo {
				t.Error("Expected to find 'apprise-go' in labels")
			}
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "12345", "key": "TEST-123", "self": "https://company.atlassian.net/rest/api/3/issue/12345"}`))
	}))
	defer server.Close()

	// Configure service for direct API mode
	service := NewJiraService().(*JiraService)
	service.serverURL = server.URL
	service.username = "test@example.com"
	service.apiToken = "test-api-token"
	service.projectKey = "TEST"
	service.issueType = "Bug"
	service.priority = "High"

	req := NotificationRequest{
		Title:      "API Test Issue",
		Body:       "Direct API integration test",
		NotifyType: NotifyTypeError,
		Tags:       []string{"api", "test"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("API Send failed: %v", err)
	}
}

func TestJiraService_BuildIssue(t *testing.T) {
	service := &JiraService{
		projectKey: "TEST",
		issueType:  "Bug",
		priority:   "High",
		labels:     []string{"service:api"},
		components: []string{"backend", "database"},
	}

	req := NotificationRequest{
		Title:      "Test Issue",
		Body:       "Test issue description",
		NotifyType: NotifyTypeError,
		Tags:       []string{"urgent", "production"},
	}

	issue := service.buildIssue(req)

	if issue.Fields["summary"] != req.Title {
		t.Errorf("Expected summary '%s', got '%v'", req.Title, issue.Fields["summary"])
	}

	if issue.Fields["description"] != req.Body {
		t.Errorf("Expected description '%s', got '%v'", req.Body, issue.Fields["description"])
	}

	// Check issue type
	if issueType, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		if issueType["name"] != "Bug" {
			t.Errorf("Expected issue type 'Bug', got '%v'", issueType["name"])
		}
	} else {
		t.Error("Expected issuetype to be present")
	}

	// Check project
	if project, ok := issue.Fields["project"].(map[string]interface{}); ok {
		if project["key"] != "TEST" {
			t.Errorf("Expected project key 'TEST', got '%v'", project["key"])
		}
	} else {
		t.Error("Expected project to be present")
	}

	// Check priority
	if priority, ok := issue.Fields["priority"].(map[string]interface{}); ok {
		if priority["name"] != "High" {
			t.Errorf("Expected priority 'High', got '%v'", priority["name"])
		}
	} else {
		t.Error("Expected priority to be present")
	}

	// Check labels (should include default + request tags + apprise-go)
	labels, ok := issue.Fields["labels"].([]string)
	if !ok {
		t.Error("Expected labels to be string array")
	} else {
		expectedLabels := []string{"service:api", "urgent", "production", "apprise-go"}
		if len(labels) != len(expectedLabels) {
			t.Errorf("Expected %d labels, got %d", len(expectedLabels), len(labels))
		}
	}

	// Check components
	components, ok := issue.Fields["components"].([]map[string]interface{})
	if !ok {
		t.Error("Expected components to be array of maps")
	} else {
		if len(components) != 2 {
			t.Errorf("Expected 2 components, got %d", len(components))
		}
		if components[0]["name"] != "backend" {
			t.Errorf("Expected first component 'backend', got '%v'", components[0]["name"])
		}
	}
}

func TestJiraService_BuildComment(t *testing.T) {
	service := &JiraService{}

	req := NotificationRequest{
		Title:      "Test Comment",
		Body:       "Test comment body",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"comment", "update"},
	}

	comment := service.buildComment(req)

	expectedStart := "*Test Comment*\n\nTest comment body"
	if !strings.HasPrefix(comment.Body, expectedStart) {
		t.Errorf("Expected comment to start with '%s', got '%s'", expectedStart, comment.Body)
	}

	if !strings.Contains(comment.Body, "Tags: comment, update") {
		t.Errorf("Expected comment to contain tags, got '%s'", comment.Body)
	}

	if !strings.Contains(comment.Body, "_Created by Apprise-Go_") {
		t.Errorf("Expected comment to contain source attribution, got '%s'", comment.Body)
	}
}

func TestJiraService_HelperMethods(t *testing.T) {
	service := &JiraService{}

	// Test priority validation
	validPriorities := []string{"Highest", "High", "Medium", "Low", "Lowest"}
	for _, priority := range validPriorities {
		if !service.isValidPriority(priority) {
			t.Errorf("Expected priority '%s' to be valid", priority)
		}
	}

	invalidPriorities := []string{"Invalid", "Critical", ""}
	for _, priority := range invalidPriorities {
		if service.isValidPriority(priority) {
			t.Errorf("Expected priority '%s' to be invalid", priority)
		}
	}

	// Test issue key extraction
	testCases := []struct {
		url         string
		expectedKey string
	}{
		{"https://company.atlassian.net/browse/PROJ-123", "PROJ-123"},
		{"https://jira.company.com/browse/TEST-456", "TEST-456"},
		{"https://company.atlassian.net/browse/PROJ-123?focusedCommentId=12345", "PROJ-123"},
		{"https://company.atlassian.net/browse/PROJ-123#comment-67890", "PROJ-123"},
		{"https://company.atlassian.net/projects/PROJ", ""},
		{"invalid-url", ""},
	}

	for _, tc := range testCases {
		result := service.extractIssueKeyFromURL(tc.url)
		if result != tc.expectedKey {
			t.Errorf("Expected key '%s' from URL '%s', got '%s'", tc.expectedKey, tc.url, result)
		}
	}
}

func TestJiraService_WithAttachments(t *testing.T) {
	service := &JiraService{
		projectKey: "TEST",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test data"), "test.txt", "text/plain")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	err = attachmentMgr.AddData([]byte("image data"), "test.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Test With Attachments",
		Body:          "This has attachments",
		NotifyType:    NotifyTypeInfo,
		AttachmentMgr: attachmentMgr,
	}

	// Test issue with attachments
	issue := service.buildIssue(req)
	description := issue.Fields["description"].(string)

	if !strings.Contains(description, "Attachments (2):") {
		t.Error("Expected issue description to contain attachment count")
	}

	if !strings.Contains(description, "- test.txt (text/plain)") {
		t.Error("Expected issue description to contain first attachment info")
	}

	if !strings.Contains(description, "- test.jpg (image/jpeg)") {
		t.Error("Expected issue description to contain second attachment info")
	}

	// Test comment with attachments
	comment := service.buildComment(req)

	if !strings.Contains(comment.Body, "Attachments (2):") {
		t.Error("Expected comment body to contain attachment count")
	}

	if !strings.Contains(comment.Body, "- test.txt (text/plain)") {
		t.Error("Expected comment body to contain attachment info")
	}
}