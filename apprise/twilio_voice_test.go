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

func TestTwilioVoiceService_GetServiceID(t *testing.T) {
	service := NewTwilioVoiceService()
	if service.GetServiceID() != "twilio-voice" {
		t.Errorf("Expected service ID 'twilio-voice', got '%s'", service.GetServiceID())
	}
}

func TestTwilioVoiceService_GetDefaultPort(t *testing.T) {
	service := NewTwilioVoiceService()
	if service.GetDefaultPort() != 443 {
		t.Errorf("Expected default port 443, got %d", service.GetDefaultPort())
	}
}

func TestTwilioVoiceService_SupportsAttachments(t *testing.T) {
	service := NewTwilioVoiceService()
	if service.SupportsAttachments() {
		t.Error("Twilio Voice should not support attachments")
	}
}

func TestTwilioVoiceService_GetMaxBodyLength(t *testing.T) {
	service := NewTwilioVoiceService()
	expected := 4096
	if service.GetMaxBodyLength() != expected {
		t.Errorf("Expected max body length %d, got %d", expected, service.GetMaxBodyLength())
	}
}

func TestTwilioVoiceService_ParseURL(t *testing.T) {
	tests := []struct {
		name                  string
		url                   string
		expectError           bool
		expectedAccountSID    string
		expectedAuthToken     string
		expectedFromNumber    string
		expectedToNumbers     []string
		expectedVoiceLanguage string
		expectedVoiceGender   string
		expectedWebhook       string
		expectedProxyKey      string
	}{
		{
			name:                  "Basic Twilio Voice URL",
			url:                   "twilio-voice://AC123:token456@api.twilio.com/+12345678901/+19876543210",
			expectError:           false,
			expectedAccountSID:    "AC123",
			expectedAuthToken:     "token456",
			expectedFromNumber:    "+12345678901",
			expectedToNumbers:     []string{"+19876543210"},
			expectedVoiceLanguage: "en-US", // default
			expectedVoiceGender:   "female", // default
		},
		{
			name:                  "Multiple destination numbers in path",
			url:                   "twilio-voice://AC123:token@api.twilio.com/+12345678901/+19876543210/+14567890123",
			expectError:           false,
			expectedAccountSID:    "AC123",
			expectedAuthToken:     "token",
			expectedFromNumber:    "+12345678901",
			expectedToNumbers:     []string{"+19876543210", "+14567890123"},
			expectedVoiceLanguage: "en-US",
			expectedVoiceGender:   "female",
		},
		{
			name:                  "With query parameters",
			url:                   "twilio-voice://AC123:token@api.twilio.com/+12345678901?to=+19876543210,+14567890123&language=es-ES&gender=male",
			expectError:           false,
			expectedAccountSID:    "AC123",
			expectedAuthToken:     "token",
			expectedFromNumber:    "+12345678901",
			expectedToNumbers:     []string{"+19876543210", "+14567890123"},
			expectedVoiceLanguage: "es-ES",
			expectedVoiceGender:   "male",
		},
		{
			name:                  "Webhook proxy mode",
			url:                   "twilio-voice://proxy-key@webhook.example.com/twilio-voice?account_sid=AC123&auth_token=token&from=+12345678901&to=+19876543210",
			expectError:           false,
			expectedWebhook:       "https://webhook.example.com/twilio-voice",
			expectedProxyKey:      "proxy-key",
			expectedAccountSID:    "AC123",
			expectedAuthToken:     "token",
			expectedFromNumber:    "+12345678901",
			expectedToNumbers:     []string{"+19876543210"},
			expectedVoiceLanguage: "en-US",
			expectedVoiceGender:   "female",
		},
		{
			name:        "Invalid scheme",
			url:         "http://AC123:token@api.twilio.com/+12345678901",
			expectError: true,
		},
		{
			name:        "Missing account SID",
			url:         "twilio-voice://:token@api.twilio.com/+12345678901",
			expectError: true,
		},
		{
			name:        "Missing auth token",
			url:         "twilio-voice://AC123@api.twilio.com/+12345678901",
			expectError: true,
		},
		{
			name:        "Invalid from phone number",
			url:         "twilio-voice://AC123:token@api.twilio.com/invalid_number",
			expectError: true,
		},
		{
			name:        "Invalid to phone number",
			url:         "twilio-voice://AC123:token@api.twilio.com/+12345678901/invalid_number",
			expectError: true,
		},
		{
			name:        "Invalid language",
			url:         "twilio-voice://AC123:token@api.twilio.com/+12345678901/+19876543210?language=invalid",
			expectError: true,
		},
		{
			name:        "Invalid gender",
			url:         "twilio-voice://AC123:token@api.twilio.com/+12345678901/+19876543210?gender=invalid",
			expectError: true,
		},
		{
			name:        "Missing destination numbers",
			url:         "twilio-voice://AC123:token@api.twilio.com/+12345678901",
			expectError: true,
		},
		{
			name:        "Webhook missing account SID",
			url:         "twilio-voice://proxy@webhook.example.com/twilio-voice?auth_token=token&from=+12345678901&to=+19876543210",
			expectError: true,
		},
		{
			name:        "Webhook missing auth token",
			url:         "twilio-voice://proxy@webhook.example.com/twilio-voice?account_sid=AC123&from=+12345678901&to=+19876543210",
			expectError: true,
		},
		{
			name:        "Webhook missing from number",
			url:         "twilio-voice://proxy@webhook.example.com/twilio-voice?account_sid=AC123&auth_token=token&to=+19876543210",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTwilioVoiceService().(*TwilioVoiceService)
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

			if service.accountSID != tt.expectedAccountSID {
				t.Errorf("Expected account SID '%s', got '%s'", tt.expectedAccountSID, service.accountSID)
			}

			if service.authToken != tt.expectedAuthToken {
				t.Errorf("Expected auth token '%s', got '%s'", tt.expectedAuthToken, service.authToken)
			}

			if service.fromNumber != tt.expectedFromNumber {
				t.Errorf("Expected from number '%s', got '%s'", tt.expectedFromNumber, service.fromNumber)
			}

			if len(service.toNumbers) != len(tt.expectedToNumbers) {
				t.Errorf("Expected %d to numbers, got %d", len(tt.expectedToNumbers), len(service.toNumbers))
			} else {
				for i, expected := range tt.expectedToNumbers {
					if service.toNumbers[i] != expected {
						t.Errorf("Expected to number[%d] '%s', got '%s'", i, expected, service.toNumbers[i])
					}
				}
			}

			if service.voiceLanguage != tt.expectedVoiceLanguage {
				t.Errorf("Expected voice language '%s', got '%s'", tt.expectedVoiceLanguage, service.voiceLanguage)
			}

			if service.voiceGender != tt.expectedVoiceGender {
				t.Errorf("Expected voice gender '%s', got '%s'", tt.expectedVoiceGender, service.voiceGender)
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

func TestTwilioVoiceService_TestURL(t *testing.T) {
	service := NewTwilioVoiceService()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Valid Twilio Voice URL",
			url:         "twilio-voice://AC123:token@api.twilio.com/+12345678901/+19876543210",
			expectError: false,
		},
		{
			name:        "Valid webhook URL",
			url:         "twilio-voice://proxy@webhook.example.com/twilio-voice?account_sid=AC123&auth_token=token&from=+12345678901&to=+19876543210",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong scheme",
			url:         "http://AC123:token@api.twilio.com/+12345678901/+19876543210",
			expectError: true,
		},
		{
			name:        "Missing credentials",
			url:         "twilio-voice://api.twilio.com/+12345678901/+19876543210",
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

func TestTwilioVoiceService_SendWebhook(t *testing.T) {
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
		var payload TwilioVoiceWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify payload structure
		if payload.Service != "twilio-voice" {
			t.Errorf("Expected service 'twilio-voice', got '%s'", payload.Service)
		}

		if payload.AccountSID != "AC123" {
			t.Errorf("Expected account SID 'AC123', got '%s'", payload.AccountSID)
		}

		if len(payload.Calls) == 0 {
			t.Error("Expected calls to be present")
		}

		if len(payload.Calls) > 0 {
			call := payload.Calls[0]
			if call.From != "+12345678901" {
				t.Errorf("Expected from number '+12345678901', got '%s'", call.From)
			}
			if call.To != "+19876543210" {
				t.Errorf("Expected to number '+19876543210', got '%s'", call.To)
			}
			if !strings.Contains(call.Twiml, "Test Voice Alert") {
				t.Errorf("Expected TwiML to contain 'Test Voice Alert', got '%s'", call.Twiml)
			}
		}

		if payload.Language != "en-US" {
			t.Errorf("Expected language 'en-US', got '%s'", payload.Language)
		}

		if payload.Gender != "female" {
			t.Errorf("Expected gender 'female', got '%s'", payload.Gender)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Configure service for webhook
	service := NewTwilioVoiceService().(*TwilioVoiceService)
	service.webhookURL = server.URL
	service.proxyAPIKey = "test-proxy-key"
	service.accountSID = "AC123"
	service.authToken = "token456"
	service.fromNumber = "+12345678901"
	service.toNumbers = []string{"+19876543210"}
	service.voiceLanguage = "en-US"
	service.voiceGender = "female"

	req := NotificationRequest{
		Title:      "Test Voice Alert",
		Body:       "This is a test voice notification",
		NotifyType: NotifyTypeError,
		Tags:       []string{"voice", "urgent"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Send(ctx, req)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestTwilioVoiceService_SendAPICall(t *testing.T) {
	// Create mock Twilio API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		// Verify authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be present")
		}
		if username != "AC123" {
			t.Errorf("Expected username 'AC123', got '%s'", username)
		}
		if password != "token456" {
			t.Errorf("Expected password 'token456', got '%s'", password)
		}

		// Verify URL path
		expectedPath := "/2010-04-01/Accounts/AC123/Calls.json"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form data: %v", err)
		}

		// Verify form fields
		if r.FormValue("From") != "+12345678901" {
			t.Errorf("Expected From '+12345678901', got '%s'", r.FormValue("From"))
		}

		if r.FormValue("To") != "+19876543210" {
			t.Errorf("Expected To '+19876543210', got '%s'", r.FormValue("To"))
		}

		twiml := r.FormValue("Twiml")
		if !strings.Contains(twiml, "API Test Call") {
			t.Errorf("Expected TwiML to contain 'API Test Call', got '%s'", twiml)
		}

		// Check TwiML structure
		if !strings.Contains(twiml, "<Response>") || !strings.Contains(twiml, "</Response>") {
			t.Error("Expected valid TwiML response structure")
		}

		if !strings.Contains(twiml, "<Say") || !strings.Contains(twiml, "</Say>") {
			t.Error("Expected TwiML to contain Say element")
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid": "CA123456", "status": "queued"}`))
	}))
	defer server.Close()

	// Configure service for direct API mode
	service := NewTwilioVoiceService().(*TwilioVoiceService)
	service.accountSID = "AC123"
	service.authToken = "token456"
	service.fromNumber = "+12345678901"
	service.toNumbers = []string{"+19876543210"}

	// Override the Twilio API URL for testing
	originalURL := "https://api.twilio.com/2010-04-01/Accounts/AC123/Calls.json"
	testURL := server.URL + "/2010-04-01/Accounts/AC123/Calls.json"

	// Create call directly for testing
	call := TwilioVoiceCall{
		From:  service.fromNumber,
		To:    service.toNumbers[0],
		Twiml: service.generateTwiML(NotificationRequest{Title: "API Test Call", Body: "Testing direct API integration", NotifyType: NotifyTypeInfo}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.sendSingleCall(ctx, testURL, call)
	if err != nil {
		t.Fatalf("API Send failed: %v", err)
	}

	// Also test the URL handling
	_ = originalURL // Prevent unused variable warning
}

func TestTwilioVoiceService_GenerateTwiML(t *testing.T) {
	service := &TwilioVoiceService{
		voiceLanguage: "en-US",
		voiceGender:   "female",
	}

	tests := []struct {
		name         string
		req          NotificationRequest
		expectInTwiML []string
	}{
		{
			name: "Basic notification",
			req: NotificationRequest{
				Title:      "Test Alert",
				Body:       "System status update",
				NotifyType: NotifyTypeInfo,
			},
			expectInTwiML: []string{
				"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
				"<Response>",
				"<Say voice=\"woman\" language=\"en-US\">",
				"Test Alert. System status update",
				"</Say>",
				"</Response>",
			},
		},
		{
			name: "Error notification",
			req: NotificationRequest{
				Title:      "System Failure",
				Body:       "Database connection lost",
				NotifyType: NotifyTypeError,
			},
			expectInTwiML: []string{
				"Alert: System Failure. Database connection lost",
			},
		},
		{
			name: "Warning notification",
			req: NotificationRequest{
				Title:      "High CPU Usage",
				Body:       "CPU at 90%",
				NotifyType: NotifyTypeWarning,
			},
			expectInTwiML: []string{
				"Warning: High CPU Usage. CPU at 90%",
			},
		},
		{
			name: "Success notification",
			req: NotificationRequest{
				Title:      "Deployment Complete",
				Body:       "",
				NotifyType: NotifyTypeSuccess,
			},
			expectInTwiML: []string{
				"Success: Deployment Complete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			twiml := service.generateTwiML(tt.req)

			for _, expected := range tt.expectInTwiML {
				if !strings.Contains(twiml, expected) {
					t.Errorf("Expected TwiML to contain '%s', got '%s'", expected, twiml)
				}
			}
		})
	}
}

func TestTwilioVoiceService_CleanMessageForVoice(t *testing.T) {
	service := &TwilioVoiceService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello & goodbye", "Hello and goodbye"},
		{"Value < 10", "Value less than 10"},
		{"Value > 5", "Value greater than 5"},
		{"He said \"hello\"", "He said quote hello quote"},
		{"Don't worry", "Dont worry"},
		{"Multiple  spaces   here", "Multiple spaces here"},
		{"  Trimmed spaces  ", "Trimmed spaces"},
	}

	for _, tt := range tests {
		result := service.cleanMessageForVoice(tt.input)
		if result != tt.expected {
			t.Errorf("cleanMessageForVoice(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestTwilioVoiceService_ValidationMethods(t *testing.T) {
	service := &TwilioVoiceService{}

	// Test phone number validation
	validNumbers := []string{"+12345678901", "+447700900123", "+8613800138000"}
	for _, number := range validNumbers {
		if !service.isValidPhoneNumber(number) {
			t.Errorf("Expected phone number '%s' to be valid", number)
		}
	}

	invalidNumbers := []string{"12345678901", "+abc123", "+", "++12345", "+123456789012345678"}
	for _, number := range invalidNumbers {
		if service.isValidPhoneNumber(number) {
			t.Errorf("Expected phone number '%s' to be invalid", number)
		}
	}

	// Test language validation
	validLanguages := []string{"en-US", "es-ES", "fr-FR", "de-DE", "EN-US"}
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

	// Test gender validation
	validGenders := []string{"male", "female", "MALE", "Female"}
	for _, gender := range validGenders {
		if !service.isValidGender(gender) {
			t.Errorf("Expected gender '%s' to be valid", gender)
		}
	}

	invalidGenders := []string{"invalid", "m", "f", ""}
	for _, gender := range invalidGenders {
		if service.isValidGender(gender) {
			t.Errorf("Expected gender '%s' to be invalid", gender)
		}
	}
}

func TestTwilioVoiceService_GetVoiceName(t *testing.T) {
	tests := []struct {
		language     string
		gender       string
		expectedVoice string
	}{
		{"en-US", "male", "man"},
		{"en-US", "female", "woman"},
		{"en-GB", "male", "man"},
		{"en-GB", "female", "woman"},
		{"es-ES", "male", "man"},
		{"es-ES", "female", "woman"},
	}

	for _, tt := range tests {
		service := &TwilioVoiceService{
			voiceLanguage: tt.language,
			voiceGender:   tt.gender,
		}

		voice := service.getVoiceName()
		if voice != tt.expectedVoice {
			t.Errorf("Expected voice '%s' for language '%s' and gender '%s', got '%s'", 
				tt.expectedVoice, tt.language, tt.gender, voice)
		}
	}
}