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

func TestTwitterService_GetServiceID(t *testing.T) {
	service := NewTwitterService()
	if service.GetServiceID() != "twitter" {
		t.Errorf("Expected service ID 'twitter', got '%s'", service.GetServiceID())
	}
}

func TestTwitterService_GetDefaultPort(t *testing.T) {
	service := NewTwitterService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestTwitterService_SupportsAttachments(t *testing.T) {
	service := NewTwitterService()
	if !service.SupportsAttachments() {
		t.Error("Twitter should support attachments")
	}
}

func TestTwitterService_GetMaxBodyLength(t *testing.T) {
	service := NewTwitterService()
	expected := 280
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestTwitterService_ParseURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectError       bool
		expectedAPIKey    string
		expectedAPISecret string
		expectedToken     string
		expectedSecret    string
		expectedBearer    string
		expectedWebhook   string
		expectedProxyKey  string
	}{
		{
			name:              "OAuth 1.0a credentials",
			url:               "twitter://api_key:api_secret:access_token:access_secret@api.twitter.com/1.1/statuses/update.json",
			expectError:       false,
			expectedAPIKey:    "api_key",
			expectedAPISecret: "api_secret",
			expectedToken:     "access_token",
			expectedSecret:    "access_secret",
		},
		{
			name:           "Bearer token authentication",
			url:            "twitter://bearer_token@api.twitter.com/2/tweets",
			expectError:    false,
			expectedBearer: "bearer_token",
		},
		{
			name:              "Webhook proxy with OAuth 1.0a",
			url:               "twitter://proxy-key@webhook.example.com/twitter?api_key=key&api_secret=secret&access_token=token&access_secret=secret",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/twitter",
			expectedProxyKey:  "proxy-key",
			expectedAPIKey:    "key",
			expectedAPISecret: "secret",
			expectedToken:     "token",
			expectedSecret:    "secret",
		},
		{
			name:              "Webhook proxy with bearer token",
			url:               "twitter://proxy@webhook.example.com/twitter?api_key=key&api_secret=secret&bearer_token=bearer123",
			expectError:       false,
			expectedWebhook:   "https://webhook.example.com/twitter",
			expectedProxyKey:  "proxy",
			expectedAPIKey:    "key",
			expectedAPISecret: "secret",
			expectedBearer:    "bearer123",
		},
		{
			name:        "Invalid scheme",
			url:         "http://api_key:api_secret:token:secret@api.twitter.com/1.1/statuses/update.json",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "twitter://api.twitter.com/2/tweets",
			expectError: true,
		},
		{
			name:        "Incomplete OAuth credentials",
			url:         "twitter://api_key@api.twitter.com/1.1/statuses/update.json",
			expectError: true,
		},
		{
			name:        "Webhook missing API key",
			url:         "twitter://proxy@webhook.example.com/twitter?api_secret=secret&access_token=token&access_secret=secret",
			expectError: true,
		},
		{
			name:        "Webhook missing API secret",
			url:         "twitter://proxy@webhook.example.com/twitter?api_key=key&access_token=token&access_secret=secret",
			expectError: true,
		},
		{
			name:        "Webhook with no auth method",
			url:         "twitter://proxy@webhook.example.com/twitter?api_key=key&api_secret=secret",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTwitterService().(*TwitterService)
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

			if service.apiKey != tt.expectedAPIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.expectedAPIKey, service.apiKey)
			}

			if service.apiKeySecret != tt.expectedAPISecret {
				t.Errorf("Expected API secret '%s', got '%s'", tt.expectedAPISecret, service.apiKeySecret)
			}

			if service.accessToken != tt.expectedToken {
				t.Errorf("Expected access token '%s', got '%s'", tt.expectedToken, service.accessToken)
			}

			if service.accessSecret != tt.expectedSecret {
				t.Errorf("Expected access secret '%s', got '%s'", tt.expectedSecret, service.accessSecret)
			}

			if service.bearerToken != tt.expectedBearer {
				t.Errorf("Expected bearer token '%s', got '%s'", tt.expectedBearer, service.bearerToken)
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

func TestTwitterService_TestURL(t *testing.T) {
	service := NewTwitterService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid OAuth URL",
			url:         "twitter://key:secret:token:secret@api.twitter.com/1.1/statuses/update.json",
			expectError: false,
		},
		{
			name:        "Valid Bearer token URL",
			url:         "twitter://bearer@api.twitter.com/2/tweets",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "twitter://proxy@webhook.example.com/twitter?api_key=key&api_secret=secret&access_token=token&access_secret=secret",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://key:secret:token:secret@api.twitter.com/1.1/statuses/update.json",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "twitter://api.twitter.com/2/tweets",
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

func TestTwitterService_SendWebhook(t *testing.T) {
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
		var payload TwitterWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "twitter" {
			t.Errorf("Expected service 'twitter', got '%s'", payload.Service)
		}

		if payload.APIKey != "test-api-key" {
			t.Errorf("Expected API key 'test-api-key', got '%s'", payload.APIKey)
		}

		if payload.APIKeySecret != "test-api-secret" {
			t.Errorf("Expected API secret 'test-api-secret', got '%s'", payload.APIKeySecret)
		}

		if payload.AccessToken != "test-access-token" {
			t.Errorf("Expected access token 'test-access-token', got '%s'", payload.AccessToken)
		}

		if payload.AccessSecret != "test-access-secret" {
			t.Errorf("Expected access secret 'test-access-secret', got '%s'", payload.AccessSecret)
		}

		if payload.MessageType != "tweet" {
			t.Errorf("Expected message type 'tweet', got '%s'", payload.MessageType)
		}

		// Verify message content
		if !strings.Contains(payload.Message.Text, "Critical System Alert") {
			t.Errorf("Expected tweet text to contain 'Critical System Alert', got '%s'", payload.Message.Text)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": {"id": "1234567890", "text": "Tweet posted successfully"}}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewTwitterService().(*TwitterService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.apiKey = "test-api-key"
	service.apiKeySecret = "test-api-secret"
	service.accessToken = "test-access-token"
	service.accessSecret = "test-access-secret"

	req := NotificationRequest{
		Title:      "Critical System Alert",
		Body:       "Database connection failed",
		NotifyType: NotifyTypeError,
		Tags:       []string{"critical", "database", "system"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestTwitterService_FormatTweetText(t *testing.T) {
	service := &TwitterService{}

	tests := []struct {
		name     string
		req      NotificationRequest
		expected string
		contains []string
	}{
		{
			name: "Basic title and body",
			req: NotificationRequest{
				Title:      "Test Alert",
				Body:       "This is a test message",
				NotifyType: NotifyTypeInfo,
			},
			expected: "ℹ️ Test Alert: This is a test message",
		},
		{
			name: "Error with emoji",
			req: NotificationRequest{
				Title:      "System Error",
				Body:       "Critical failure detected",
				NotifyType: NotifyTypeError,
			},
			expected: "🚨 System Error: Critical failure detected",
		},
		{
			name: "Warning with tags",
			req: NotificationRequest{
				Title:      "High CPU Usage",
				Body:       "CPU usage exceeded 90%",
				NotifyType: NotifyTypeWarning,
				Tags:       []string{"cpu", "performance"},
			},
			contains: []string{"⚠️", "High CPU Usage", "#cpu", "#performance"},
		},
		{
			name: "Success notification",
			req: NotificationRequest{
				Title:      "Deployment Complete",
				Body:       "Version 2.1.0 deployed successfully",
				NotifyType: NotifyTypeSuccess,
				Tags:       []string{"deployment", "v2.1.0"},
			},
			contains: []string{"✅", "Deployment Complete", "#deployment"},
		},
		{
			name: "With URL",
			req: NotificationRequest{
				Title:      "Build Status",
				Body:       "Build completed",
				NotifyType: NotifyTypeInfo,
				URL:        "https://ci.example.com/build/123",
			},
			contains: []string{"ℹ️", "Build Status", "https://ci.example.com/build/123"},
		},
		{
			name: "Long text truncation",
			req: NotificationRequest{
				Title:      "Very Long Title That Exceeds Twitter Character Limit",
				Body:       strings.Repeat("This is a very long message that will definitely exceed the Twitter character limit and should be truncated appropriately. ", 5),
				NotifyType: NotifyTypeInfo,
			},
			contains: []string{"ℹ️", "Very Long Title", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.formatTweetText(tt.req)

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
			if len(result) > 280 {
				t.Errorf("Tweet text exceeds 280 characters: %d", len(result))
			}
		})
	}
}

func TestTwitterService_BuildTwitterMessage(t *testing.T) {
	service := &TwitterService{}

	// Create attachment manager with test data
	attachmentMgr := NewAttachmentManager()
	err := attachmentMgr.AddData([]byte("test image"), "screenshot.png", "image/png")
	if err != nil {
		t.Fatalf("Failed to add test attachment: %v", err)
	}

	req := NotificationRequest{
		Title:         "Build Failed",
		Body:          "Unit tests failed on master branch",
		NotifyType:    NotifyTypeError,
		Tags:          []string{"build", "tests", "master"},
		URL:           "https://ci.example.com/build/456",
		AttachmentMgr: attachmentMgr,
	}

	message := service.buildTwitterMessage(req)

	// Check that message contains expected content
	if !strings.Contains(message.Text, "Build Failed") {
		t.Errorf("Expected message text to contain 'Build Failed', got '%s'", message.Text)
	}

	if !strings.Contains(message.Text, "🚨") {
		t.Errorf("Expected error emoji in message text, got '%s'", message.Text)
	}

	if !strings.Contains(message.Text, "[1 attachments]") {
		t.Errorf("Expected attachment info in message text, got '%s'", message.Text)
	}

	// Check message structure
	if message.Text == "" {
		t.Error("Message text should not be empty")
	}

	if len(message.Text) > 280 {
		t.Errorf("Message text exceeds Twitter limit: %d characters", len(message.Text))
	}
}

func TestTwitterService_OAuth1Signature(t *testing.T) {
	service := &TwitterService{
		apiKey:       "test-api-key",
		apiKeySecret: "test-api-secret",
		accessToken:  "test-access-token",
		accessSecret: "test-access-secret",
	}

	// Create a test request
	reqURL := "https://api.twitter.com/1.1/statuses/update.json"
	data := url.Values{}
	data.Set("status", "Test tweet")

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Sign the request
	err = service.signOAuth1Request(req, data)
	if err != nil {
		t.Fatalf("Failed to sign request: %v", err)
	}

	// Verify Authorization header is set
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		t.Error("Authorization header not set")
	}

	if !strings.HasPrefix(authHeader, "OAuth ") {
		t.Errorf("Expected Authorization header to start with 'OAuth ', got '%s'", authHeader)
	}

	// Verify required OAuth parameters are present
	requiredParams := []string{
		"oauth_consumer_key",
		"oauth_token",
		"oauth_signature_method",
		"oauth_timestamp",
		"oauth_nonce",
		"oauth_version",
		"oauth_signature",
	}

	for _, param := range requiredParams {
		if !strings.Contains(authHeader, param) {
			t.Errorf("Authorization header missing required parameter: %s", param)
		}
	}
}

func TestTwitterService_GenerateNonce(t *testing.T) {
	service := &TwitterService{}

	nonce1 := service.generateNonce()
	nonce2 := service.generateNonce()

	// Check nonce length
	if len(nonce1) != 32 {
		t.Errorf("Expected nonce length 32, got %d", len(nonce1))
	}

	// Check nonces are different
	if nonce1 == nonce2 {
		t.Error("Generated nonces should be different")
	}

	// Check nonce contains only valid characters
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range nonce1 {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("Nonce contains invalid character: %c", char)
		}
	}
}

func TestTwitterService_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		service     *TwitterService
		expectError bool
	}{
		{
			name: "Valid OAuth 1.0a credentials",
			service: &TwitterService{
				apiKey:       "key",
				apiKeySecret: "secret",
				accessToken:  "token",
				accessSecret: "secret",
			},
			expectError: false,
		},
		{
			name: "Valid Bearer token",
			service: &TwitterService{
				bearerToken: "bearer123",
			},
			expectError: false,
		},
		{
			name: "Missing all credentials",
			service: &TwitterService{},
			expectError: true,
		},
		{
			name: "Incomplete OAuth credentials",
			service: &TwitterService{
				apiKey:      "key",
				accessToken: "token",
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

func TestTwitterService_DirectAPICall(t *testing.T) {
	service := &TwitterService{
		bearerToken: "test-bearer-token",
		client:      GetCloudHTTPClient("twitter"),
	}

	message := TwitterMessage{
		Text: "Test tweet from unit test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will fail with authentication error since we're using a fake token
	err := service.sendTweetV2(ctx, message)
	if err == nil {
		t.Error("Expected error for fake bearer token")
	}

	// The error should be related to authentication or network
	if !strings.Contains(err.Error(), "twitter api error") && !strings.Contains(err.Error(), "failed to send tweet") {
		t.Errorf("Unexpected error type: %v", err)
	}
}