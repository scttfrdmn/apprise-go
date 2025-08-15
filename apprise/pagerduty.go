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
)

// PagerDutyService implements PagerDuty Events API v2 notifications
type PagerDutyService struct {
	integrationKey string
	region         string // "us" or "eu"
	source         string
	component      string
	group          string
	class          string
	client         *http.Client
}

// NewPagerDutyService creates a new PagerDuty service instance
func NewPagerDutyService() Service {
	return &PagerDutyService{
		client: &http.Client{},
		region: "us", // Default to US region
	}
}

// GetServiceID returns the service identifier
func (p *PagerDutyService) GetServiceID() string {
	return "pagerduty"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (p *PagerDutyService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a PagerDuty service URL
// Format: pagerduty://integration_key@region?source=source&component=component
// Format: pagerduty://integration_key (defaults to US region)
func (p *PagerDutyService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "pagerduty" {
		return fmt.Errorf("invalid scheme: expected 'pagerduty', got '%s'", serviceURL.Scheme)
	}

	// Extract integration key from user info or host
	if serviceURL.User != nil {
		p.integrationKey = serviceURL.User.Username()
		// Region can be specified in the host when using user@host format
		if serviceURL.Host != "" {
			p.region = serviceURL.Host
		}
	} else {
		// Integration key in host
		p.integrationKey = serviceURL.Host
	}

	if p.integrationKey == "" {
		return fmt.Errorf("PagerDuty integration key is required")
	}

	// Parse query parameters
	query := serviceURL.Query()

	if region := query.Get("region"); region != "" {
		p.region = strings.ToLower(region)
	}

	// Validate region
	if p.region != "us" && p.region != "eu" {
		return fmt.Errorf("invalid region '%s': must be 'us' or 'eu'", p.region)
	}

	if source := query.Get("source"); source != "" {
		p.source = source
	}

	if component := query.Get("component"); component != "" {
		p.component = component
	}

	if group := query.Get("group"); group != "" {
		p.group = group
	}

	if class := query.Get("class"); class != "" {
		p.class = class
	}

	return nil
}

// PagerDutyPayload represents the PagerDuty Events API v2 payload structure
type PagerDutyPayload struct {
	RoutingKey  string                    `json:"routing_key"`
	EventAction string                    `json:"event_action"`
	Client      string                    `json:"client,omitempty"`
	Payload     PagerDutyPayloadDetails   `json:"payload"`
	Links       []PagerDutyLink          `json:"links,omitempty"`
	Images      []PagerDutyImage         `json:"images,omitempty"`
}

// PagerDutyPayloadDetails represents the payload details
type PagerDutyPayloadDetails struct {
	Summary       string                 `json:"summary"`
	Source        string                 `json:"source"`
	Severity      string                 `json:"severity"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Group         string                 `json:"group,omitempty"`
	Class         string                 `json:"class,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// PagerDutyLink represents a link in the payload
type PagerDutyLink struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// PagerDutyImage represents an image in the payload
type PagerDutyImage struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

// PagerDutyResponse represents the PagerDuty API response
type PagerDutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}

// Send sends a notification to PagerDuty
func (p *PagerDutyService) Send(ctx context.Context, req NotificationRequest) error {
	apiURL := p.getAPIURL()

	payload := PagerDutyPayload{
		RoutingKey:  p.integrationKey,
		EventAction: "trigger",
		Client:      GetUserAgent(),
		Payload: PagerDutyPayloadDetails{
			Summary:   p.formatSummary(req.Title, req.Body),
			Source:    p.getSource(),
			Severity:  p.mapSeverity(req.NotifyType),
			Component: p.component,
			Group:     p.group,
			Class:     p.class,
		},
	}

	// Add custom details if title is present
	if req.Title != "" {
		payload.Payload.CustomDetails = map[string]interface{}{
			"title": req.Title,
			"body":  req.Body,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PagerDuty payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send PagerDuty notification: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var result PagerDutyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse PagerDuty response: %w", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PagerDuty API error (status %d): %s", resp.StatusCode, result.Message)
	}

	if result.Status != "success" {
		return fmt.Errorf("PagerDuty API error: %s", result.Message)
	}

	return nil
}

// getAPIURL returns the appropriate API URL based on region
func (p *PagerDutyService) getAPIURL() string {
	switch p.region {
	case "eu":
		return "https://events.eu.pagerduty.com/v2/enqueue"
	default: // "us"
		return "https://events.pagerduty.com/v2/enqueue"
	}
}

// getSource returns the source for the alert
func (p *PagerDutyService) getSource() string {
	if p.source != "" {
		return p.source
	}
	return "apprise-go"
}

// formatSummary formats the title and body into a summary
func (p *PagerDutyService) formatSummary(title, body string) string {
	if title != "" {
		return title
	}
	if body != "" {
		// Truncate body if too long for summary
		if len(body) > 1024 {
			return body[:1021] + "..."
		}
		return body
	}
	return "Alert from Apprise-Go"
}

// mapSeverity maps NotifyType to PagerDuty severity levels
func (p *PagerDutyService) mapSeverity(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "info"
	case NotifyTypeInfo:
		return "info"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeError:
		return "error"
	default:
		return "error"
	}
}

// TestURL validates a PagerDuty service URL
func (p *PagerDutyService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return p.ParseURL(parsedURL)
}

// SupportsAttachments returns false since PagerDuty Events API doesn't support file attachments
func (p *PagerDutyService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns PagerDuty's summary length limit
func (p *PagerDutyService) GetMaxBodyLength() int {
	return 1024 // PagerDuty summary field limit
}

// Example usage and URL formats:
// pagerduty://integration_key
// pagerduty://integration_key@us
// pagerduty://integration_key@eu  
// pagerduty://integration_key?region=eu&source=monitoring&component=api
// pagerduty://integration_key?source=server-01&component=database&group=production