package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GitLabService implements GitLab webhook notifications
type GitLabService struct {
	serverURL    string // GitLab server URL (gitlab.com or self-hosted)
	token        string // GitLab personal access token or webhook token
	projectID    string // GitLab project ID
	webhookURL   string // Webhook proxy URL for secure credential management
	proxyAPIKey  string // API key for webhook authentication
	eventTypes   []string // Event types to filter (push, merge_request, issue, etc.)
	branches     []string // Branch filters for push events
	labels       []string // Label filters for issues/MRs
	client       *http.Client
}

// GitLabEvent represents a GitLab webhook event
type GitLabEvent struct {
	EventType    string                 `json:"event_type"`
	ObjectKind   string                 `json:"object_kind,omitempty"`
	ProjectID    int                    `json:"project_id,omitempty"`
	ProjectName  string                 `json:"project_name,omitempty"`
	ProjectPath  string                 `json:"project_path,omitempty"`
	Repository   map[string]interface{} `json:"repository,omitempty"`
	User         map[string]interface{} `json:"user,omitempty"`
	Commits      []map[string]interface{} `json:"commits,omitempty"`
	MergeRequest map[string]interface{} `json:"merge_request,omitempty"`
	Issue        map[string]interface{} `json:"issue,omitempty"`
	Pipeline     map[string]interface{} `json:"pipeline,omitempty"`
	Build        map[string]interface{} `json:"build,omitempty"`
	Wiki         map[string]interface{} `json:"wiki,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
	Branch       string                 `json:"branch,omitempty"`
	Ref          string                 `json:"ref,omitempty"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	URL          string                 `json:"url,omitempty"`
	State        string                 `json:"state,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Labels       []string               `json:"labels,omitempty"`
	Assignees    []map[string]interface{} `json:"assignees,omitempty"`
	Reviewers    []map[string]interface{} `json:"reviewers,omitempty"`
}

// GitLabWebhookPayload represents webhook proxy payload
type GitLabWebhookPayload struct {
	Service     string       `json:"service"`
	ProjectID   string       `json:"project_id"`
	ServerURL   string       `json:"server_url"`
	Event       *GitLabEvent `json:"event"`
	Timestamp   string       `json:"timestamp"`
	Source      string       `json:"source"`
	Version     string       `json:"version"`
}

// NewGitLabService creates a new GitLab service instance
func NewGitLabService() Service {
	return &GitLabService{
		client:     GetCloudHTTPClient("gitlab"),
		serverURL:  "https://gitlab.com", // Default to GitLab.com
		eventTypes: []string{"push", "merge_request", "issue", "pipeline"}, // Default events
	}
}

// GetServiceID returns the service identifier
func (g *GitLabService) GetServiceID() string {
	return "gitlab"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (g *GitLabService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a GitLab service URL
// Format: gitlab://token@gitlab.com/project_id?events=push,mr,issue&branches=main,develop
// Format: gitlab://token@self-hosted.gitlab.com/project_id?events=pipeline&labels=bug,feature
// Format: gitlab://proxy-key@webhook.example.com/gitlab?token=gl_token&project_id=123&events=all
func (g *GitLabService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "gitlab" {
		return fmt.Errorf("invalid scheme: expected 'gitlab', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/gitlab") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		g.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			g.proxyAPIKey = serviceURL.User.Username()
		}

		// Get GitLab token from query parameters
		g.token = query.Get("token")
		if g.token == "" {
			return fmt.Errorf("token parameter is required for webhook mode")
		}

		// Get project ID from query parameters
		g.projectID = query.Get("project_id")
		if g.projectID == "" {
			return fmt.Errorf("project_id parameter is required")
		}

		// Get server URL if specified
		if serverURL := query.Get("server_url"); serverURL != "" {
			g.serverURL = serverURL
		}
	} else {
		// Direct GitLab API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: token must be provided")
		}

		g.token = serviceURL.User.Username()
		if g.token == "" {
			return fmt.Errorf("gitLab token is required")
		}

		// Set server URL from host
		if serviceURL.Host != "" {
			scheme := "https"
			if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
				scheme = "http"
			}
			g.serverURL = fmt.Sprintf("%s://%s", scheme, serviceURL.Host)
		}

		// Extract project ID from path
		if serviceURL.Path != "" {
			// Remove leading slash and use as project ID
			g.projectID = strings.TrimPrefix(serviceURL.Path, "/")
		}

		if g.projectID == "" {
			return fmt.Errorf("project ID is required (specify as path: gitlab://token@host/project_id)")
		}
	}

	// Parse event types
	if events := query.Get("events"); events != "" {
		if events == "all" {
			g.eventTypes = []string{"push", "tag_push", "merge_request", "issue", "pipeline", "job", "release", "wiki", "deployment"}
		} else {
			g.eventTypes = strings.Split(events, ",")
			for i, event := range g.eventTypes {
				g.eventTypes[i] = strings.TrimSpace(event)
				// Handle common aliases
				switch g.eventTypes[i] {
				case "mr":
					g.eventTypes[i] = "merge_request"
				case "pr":
					g.eventTypes[i] = "merge_request"
				case "ci":
					g.eventTypes[i] = "pipeline"
				}
			}
		}
	}

	// Parse branch filters
	if branches := query.Get("branches"); branches != "" {
		g.branches = strings.Split(branches, ",")
		for i, branch := range g.branches {
			g.branches[i] = strings.TrimSpace(branch)
		}
	}

	// Parse label filters
	if labels := query.Get("labels"); labels != "" {
		g.labels = strings.Split(labels, ",")
		for i, label := range g.labels {
			g.labels[i] = strings.TrimSpace(label)
		}
	}

	return nil
}

// Send sends a notification to GitLab (creates an issue comment or note)
func (g *GitLabService) Send(ctx context.Context, req NotificationRequest) error {
	// Create GitLab event from notification
	event := g.createEvent(req)

	if g.webhookURL != "" {
		// Send via webhook proxy
		return g.sendViaWebhook(ctx, event)
	} else {
		// Send directly to GitLab API (create project note/comment)
		return g.sendDirectly(ctx, event)
	}
}

// createEvent creates a GitLab event from notification request
func (g *GitLabService) createEvent(req NotificationRequest) *GitLabEvent {
	event := &GitLabEvent{
		EventType:   "notification",
		ObjectKind:  "note",
		Title:       req.Title,
		Description: req.Body,
		Timestamp:   time.Now().Unix(),
		State:       g.getStateForNotifyType(req.NotifyType),
		Action:      "created",
		Labels:      req.Tags,
	}

	// Add project information if available
	if g.projectID != "" {
		if projectID, err := strconv.Atoi(g.projectID); err == nil {
			event.ProjectID = projectID
		}
	}

	// Add URL if this is a webhook or external notification
	if req.URL != "" {
		event.URL = req.URL
	}

	// Add attachment info as labels
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		event.Labels = append(event.Labels, fmt.Sprintf("attachments:%d", req.AttachmentMgr.Count()))
	}

	return event
}

// sendViaWebhook sends data via webhook proxy
func (g *GitLabService) sendViaWebhook(ctx context.Context, event *GitLabEvent) error {
	payload := GitLabWebhookPayload{
		Service:   "gitlab",
		ProjectID: g.projectID,
		ServerURL: g.serverURL,
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Source:    "apprise-go",
		Version:   GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GitLab webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GitLab webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if g.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", g.proxyAPIKey)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send GitLab webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitLab webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendDirectly sends data directly to GitLab API
func (g *GitLabService) sendDirectly(ctx context.Context, event *GitLabEvent) error {
	// Create a project note/comment via GitLab API
	noteURL := fmt.Sprintf("%s/api/v4/projects/%s/notes", g.serverURL, g.projectID)

	noteBody := fmt.Sprintf("**%s**\n\n%s", event.Title, event.Description)
	if len(event.Labels) > 0 {
		noteBody += fmt.Sprintf("\n\n*Tags: %s*", strings.Join(event.Labels, ", "))
	}

	noteData := map[string]interface{}{
		"body": noteBody,
	}

	jsonData, err := json.Marshal(noteData)
	if err != nil {
		return fmt.Errorf("failed to marshal note data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", noteURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create note request: %w", err)
	}

	g.setAuthHeaders(httpReq)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send note: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitLab API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (g *GitLabService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("Private-Token", g.token)
}

func (g *GitLabService) getStateForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "failed"
	case NotifyTypeWarning:
		return "warning"
	case NotifyTypeSuccess:
		return "success"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "pending"
	}
}

// matchesEventFilter checks if an event matches the configured filters
func (g *GitLabService) matchesEventFilter(eventType string) bool {
	if len(g.eventTypes) == 0 {
		return true // No filter, allow all
	}

	for _, allowedType := range g.eventTypes {
		if eventType == allowedType {
			return true
		}
	}
	return false
}

// matchesBranchFilter checks if a branch matches the configured filters
func (g *GitLabService) matchesBranchFilter(branch string) bool {
	if len(g.branches) == 0 {
		return true // No filter, allow all
	}

	for _, allowedBranch := range g.branches {
		if branch == allowedBranch || strings.Contains(branch, allowedBranch) {
			return true
		}
	}
	return false
}

// matchesLabelFilter checks if labels match the configured filters
func (g *GitLabService) matchesLabelFilter(labels []string) bool {
	if len(g.labels) == 0 {
		return true // No filter, allow all
	}

	for _, requiredLabel := range g.labels {
		for _, label := range labels {
			if label == requiredLabel {
				return true
			}
		}
	}
	return false
}

// TestURL validates a GitLab service URL
func (g *GitLabService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return g.ParseURL(parsedURL)
}

// SupportsAttachments returns true for GitLab (supports metadata in webhook events)
func (g *GitLabService) SupportsAttachments() bool {
	return true // GitLab supports attachment metadata in webhook events
}

// GetMaxBodyLength returns GitLab's content length limit
func (g *GitLabService) GetMaxBodyLength() int {
	return 16384 // GitLab supports reasonably large content (16KB)
}

// Example usage and URL formats:
// gitlab://token@gitlab.com/project_id?events=push,mr,issue&branches=main,develop
// gitlab://token@self-hosted.gitlab.com/project_id?events=pipeline&labels=bug,feature
// gitlab://proxy-key@webhook.example.com/gitlab?token=gl_token&project_id=123&events=all