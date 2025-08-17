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

// AWSSESService implements Amazon SES notifications via API Gateway webhook
type AWSSESService struct {
	webhookURL   string
	region       string
	fromEmail    string
	fromName     string
	toEmails     []string
	ccEmails     []string
	bccEmails    []string
	replyTo      string
	subject      string
	template     string
	templateData map[string]interface{}
	apiKey       string
	client       *http.Client
}

// NewAWSSESService creates a new AWS SES service instance
func NewAWSSESService() Service {
	return &AWSSESService{
		client:       GetCloudHTTPClient("aws-ses"),
		region:       "us-east-1",
		templateData: make(map[string]interface{}),
	}
}

// GetServiceID returns the service identifier
func (a *AWSSESService) GetServiceID() string {
	return "ses"
}

// GetDefaultPort returns the default port (443 for HTTPS)
func (a *AWSSESService) GetDefaultPort() int {
	return 443
}

// ParseURL parses an AWS SES service URL
// Format: ses://webhook.url/ses-proxy?from=sender@domain.com&to=recipient@domain.com
// Format: ses://api-key@api.gateway.url/prod/ses?from=noreply@company.com&to=admin@company.com,alerts@company.com
// Format: ses://webhook.url/ses?template=alert-template&from=alerts@company.com&to=team@company.com
func (a *AWSSESService) ParseURL(serviceURL *url.URL) error {
	if serviceURL.Scheme != "ses" {
		return fmt.Errorf("invalid scheme: expected 'ses', got '%s'", serviceURL.Scheme)
	}

	// Extract API key from userinfo if provided
	if serviceURL.User != nil {
		a.apiKey = serviceURL.User.Username()
	}

	if serviceURL.Host == "" {
		return fmt.Errorf("webhook host is required")
	}

	// Build webhook URL - use HTTP for testing if specified
	scheme := "https" // Default to HTTPS for production
	if serviceURL.Query().Get("test_mode") == "true" {
		scheme = "http" // Use HTTP for testing
	}
	a.webhookURL = fmt.Sprintf("%s://%s%s", scheme, serviceURL.Host, serviceURL.Path)

	// Parse query parameters
	queryParams := serviceURL.Query()

	// Required: from email
	fromEmail := queryParams.Get("from")
	if fromEmail == "" {
		return fmt.Errorf("from email parameter is required")
	}
	a.fromEmail = fromEmail

	// Optional: from name
	if fromName := queryParams.Get("name"); fromName != "" {
		a.fromName = fromName
	}

	// Required: to email(s)
	toEmails := queryParams.Get("to")
	if toEmails == "" {
		return fmt.Errorf("to email parameter is required")
	}
	a.toEmails = parseEmailList(toEmails)

	// Optional: CC emails
	if ccEmails := queryParams.Get("cc"); ccEmails != "" {
		a.ccEmails = parseEmailList(ccEmails)
	}

	// Optional: BCC emails
	if bccEmails := queryParams.Get("bcc"); bccEmails != "" {
		a.bccEmails = parseEmailList(bccEmails)
	}

	// Optional: reply-to
	if replyTo := queryParams.Get("reply_to"); replyTo != "" {
		a.replyTo = replyTo
	}

	// Optional: custom subject
	if subject := queryParams.Get("subject"); subject != "" {
		a.subject = subject
	}

	// Optional: template name
	if template := queryParams.Get("template"); template != "" {
		a.template = template
	}

	// Optional: region
	if region := queryParams.Get("region"); region != "" {
		a.region = region
	}

	// Parse template data (prefix: data_)
	for key, values := range queryParams {
		if strings.HasPrefix(key, "data_") && len(values) > 0 {
			dataKey := strings.TrimPrefix(key, "data_")
			a.templateData[dataKey] = values[0]
		}
	}

	return nil
}

// parseEmailList splits comma-separated email addresses
func parseEmailList(emailStr string) []string {
	emails := strings.Split(emailStr, ",")
	var result []string
	for _, email := range emails {
		if trimmed := strings.TrimSpace(email); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// TestURL validates the URL format
func (a *AWSSESService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	tempService := NewAWSSESService().(*AWSSESService)
	return tempService.ParseURL(parsedURL)
}

// SupportsAttachments returns whether this service supports attachments
func (a *AWSSESService) SupportsAttachments() bool {
	return true // SES supports attachments
}

// GetMaxBodyLength returns the maximum body length (10MB for SES)
func (a *AWSSESService) GetMaxBodyLength() int {
	return 10 * 1024 * 1024 // 10MB
}

// Send sends a notification via AWS SES webhook
func (a *AWSSESService) Send(ctx context.Context, req NotificationRequest) error {
	// Prepare the email content
	subject := a.subject
	if subject == "" {
		subject = req.Title
		if subject == "" {
			subject = fmt.Sprintf("Notification (%s)", req.NotifyType.String())
		}
	}

	// Create payload for SES webhook
	payload := map[string]interface{}{
		"region":      a.region,
		"source":      a.buildFromEmail(),
		"destination": a.buildDestination(),
		"message":     a.buildMessage(req, subject),
	}

	// Add reply-to if specified
	if a.replyTo != "" {
		payload["replyToAddresses"] = []string{a.replyTo}
	}

	// Add template data if using templates
	if a.template != "" {
		payload["template"] = a.template
		if len(a.templateData) > 0 {
			// Add notification data to template data
			templateData := make(map[string]interface{})
			for k, v := range a.templateData {
				templateData[k] = v
			}
			templateData["title"] = req.Title
			templateData["body"] = req.Body
			templateData["notifyType"] = req.NotifyType.String()
			templateData["timestamp"] = time.Now().UTC().Format(time.RFC3339)

			payload["templateData"] = templateData
		}
	}

	// Add attachments if present
	if req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0 {
		attachments, err := a.processAttachments(req.AttachmentMgr)
		if err != nil {
			return fmt.Errorf("failed to process attachments: %w", err)
		}
		payload["attachments"] = attachments
	}

	return a.sendWebhookRequest(ctx, payload)
}

// buildFromEmail constructs the from email address with optional name
func (a *AWSSESService) buildFromEmail() string {
	if a.fromName != "" {
		return fmt.Sprintf("%s <%s>", a.fromName, a.fromEmail)
	}
	return a.fromEmail
}

// buildDestination constructs the SES destination object
func (a *AWSSESService) buildDestination() map[string]interface{} {
	destination := make(map[string]interface{})

	if len(a.toEmails) > 0 {
		destination["toAddresses"] = a.toEmails
	}

	if len(a.ccEmails) > 0 {
		destination["ccAddresses"] = a.ccEmails
	}

	if len(a.bccEmails) > 0 {
		destination["bccAddresses"] = a.bccEmails
	}

	return destination
}

// buildMessage constructs the SES message object
func (a *AWSSESService) buildMessage(req NotificationRequest, subject string) map[string]interface{} {
	// Don't include message content if using templates
	if a.template != "" {
		return map[string]interface{}{
			"subject": map[string]interface{}{
				"data":    subject,
				"charset": "UTF-8",
			},
		}
	}

	// Format the body content
	htmlBody, textBody := a.formatMessageBody(req.Title, req.Body, req.NotifyType)

	message := map[string]interface{}{
		"subject": map[string]interface{}{
			"data":    subject,
			"charset": "UTF-8",
		},
		"body": map[string]interface{}{
			"html": map[string]interface{}{
				"data":    htmlBody,
				"charset": "UTF-8",
			},
			"text": map[string]interface{}{
				"data":    textBody,
				"charset": "UTF-8",
			},
		},
	}

	return message
}

// formatMessageBody creates HTML and text versions of the message
func (a *AWSSESService) formatMessageBody(title, body string, notifyType NotifyType) (html, text string) {
	// Get emoji for notification type
	emoji := a.getEmojiForNotifyType(notifyType)
	color := a.getColorForNotifyType(notifyType)

	// Create text version
	var textBuilder strings.Builder
	textBuilder.WriteString(emoji + " ")
	if title != "" {
		textBuilder.WriteString(title)
		if body != "" {
			textBuilder.WriteString("\n\n")
		}
	}
	if body != "" {
		textBuilder.WriteString(body)
	}
	textBuilder.WriteString(fmt.Sprintf("\n\n---\nSent via Apprise-Go (%s)", time.Now().Format("2006-01-02 15:04:05 UTC")))
	text = textBuilder.String()

	// Create HTML version
	var htmlBuilder strings.Builder
	htmlBuilder.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>`)
	htmlBuilder.WriteString(`<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">`)

	if title != "" {
		htmlBuilder.WriteString(fmt.Sprintf(`<h2 style="color: %s; margin-bottom: 20px;">%s %s</h2>`,
			color, emoji, htmlEscape(title)))
	}

	if body != "" {
		// Convert newlines to HTML
		htmlBody := strings.ReplaceAll(htmlEscape(body), "\n", "<br>")
		htmlBuilder.WriteString(fmt.Sprintf(`<div style="margin-bottom: 30px; line-height: 1.6;">%s</div>`, htmlBody))
	}

	htmlBuilder.WriteString(`<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">`)
	htmlBuilder.WriteString(fmt.Sprintf(`<p style="color: #666; font-size: 12px; margin: 0;">Sent via Apprise-Go on %s</p>`,
		time.Now().Format("January 2, 2006 at 15:04:05 UTC")))
	htmlBuilder.WriteString(`</div></body></html>`)
	html = htmlBuilder.String()

	return html, text
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (a *AWSSESService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "✅"
	case NotifyTypeWarning:
		return "⚠️"
	case NotifyTypeError:
		return "❌"
	default:
		return "ℹ️"
	}
}

// getColorForNotifyType returns appropriate color for notification type
func (a *AWSSESService) getColorForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "#28a745"
	case NotifyTypeWarning:
		return "#ffc107"
	case NotifyTypeError:
		return "#dc3545"
	default:
		return "#17a2b8"
	}
}

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// processAttachments converts attachment manager attachments to SES format
func (a *AWSSESService) processAttachments(mgr *AttachmentManager) ([]map[string]interface{}, error) {
	attachments := mgr.GetAll()
	var sesAttachments []map[string]interface{}

	for _, attachment := range attachments {
		if !attachment.Exists() {
			continue // Skip non-existent attachments
		}

		// Get base64 encoded data
		base64Data, err := attachment.Base64()
		if err != nil {
			return nil, fmt.Errorf("failed to encode attachment %s: %w", attachment.GetName(), err)
		}

		sesAttachment := map[string]interface{}{
			"filename":    attachment.GetName(),
			"contentType": attachment.GetMimeType(),
			"data":        base64Data,
			"size":        attachment.GetSize(),
		}

		sesAttachments = append(sesAttachments, sesAttachment)
	}

	return sesAttachments, nil
}

// sendWebhookRequest sends the webhook request to the SES gateway
func (a *AWSSESService) sendWebhookRequest(ctx context.Context, payload map[string]interface{}) error {
	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GetUserAgent())

	// Add API key if provided
	if a.apiKey != "" {
		req.Header.Set("X-API-Key", a.apiKey)
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	// Send the request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SES webhook request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SES webhook error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
