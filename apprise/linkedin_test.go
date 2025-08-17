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

func TestLinkedInService_GetServiceID(t *testing.T) {
	service := NewLinkedInService()
	if service.GetServiceID() != "linkedin" {
		t.Errorf("Expected service ID 'linkedin', got '%s'", service.GetServiceID())
	}
}

func TestLinkedInService_GetDefaultPort(t *testing.T) {
	service := NewLinkedInService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestLinkedInService_SupportsAttachments(t *testing.T) {
	service := NewLinkedInService()
	if !service.SupportsAttachments() {
		t.Error("LinkedIn should support attachments")
	}
}

func TestLinkedInService_GetMaxBodyLength(t *testing.T) {
	service := NewLinkedInService()
	expected := 3000
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestLinkedInService_ParseURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectError      bool
		expectedToken    string
		expectedClientID string
		expectedSecret   string
		expectedUserID   string
		expectedPageID   string
		expectedWebhook  string
		expectedProxyKey string
	}{
		{
			name:          "Access token only",
			url:           "linkedin://access_token_123@api.linkedin.com/v2/ugcPosts",
			expectError:   false,
			expectedToken: "access_token_123",
			expectedUserID: "me",
		},
		{
			name:             "Full OAuth credentials",
			url:              "linkedin://client_id:client_secret:access_token@api.linkedin.com/v2/ugcPosts?user_id=user123&page_id=page456",
			expectError:      false,
			expectedClientID: "client_id",
			expectedSecret:   "client_secret",
			expectedToken:    "access_token",
			expectedUserID:   "user123",
			expectedPageID:   "page456",
		},
		{
			name:            "Webhook proxy mode",
			url:             "linkedin://proxy-key@webhook.example.com/linkedin?access_token=token123&client_id=client&client_secret=secret&user_id=user",
			expectError:     false,
			expectedWebhook: "https://webhook.example.com/linkedin",
			expectedProxyKey: "proxy-key",
			expectedToken:   "token123",
			expectedClientID: "client",
			expectedSecret:  "secret",
			expectedUserID:  "user",
		},
		{
			name:            "Webhook with page ID",
			url:             "linkedin://proxy@webhook.example.com/linkedin?access_token=token&client_id=id&client_secret=secret&user_id=user&page_id=page123",
			expectError:     false,
			expectedWebhook: "https://webhook.example.com/linkedin",
			expectedProxyKey: "proxy",
			expectedToken:   "token",
			expectedClientID: "id",
			expectedSecret:  "secret",
			expectedUserID:  "user",
			expectedPageID:  "page123",
		},
		{
			name:        "Invalid scheme",
			url:         "http://access_token@api.linkedin.com/v2/ugcPosts",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "linkedin://api.linkedin.com/v2/ugcPosts",
			expectError: true,
		},
		{
			name:        "Incomplete OAuth credentials",
			url:         "linkedin://client_id:access_token@api.linkedin.com/v2/ugcPosts",
			expectError: true,
		},
		{
			name:        "Webhook missing access token",
			url:         "linkedin://proxy@webhook.example.com/linkedin?client_id=id&client_secret=secret",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLinkedInService().(*LinkedInService)
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

			if service.accessToken != tt.expectedToken {
				t.Errorf("Expected access token '%s', got '%s'", tt.expectedToken, service.accessToken)
			}

			if service.clientID != tt.expectedClientID {
				t.Errorf("Expected client ID '%s', got '%s'", tt.expectedClientID, service.clientID)
			}

			if service.clientSecret != tt.expectedSecret {
				t.Errorf("Expected client secret '%s', got '%s'", tt.expectedSecret, service.clientSecret)
			}

			if service.userID != tt.expectedUserID {
				t.Errorf("Expected user ID '%s', got '%s'", tt.expectedUserID, service.userID)
			}

			if tt.expectedPageID != "" && service.pageID != tt.expectedPageID {
				t.Errorf("Expected page ID '%s', got '%s'", tt.expectedPageID, service.pageID)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedProxyKey != "" && service.proxyAPIKey != tt.expectedProxyKey {
				t.Errorf("Expected proxy key '%s', got '%s'", tt.expectedProxyKey, service.proxyAPIKey)
			}
		})
	}
}

func TestLinkedInService_TestURL(t *testing.T) {
	service := NewLinkedInService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid access token URL",
			url:         "linkedin://token123@api.linkedin.com/v2/ugcPosts",
			expectError: false,
		},
		{
			name:        "Valid OAuth URL",
			url:         "linkedin://client:secret:token@api.linkedin.com/v2/ugcPosts?user_id=user",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "linkedin://proxy@webhook.example.com/linkedin?access_token=token&client_id=id&client_secret=secret",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://token@api.linkedin.com/v2/ugcPosts",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "linkedin://api.linkedin.com/v2/ugcPosts",
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

func TestLinkedInService_SendWebhook(t *testing.T) {
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
		var payload LinkedInWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "linkedin" {
			t.Errorf("Expected service 'linkedin', got '%s'", payload.Service)
		}

		if payload.AccessToken != "test-access-token" {
			t.Errorf("Expected access token 'test-access-token', got '%s'", payload.AccessToken)
		}

		if payload.ClientID != "test-client-id" {
			t.Errorf("Expected client ID 'test-client-id', got '%s'", payload.ClientID)
		}

		if payload.UserID != "test-user" {
			t.Errorf("Expected user ID 'test-user', got '%s'", payload.UserID)
		}

		// Verify message content
		if !strings.Contains(payload.Message.Text.Text, "Professional Update") {
			t.Errorf("Expected message text to contain 'Professional Update', got '%s'", payload.Message.Text.Text)
		}

		if payload.Message.LifecycleState != "PUBLISHED" {
			t.Errorf("Expected lifecycle state 'PUBLISHED', got '%s'", payload.Message.LifecycleState)
		}

		if payload.Message.Distribution.FeedDistribution != "MAIN_FEED" {
			t.Errorf("Expected feed distribution 'MAIN_FEED', got '%s'", payload.Message.Distribution.FeedDistribution)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "urn:li:ugcPost:12345", "created": {"time": "2025-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewLinkedInService().(*LinkedInService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.accessToken = "test-access-token"
	service.clientID = "test-client-id"
	service.clientSecret = "test-client-secret"
	service.userID = "test-user"

	req := NotificationRequest{
		Title:      "Professional Update",
		Body:       "Sharing an important announcement with my network",
		NotifyType: NotifyTypeInfo,
		Tags:       []string{"announcement", "professional", "networking"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestLinkedInService_FormatLinkedInText(t *testing.T) {
	service := &LinkedInService{}

	tests := []struct {
		name     string
		req      NotificationRequest
		expected string
		contains []string
	}{
		{
			name: "Basic title and body",
			req: NotificationRequest{
				Title:      "Project Launch",
				Body:       "Excited to announce the launch of our new product",
				NotifyType: NotifyTypeInfo,
			},
			contains: []string{"📢 Update:", "Project Launch", "Excited to announce"},
		},
		{
			name: "Success notification",
			req: NotificationRequest{
				Title:      "Milestone Achieved",
				Body:       "We've reached 1M users!",
				NotifyType: NotifyTypeSuccess,
			},
			contains: []string{"✅ Success:", "Milestone Achieved", "1M users"},
		},
		{
			name: "Warning with professional context",
			req: NotificationRequest{
				Title:      "System Maintenance",
				Body:       "Scheduled maintenance window tonight",
				NotifyType: NotifyTypeWarning,
			},
			contains: []string{"⚠️ Important:", "System Maintenance", "tonight"},
		},
		{
			name: "With hashtags",
			req: NotificationRequest{
				Title:      "Tech Conference",
				Body:       "Great insights from today's sessions",
				NotifyType: NotifyTypeInfo,
				Tags:       []string{"technology", "conference", "AI"},
			},
			contains: []string{"📢 Update:", "Tech Conference", "#technology", "#conference", "#AI"},
		},
		{
			name: "Long text truncation",
			req: NotificationRequest{
				Title:      "Very Long Professional Update That Should Be Truncated",
				Body:       strings.Repeat("This is a very long professional update that will exceed LinkedIn's character limit and should be properly truncated to maintain readability and compliance with platform requirements. ", 20),
				NotifyType: NotifyTypeInfo,
			},
			contains: []string{"📢 Update:", "Very Long Professional", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.formatLinkedInText(tt.req)

			if tt.expected != "" {
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}

			for _, contain := range tt.contains {
				if !strings.Contains(result, contain) {
					t.Errorf("Expected result to contain '%s', got '%s'", contain, result)
				}
			}

			// Check character limit
			if len(result) > 3000 {
				t.Errorf("LinkedIn text exceeds 3000 characters: %d", len(result))
			}
		})
	}
}

func TestLinkedInService_BuildLinkedInMessage(t *testing.T) {
	service := &LinkedInService{
		userID: "test-user-123",
	}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("presentation data"), "presentation.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Q4 Results Presentation",
		Body:          "Sharing our outstanding Q4 performance with the team",
		NotifyType:    NotifyTypeSuccess,
		Tags:          []string{"Q4", "results", "performance"},
		URL:           "https://company.com/q4-results",
		AttachmentMgr: attachmentMgr,
	}

	message := service.buildLinkedInMessage(req)

	// Check basic message structure
	if message.LifecycleState != "PUBLISHED" {
		t.Errorf("Expected lifecycle state 'PUBLISHED', got '%s'", message.LifecycleState)
	}

	if message.Distribution.FeedDistribution != "MAIN_FEED" {
		t.Errorf("Expected feed distribution 'MAIN_FEED', got '%s'", message.Distribution.FeedDistribution)
	}

	if message.Visibility.Code != "PUBLIC" {
		t.Errorf("Expected visibility 'PUBLIC', got '%s'", message.Visibility.Code)
	}

	// Check author format
	expectedAuthor := "urn:li:person:test-user-123"
	if message.Author != expectedAuthor {
		t.Errorf("Expected author '%s', got '%s'", expectedAuthor, message.Author)
	}

	// Check text content
	if !strings.Contains(message.Text.Text, "Q4 Results Presentation") {
		t.Errorf("Expected message text to contain 'Q4 Results Presentation', got '%s'", message.Text.Text)
	}

	if !strings.Contains(message.Text.Text, "#Q4") {
		t.Errorf("Expected message text to contain '#Q4', got '%s'", message.Text.Text)
	}

	// Check rich content
	if message.Content == nil {
		t.Error("Expected rich content to be set")
	} else {
		if len(message.Content.ContentEntities) != 1 {
			t.Errorf("Expected 1 content entity, got %d", len(message.Content.ContentEntities))
		} else {
			entity := message.Content.ContentEntities[0]
			if entity.EntityLocation != req.URL {
				t.Errorf("Expected entity location '%s', got '%s'", req.URL, entity.EntityLocation)
			}
		}

		if message.Content.Title != req.Title {
			t.Errorf("Expected content title '%s', got '%s'", req.Title, message.Content.Title)
		}
	}

	// Check attachment reference in text
	if !strings.Contains(message.Text.Text, "attachment(s)") {
		t.Error("Expected attachment reference in message text")
	}
}

func TestLinkedInService_OrganizationPost(t *testing.T) {
	service := &LinkedInService{
		userID: "user123",
		pageID: "company456",
	}

	req := NotificationRequest{
		Title:      "Company Announcement",
		Body:       "Important update from our leadership team",
		NotifyType: NotifyTypeInfo,
	}

	message := service.buildLinkedInMessage(req)

	// Should use organization author format when pageID is set
	expectedAuthor := "urn:li:organization:company456"
	if message.Author != expectedAuthor {
		t.Errorf("Expected organization author '%s', got '%s'", expectedAuthor, message.Author)
	}
}

func TestLinkedInService_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		service     *LinkedInService
		expectError bool
	}{
		{
			name: "Valid access token",
			service: &LinkedInService{
				accessToken: "valid-token",
			},
			expectError: false,
		},
		{
			name: "Missing access token",
			service: &LinkedInService{
				accessToken: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.service.validateCredentials()
			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestLinkedInService_DirectAPICall(t *testing.T) {
	service := &LinkedInService{
		accessToken: "test-access-token",
		userID:      "me",
		client:      GetCloudHTTPClient("linkedin"),
	}

	message := LinkedInMessage{
		Author: "urn:li:person:me",
		Text: LinkedInTextContent{
			Text: "Test LinkedIn post from unit test",
		},
		LifecycleState: "PUBLISHED",
		Distribution: LinkedInDistribution{
			FeedDistribution: "MAIN_FEED",
		},
		Visibility: LinkedInVisibility{
			Code: "PUBLIC",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will fail with authentication error since we're using a fake token
	err := service.sendToLinkedInDirectly(ctx, message)
	if err == nil {
		t.Error("Expected error for fake access token")
	}

	// The error should be related to authentication or network
	if !strings.Contains(err.Error(), "linkedin api error") && !strings.Contains(err.Error(), "failed to send") {
		t.Errorf("Unexpected error type: %v", err)
	}
}