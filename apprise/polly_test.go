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

func TestPollyService_GetServiceID(t *testing.T) {
	service := NewPollyService()
	if service.GetServiceID() != "polly" {
		t.Errorf("Expected service ID 'polly', got '%s'", service.GetServiceID())
	}
}

func TestPollyService_GetDefaultPort(t *testing.T) {
	service := NewPollyService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestPollyService_SupportsAttachments(t *testing.T) {
	service := NewPollyService()
	if service.SupportsAttachments() {
		t.Error("Amazon Polly should not support attachments")
	}
}

func TestPollyService_GetMaxBodyLength(t *testing.T) {
	service := NewPollyService()
	expected := 3000
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestPollyService_ParseURL(t *testing.T) {
	tests := []struct {
		name                    string
		url                     string
		expectError             bool
		expectedAccessKeyID     string
		expectedSecretAccessKey string
		expectedRegion          string
		expectedVoiceID         string
		expectedOutputFormat    string
		expectedLanguageCode    string
		expectedWebhook         string
		expectedProxyKey        string
		expectedS3Bucket        string
	}{
		{
			name:                    "Basic AWS Polly URL",
			url:                     "polly://AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY@polly.us-east-1.amazonaws.com/",
			expectError:             false,
			expectedAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			expectedSecretAccessKey: "wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY",
			expectedRegion:          "us-east-1",
			expectedVoiceID:         "Joanna", // default
			expectedOutputFormat:    "mp3",    // default
			expectedLanguageCode:    "en-US",  // default
		},
		{
			name:                    "With voice and format parameters",
			url:                     "polly://AKIAIOSFODNN7EXAMPLE:secret@polly.eu-west-1.amazonaws.com/?voice=Matthew&format=ogg_vorbis&language=en-GB",
			expectError:             false,
			expectedAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "eu-west-1",
			expectedVoiceID:         "Matthew",
			expectedOutputFormat:    "ogg_vorbis",
			expectedLanguageCode:    "en-GB",
		},
		{
			name:                    "With S3 storage options",
			url:                     "polly://key:secret@polly.us-west-2.amazonaws.com/?voice=Joanna&s3_bucket=audio-files&s3_prefix=notifications/",
			expectError:             false,
			expectedAccessKeyID:     "key",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "us-west-2",
			expectedVoiceID:         "Joanna",
			expectedOutputFormat:    "mp3",   // default
			expectedLanguageCode:    "en-US", // default
			expectedS3Bucket:        "audio-files",
		},
		{
			name:                    "Webhook proxy mode",
			url:                     "polly://proxy-key@webhook.example.com/polly?access_key=AKIATEST&secret_key=secrettest&region=us-east-1&voice=Salli",
			expectError:             false,
			expectedWebhook:         "https://webhook.example.com/polly",
			expectedProxyKey:        "proxy-key",
			expectedAccessKeyID:     "AKIATEST",
			expectedSecretAccessKey: "secrettest",
			expectedRegion:          "us-east-1",
			expectedVoiceID:         "Salli",
			expectedOutputFormat:    "mp3",
			expectedLanguageCode:    "en-US",
		},
		{
			name:                    "Webhook with full parameters",
			url:                     "polly://proxy@webhook.example.com/polly?access_key=key&secret_key=secret&region=eu-west-1&voice=Amy&format=pcm&language=en-GB&s3_bucket=speech-files",
			expectError:             false,
			expectedWebhook:         "https://webhook.example.com/polly",
			expectedProxyKey:        "proxy",
			expectedAccessKeyID:     "key",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "eu-west-1",
			expectedVoiceID:         "Amy",
			expectedOutputFormat:    "pcm",
			expectedLanguageCode:    "en-GB",
			expectedS3Bucket:        "speech-files",
		},
		{
			name:        "Invalid scheme",
			url:         "http://key:secret@polly.us-east-1.amazonaws.com/",
			expectError: true,
		},
		{
			name:        "Missing access key",
			url:         "polly://:secret@polly.us-east-1.amazonaws.com/",
			expectError: true,
		},
		{
			name:        "Missing secret key",
			url:         "polly://key@polly.us-east-1.amazonaws.com/",
			expectError: true,
		},
		{
			name:                    "Invalid region in hostname",
			url:                     "polly://key:secret@polly.invalid-region.amazonaws.com/",
			expectError:             false, // This won't fail because we don't validate hostname regions automatically
			expectedAccessKeyID:     "key",
			expectedSecretAccessKey: "secret",
			expectedRegion:          "us-east-1", // falls back to default
			expectedVoiceID:         "Joanna",    // default
			expectedOutputFormat:    "mp3",       // default
			expectedLanguageCode:    "en-US",     // default
		},
		{
			name:        "Invalid voice",
			url:         "polly://key:secret@polly.us-east-1.amazonaws.com/?voice=InvalidVoice",
			expectError: true,
		},
		{
			name:        "Invalid format",
			url:         "polly://key:secret@polly.us-east-1.amazonaws.com/?format=invalid",
			expectError: true,
		},
		{
			name:        "Invalid language",
			url:         "polly://key:secret@polly.us-east-1.amazonaws.com/?language=invalid",
			expectError: true,
		},
		{
			name:        "Webhook missing access key",
			url:         "polly://proxy@webhook.example.com/polly?secret_key=secret&region=us-east-1",
			expectError: true,
		},
		{
			name:        "Webhook missing secret key",
			url:         "polly://proxy@webhook.example.com/polly?access_key=key&region=us-east-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPollyService().(*PollyService)
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

			if service.accessKeyID != tt.expectedAccessKeyID {
				t.Errorf("Expected access key ID '%s', got '%s'", tt.expectedAccessKeyID, service.accessKeyID)
			}

			if service.secretAccessKey != tt.expectedSecretAccessKey {
				t.Errorf("Expected secret access key '%s', got '%s'", tt.expectedSecretAccessKey, service.secretAccessKey)
			}

			if service.region != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, service.region)
			}

			if service.voiceID != tt.expectedVoiceID {
				t.Errorf("Expected voice ID '%s', got '%s'", tt.expectedVoiceID, service.voiceID)
			}

			if service.outputFormat != tt.expectedOutputFormat {
				t.Errorf("Expected output format '%s', got '%s'", tt.expectedOutputFormat, service.outputFormat)
			}

			if service.languageCode != tt.expectedLanguageCode {
				t.Errorf("Expected language code '%s', got '%s'", tt.expectedLanguageCode, service.languageCode)
			}

			if tt.expectedWebhook != "" && service.webhookURL != tt.expectedWebhook {
				t.Errorf("Expected webhook URL '%s', got '%s'", tt.expectedWebhook, service.webhookURL)
			}

			if tt.expectedProxyKey != "" && service.proxyAPIKey != tt.expectedProxyKey {
				t.Errorf("Expected proxy key '%s', got '%s'", tt.expectedProxyKey, service.proxyAPIKey)
			}

			if tt.expectedS3Bucket != "" && service.s3Bucket != tt.expectedS3Bucket {
				t.Errorf("Expected S3 bucket '%s', got '%s'", tt.expectedS3Bucket, service.s3Bucket)
			}
		})
	}
}

func TestPollyService_TestURL(t *testing.T) {
	service := NewPollyService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Polly URL",
			url:         "polly://key:secret@polly.us-east-1.amazonaws.com/?voice=Joanna",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "polly://proxy@webhook.example.com/polly?access_key=key&secret_key=secret&region=us-east-1",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://key:secret@polly.us-east-1.amazonaws.com/",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "polly://polly.us-east-1.amazonaws.com/",
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

func TestPollyService_SendWebhook(t *testing.T) {
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
		var payload PollyWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "polly" {
			t.Errorf("Expected service 'polly', got '%s'", payload.Service)
		}

		if payload.Region != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got '%s'", payload.Region)
		}

		if payload.AccessKeyID != "AKIATEST" {
			t.Errorf("Expected access key 'AKIATEST', got '%s'", payload.AccessKeyID)
		}

		if payload.SecretAccessKey != "secrettest" {
			t.Errorf("Expected secret key 'secrettest', got '%s'", payload.SecretAccessKey)
		}

		if payload.Request.VoiceId != "Joanna" {
			t.Errorf("Expected voice ID 'Joanna', got '%s'", payload.Request.VoiceId)
		}

		if payload.Request.OutputFormat != "mp3" {
			t.Errorf("Expected output format 'mp3', got '%s'", payload.Request.OutputFormat)
		}

		if payload.Request.LanguageCode != "en-US" {
			t.Errorf("Expected language code 'en-US', got '%s'", payload.Request.LanguageCode)
		}

		if !strings.Contains(payload.Request.Text, "Test Speech Alert") {
			t.Errorf("Expected text to contain 'Test Speech Alert', got '%s'", payload.Request.Text)
		}

		if !strings.Contains(payload.Request.Text, "Alert.") {
			t.Error("Expected text to contain notification type prefix")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true, "audio_url": "https://s3.amazonaws.com/bucket/audio.mp3"}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewPollyService().(*PollyService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.accessKeyID = "AKIATEST"
	service.secretAccessKey = "secrettest"
	service.region = "us-east-1"
	service.voiceID = "Joanna"
	service.outputFormat = "mp3"
	service.languageCode = "en-US"

	req := NotificationRequest{
		Title:      "Test Speech Alert",
		Body:       "This is a test text-to-speech notification",
		NotifyType: NotifyTypeError,
		Tags:       []string{"speech", "tts"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestPollyService_BuildSpeechText(t *testing.T) {
	service := &PollyService{}

	tests := []struct {
		name            string
		req             NotificationRequest
		expectedContent []string
	}{
		{
			name: "Basic notification",
			req: NotificationRequest{
				Title:      "System Alert",
				Body:       "Database connection failed",
				NotifyType: NotifyTypeError,
			},
			expectedContent: []string{
				"Alert. System Alert. Database connection failed",
			},
		},
		{
			name: "Warning with tags",
			req: NotificationRequest{
				Title:      "High CPU Usage",
				Body:       "CPU usage is above 80%",
				NotifyType: NotifyTypeWarning,
				Tags:       []string{"performance", "cpu"},
			},
			expectedContent: []string{
				"Warning. High CPU Usage. CPU usage is above 80 percent .",
				"Tags: performance, cpu",
			},
		},
		{
			name: "Success notification",
			req: NotificationRequest{
				Title:      "Deployment Complete",
				Body:       "",
				NotifyType: NotifyTypeSuccess,
			},
			expectedContent: []string{
				"Success. Deployment Complete",
			},
		},
		{
			name: "Info with special characters",
			req: NotificationRequest{
				Title:      "Status Update",
				Body:       "Memory usage: 75% & CPU: 45%",
				NotifyType: NotifyTypeInfo,
			},
			expectedContent: []string{
				"Information. Status Update. Memory usage: 75 percent and CPU: 45 percent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := service.buildSpeechText(tt.req)

			for _, expected := range tt.expectedContent {
				if !strings.Contains(text, expected) {
					t.Errorf("Expected speech text to contain '%s', got '%s'", expected, text)
				}
			}
		})
	}
}

func TestPollyService_CleanTextForSpeech(t *testing.T) {
	service := &PollyService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello & goodbye", "Hello and goodbye"},
		{"Value < 10", "Value less than 10"},
		{"Value > 5", "Value greater than 5"},
		{"Email: user@domain.com", "Email: user at domain.com"},
		{"Tag #important", "Tag hash important"},
		{"95% complete", "95 percent complete"},
		{"He said \"hello\"", "He said hello"},
		{"Don't worry", "Dont worry"},
		{"Multiple  spaces   here", "Multiple spaces here"},
		{"  Trimmed spaces  ", "Trimmed spaces"},
	}

	for _, tt := range tests {
		result := service.cleanTextForSpeech(tt.input)
		if result != tt.expected {
			t.Errorf("cleanTextForSpeech(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestPollyService_ValidationMethods(t *testing.T) {
	service := &PollyService{}

	// Test region validation
	validRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	for _, region := range validRegions {
		if !service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be valid", region)
		}
	}

	invalidRegions := []string{"invalid", "us-invalid-1", "eu-invalid", ""}
	for _, region := range invalidRegions {
		if service.isValidRegion(region) {
			t.Errorf("Expected region '%s' to be invalid", region)
		}
	}

	// Test voice validation
	validVoices := []string{"Joanna", "Matthew", "Amy", "Brian", "Penelope", "Celine"}
	for _, voice := range validVoices {
		if !service.isValidVoice(voice) {
			t.Errorf("Expected voice '%s' to be valid", voice)
		}
	}

	invalidVoices := []string{"InvalidVoice", "Unknown", ""}
	for _, voice := range invalidVoices {
		if service.isValidVoice(voice) {
			t.Errorf("Expected voice '%s' to be invalid", voice)
		}
	}

	// Test format validation
	validFormats := []string{"mp3", "ogg_vorbis", "pcm"}
	for _, format := range validFormats {
		if !service.isValidFormat(format) {
			t.Errorf("Expected format '%s' to be valid", format)
		}
	}

	invalidFormats := []string{"wav", "flac", "invalid", ""}
	for _, format := range invalidFormats {
		if service.isValidFormat(format) {
			t.Errorf("Expected format '%s' to be invalid", format)
		}
	}

	// Test language validation
	validLanguages := []string{"en-US", "en-GB", "es-ES", "fr-FR", "de-DE", "ja-JP"}
	for _, language := range validLanguages {
		if !service.isValidLanguage(language) {
			t.Errorf("Expected language '%s' to be valid", language)
		}
	}

	invalidLanguages := []string{"invalid", "en", "us", ""}
	for _, language := range invalidLanguages {
		if service.isValidLanguage(language) {
			t.Errorf("Expected language '%s' to be invalid", language)
		}
	}
}

func TestPollyService_SendToPollyDirectly(t *testing.T) {
	service := &PollyService{
		region: "us-east-1",
	}

	pollyReq := PollyRequest{
		Text:         "Test direct API call",
		VoiceId:      "Joanna",
		OutputFormat: "mp3",
		LanguageCode: "en-US",
		TextType:     "text",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should return an error indicating AWS Signature V4 is required
	err := service.sendToPollyDirectly(ctx, pollyReq)
	if err == nil {
		t.Error("Expected error for direct API call without AWS Signature V4")
	}

	if !strings.Contains(err.Error(), "AWS Signature V4") {
		t.Errorf("Expected error message about AWS Signature V4, got: %v", err)
	}
}