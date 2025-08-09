package apprise

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// DesktopService implements desktop notifications for various platforms
type DesktopService struct {
	platform string
	sound    string
	duration int // Duration in seconds (Windows only)
	image    string
}

// NewDesktopService creates a new desktop notification service
func NewDesktopService() *DesktopService {
	return &DesktopService{
		platform: runtime.GOOS,
		duration: 12, // Default 12 seconds for Windows
	}
}

func (d *DesktopService) GetServiceID() string {
	switch d.platform {
	case "darwin":
		return "macosx"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return "desktop"
	}
}

func (d *DesktopService) GetDefaultPort() int {
	return 0 // Not applicable for desktop notifications
}

func (d *DesktopService) ParseURL(serviceURL *url.URL) error {
	// Store the original platform detection
	d.platform = runtime.GOOS
	
	// Override platform if specified in scheme
	switch serviceURL.Scheme {
	case "macosx":
		d.platform = "darwin"
	case "windows":
		d.platform = "windows"
	case "linux", "dbus", "gnome", "kde", "glib", "qt":
		d.platform = "linux"
	}
	
	// Parse query parameters
	query := serviceURL.Query()
	
	// Sound parameter
	if sound := query.Get("sound"); sound != "" {
		d.sound = sound
	}
	
	// Duration parameter (Windows)
	if durationStr := query.Get("duration"); durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil && duration > 0 {
			d.duration = duration
		}
	}
	
	// Image parameter
	if image := query.Get("image"); image != "" {
		d.image = image
	}
	
	return nil
}

func (d *DesktopService) Send(ctx context.Context, req NotificationRequest) error {
	// Limit message length to 250 characters as per original Apprise
	title := req.Title
	body := req.Body
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	if len(body) > 250 {
		body = body[:247] + "..."
	}
	
	switch d.platform {
	case "darwin":
		return d.sendMacOS(ctx, title, body)
	case "windows":
		return d.sendWindows(ctx, title, body)
	case "linux":
		return d.sendLinux(ctx, title, body)
	default:
		return fmt.Errorf("desktop notifications not supported on platform: %s", d.platform)
	}
}

func (d *DesktopService) sendMacOS(ctx context.Context, title, body string) error {
	// Check if terminal-notifier is available
	if _, err := exec.LookPath("terminal-notifier"); err != nil {
		return fmt.Errorf("terminal-notifier not found - install with: brew install terminal-notifier")
	}
	
	args := []string{
		"-title", title,
		"-message", body,
	}
	
	// Add sound if specified
	if d.sound != "" {
		args = append(args, "-sound", d.sound)
	}
	
	// Add image if specified
	if d.image != "" {
		args = append(args, "-contentImage", d.image)
	}
	
	cmd := exec.CommandContext(ctx, "terminal-notifier", args...)
	return cmd.Run()
}

func (d *DesktopService) sendWindows(ctx context.Context, title, body string) error {
	// Use PowerShell to show Windows toast notifications
	// This approach works without additional dependencies
	script := fmt.Sprintf(`
		Add-Type -AssemblyName System.Windows.Forms
		$balloon = New-Object System.Windows.Forms.NotifyIcon
		$balloon.Icon = [System.Drawing.SystemIcons]::Information
		$balloon.BalloonTipTitle = '%s'
		$balloon.BalloonTipText = '%s'
		$balloon.BalloonTipIcon = 'Info'
		$balloon.Visible = $true
		$balloon.ShowBalloonTip(%d)
		Start-Sleep -Seconds %d
		$balloon.Dispose()
	`, 
		escapeString(title), 
		escapeString(body), 
		d.duration*1000, // Convert to milliseconds
		d.duration,
	)
	
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	return cmd.Run()
}

func (d *DesktopService) sendLinux(ctx context.Context, title, body string) error {
	// Try notify-send first (most common)
	if _, err := exec.LookPath("notify-send"); err == nil {
		args := []string{title, body}
		
		// Add image if specified
		if d.image != "" {
			args = append([]string{"-i", d.image}, args...)
		}
		
		cmd := exec.CommandContext(ctx, "notify-send", args...)
		return cmd.Run()
	}
	
	// Try zenity as fallback
	if _, err := exec.LookPath("zenity"); err == nil {
		args := []string{
			"--notification",
			"--text", fmt.Sprintf("%s\n%s", title, body),
		}
		
		cmd := exec.CommandContext(ctx, "zenity", args...)
		return cmd.Run()
	}
	
	// Try kdialog for KDE environments
	if _, err := exec.LookPath("kdialog"); err == nil {
		cmd := exec.CommandContext(ctx, "kdialog", "--passivepopup", fmt.Sprintf("%s\n%s", title, body), "5")
		return cmd.Run()
	}
	
	return fmt.Errorf("no desktop notification tool found - install notify-send, zenity, or kdialog")
}

func (d *DesktopService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid desktop notification URL: %w", err)
	}
	
	// Validate scheme
	validSchemes := []string{"desktop", "macosx", "windows", "linux", "dbus", "gnome", "kde", "glib", "qt"}
	valid := false
	for _, scheme := range validSchemes {
		if parsedURL.Scheme == scheme {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("unsupported desktop notification scheme: %s", parsedURL.Scheme)
	}
	
	return d.ParseURL(parsedURL)
}

func (d *DesktopService) SupportsAttachments() bool {
	// Desktop notifications generally don't support file attachments
	// Images can be shown via the image parameter but not as attachments
	return false
}

func (d *DesktopService) GetMaxBodyLength() int {
	return 250 // Match original Apprise limit
}

// escapeString escapes single quotes in PowerShell strings
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// LinuxDBusService provides Linux DBus notifications
type LinuxDBusService struct {
	*DesktopService
	interfaceType string // "qt", "glib", or auto-detect
}

// NewLinuxDBusService creates a new Linux DBus notification service
func NewLinuxDBusService() *LinuxDBusService {
	return &LinuxDBusService{
		DesktopService: NewDesktopService(),
		interfaceType:  "auto",
	}
}

func (l *LinuxDBusService) GetServiceID() string {
	return "dbus"
}

func (l *LinuxDBusService) ParseURL(serviceURL *url.URL) error {
	// Determine interface type from scheme
	switch serviceURL.Scheme {
	case "qt", "kde":
		l.interfaceType = "qt"
	case "glib", "gnome":
		l.interfaceType = "glib"
	case "dbus":
		l.interfaceType = "auto"
	}
	
	// Parse common desktop parameters
	return l.DesktopService.ParseURL(serviceURL)
}

func (l *LinuxDBusService) Send(ctx context.Context, req NotificationRequest) error {
	// Force Linux platform for DBus
	l.platform = "linux"
	return l.DesktopService.Send(ctx, req)
}

// GotifyService implements self-hosted Gotify notifications
type GotifyService struct {
	serverURL string
	appToken  string
	priority  int
	secure    bool
}

// NewGotifyService creates a new Gotify service
func NewGotifyService() *GotifyService {
	return &GotifyService{
		priority: 5, // Default priority
	}
}

func (g *GotifyService) GetServiceID() string {
	return "gotify"
}

func (g *GotifyService) GetDefaultPort() int {
	if g.secure {
		return 443
	}
	return 80
}

func (g *GotifyService) ParseURL(serviceURL *url.URL) error {
	// URL format: gotify://hostname/token or gotifys://hostname/token
	g.secure = serviceURL.Scheme == "gotifys"
	
	// Extract server URL
	port := serviceURL.Port()
	if port == "" {
		port = fmt.Sprintf("%d", g.GetDefaultPort())
	}
	
	protocol := "http"
	if g.secure {
		protocol = "https"
	}
	
	g.serverURL = fmt.Sprintf("%s://%s:%s", protocol, serviceURL.Hostname(), port)
	
	// Extract token from path
	if serviceURL.Path == "" || serviceURL.Path == "/" {
		return fmt.Errorf("gotify token required in URL path")
	}
	
	g.appToken = strings.TrimPrefix(serviceURL.Path, "/")
	
	// Parse priority from query
	if priorityStr := serviceURL.Query().Get("priority"); priorityStr != "" {
		if priority, err := strconv.Atoi(priorityStr); err == nil && priority >= 0 && priority <= 10 {
			g.priority = priority
		}
	}
	
	return nil
}

func (g *GotifyService) Send(ctx context.Context, req NotificationRequest) error {
	// Create Gotify message payload
	payload := map[string]interface{}{
		"title":    req.Title,
		"message":  req.Body,
		"priority": g.priority,
	}
	
	// Add extras based on notification type
	extras := make(map[string]interface{})
	switch req.NotifyType {
	case NotifyTypeSuccess:
		extras["client::notification"] = map[string]string{"color": "#4CAF50"}
	case NotifyTypeWarning:
		extras["client::notification"] = map[string]string{"color": "#FF9800"}
	case NotifyTypeError:
		extras["client::notification"] = map[string]string{"color": "#F44336"}
	default:
		extras["client::notification"] = map[string]string{"color": "#2196F3"}
	}
	
	if len(extras) > 0 {
		payload["extras"] = extras
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Gotify payload: %w", err)
	}
	
	// Create HTTP request
	url := fmt.Sprintf("%s/message?token=%s", g.serverURL, g.appToken)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Gotify request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", GetUserAgent())
	
	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send Gotify notification: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Gotify API error: %s", resp.Status)
	}
	
	return nil
}

func (g *GotifyService) TestURL(serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("invalid Gotify URL: %w", err)
	}
	
	if parsedURL.Scheme != "gotify" && parsedURL.Scheme != "gotifys" {
		return fmt.Errorf("invalid Gotify scheme: %s", parsedURL.Scheme)
	}
	
	return g.ParseURL(parsedURL)
}

func (g *GotifyService) SupportsAttachments() bool {
	return false // Gotify doesn't support file attachments
}

func (g *GotifyService) GetMaxBodyLength() int {
	return 0 // No specific limit for Gotify
}