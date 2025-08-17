package apprise

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// EmailService implements SMTP email notifications
type EmailService struct {
	smtpHost    string
	smtpPort    int
	username    string
	password    string
	fromEmail   string
	fromName    string
	toEmails    []string
	ccEmails    []string
	bccEmails   []string
	subject     string
	useTLS      bool
	useSTARTTLS bool
	skipVerify  bool
	timeout     time.Duration
}

// NewEmailService creates a new email service instance
func NewEmailService() Service {
	return &EmailService{
		smtpPort:    587, // Default SMTP submission port
		useTLS:      false,
		useSTARTTLS: true,
		skipVerify:  false,
		timeout:     30 * time.Second,
	}
}

// GetServiceID returns the service identifier
func (e *EmailService) GetServiceID() string {
	return "email"
}

// GetDefaultPort returns the default SMTP port
func (e *EmailService) GetDefaultPort() int {
	return 587 // SMTP submission port
}

// ParseURL parses an email service URL
// Format: mailto://username:password@smtp.server.com:port/to@email.com
// Format: mailtos://username:password@smtp.server.com:port/to@email.com (TLS)
// Format: mailto://username:password@smtp.server.com/to@email.com?cc=cc@email.com&bcc=bcc@email.com
func (e *EmailService) ParseURL(serviceURL *url.URL) error {
	if err := e.parseScheme(serviceURL); err != nil {
		return err
	}
	if err := e.parseHost(serviceURL); err != nil {
		return err
	}
	if err := e.parseCredentials(serviceURL); err != nil {
		return err
	}
	if err := e.parseRecipients(serviceURL); err != nil {
		return err
	}
	if err := e.parseQueryParams(serviceURL); err != nil {
		return err
	}
	e.setDefaults()
	return nil
}

func (e *EmailService) parseScheme(serviceURL *url.URL) error {
	if serviceURL.Scheme != "mailto" && serviceURL.Scheme != "mailtos" {
		return fmt.Errorf("invalid scheme: expected 'mailto' or 'mailtos', got '%s'", serviceURL.Scheme)
	}
	e.useTLS = serviceURL.Scheme == "mailtos"
	return nil
}

func (e *EmailService) parseHost(serviceURL *url.URL) error {
	e.smtpHost = serviceURL.Hostname()
	if e.smtpHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	portStr := serviceURL.Port()
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		e.smtpPort = port
	} else {
		if e.useTLS {
			e.smtpPort = 465 // SMTPS port
		} else {
			e.smtpPort = 587 // SMTP submission port
		}
	}
	return nil
}

func (e *EmailService) parseCredentials(serviceURL *url.URL) error {
	if serviceURL.User != nil {
		e.username = serviceURL.User.Username()
		if password, hasPassword := serviceURL.User.Password(); hasPassword {
			e.password = password
		}
	}
	return nil
}

func (e *EmailService) parseRecipients(serviceURL *url.URL) error {
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	for _, part := range pathParts {
		if part != "" && e.isValidEmail(part) {
			e.toEmails = append(e.toEmails, part)
		}
	}

	if len(e.toEmails) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}
	return nil
}

func (e *EmailService) parseQueryParams(serviceURL *url.URL) error {
	query := serviceURL.Query()

	if from := query.Get("from"); from != "" && e.isValidEmail(from) {
		e.fromEmail = from
	}
	if fromName := query.Get("name"); fromName != "" {
		e.fromName = fromName
	}

	e.parseEmailList(query.Get("cc"), &e.ccEmails)
	e.parseEmailList(query.Get("bcc"), &e.bccEmails)

	if subject := query.Get("subject"); subject != "" {
		e.subject = subject
	}
	if skipVerify := query.Get("skip_verify"); skipVerify == "true" || skipVerify == "yes" {
		e.skipVerify = true
	}
	if noTLS := query.Get("no_tls"); noTLS == "true" || noTLS == "yes" {
		e.useSTARTTLS = false
	}
	return nil
}

func (e *EmailService) parseEmailList(emailStr string, target *[]string) {
	if emailStr == "" {
		return
	}
	emails := strings.Split(emailStr, ",")
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if e.isValidEmail(email) {
			*target = append(*target, email)
		}
	}
}

func (e *EmailService) setDefaults() {
	if e.fromEmail == "" && e.username != "" {
		e.fromEmail = e.username
	}
}

// Send sends an email notification
func (e *EmailService) Send(ctx context.Context, req NotificationRequest) error {
	// Create the email message
	message, err := e.createMessage(req)
	if err != nil {
		return fmt.Errorf("failed to create email message: %w", err)
	}

	// Create all recipients list (TO + CC + BCC)
	var allRecipients []string
	allRecipients = append(allRecipients, e.toEmails...)
	allRecipients = append(allRecipients, e.ccEmails...)
	allRecipients = append(allRecipients, e.bccEmails...)

	// Connect to SMTP server
	client, err := e.connectSMTP(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Authenticate if credentials provided
	if e.username != "" && e.password != "" {
		auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to initiate data transfer: %w", err)
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize message: %w", err)
	}

	return nil
}

// connectSMTP establishes connection to SMTP server
func (e *EmailService) connectSMTP(ctx context.Context) (*smtp.Client, error) {
	address := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	var conn net.Conn
	var err error

	if e.useTLS {
		// Direct TLS connection
		tlsConfig := &tls.Config{
			ServerName:         e.smtpHost,
			InsecureSkipVerify: e.skipVerify,
		}

		dialer := &net.Dialer{Timeout: e.timeout}
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		// Plain connection
		dialer := &net.Dialer{Timeout: e.timeout}
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}

	if err != nil {
		return nil, err
	}

	client, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Use STARTTLS if not already using TLS and STARTTLS is enabled
	if !e.useTLS && e.useSTARTTLS {
		tlsConfig := &tls.Config{
			ServerName:         e.smtpHost,
			InsecureSkipVerify: e.skipVerify,
		}

		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

// createMessage creates the email message with headers, body, and attachments
func (e *EmailService) createMessage(req NotificationRequest) (string, error) {
	var message strings.Builder

	// From header
	fromAddr := e.fromEmail
	if e.fromName != "" {
		fromAddr = fmt.Sprintf("%s <%s>", e.fromName, e.fromEmail)
	}
	message.WriteString(fmt.Sprintf("From: %s\r\n", fromAddr))

	// To header
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.toEmails, ", ")))

	// CC header
	if len(e.ccEmails) > 0 {
		message.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(e.ccEmails, ", ")))
	}

	// Subject
	subject := e.subject
	if subject == "" && req.Title != "" {
		subject = req.Title
	}
	if subject == "" {
		subject = fmt.Sprintf("Notification (%s)", req.NotifyType.String())
	}
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Additional headers
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	message.WriteString(fmt.Sprintf("X-Mailer: %s\r\n", GetUserAgent()))

	// Check if we have attachments
	hasAttachments := req.AttachmentMgr != nil && req.AttachmentMgr.Count() > 0

	if hasAttachments {
		// Generate boundary for multipart message
		boundary, err := e.generateBoundary()
		if err != nil {
			return "", fmt.Errorf("failed to generate MIME boundary: %w", err)
		}

		// Multipart message with attachments
		message.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		message.WriteString("\r\n")
		message.WriteString("This is a multi-part message in MIME format.\r\n")

		// Add text/HTML part
		message.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
		if req.BodyFormat == "html" {
			message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		message.WriteString("\r\n")

		// Add formatted body
		body := e.formatMessageBody(req.Title, req.Body, req.NotifyType, req.BodyFormat)
		message.WriteString(body)

		// Add attachment parts
		if err := e.addAttachments(&message, boundary, req.AttachmentMgr); err != nil {
			return "", fmt.Errorf("failed to add attachments: %w", err)
		}

		// Close multipart message
		message.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		// Simple message without attachments
		if req.BodyFormat == "html" {
			message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		} else {
			message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		}
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n")
		message.WriteString("\r\n")

		// Add formatted body
		body := e.formatMessageBody(req.Title, req.Body, req.NotifyType, req.BodyFormat)
		message.WriteString(body)
	}

	return message.String(), nil
}

// formatMessageBody formats the email body based on the specified format
func (e *EmailService) formatMessageBody(title, body string, notifyType NotifyType, format string) string {
	var result strings.Builder

	if format == "html" {
		// HTML format
		result.WriteString("<html><body>\r\n")

		if title != "" {
			emoji := e.getEmojiForNotifyType(notifyType)
			result.WriteString(fmt.Sprintf("<h2>%s %s</h2>\r\n", emoji, title))
		}

		if body != "" {
			// Convert line breaks to <br> for HTML
			htmlBody := strings.ReplaceAll(body, "\n", "<br>\r\n")
			result.WriteString(fmt.Sprintf("<p>%s</p>\r\n", htmlBody))
		}

		result.WriteString(fmt.Sprintf("<hr><small>Notification Type: %s</small>\r\n", notifyType.String()))
		result.WriteString("</body></html>\r\n")
	} else {
		// Plain text format
		if title != "" {
			emoji := e.getEmojiForNotifyType(notifyType)
			result.WriteString(fmt.Sprintf("%s %s\r\n\r\n", emoji, title))
		}

		if body != "" {
			result.WriteString(body + "\r\n")
		}

		result.WriteString(fmt.Sprintf("\r\n---\r\nNotification Type: %s\r\n", notifyType.String()))
	}

	return result.String()
}

// getEmojiForNotifyType returns appropriate emoji for notification type
func (e *EmailService) getEmojiForNotifyType(notifyType NotifyType) string {
	switch notifyType {
	case NotifyTypeSuccess:
		return "✅"
	case NotifyTypeWarning:
		return "⚠️"
	case NotifyTypeError:
		return "❌"
	case NotifyTypeInfo:
		return "ℹ️"
	default:
		return "📧"
	}
}

// isValidEmail performs basic email validation
func (e *EmailService) isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// TestURL validates an email service URL
func (e *EmailService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return e.ParseURL(parsedURL)
}

// SupportsAttachments returns true - SMTP supports attachments via MIME multipart
func (e *EmailService) SupportsAttachments() bool {
	return true // Full MIME multipart support with attachments
}

// GetMaxBodyLength returns email body length limit (effectively unlimited for most servers)
func (e *EmailService) GetMaxBodyLength() int {
	return 0 // No practical limit for email body
}

// generateBoundary generates a random MIME boundary string
func (e *EmailService) generateBoundary() (string, error) {
	boundary := make([]byte, 16)
	_, err := rand.Read(boundary)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("----=_Part_%x", boundary), nil
}

// addAttachments adds all attachments to the MIME message
func (e *EmailService) addAttachments(message *strings.Builder, boundary string, attachmentMgr *AttachmentManager) error {
	if attachmentMgr == nil {
		return nil
	}

	attachments := attachmentMgr.GetAll()
	for _, attachment := range attachments {
		if !attachment.Exists() {
			continue // Skip non-existent attachments
		}

		if err := e.addSingleAttachment(message, boundary, attachment); err != nil {
			return fmt.Errorf("failed to add attachment %s: %w", attachment.GetName(), err)
		}
	}

	return nil
}

// addSingleAttachment adds a single attachment to the MIME message
func (e *EmailService) addSingleAttachment(message *strings.Builder, boundary string, attachment AttachmentInterface) error {
	// Get attachment content as base64
	base64Content, err := attachment.Base64()
	if err != nil {
		return fmt.Errorf("failed to encode attachment: %w", err)
	}

	// Get MIME type
	mimeType := attachment.GetMimeType()
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get filename
	filename := attachment.GetName()
	if filename == "" {
		filename = "attachment"
	}

	// Determine if this is an inline attachment (image)
	isInline := strings.HasPrefix(mimeType, "image/")

	// Write attachment headers
	message.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	message.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", mimeType, filename))
	message.WriteString("Content-Transfer-Encoding: base64\r\n")

	if isInline {
		message.WriteString(fmt.Sprintf("Content-Disposition: inline; filename=\"%s\"\r\n", filename))
		message.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", filename))
	} else {
		message.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
	}

	message.WriteString("\r\n")

	// Write base64 content in 76-character lines (RFC 2045)
	e.writeBase64Lines(message, base64Content)

	return nil
}

// writeBase64Lines writes base64 content with proper line breaks
func (e *EmailService) writeBase64Lines(message *strings.Builder, base64Content string) {
	const lineLength = 76
	for i := 0; i < len(base64Content); i += lineLength {
		end := i + lineLength
		if end > len(base64Content) {
			end = len(base64Content)
		}
		message.WriteString(base64Content[i:end])
		message.WriteString("\r\n")
	}
}

// Example usage and URL formats:
// mailto://user:pass@smtp.gmail.com:587/recipient@domain.com
// mailtos://user:pass@smtp.gmail.com:465/recipient@domain.com
// mailto://user:pass@smtp.server.com/to@domain.com?cc=cc@domain.com&bcc=bcc@domain.com
// mailto://user:pass@smtp.server.com/to@domain.com?from=sender@domain.com&name=Sender%20Name
// mailto://user:pass@smtp.server.com/to@domain.com?subject=Custom%20Subject&no_tls=yes
