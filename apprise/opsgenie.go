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

// OpsgenieService implements Opsgenie alerting and incident management
type OpsgenieService struct {
	apiKey     string
	region     string // us, eu
	targets    []string
	tags       []string
	teams      []string
	priority   string // P1, P2, P3, P4, P5
	alias      string
	entity     string
	source     string
	user       string
	note       string
	client     *http.Client
}

// NewOpsgenieService creates a new Opsgenie service instance
func NewOpsgenieService() Service {
	return &OpsgenieService{
		client: &http.Client{},
		region: "us", // Default to US region
	}
}

// GetServiceID returns the service identifier
func (o *OpsgenieService) GetServiceID() string {
	return "opsgenie"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (o *OpsgenieService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an Opsgenie service URL
// Format: opsgenie://api_key@region/target1/target2
// Format: opsgenie://api_key@region
func (o *OpsgenieService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "opsgenie" {
		return fmt.Errorf("invalid scheme: expected 'opsgenie', got '%s'", serviceURL.Scheme)
	}

	// Extract API key from user info or host (if no @ in URL)
	if serviceURL.User != nil {
		o.apiKey = serviceURL.User.Username()
		if o.apiKey == "" {
			return fmt.Errorf("opsgenie API key is required")
		}
		
		// Extract region from host (optional, defaults to 'us')
		if serviceURL.Host != "" {
			region := strings.ToLower(serviceURL.Host)
			if region != "us" && region != "eu" {
				return fmt.Errorf("invalid opsgenie region: must be 'us' or 'eu', got '%s'", region)
			}
			o.region = region
		}
	} else if serviceURL.Host != "" {
		// No @ in URL, so host contains the API key
		o.apiKey = serviceURL.Host
		// Region stays default (us)
	} else {
		return fmt.Errorf("opsgenie API key is required")
	}

	// Extract targets from path
	if serviceURL.Path != "" && serviceURL.Path != "/" {
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		for _, part := range pathParts {
			if part != "" {
				o.targets = append(o.targets, part)
			}
		}
	}

	// Parse query parameters
	query := serviceURL.Query()

	if region := query.Get("region"); region != "" {
		region = strings.ToLower(region)
		if region != "us" && region != "eu" {
			return fmt.Errorf("invalid opsgenie region: must be 'us' or 'eu', got '%s'", region)
		}
		o.region = region
	}

	if tags := query.Get("tags"); tags != "" {
		o.tags = strings.Split(tags, ",")
		// Trim whitespace from tags
		for i, tag := range o.tags {
			o.tags[i] = strings.TrimSpace(tag)
		}
	}

	if teams := query.Get("teams"); teams != "" {
		o.teams = strings.Split(teams, ",")
		// Trim whitespace from teams
		for i, team := range o.teams {
			o.teams[i] = strings.TrimSpace(team)
		}
	}

	if priority := query.Get("priority"); priority != "" {
		priority = strings.ToUpper(priority)
		if !isValidOpsgeniePriority(priority) {
			return fmt.Errorf("invalid opsgenie priority: must be P1-P5, got '%s'", priority)
		}
		o.priority = priority
	}

	if alias := query.Get("alias"); alias != "" {
		o.alias = alias
	}

	if entity := query.Get("entity"); entity != "" {
		o.entity = entity
	}

	if source := query.Get("source"); source != "" {
		o.source = source
	}

	if user := query.Get("user"); user != "" {
		o.user = user
	}

	if note := query.Get("note"); note != "" {
		o.note = note
	}

	return nil
}

// isValidOpsgeniePriority validates Opsgenie priority levels
func isValidOpsgeniePriority(priority string) bool {
	validPriorities := []string{"P1", "P2", "P3", "P4", "P5"}
	for _, valid := range validPriorities {
		if priority == valid {
			return true
		}
	}
	return false
}

// OpsgenieAlert represents an Opsgenie alert payload
type OpsgenieAlert struct {
	Message     string                 `json:"message"`
	Alias       string                 `json:"alias,omitempty"`
	Description string                 `json:"description,omitempty"`
	Responders  []OpsgenieResponder    `json:"responders,omitempty"`
	VisibleTo   []OpsgenieResponder    `json:"visibleTo,omitempty"`
	Actions     []string               `json:"actions,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Entity      string                 `json:"entity,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	User        string                 `json:"user,omitempty"`
	Note        string                 `json:"note,omitempty"`
}

// OpsgenieResponder represents a responder (user, team, escalation, schedule)
type OpsgenieResponder struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// OpsgenieResponse represents the API response
type OpsgenieResponse struct {
	Result  string `json:"result"`
	Took    int    `json:"took"`
	Request string `json:"requestId"`
}

// Send sends an alert to Opsgenie
func (o *OpsgenieService) Send(ctx context.Context, req NotificationRequest) error {
	alert := OpsgenieAlert{
		Message:     req.Title,
		Description: req.Body,
		Alias:       o.alias,
		Entity:      o.entity,
		Source:      o.source,
		Priority:    o.priority,
		User:        o.user,
		Note:        o.note,
		Tags:        o.tags,
	}

	// If no explicit priority, map from notification type
	if o.priority == "" {
		alert.Priority = o.mapNotifyTypeToPriority(req.NotifyType)
	}

	// If no explicit source, use default
	if o.source == "" {
		alert.Source = "apprise-go"
	}

	// Add notification type as tag if no custom tags
	if len(o.tags) == 0 {
		alert.Tags = []string{req.NotifyType.String()}
	}

	// Add responders from targets
	if len(o.targets) > 0 {
		for _, target := range o.targets {
			// Determine responder type based on target format
			if strings.Contains(target, "@") {
				// Email address - user
				alert.Responders = append(alert.Responders, OpsgenieResponder{
					Type: "user",
					Name: target,
				})
			} else {
				// Assume it's a team name
				alert.Responders = append(alert.Responders, OpsgenieResponder{
					Type: "team",
					Name: target,
				})
			}
		}
	}

	// Add teams from query parameter
	for _, team := range o.teams {
		alert.Responders = append(alert.Responders, OpsgenieResponder{
			Type: "team",
			Name: team,
		})
	}

	// Add notification details
	alert.Details = map[string]interface{}{
		"notifyType": req.NotifyType.String(),
		"timestamp":  fmt.Sprintf("%d", ctx.Value("timestamp")),
	}

	jsonData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal Opsgenie alert: %w", err)
	}

	// Build API URL based on region
	apiURL := o.getAPIURL()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "GenieKey "+o.apiKey)
	httpReq.Header.Set("User-Agent", GetUserAgent())

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Opsgenie alert: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response for error details
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("opsgenie API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// getAPIURL returns the appropriate API URL based on region
func (o *OpsgenieService) getAPIURL() string {
	if o.region == "eu" {
		return "https://api.eu.opsgenie.com/v2/alerts"
	}
	return "https://api.opsgenie.com/v2/alerts"
}

// mapNotifyTypeToPriority maps notification types to Opsgenie priority levels
func (o *OpsgenieService) mapNotifyTypeToPriority(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "P1" // Critical
	case NotifyTypeWarning:
		return "P2" // High
	case NotifyTypeSuccess:
		return "P4" // Low
	default:
		return "P3" // Moderate (info)
	}
}

// TestURL validates an Opsgenie service URL
func (o *OpsgenieService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return o.ParseURL(parsedURL)
}

// SupportsAttachments returns false since Opsgenie doesn't support file attachments in alerts
func (o *OpsgenieService) SupportsAttachments() bool {
	return false
}

// GetMaxBodyLength returns Opsgenie's description length limit
func (o *OpsgenieService) GetMaxBodyLength() int {
	return 15000 // Opsgenie description limit is 15000 characters
}

// Example usage and URL formats:
// opsgenie://api_key@us
// opsgenie://api_key@eu/team-backend/user@example.com
// opsgenie://api_key@us?priority=P1&tags=critical,production
// opsgenie://api_key@us?teams=devops,backend&priority=P2&entity=web-server&source=monitoring
// opsgenie://api_key@us/oncall-team?alias=db-alert&note=Database%20performance%20issue