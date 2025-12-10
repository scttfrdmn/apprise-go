package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ElasticsearchService implements Elasticsearch/OpenSearch notification indexing
// Elasticsearch is the leading search and analytics engine with 250M+ downloads
// OpenSearch is the open-source fork with AWS backing
type ElasticsearchService struct {
	scheme   string // http or https
	host     string
	port     string
	index    string // Index name for alert documents
	username string // Basic auth username
	password string // Basic auth password
	apiKey   string // API key authentication
	client   *http.Client
}

// ElasticsearchDocument represents an alert document to be indexed
type ElasticsearchDocument struct {
	Timestamp    string            `json:"@timestamp"`
	Title        string            `json:"title"`
	Message      string            `json:"message"`
	Severity     string            `json:"severity"`
	NotifyType   string            `json:"notify_type"`
	Tags         []string          `json:"tags,omitempty"`
	Source       string            `json:"source"`
	Host         string            `json:"host,omitempty"`
	Environment  string            `json:"environment,omitempty"`
	Application  string            `json:"application,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ElasticsearchIndexResponse represents the response from indexing a document
type ElasticsearchIndexResponse struct {
	Index   string `json:"_index"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Result  string `json:"result"` // "created" or "updated"
	Shards  struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
}

// ElasticsearchErrorResponse represents an error response
type ElasticsearchErrorResponse struct {
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
	Status int `json:"status"`
}

// NewElasticsearchService creates a new Elasticsearch service instance
func NewElasticsearchService() Service {
	return &ElasticsearchService{
		scheme: "http",
		index:  "apprise-notifications", // Default index name
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetServiceID returns the service identifier
func (e *ElasticsearchService) GetServiceID() string {
	return "elasticsearch"
}

// GetDefaultPort returns the default port (9200 for Elasticsearch)
func (e *ElasticsearchService) GetDefaultPort() int {
	return 9200
}

// ParseURL parses an Elasticsearch service URL
// Format: elasticsearch://host:port/index
// Format: elasticsearch://username:password@host:port/index
// Format: elasticsearch://host:port/index?apikey=key
// Format: opensearch://host:port/index (alias)
//
// Parameters:
//   - host: Elasticsearch/OpenSearch host
//   - port: Port (default: 9200)
//   - index: Index name (default: apprise-notifications)
//
// Query Parameters:
//   - apikey=key - API key authentication (preferred over basic auth)
//   - environment=prod - Add environment tag
//   - application=myapp - Add application name
//
// Examples:
//   elasticsearch://localhost:9200/alerts
//   elasticsearch://user:pass@es.example.com:9200/notifications
//   elasticsearch://es.example.com/logs?apikey=abc123
//   opensearch://opensearch.example.com:9200/apprise
func (e *ElasticsearchService) ParseURL(serviceURL *url.URL) error {
	scheme := serviceURL.Scheme
	if scheme != "elasticsearch" && scheme != "opensearch" && scheme != "es" {
		return fmt.Errorf("invalid scheme: expected 'elasticsearch', 'opensearch', or 'es', got '%s'", scheme)
	}

	// Determine if HTTPS should be used
	if scheme == "elasticsearch" || scheme == "es" {
		e.scheme = "http"
	} else {
		e.scheme = "http" // Can be overridden by query param
	}

	// Extract host
	e.host = serviceURL.Hostname()
	if e.host == "" {
		return fmt.Errorf("missing host in URL")
	}

	// Extract port
	e.port = serviceURL.Port()
	if e.port == "" {
		e.port = "9200" // Default Elasticsearch port
	}

	// Extract index from path
	path := strings.Trim(serviceURL.Path, "/")
	if path != "" {
		e.index = path
	} else {
		e.index = "apprise-notifications" // Default index
	}

	// Extract authentication
	if serviceURL.User != nil {
		e.username = serviceURL.User.Username()
		e.password, _ = serviceURL.User.Password()
	}

	// Parse query parameters
	query := serviceURL.Query()

	// API key authentication (preferred)
	if apiKey := query.Get("apikey"); apiKey != "" {
		e.apiKey = apiKey
	}

	// Secure connection
	if query.Get("secure") == "true" || query.Get("ssl") == "true" {
		e.scheme = "https"
	}

	return nil
}

// Send sends a notification by indexing it as a document in Elasticsearch
func (e *ElasticsearchService) Send(ctx context.Context, req NotificationRequest) error {
	// Build document
	doc := e.buildDocument(req)

	// Marshal to JSON
	payload, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Build URL for indexing
	indexURL := fmt.Sprintf("%s://%s:%s/%s/_doc", e.scheme, e.host, e.port, e.index)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", indexURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set authentication
	if e.apiKey != "" {
		// API key authentication
		httpReq.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", e.apiKey))
	} else if e.username != "" {
		// Basic authentication
		httpReq.SetBasicAuth(e.username, e.password)
	}

	// Send request
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp ElasticsearchErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error.Reason != "" {
			return fmt.Errorf("elasticsearch error: %s (type: %s, status: %d)",
				errorResp.Error.Reason, errorResp.Error.Type, resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var indexResp ElasticsearchIndexResponse
	if err := json.NewDecoder(resp.Body).Decode(&indexResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Verify successful indexing
	if indexResp.Shards.Failed > 0 {
		return fmt.Errorf("document indexed but %d shard(s) failed", indexResp.Shards.Failed)
	}

	return nil
}

// buildDocument constructs an Elasticsearch document from the notification request
func (e *ElasticsearchService) buildDocument(req NotificationRequest) ElasticsearchDocument {
	doc := ElasticsearchDocument{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Title:      req.Title,
		Message:    req.Body,
		NotifyType: req.NotifyType.String(),
		Severity:   e.mapNotifyTypeToSeverity(req.NotifyType),
		Tags:       req.Tags,
		Source:     "apprise-go",
	}

	// Add default title if missing
	if doc.Title == "" && doc.Message != "" {
		// Use first line of message as title
		lines := strings.Split(doc.Message, "\n")
		doc.Title = lines[0]
		if len(doc.Title) > 100 {
			doc.Title = doc.Title[:100] + "..."
		}
	}

	// Add default message if missing
	if doc.Message == "" {
		doc.Message = "Notification from apprise-go"
	}

	return doc
}

// mapNotifyTypeToSeverity maps notification types to Elasticsearch severity levels
func (e *ElasticsearchService) mapNotifyTypeToSeverity(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "error"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeSuccess:
		return "success"
	default:
		return "info"
	}
}

// TestURL validates the Elasticsearch service URL format
func (e *ElasticsearchService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return e.ParseURL(parsedURL)
}

// SupportsAttachments returns false as Elasticsearch doesn't support attachments
// Note: Binary data can be base64-encoded and stored, but not recommended
func (e *ElasticsearchService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns 32768 (32KB reasonable limit for log messages)
func (e *ElasticsearchService) GetMaxBodyLength() int {
	return 32768
}
