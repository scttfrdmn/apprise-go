package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PollyService implements Amazon Polly text-to-speech notifications
type PollyService struct {
	accessKeyID     string // AWS Access Key ID
	secretAccessKey string // AWS Secret Access Key
	region          string // AWS region (us-east-1, eu-west-1, etc.)
	voiceID         string // Polly voice ID (Joanna, Matthew, etc.)
	outputFormat    string // Audio format (mp3, ogg_vorbis, pcm)
	languageCode    string // Language code (en-US, es-ES, etc.)
	webhookURL      string // Webhook proxy URL for secure credential management
	proxyAPIKey     string // API key for webhook authentication
	s3Bucket        string // S3 bucket for audio file storage (optional)
	s3KeyPrefix     string // S3 key prefix for audio files
	client          *http.Client
}

// PollyRequest represents a Polly text-to-speech request
type PollyRequest struct {
	Text         string `json:"Text"`
	VoiceId      string `json:"VoiceId"`
	OutputFormat string `json:"OutputFormat"`
	LanguageCode string `json:"LanguageCode,omitempty"`
	TextType     string `json:"TextType,omitempty"` // text or ssml
	SampleRate   string `json:"SampleRate,omitempty"`
}

// PollyWebhookPayload represents webhook proxy payload
type PollyWebhookPayload struct {
	Service         string       `json:"service"`
	Region          string       `json:"region"`
	AccessKeyID     string       `json:"access_key_id"`
	SecretAccessKey string       `json:"secret_access_key"`
	Request         PollyRequest `json:"polly_request"`
	S3Bucket        string       `json:"s3_bucket,omitempty"`
	S3KeyPrefix     string       `json:"s3_key_prefix,omitempty"`
	Timestamp       string       `json:"timestamp"`
	Source          string       `json:"source"`
	Version         string       `json:"version"`
}

// PollyResponse represents Polly API response
type PollyResponse struct {
	AudioStream      []byte `json:"AudioStream,omitempty"`
	ContentType      string `json:"ContentType"`
	RequestCharacters int    `json:"RequestCharacters"`
}

// NewPollyService creates a new Amazon Polly service instance
func NewPollyService() Service {
	return &PollyService{
		client:       GetCloudHTTPClient("polly"),
		region:       "us-east-1", // Default region
		voiceID:      "Joanna",    // Default voice
		outputFormat: "mp3",       // Default format
		languageCode: "en-US",     // Default language
	}
}

// GetServiceID returns the service identifier
func (p *PollyService) GetServiceID() string {
	return "polly"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PollyService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an Amazon Polly service URL
// Format: polly://access_key:secret_key@polly.us-east-1.amazonaws.com/?voice=Joanna&format=mp3&language=en-US
// Format: polly://proxy-key@webhook.example.com/polly?access_key=key&secret_key=secret&region=us-east-1&voice=Joanna
func (p *PollyService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "polly" {
		return fmt.Errorf("invalid scheme: expected 'polly', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/polly") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		p.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			p.proxyAPIKey = serviceURL.User.Username()
		}

		// Get AWS credentials from query parameters
		p.accessKeyID = query.Get("access_key")
		if p.accessKeyID == "" {
			return fmt.Errorf("access_key parameter is required for webhook mode")
		}

		p.secretAccessKey = query.Get("secret_key")
		if p.secretAccessKey == "" {
			return fmt.Errorf("secret_key parameter is required for webhook mode")
		}

		// Get region
		if region := query.Get("region"); region != "" {
			if p.isValidRegion(region) {
				p.region = region
			} else {
				return fmt.Errorf("invalid AWS region: %s", region)
			}
		}
	} else {
		// Direct AWS API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: access_key and secret_key must be provided")
		}

		p.accessKeyID = serviceURL.User.Username()
		if p.accessKeyID == "" {
			return fmt.Errorf("AWS access key is required")
		}

		if secretKey, hasSecret := serviceURL.User.Password(); hasSecret {
			p.secretAccessKey = secretKey
		}
		if p.secretAccessKey == "" {
			return fmt.Errorf("AWS secret key is required")
		}

		// Extract region from host if present
		if serviceURL.Host != "" {
			hostParts := strings.Split(serviceURL.Host, ".")
			if len(hostParts) >= 3 && hostParts[0] == "polly" {
				if p.isValidRegion(hostParts[1]) {
					p.region = hostParts[1]
				}
			}
		}
	}

	// Parse optional parameters
	if voice := query.Get("voice"); voice != "" {
		if p.isValidVoice(voice) {
			p.voiceID = voice
		} else {
			return fmt.Errorf("invalid Polly voice: %s", voice)
		}
	}

	if format := query.Get("format"); format != "" {
		if p.isValidFormat(format) {
			p.outputFormat = format
		} else {
			return fmt.Errorf("invalid output format: %s (valid: mp3, ogg_vorbis, pcm)", format)
		}
	}

	if language := query.Get("language"); language != "" {
		if p.isValidLanguage(language) {
			p.languageCode = language
		} else {
			return fmt.Errorf("invalid language code: %s", language)
		}
	}

	// S3 storage options
	p.s3Bucket = query.Get("s3_bucket")
	p.s3KeyPrefix = query.Get("s3_prefix")

	return nil
}

// Send sends a text-to-speech notification via Amazon Polly
func (p *PollyService) Send(ctx context.Context, req NotificationRequest) error {
	// Build speech text
	text := p.buildSpeechText(req)
	
	pollyReq := PollyRequest{
		Text:         text,
		VoiceId:      p.voiceID,
		OutputFormat: p.outputFormat,
		LanguageCode: p.languageCode,
		TextType:     "text", // Could be enhanced to support SSML
	}

	if p.webhookURL != "" {
		// Send via webhook proxy
		return p.sendViaWebhook(ctx, pollyReq)
	} else {
		// Send directly to AWS Polly API
		return p.sendToPollyDirectly(ctx, pollyReq)
	}
}

// buildSpeechText creates optimized text for speech synthesis
func (p *PollyService) buildSpeechText(req NotificationRequest) string {
	var text strings.Builder

	// Add notification type prefix
	switch req.NotifyType {
	case NotifyTypeError:
		text.WriteString("Alert. ")
	case NotifyTypeWarning:
		text.WriteString("Warning. ")
	case NotifyTypeSuccess:
		text.WriteString("Success. ")
	case NotifyTypeInfo:
		text.WriteString("Information. ")
	}

	// Add title
	if req.Title != "" {
		text.WriteString(req.Title)
		if req.Body != "" {
			text.WriteString(". ")
		}
	}

	// Add body
	if req.Body != "" {
		text.WriteString(req.Body)
	}

	// Add tags if present
	if len(req.Tags) > 0 {
		text.WriteString(". Tags: ")
		text.WriteString(strings.Join(req.Tags, ", "))
	}

	// Clean text for speech
	return p.cleanTextForSpeech(text.String())
}

// sendViaWebhook sends request via webhook proxy
func (p *PollyService) sendViaWebhook(ctx context.Context, pollyReq PollyRequest) error {
	payload := PollyWebhookPayload{
		Service:         "polly",
		Region:          p.region,
		AccessKeyID:     p.accessKeyID,
		SecretAccessKey: p.secretAccessKey,
		Request:         pollyReq,
		S3Bucket:        p.s3Bucket,
		S3KeyPrefix:     p.s3KeyPrefix,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Source:          "apprise-go",
		Version:         GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Polly webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Polly webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if p.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", p.proxyAPIKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Polly webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("polly webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendToPollyDirectly sends request directly to AWS Polly API
func (p *PollyService) sendToPollyDirectly(ctx context.Context, pollyReq PollyRequest) error {
	pollyURL := fmt.Sprintf("https://polly.%s.amazonaws.com/v1/speech", p.region)

	jsonData, err := json.Marshal(pollyReq)
	if err != nil {
		return fmt.Errorf("failed to marshal Polly request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", pollyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Polly request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-amz-json-1.0")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	// AWS Signature V4 would normally be required here
	// For now, return an error indicating webhook mode should be used
	return fmt.Errorf("direct AWS API access requires AWS Signature V4 authentication - please use webhook proxy mode")
}

// Helper methods

func (p *PollyService) cleanTextForSpeech(text string) string {
	// Remove or replace problematic characters for speech synthesis
	text = strings.ReplaceAll(text, "&", " and ")
	text = strings.ReplaceAll(text, "<", " less than ")
	text = strings.ReplaceAll(text, ">", " greater than ")
	text = strings.ReplaceAll(text, "@", " at ")
	text = strings.ReplaceAll(text, "#", " hash ")
	text = strings.ReplaceAll(text, "%", " percent ")
	text = strings.ReplaceAll(text, "\"", "")
	text = strings.ReplaceAll(text, "'", "")
	
	// Replace multiple spaces with single space
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	
	return strings.TrimSpace(text)
}

func (p *PollyService) isValidRegion(region string) bool {
	// Common AWS regions that support Polly
	validRegions := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2",
		"ap-south-1", "ca-central-1", "sa-east-1",
	}
	
	for _, valid := range validRegions {
		if strings.EqualFold(region, valid) {
			return true
		}
	}
	return false
}

func (p *PollyService) isValidVoice(voice string) bool {
	// Common Polly voices (subset for validation)
	validVoices := []string{
		// English (US)
		"Joanna", "Matthew", "Ivy", "Justin", "Kendra", "Kimberly", "Salli", "Joey",
		// English (GB)
		"Amy", "Emma", "Brian",
		// Spanish
		"Penelope", "Miguel", "Conchita", "Enrique",
		// French
		"Celine", "Mathieu", "Lea",
		// German
		"Marlene", "Hans", "Vicki",
		// Italian
		"Carla", "Giorgio", "Bianca",
		// Portuguese
		"Vitoria", "Ricardo", "Camila",
		// Japanese
		"Mizuki", "Takumi",
		// Korean
		"Seoyeon",
		// Chinese
		"Zhiyu",
	}
	
	for _, valid := range validVoices {
		if strings.EqualFold(voice, valid) {
			return true
		}
	}
	return false
}

func (p *PollyService) isValidFormat(format string) bool {
	validFormats := []string{"mp3", "ogg_vorbis", "pcm"}
	for _, valid := range validFormats {
		if strings.EqualFold(format, valid) {
			return true
		}
	}
	return false
}

func (p *PollyService) isValidLanguage(language string) bool {
	// Common language codes supported by Polly
	validLanguages := []string{
		"en-US", "en-GB", "en-AU", "en-IN",
		"es-ES", "es-MX", "es-US",
		"fr-FR", "fr-CA",
		"de-DE",
		"it-IT",
		"pt-BR", "pt-PT",
		"ja-JP",
		"ko-KR",
		"zh-CN",
		"da-DK", "nl-NL", "nb-NO", "pl-PL", "ro-RO", "sv-SE", "tr-TR",
	}
	
	for _, valid := range validLanguages {
		if strings.EqualFold(language, valid) {
			return true
		}
	}
	return false
}

// TestURL validates an Amazon Polly service URL
func (p *PollyService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns false (voice synthesis doesn't support file attachments)
func (p *PollyService) SupportsAttachments() bool {
	return false // Voice synthesis cannot include file attachments
}

// GetMaxBodyLength returns Polly's content length limit
func (p *PollyService) GetMaxBodyLength() int {
	return 3000 // AWS Polly limit is 3000 characters for text input
}

// Example usage and URL formats:
// polly://access_key:secret_key@polly.us-east-1.amazonaws.com/?voice=Joanna&format=mp3&language=en-US
// polly://proxy-key@webhook.example.com/polly?access_key=key&secret_key=secret&region=us-east-1&voice=Matthew&s3_bucket=audio-files