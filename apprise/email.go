package apprise

import (
	"context"
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
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	fromEmail    string
	fromName     string
	toEmails     []string
	ccEmails     []string
	bccEmails    []string
	subject      string
	useTLS       bool
	useSTARTTLS  bool
	skipVerify   bool
	timeout      time.Duration
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
	if serviceURL.Scheme != "mailto" && serviceURL.Scheme != "mailtos" {
		return fmt.Errorf("invalid scheme: expected 'mailto' or 'mailtos', got '%s'", serviceURL.Scheme)
	}

	// Set TLS based on scheme
	e.useTLS = serviceURL.Scheme == "mailtos"

	// Extract SMTP server and port
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
		// Set default port based on TLS
		if e.useTLS {
			e.smtpPort = 465 // SMTPS port
		} else {
			e.smtpPort = 587 // SMTP submission port
		}
	}

	// Extract credentials
	if serviceURL.User != nil {
		e.username = serviceURL.User.Username()
		if password, hasPassword := serviceURL.User.Password(); hasPassword {
			e.password = password
		}
	}

	// Extract recipient email(s) from path
	pathParts := strings.Split(strings.Trim(serviceURL.Path, "/"), "/")
	for _, part := range pathParts {
		if part != "" && e.isValidEmail(part) {
			e.toEmails = append(e.toEmails, part)
		}
	}

	if len(e.toEmails) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	// Parse query parameters
	query := serviceURL.Query()
	
	if from := query.Get("from"); from != "" && e.isValidEmail(from) {
		e.fromEmail = from
	}
	
	if fromName := query.Get("name"); fromName != "" {
		e.fromName = fromName
	}
	
	if cc := query.Get("cc"); cc != "" {
		ccEmails := strings.Split(cc, ",")
		for _, email := range ccEmails {
			email = strings.TrimSpace(email)
			if e.isValidEmail(email) {
				e.ccEmails = append(e.ccEmails, email)
			}
		}
	}
	
	if bcc := query.Get("bcc"); bcc != "" {
		bccEmails := strings.Split(bcc, ",")
		for _, email := range bccEmails {
			email = strings.TrimSpace(email)
			if e.isValidEmail(email) {
				e.bccEmails = append(e.bccEmails, email)
			}
		}
	}
	
	if subject := query.Get("subject"); subject != "" {
		e.subject = subject
	}
	
	if skipVerify := query.Get("skip_verify"); skipVerify == "true" || skipVerify == "yes" {
		e.skipVerify = true
	}
	
	if noTLS := query.Get("no_tls"); noTLS == "true" || noTLS == "yes" {
		e.useSTARTTLS = false
	}

	// Set default from email if not specified
	if e.fromEmail == "" && e.username != "" {
		e.fromEmail = e.username
	}

	return nil
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
	defer client.Close()

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
		conn.Close()
		return nil, err
	}

	// Use STARTTLS if not already using TLS and STARTTLS is enabled
	if !e.useTLS && e.useSTARTTLS {
		tlsConfig := &tls.Config{
			ServerName:         e.smtpHost,
			InsecureSkipVerify: e.skipVerify,
		}
		
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return nil, err
		}
	}

	return client, nil
}

// createMessage creates the email message with headers and body
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
	
	// Determine content type based on body format
	contentType := "text/plain; charset=UTF-8"
	if req.BodyFormat == "html" {
		contentType = "text/html; charset=UTF-8"
	}
	message.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	message.WriteString(fmt.Sprintf("X-Mailer: Go-Apprise/1.0\r\n"))
	
	// Empty line separating headers from body
	message.WriteString("\r\n")

	// Format message body
	body := e.formatMessageBody(req.Title, req.Body, req.NotifyType, req.BodyFormat)
	message.WriteString(body)

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

// SupportsAttachments returns false for basic SMTP (can be extended later)
func (e *EmailService) SupportsAttachments() bool {
	return false // Basic implementation, can be extended with MIME multipart
}

// GetMaxBodyLength returns email body length limit (effectively unlimited for most servers)
func (e *EmailService) GetMaxBodyLength() int {
	return 0 // No practical limit for email body
}

// Example usage and URL formats:
// mailto://user:pass@smtp.gmail.com:587/recipient@domain.com
// mailtos://user:pass@smtp.gmail.com:465/recipient@domain.com
// mailto://user:pass@smtp.server.com/to@domain.com?cc=cc@domain.com&bcc=bcc@domain.com
// mailto://user:pass@smtp.server.com/to@domain.com?from=sender@domain.com&name=Sender%20Name
// mailto://user:pass@smtp.server.com/to@domain.com?subject=Custom%20Subject&no_tls=yes