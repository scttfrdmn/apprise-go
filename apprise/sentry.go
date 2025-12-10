package apprise

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SentryService implements Sentry.io error tracking notifications
// Sentry is the leading application monitoring platform with 3M+ developers
// Perfect for error tracking, performance monitoring, and release health
type SentryService struct {
	protocol  string // Protocol (http or https)
	publicKey string // DSN public key
	host      string // Sentry host (e.g., o123456.ingest.sentry.io or self-hosted)
	projectID string // Project ID
	dsn       string // Full DSN URL
	client    *http.Client
}

// SentryEvent represents a Sentry event payload
type SentryEvent struct {
	EventID     string                 `json:"event_id"`
	Timestamp   string                 `json:"timestamp"`
	Platform    string                 `json:"platform"`
	Level       string                 `json:"level"`
	Logger      string                 `json:"logger,omitempty"`
	Transaction string                 `json:"transaction,omitempty"`
	ServerName  string                 `json:"server_name,omitempty"`
	Release     string                 `json:"release,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Message     *SentryMessage         `json:"message,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
	Contexts    map[string]interface{} `json:"contexts,omitempty"`
}

// SentryMessage represents the message interface
type SentryMessage struct {
	Formatted string `json:"formatted"`
	Message   string `json:"message,omitempty"`
}

// SentryEnvelopeHeader represents the envelope header
type SentryEnvelopeHeader struct {
	EventID string `json:"event_id,omitempty"`
	SentAt  string `json:"sent_at,omitempty"`
}

// SentryItemHeader represents an item header in the envelope
type SentryItemHeader struct {
	Type        string `json:"type"`
	ContentType string `json:"content_type,omitempty"`
	Length      int    `json:"length,omitempty"`
}

// NewSentryService creates a new Sentry service instance
func NewSentryService() Service {
	return &SentryService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (s *SentryService) GetServiceID() string {
	return "sentry"
}

// GetDefaultPort returns the default port (443 for https, 80 for http)
func (s *SentryService) GetDefaultPort() int {
	if s.protocol == "https" {
		return 443
	}
	return 80
}

// ParseURL parses a Sentry DSN URL
// Format: sentry://public_key@host/project_id
// Format: sentry://public_key@host:port/project_id
// Format: https://public_key@o123456.ingest.sentry.io/project_id
//
// Examples:
//   sentry://abc123def456@o123456.ingest.sentry.io/789012
//   https://public_key@sentry.example.com/1
//   sentry://key@self-hosted.com:8080/project
func (s *SentryService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme

	// Determine protocol
	if scheme == "sentry" || scheme == "sentries" {
		s.protocol = "https"
	} else if scheme == "http" || scheme == "https" {
		s.protocol = scheme
	} else {
		return fmt.Errorf("invalid scheme: expected 'sentry', 'sentries', 'http', or 'https', got '%s'", scheme)
	}

	// Extract public key from user info
	if serviceURL.User == nil || serviceURL.User.Username() == "" {
		return fmt.Errorf("missing public key in DSN")
	}
	s.publicKey = serviceURL.User.Username()

	// Extract host
	host := serviceURL.Host
	if host == "" {
		return fmt.Errorf("missing host in DSN")
	}
	s.host = host

	// Extract project ID from path
	path := strings.Trim(serviceURL.Path, "/")
	if path == "" {
		return fmt.Errorf("missing project ID in DSN")
	}
	s.projectID = path

	// Build full DSN for reference
	s.dsn = fmt.Sprintf("%s://%s@%s/%s", s.protocol, s.publicKey, s.host, s.projectID)

	return nil
}

// Send sends a notification to Sentry as an error event
func (s *SentryService) Send(ctx context.Context, req NotificationRequest) error {
	// Generate event ID
	eventID, err := generateEventID()
	if err != nil {
		return fmt.Errorf("failed to generate event ID: %w", err)
	}

	// Map notification type to Sentry level
	level := s.mapNotifyTypeToLevel(req.NotifyType)

	// Build message
	message := s.buildMessage(req)

	// Create event
	event := SentryEvent{
		EventID:   eventID,
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05"),
		Platform:  "go",
		Level:     level,
		Logger:    "apprise-go",
		Message: &SentryMessage{
			Formatted: message,
			Message:   message,
		},
	}

	// Add tags if present
	if len(req.Tags) > 0 {
		event.Tags = make(map[string]string)
		for i, tag := range req.Tags {
			event.Tags[fmt.Sprintf("tag_%d", i)] = tag
		}
	}

	// Add extra context
	event.Extra = map[string]interface{}{
		"apprise_notification_type": req.NotifyType.String(),
		"source":                    "apprise-go",
	}

	// Send using envelope format
	return s.sendEnvelope(ctx, event)
}

// sendEnvelope sends the event using Sentry's envelope format
func (s *SentryService) sendEnvelope(ctx context.Context, event SentryEvent) error {
	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Build envelope
	// Format:
	// {envelope_header}\n
	// {item_header}\n
	// {item_payload}\n
	var envelope bytes.Buffer

	// Envelope header
	envelopeHeader := SentryEnvelopeHeader{
		EventID: event.EventID,
		SentAt:  time.Now().UTC().Format(time.RFC3339),
	}
	envelopeHeaderJSON, err := json.Marshal(envelopeHeader)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope header: %w", err)
	}
	envelope.Write(envelopeHeaderJSON)
	envelope.WriteString("\n")

	// Item header
	itemHeader := SentryItemHeader{
		Type:        "event",
		ContentType: "application/json",
		Length:      len(eventJSON),
	}
	itemHeaderJSON, err := json.Marshal(itemHeader)
	if err != nil {
		return fmt.Errorf("failed to marshal item header: %w", err)
	}
	envelope.Write(itemHeaderJSON)
	envelope.WriteString("\n")

	// Item payload
	envelope.Write(eventJSON)
	envelope.WriteString("\n")

	// Build URL
	envelopeURL := fmt.Sprintf("%s://%s/api/%s/envelope/", s.protocol, s.host, s.projectID)

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", envelopeURL, &envelope)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-sentry-envelope")
	httpReq.Header.Set("X-Sentry-Auth", s.buildAuthHeader(event.EventID))

	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// buildAuthHeader builds the X-Sentry-Auth header
func (s *SentryService) buildAuthHeader(eventID string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("Sentry sentry_version=7,sentry_key=%s,sentry_timestamp=%d,sentry_client=apprise-go/1.0",
		s.publicKey, timestamp)
}

// buildMessage constructs the event message
func (s *SentryService) buildMessage(req NotificationRequest) string {
	var builder strings.Builder

	// Add title if present
	if req.Title != "" {
		builder.WriteString(req.Title)
		if req.Body != "" {
			builder.WriteString(": ")
		}
	}

	// Add body
	if req.Body != "" {
		builder.WriteString(req.Body)
	}

	return builder.String()
}

// mapNotifyTypeToLevel maps AppRise notification types to Sentry levels
func (s *SentryService) mapNotifyTypeToLevel(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeSuccess:
		return "info"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeError:
		return "error"
	default:
		return "info"
	}
}

// generateEventID generates a random event ID (UUID v4 format)
func generateEventID() (string, error) {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return "", err
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10

	return hex.EncodeToString(uuid), nil
}

// TestURL validates the Sentry DSN format
func (s *SentryService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return s.ParseURL(parsedURL)
}

// SupportsAttachments returns false as Sentry envelope format supports attachments
// but we implement basic event sending only for notification purposes
func (s *SentryService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 0 (Sentry has generous limits for event payloads)
func (s *SentryService) GetMaxBodyLength() int {
	return 0 // No strict limit enforced
}
