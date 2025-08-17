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

// JiraService implements Jira issue tracking notifications
type JiraService struct {
	serverURL    string // Jira server URL (cloud or self-hosted)
	username     string // Jira username or email
	apiToken     string // Jira API token or password
	projectKey   string // Jira project key (optional, for creating issues)
	issueType    string // Default issue type (Bug, Task, Story, etc.)
	webhookURL   string // Webhook proxy URL for secure credential management
	proxyAPIKey  string // API key for webhook authentication
	priority     string // Default priority (Highest, High, Medium, Low, Lowest)
	labels       []string // Default labels to apply to issues
	components   []string // Default components to assign
	client       *http.Client
}

// JiraIssue represents a Jira issue
type JiraIssue struct {
	Key    string                 `json:"key,omitempty"`
	ID     string                 `json:"id,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

// JiraComment represents a Jira comment
type JiraComment struct {
	Body   string                 `json:"body"`
	Author map[string]interface{} `json:"author,omitempty"`
}

// JiraWebhookPayload represents webhook proxy payload
type JiraWebhookPayload struct {
	Service     string       `json:"service"`
	ServerURL   string       `json:"server_url"`
	ProjectKey  string       `json:"project_key,omitempty"`
	Issue       *JiraIssue   `json:"issue,omitempty"`
	Comment     *JiraComment `json:"comment,omitempty"`
	Action      string       `json:"action"` // "create_issue", "add_comment", "update_issue"
	Timestamp   string       `json:"timestamp"`
	Source      string       `json:"source"`
	Version     string       `json:"version"`
}

// NewJiraService creates a new Jira service instance
func NewJiraService() Service {
	return &JiraService{
		client:    GetCloudHTTPClient("jira"),
		issueType: "Task", // Default issue type
		priority:  "Medium", // Default priority
	}
}

// GetServiceID returns the service identifier
func (j *JiraService) GetServiceID() string {
	return "jira"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (j *JiraService) GetDefaultPort() int {
	return 443
}

// ParseURL parses a Jira service URL
// Format: jira://username:token@company.atlassian.net/project_key?issue_type=Bug&priority=High&labels=urgent,api
// Format: jira://username:token@jira.company.com/project_key?issue_type=Story&components=backend,api
// Format: jira://proxy-key@webhook.example.com/jira?username=user&token=token&server_url=https://company.atlassian.net&project_key=PROJ
func (j *JiraService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "jira" {
		return fmt.Errorf("invalid scheme: expected 'jira', got '%s'", serviceURL.Scheme)
	}

	query := serviceURL.Query()

	// Check if this is a webhook proxy URL
	if strings.Contains(serviceURL.Host, "webhook") || strings.Contains(serviceURL.Path, "webhook") || strings.Contains(serviceURL.Path, "/jira") {
		// Webhook proxy mode
		scheme := "https"
		if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
			scheme = "http"
		}
		j.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

		// Extract proxy API key from user info
		if serviceURL.User != nil {
			j.proxyAPIKey = serviceURL.User.Username()
		}

		// Get Jira credentials from query parameters
		j.username = query.Get("username")
		if j.username == "" {
			return fmt.Errorf("username parameter is required for webhook mode")
		}

		j.apiToken = query.Get("token")
		if j.apiToken == "" {
			return fmt.Errorf("token parameter is required for webhook mode")
		}

		// Get server URL
		serverURL := query.Get("server_url")
		if serverURL == "" {
			return fmt.Errorf("server_url parameter is required for webhook mode")
		}
		j.serverURL = serverURL

		// Get project key
		j.projectKey = query.Get("project_key")
	} else {
		// Direct Jira API mode
		if serviceURL.User == nil {
			return fmt.Errorf("authentication required: username and token must be provided")
		}

		j.username = serviceURL.User.Username()
		if j.username == "" {
			return fmt.Errorf("jira username is required")
		}

		if token, hasToken := serviceURL.User.Password(); hasToken {
			j.apiToken = token
		}
		if j.apiToken == "" {
			return fmt.Errorf("jira API token is required")
		}

		// Build server URL from host
		if serviceURL.Host != "" {
			scheme := "https"
			if strings.Contains(serviceURL.Host, "127.0.0.1") || strings.Contains(serviceURL.Host, "localhost") {
				scheme = "http"
			}
			j.serverURL = fmt.Sprintf("%s://%s", scheme, serviceURL.Host)
		}

		// Extract project key from path
		if serviceURL.Path != "" && serviceURL.Path != "/" {
			j.projectKey = strings.TrimPrefix(serviceURL.Path, "/")
		}
	}

	// Parse optional parameters
	if issueType := query.Get("issue_type"); issueType != "" {
		j.issueType = issueType
	}

	if priority := query.Get("priority"); priority != "" {
		if j.isValidPriority(priority) {
			j.priority = priority
		} else {
			return fmt.Errorf("invalid priority: %s (valid: Highest, High, Medium, Low, Lowest)", priority)
		}
	}

	// Parse labels
	if labelsStr := query.Get("labels"); labelsStr != "" {
		j.labels = strings.Split(labelsStr, ",")
		for i, label := range j.labels {
			j.labels[i] = strings.TrimSpace(label)
		}
	}

	// Parse components
	if componentsStr := query.Get("components"); componentsStr != "" {
		j.components = strings.Split(componentsStr, ",")
		for i, component := range j.components {
			j.components[i] = strings.TrimSpace(component)
		}
	}

	return nil
}

// Send sends a notification to Jira (creates an issue or adds a comment)
func (j *JiraService) Send(ctx context.Context, req NotificationRequest) error {
	// Decide whether to create an issue or add a comment
	action := "create_issue"
	if req.URL != "" && strings.Contains(req.URL, "/browse/") {
		// If URL contains a Jira issue key, add a comment instead
		action = "add_comment"
	}

	if action == "create_issue" {
		return j.createIssue(ctx, req)
	} else {
		return j.addComment(ctx, req)
	}
}

// createIssue creates a new Jira issue
func (j *JiraService) createIssue(ctx context.Context, req NotificationRequest) error {
	issue := j.buildIssue(req)

	if j.webhookURL != "" {
		// Send via webhook proxy
		return j.sendViaWebhook(ctx, issue, nil, "create_issue")
	} else {
		// Send directly to Jira API
		return j.sendIssueDirectly(ctx, issue)
	}
}

// addComment adds a comment to an existing Jira issue
func (j *JiraService) addComment(ctx context.Context, req NotificationRequest) error {
	comment := j.buildComment(req)
	
	if j.webhookURL != "" {
		// Send via webhook proxy
		return j.sendViaWebhook(ctx, nil, comment, "add_comment")
	} else {
		// Extract issue key from URL and send directly
		issueKey := j.extractIssueKeyFromURL(req.URL)
		if issueKey == "" {
			return fmt.Errorf("could not extract issue key from URL: %s", req.URL)
		}
		return j.sendCommentDirectly(ctx, issueKey, comment)
	}
}

// buildIssue creates a Jira issue from notification request
func (j *JiraService) buildIssue(req NotificationRequest) *JiraIssue {
	fields := map[string]interface{}{
		"summary":     req.Title,
		"description": req.Body,
		"issuetype": map[string]interface{}{
			"name": j.issueType,
		},
	}

	// Add project if specified
	if j.projectKey != "" {
		fields["project"] = map[string]interface{}{
			"key": j.projectKey,
		}
	}

	// Add priority
	if j.priority != "" {
		fields["priority"] = map[string]interface{}{
			"name": j.priority,
		}
	}

	// Add labels
	allLabels := make([]string, 0, len(j.labels)+len(req.Tags)+1)
	allLabels = append(allLabels, j.labels...)
	allLabels = append(allLabels, req.Tags...)
	allLabels = append(allLabels, "apprise-go") // Add source identifier
	if len(allLabels) > 0 {
		fields["labels"] = allLabels
	}

	// Add components
	if len(j.components) > 0 {
		components := make([]map[string]interface{}, len(j.components))
		for i, component := range j.components {
			components[i] = map[string]interface{}{
				"name": component,
			}
		}
		fields["components"] = components
	}

	// Add attachment info as custom field or in description
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		attachmentInfo := fmt.Sprintf("\n\nAttachments (%d):", len(attachments))
		for _, attachment := range attachments {
			attachmentInfo += fmt.Sprintf("\n- %s (%s)", attachment.GetName(), attachment.GetMimeType())
		}
		fields["description"] = fmt.Sprintf("%s%s", req.Body, attachmentInfo)
	}

	return &JiraIssue{
		Fields: fields,
	}
}

// buildComment creates a Jira comment from notification request  
func (j *JiraService) buildComment(req NotificationRequest) *JiraComment {
	body := fmt.Sprintf("*%s*\n\n%s", req.Title, req.Body)

	// Add tags as labels in comment
	if len(req.Tags) > 0 {
		body += fmt.Sprintf("\n\nTags: %s", strings.Join(req.Tags, ", "))
	}

	// Add attachment info
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments := req.AttachmentMgr.GetAll()
		body += fmt.Sprintf("\n\nAttachments (%d):", len(attachments))
		for _, attachment := range attachments {
			body += fmt.Sprintf("\n- %s (%s)", attachment.GetName(), attachment.GetMimeType())
		}
	}

	body += "\n\n_Created by Apprise-Go_"

	return &JiraComment{
		Body: body,
	}
}

// sendViaWebhook sends data via webhook proxy
func (j *JiraService) sendViaWebhook(ctx context.Context, issue *JiraIssue, comment *JiraComment, action string) error {
	payload := JiraWebhookPayload{
		Service:    "jira",
		ServerURL:  j.serverURL,
		ProjectKey: j.projectKey,
		Issue:      issue,
		Comment:    comment,
		Action:     action,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Source:     "apprise-go",
		Version:    GetVersion(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Jira webhook payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", j.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Jira webhook request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())

	if j.proxyAPIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", j.proxyAPIKey))
		httpReq.Header.Set("X-API-Key", j.proxyAPIKey)
	}

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Jira webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendIssueDirectly sends an issue directly to Jira API
func (j *JiraService) sendIssueDirectly(ctx context.Context, issue *JiraIssue) error {
	issuesURL := fmt.Sprintf("%s/rest/api/3/issue", j.serverURL)

	jsonData, err := json.Marshal(issue)
	if err != nil {
		return fmt.Errorf("failed to marshal issue: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", issuesURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create issue request: %w", err)
	}

	j.setAuthHeaders(httpReq)

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send issue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendCommentDirectly sends a comment directly to Jira API
func (j *JiraService) sendCommentDirectly(ctx context.Context, issueKey string, comment *JiraComment) error {
	commentURL := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", j.serverURL, issueKey)

	jsonData, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", commentURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create comment request: %w", err)
	}

	j.setAuthHeaders(httpReq)

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper methods

func (j *JiraService) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())
	req.Header.Set("Accept", "application/json")
	
	// Use basic authentication
	req.SetBasicAuth(j.username, j.apiToken)
}

func (j *JiraService) isValidPriority(priority string) bool {
	validPriorities := []string{"Highest", "High", "Medium", "Low", "Lowest"}
	for _, valid := range validPriorities {
		if strings.EqualFold(priority, valid) {
			return true
		}
	}
	return false
}

func (j *JiraService) extractIssueKeyFromURL(urlStr string) string {
	// Extract issue key from URLs like:
	// https://company.atlassian.net/browse/PROJ-123
	// https://jira.company.com/browse/PROJ-456
	if strings.Contains(urlStr, "/browse/") {
		parts := strings.Split(urlStr, "/browse/")
		if len(parts) > 1 {
			// Take everything after /browse/ until next slash or end
			keyPart := strings.Split(parts[1], "/")[0]
			keyPart = strings.Split(keyPart, "?")[0] // Remove query params
			keyPart = strings.Split(keyPart, "#")[0] // Remove fragments
			return keyPart
		}
	}
	return ""
}

// TestURL validates a Jira service URL
func (j *JiraService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return j.ParseURL(parsedURL)
}

// SupportsAttachments returns true for Jira (supports metadata in issues/comments)
func (j *JiraService) SupportsAttachments() bool {
	return true // Jira supports attachment metadata in issues and comments
}

// GetMaxBodyLength returns Jira's content length limit
func (j *JiraService) GetMaxBodyLength() int {
	return 32768 // Jira supports large descriptions (32KB)
}

// Example usage and URL formats:
// jira://username:token@company.atlassian.net/PROJECT?issue_type=Bug&priority=High&labels=urgent,api
// jira://username:token@jira.company.com/PROJ?issue_type=Story&components=backend,api
// jira://proxy-key@webhook.example.com/jira?username=user&token=token&server_url=https://company.atlassian.net&project_key=PROJ