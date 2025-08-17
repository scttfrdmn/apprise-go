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

// GitHubService implements GitHub webhook notifications
type GitHubService struct {
	token        string   // GitHub personal access token or app token
	owner        string   // Repository owner (user or organization)
	repo         string   // Repository name
	webhookURL   string   // Webhook proxy URL for secure credential management
	proxyAPIKey  string   // API key for webhook authentication
	eventTypes   []string // Event types to filter (push, pull_request, issues, etc.)
	branches     []string // Branch filters for push events
	labels       []string // Label filters for issues/PRs
	client       *http.Client
}

// GitHubEvent represents a GitHub webhook event
type GitHubEvent struct {
	EventType    string                 `json:"event_type"`
	Action       string                 `json:"action,omitempty"`
	Repository   map[string]interface{} `json:"repository,omitempty"`
	Sender       map[string]interface{} `json:"sender,omitempty"`
	Organization map[string]interface{} `json:"organization,omitempty"`
	Commits      []map[string]interface{} `json:"commits,omitempty"`
	PullRequest  map[string]interface{} `json:"pull_request,omitempty"`
	Issue        map[string]interface{} `json:"issue,omitempty"`
	Release      map[string]interface{} `json:"release,omitempty"`
	CheckRun     map[string]interface{} `json:"check_run,omitempty"`
	CheckSuite   map[string]interface{} `json:"check_suite,omitempty"`
	Deployment   map[string]interface{} `json:"deployment,omitempty"`
	WorkflowRun  map[string]interface{} `json:"workflow_run,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
	Ref          string                 `json:"ref,omitempty"`
	Branch       string                 `json:"branch,omitempty"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	URL          string                 `json:"url,omitempty"`
	State        string                 `json:"state,omitempty"`
	Labels       []string               `json:"labels,omitempty"`
	Assignees    []map[string]interface{} `json:"assignees,omitempty"`
	Reviewers    []map[string]interface{} `json:"reviewers,omitempty"`
	Author       map[string]interface{} `json:"author,omitempty"`
}

// GitHubWebhookPayload represents webhook proxy payload
type GitHubWebhookPayload struct {
	Service     string        `json:"service"`
	Owner       string        `json:"owner"`
	Repository  string        `json:"repository"`
	Event       *GitHubEvent  `json:"event"`
	Timestamp   string        `json:"timestamp"`
	Source      string        `json:"source"`
	Version     string        `json:"version"`
}

// NewGitHubService creates a new GitHub service instance
func NewGitHubService() Service {
	return &GitHubService{
		client:     GetCloudHTTPClient("github"),
		eventTypes: []string{"push", "pull_request", "issues", "release"}, // Default events
	}
}

// GetServiceID returns the service identifier
func (g *GitHubService) GetServiceID() string {
	return "github"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (g *GitHubService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a GitHub service URL
// Format: github://token@github.com/owner/repo?events=push,pr,issues&branches=main,develop
// Format: github://token@github.enterprise.com/owner/repo?events=release&labels=bug,feature
// Format: github://proxy-key@webhook.example.com/github?token=gh_token&owner=user&repo=myrepo&events=all
func (g *GitHubService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "github" {
		return fmt.Errorf("invalid scheme: expected 'github', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/github") {
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

		// Get GitHub token from query parameters
		g.token = query.Get("token")
		if g.token == "" {
			return fmt.Errorf("token parameter is required for webhook mode")
		}

		// Get owner and repo from query parameters
		g.owner = query.Get("owner")
		if g.owner == "" {
			return fmt.Errorf("owner parameter is required")
		}

		g.repo = query.Get("repo")
		if g.repo == "" {
			return fmt.Errorf("repo parameter is required")
		}
	} else {
		// Direct GitHub API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: token must be provided")
		}

		g.token = serviceURL.User.Username()
		if g.token == "" {
			return fmt.Errorf("gitHub token is required")
		}

		// Extract owner and repo from path
		pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
		if len(pathParts) < 2 {
			return fmt.Errorf("owner and repo are required (specify as path: github://token@host/owner/repo)")
		}

		g.owner = pathParts[0]
		g.repo = pathParts[1]
	}

	// Parse event types
	if events := query.Get("events"); events != "" {
		if events == "all" {
			g.eventTypes = []string{"push", "pull_request", "issues", "issue_comment", "release", "create", "delete", "fork", "watch", "star", "check_run", "check_suite", "deployment", "workflow_run"}
		} else {
			g.eventTypes = strings.Split(events, ",")
			for i, event := range g.eventTypes {
				g.eventTypes[i] = strings.TrimSpace(event)
				// Handle common aliases
				switch g.eventTypes[i] {
				case "pr":
					g.eventTypes[i] = "pull_request"
				case "issue":
					g.eventTypes[i] = "issues"
				case "workflow":
					g.eventTypes[i] = "workflow_run"
				case "deploy":
					g.eventTypes[i] = "deployment"
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

// Send sends a notification to GitHub (creates an issue comment or discussion)
func (g *GitHubService) Send(ctx context.Context, req NotificationRequest) error {
	// Create GitHub event from notification
	event := g.createEvent(req)

	if g.webhookURL != "" {
		// Send via webhook proxy
		return g.sendViaWebhook(ctx, event)
	} else {
		// Send directly to GitHub API (create issue comment or repository comment)
		return g.sendDirectly(ctx, event)
	}
}

// createEvent creates a GitHub event from notification request
func (g *GitHubService) createEvent(req NotificationRequest) *GitHubEvent {
	event := &GitHubEvent{
		EventType:   "notification",
		Action:      "created",
		Title:       req.Title,
		Description: req.Body,
		Timestamp:   time.Now().Unix(),
		State:       g.getStateForNotifyType(req.NotifyType),
		Labels:      req.Tags,
	}

	// Add repository information
	event.Repository = map[string]interface{}{
		"name":      g.repo,
		"full_name": fmt.Sprintf("%s/%s", g.owner, g.repo),
		"owner": map[string]interface{}{
			"login": g.owner,
		},
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
func (g *GitHubService) sendViaWebhook(ctx context.Context, event *GitHubEvent) error {
	payload := GitHubWebhookPayload{
		Service:    "github",
		Owner:      g.owner,
		Repository: g.repo,
		Event:      event,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Source:     "apprise-go",
		Version:    GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GitHub webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create GitHub webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if g.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", g.proxyAPIKey)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send GitHub webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitHub webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendDirectly sends data directly to GitHub API
func (g *GitHubService) sendDirectly(ctx context.Context, event *GitHubEvent) error {
	// Try to create a repository dispatch event or commit comment
	// First, try to create a repository dispatch event (preferred for notifications)
	dispatchURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/dispatches", g.owner, g.repo)

	dispatchData := map[string]interface{}{
		"event_type": "apprise-notification",
		"client_payload": map[string]interface{}{
			"title":       event.Title,
			"description": event.Description,
			"state":       event.State,
			"labels":      event.Labels,
			"timestamp":   event.Timestamp,
			"source":      "apprise-go",
		},
	}

	jsonData, err := json.Marshal(dispatchData)
	if err != nil {
		return fmt.Errorf("failed to marshal dispatch data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", dispatchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create dispatch request: %w", err)
	}

	g.setAuthHeaders(httpReq)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send dispatch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (g *GitHubService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	// Support both personal access tokens and GitHub App tokens
	if strings.HasPrefix(g.token, "ghp_") || strings.HasPrefix(g.token, "gho_") || strings.HasPrefix(g.token, "ghu_") {
		// Personal access token
		req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	} else if strings.HasPrefix(g.token, "ghs_") {
		// GitHub App installation token
		req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	} else {
		// Assume it's a personal access token
		req.Header.Set("Authorization", fmt.Sprintf("token %s", g.token))
	}
}

func (g *GitHubService) getStateForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeError:
		return "failure"
	case NotifyTypeWarning:
		return "pending"
	case NotifyTypeSuccess:
		return "success"
	case NotifyTypeInfo:
		fallthrough
	default:
		return "pending"
	}
}

// matchesEventFilter checks if an event matches the configured filters
func (g *GitHubService) matchesEventFilter(eventType string) bool {
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
func (g *GitHubService) matchesBranchFilter(branch string) bool {
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
func (g *GitHubService) matchesLabelFilter(labels []string) bool {
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

// TestURL validates a GitHub service URL
func (g *GitHubService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return g.ParseURL(parsedURL)
}

// SupportsAttachments returns true for GitHub (supports metadata in webhook events)
func (g *GitHubService) SupportsAttachments() bool {
	return true // GitHub supports attachment metadata in webhook events and dispatch payloads
}

// GetMaxBodyLength returns GitHub's content length limit
func (g *GitHubService) GetMaxBodyLength() int {
	return 65536 // GitHub supports large content (64KB) in repository dispatch payloads
}

// Example usage and URL formats:
// github://token@github.com/owner/repo?events=push,pr,issues&branches=main,develop
// github://token@github.enterprise.com/owner/repo?events=release&labels=bug,feature
// github://proxy-key@webhook.example.com/github?token=gh_token&owner=user&repo=myrepo&events=all