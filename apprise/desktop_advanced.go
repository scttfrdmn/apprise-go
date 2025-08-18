package apprise

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// AdvancedDesktopService provides enhanced desktop notifications with actions and rich UI
type AdvancedDesktopService struct {
	*DesktopService
	actions     []NotificationAction
	category    string
	urgent      bool
	timeout     time.Duration
	subtitle    string
	group       string
	contentURL  string
	attachment  string
	replyButton bool
}

// NotificationAction represents an action button in the notification
type NotificationAction struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

// NotificationResult contains the result of user interaction with the notification
type NotificationResult struct {
	ActionID    string            `json:"actionId,omitempty"`
	ReplyText   string            `json:"replyText,omitempty"`
	Clicked     bool              `json:"clicked"`
	Dismissed   bool              `json:"dismissed"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// NewAdvancedDesktopService creates a new advanced desktop notification service
func NewAdvancedDesktopService() *AdvancedDesktopService {
	return &AdvancedDesktopService{
		DesktopService: NewDesktopService(),
		actions:        make([]NotificationAction, 0),
		timeout:        15 * time.Second, // Default 15 seconds
	}
}

func (ads *AdvancedDesktopService) GetServiceID() string {
	return "desktop-advanced"
}

func (ads *AdvancedDesktopService) ParseURL(serviceURL *url.URL) error {
	// Parse base desktop service parameters
	if err := ads.DesktopService.ParseURL(serviceURL); err != nil {
		return err
	}

	query := serviceURL.Query()

	// Parse advanced parameters
	if category := query.Get("category"); category != "" {
		ads.category = category
	}

	if urgentStr := query.Get("urgent"); urgentStr == "true" || urgentStr == "1" {
		ads.urgent = true
	}

	if timeoutStr := query.Get("timeout"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			ads.timeout = time.Duration(timeout) * time.Second
		}
	}

	if subtitle := query.Get("subtitle"); subtitle != "" {
		ads.subtitle = subtitle
	}

	if group := query.Get("group"); group != "" {
		ads.group = group
	}

	if contentURL := query.Get("contentUrl"); contentURL != "" {
		ads.contentURL = contentURL
	}

	if attachment := query.Get("attachment"); attachment != "" {
		ads.attachment = attachment
	}

	if replyStr := query.Get("reply"); replyStr == "true" || replyStr == "1" {
		ads.replyButton = true
	}

	// Parse actions (format: action1=id:title:url,action2=id:title:url)
	for key, values := range query {
		if strings.HasPrefix(key, "action") && len(values) > 0 {
			parts := strings.SplitN(values[0], ":", 3) // Use SplitN to limit splits (URLs contain colons)
			if len(parts) >= 2 {
				action := NotificationAction{
					ID:    parts[0],
					Title: parts[1],
				}
				if len(parts) > 2 {
					action.URL = parts[2]
				}
				ads.actions = append(ads.actions, action)
			}
		}
	}

	return nil
}

func (ads *AdvancedDesktopService) Send(ctx context.Context, req NotificationRequest) error {
	switch ads.platform {
	case "darwin":
		return ads.sendAdvancedMacOS(ctx, req)
	case "windows":
		return ads.sendAdvancedWindows(ctx, req)
	case "linux":
		return ads.sendAdvancedLinux(ctx, req)
	default:
		// Fallback to basic desktop service
		return ads.DesktopService.Send(ctx, req)
	}
}

func (ads *AdvancedDesktopService) sendAdvancedMacOS(ctx context.Context, req NotificationRequest) error {
	// Check for terminal-notifier or alerter
	var cmd *exec.Cmd
	
	if _, err := exec.LookPath("alerter"); err == nil {
		// Use alerter for advanced features
		args := []string{
			"-title", req.Title,
			"-message", req.Body,
			"-timeout", strconv.Itoa(int(ads.timeout.Seconds())),
		}

		if ads.subtitle != "" {
			args = append(args, "-subtitle", ads.subtitle)
		}

		if ads.sound != "" {
			args = append(args, "-sound", ads.sound)
		}

		if ads.image != "" {
			args = append(args, "-appIcon", ads.image)
		}

		if ads.attachment != "" {
			args = append(args, "-contentImage", ads.attachment)
		}

		if ads.group != "" {
			args = append(args, "-group", ads.group)
		}

		// Add actions
		for _, action := range ads.actions {
			args = append(args, "-actions", action.Title)
		}

		if ads.replyButton {
			args = append(args, "-reply")
		}

		if ads.contentURL != "" {
			args = append(args, "-open", ads.contentURL)
		}

		cmd = exec.CommandContext(ctx, "alerter", args...)
	} else if _, err := exec.LookPath("terminal-notifier"); err == nil {
		// Fallback to terminal-notifier with limited features
		args := []string{
			"-title", req.Title,
			"-message", req.Body,
		}

		if ads.subtitle != "" {
			args = append(args, "-subtitle", ads.subtitle)
		}

		if ads.sound != "" {
			args = append(args, "-sound", ads.sound)
		}

		if ads.image != "" {
			args = append(args, "-contentImage", ads.image)
		}

		if ads.group != "" {
			args = append(args, "-group", ads.group)
		}

		if ads.contentURL != "" {
			args = append(args, "-open", ads.contentURL)
		}

		// Add single action support
		if len(ads.actions) > 0 {
			args = append(args, "-actions", ads.actions[0].Title)
		}

		cmd = exec.CommandContext(ctx, "terminal-notifier", args...)
	} else {
		return fmt.Errorf("advanced macOS notifications require 'alerter' or 'terminal-notifier' - install with: brew install alerter")
	}

	// Execute and capture output for interaction handling
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to send advanced macOS notification: %w", err)
	}

	// Handle user interaction (if any)
	if len(output) > 0 {
		ads.handleInteraction(string(output), req)
	}

	return nil
}

func (ads *AdvancedDesktopService) sendAdvancedWindows(ctx context.Context, req NotificationRequest) error {
	// Use PowerShell with Windows 10+ toast notifications for advanced features
	script := ads.generateWindowsToastScript(req)
	
	cmd := exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
	return cmd.Run()
}

func (ads *AdvancedDesktopService) generateWindowsToastScript(req NotificationRequest) string {
	// Generate Windows Toast notification script with actions
	script := `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$APP_ID = 'Apprise-Go'

$template = @"
<toast activationType="protocol" launch="action=default">
  <visual>
    <binding template="ToastGeneric">
      <text>%s</text>
      <text>%s</text>
`

	if ads.image != "" {
		script += `      <image placement="appLogoOverride" src="` + ads.image + `"/>`
	}

	if ads.attachment != "" {
		script += `      <image src="` + ads.attachment + `"/>`
	}

	script += `
    </binding>
  </visual>
`

	// Add actions
	if len(ads.actions) > 0 || ads.replyButton {
		script += `  <actions>`
		
		for _, action := range ads.actions {
			actionURL := action.URL
			if actionURL == "" {
				actionURL = "action=" + action.ID
			}
			script += fmt.Sprintf(`    <action content="%s" arguments="%s" activationType="protocol"/>`, action.Title, actionURL)
		}

		if ads.replyButton {
			script += `    <input id="textBox" type="text" placeHolderContent="Type your reply..."/>
    <action content="Reply" arguments="action=reply" activationType="protocol" hint-inputId="textBox"/>`
		}

		script += `  </actions>`
	}

	script += `
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml(($template -f '%s', '%s'))

$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
$toast.ExpirationTime = [DateTimeOffset]::Now.AddSeconds(%d)

[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($APP_ID).Show($toast)
`

	return fmt.Sprintf(script, 
		escapeXML(req.Title), 
		escapeXML(req.Body),
		escapeXML(req.Title),
		escapeXML(req.Body),
		int(ads.timeout.Seconds()))
}

func (ads *AdvancedDesktopService) sendAdvancedLinux(ctx context.Context, req NotificationRequest) error {
	// Try different Linux notification systems with increasing feature support
	
	// Try dunst/dunstify for advanced features
	if _, err := exec.LookPath("dunstify"); err == nil {
		return ads.sendDunstify(ctx, req)
	}

	// Try notify-send with available features
	if _, err := exec.LookPath("notify-send"); err == nil {
		return ads.sendNotifySend(ctx, req)
	}

	// Fallback to basic desktop service
	return ads.DesktopService.Send(ctx, req)
}

func (ads *AdvancedDesktopService) sendDunstify(ctx context.Context, req NotificationRequest) error {
	args := []string{req.Title, req.Body}

	// Add timeout
	args = append(args, "-t", strconv.Itoa(int(ads.timeout.Milliseconds())))

	// Add urgency
	if ads.urgent {
		args = append(args, "-u", "critical")
	} else {
		args = append(args, "-u", "normal")
	}

	// Add category
	if ads.category != "" {
		args = append(args, "-c", ads.category)
	}

	// Add image
	if ads.image != "" {
		args = append(args, "-i", ads.image)
	}

	// Add actions (dunst supports basic actions)
	if len(ads.actions) > 0 {
		actionList := make([]string, 0, len(ads.actions)*2)
		for _, action := range ads.actions {
			actionList = append(actionList, action.ID, action.Title)
		}
		args = append(args, "-A", strings.Join(actionList, ","))
	}

	cmd := exec.CommandContext(ctx, "dunstify", args...)
	output, err := cmd.Output()
	
	if err != nil {
		return fmt.Errorf("dunstify failed: %w", err)
	}

	// Handle action responses
	if len(output) > 0 {
		ads.handleInteraction(string(output), req)
	}

	return nil
}

func (ads *AdvancedDesktopService) sendNotifySend(ctx context.Context, req NotificationRequest) error {
	args := []string{req.Title, req.Body}

	// Add timeout
	args = append(args, "-t", strconv.Itoa(int(ads.timeout.Milliseconds())))

	// Add urgency
	if ads.urgent {
		args = append(args, "-u", "critical")
	}

	// Add category
	if ads.category != "" {
		args = append(args, "-c", ads.category)
	}

	// Add image
	if ads.image != "" {
		args = append(args, "-i", ads.image)
	}

	cmd := exec.CommandContext(ctx, "notify-send", args...)
	return cmd.Run()
}

func (ads *AdvancedDesktopService) handleInteraction(output string, req NotificationRequest) {
	// Parse interaction output and handle accordingly
	result := NotificationResult{
		Timestamp: time.Now(),
	}

	output = strings.TrimSpace(output)
	
	// Handle different response formats
	if output == "@CLOSED" {
		result.Dismissed = true
	} else if output == "@CONTENTCLICKED" || output == "@ACTIONCLICKED" {
		result.Clicked = true
	} else if strings.HasPrefix(output, "@REPLY:") {
		result.ReplyText = strings.TrimPrefix(output, "@REPLY:")
	} else {
		// Check if it's an action ID
		for _, action := range ads.actions {
			if output == action.ID {
				result.ActionID = action.ID
				result.Clicked = true
				
				// If action has URL, open it
				if action.URL != "" {
					ads.openURL(action.URL)
				}
				break
			}
		}
	}

	// Log interaction (in a real implementation, this might trigger callbacks)
	if result.ActionID != "" || result.ReplyText != "" || result.Clicked {
		fmt.Printf("Notification interaction: %+v\n", result)
	}
}

func (ads *AdvancedDesktopService) openURL(url string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform for opening URLs: %s", runtime.GOOS)
	}
	
	return cmd.Run()
}

func (ads *AdvancedDesktopService) SupportsAttachments() bool {
	return true // Advanced desktop notifications support image attachments
}

func (ads *AdvancedDesktopService) GetMaxBodyLength() int {
	return 500 // Increased limit for advanced notifications
}

// Helper function to escape XML content for Windows toast notifications
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// InteractiveDesktopService provides callback-based interaction handling
type InteractiveDesktopService struct {
	*AdvancedDesktopService
	actionCallback func(NotificationResult)
	resultChannel  chan NotificationResult
}

// NewInteractiveDesktopService creates a desktop service with interaction callbacks
func NewInteractiveDesktopService() *InteractiveDesktopService {
	return &InteractiveDesktopService{
		AdvancedDesktopService: NewAdvancedDesktopService(),
		resultChannel:          make(chan NotificationResult, 10),
	}
}

func (ids *InteractiveDesktopService) GetServiceID() string {
	return "desktop-interactive"
}

// SetActionCallback sets a callback function for handling user actions
func (ids *InteractiveDesktopService) SetActionCallback(callback func(NotificationResult)) {
	ids.actionCallback = callback
}

// GetResultChannel returns a channel for receiving interaction results
func (ids *InteractiveDesktopService) GetResultChannel() <-chan NotificationResult {
	return ids.resultChannel
}

// Override handleInteraction to use callbacks and channels
func (ids *InteractiveDesktopService) handleInteraction(output string, req NotificationRequest) {
	// Call parent method to get the result
	ids.AdvancedDesktopService.handleInteraction(output, req)
	
	// Create result object (simplified for demo)
	result := NotificationResult{
		Timestamp: time.Now(),
		Clicked:   output != "@CLOSED" && output != "",
		Dismissed: output == "@CLOSED",
	}
	
	// Check for specific action
	for _, action := range ids.actions {
		if output == action.ID {
			result.ActionID = action.ID
			break
		}
	}
	
	// Send to callback if set
	if ids.actionCallback != nil {
		go ids.actionCallback(result)
	}
	
	// Send to channel (non-blocking)
	select {
	case ids.resultChannel <- result:
	default:
		// Channel full, drop the result
	}
}

// PersistentDesktopService provides notification persistence and history
type PersistentDesktopService struct {
	*InteractiveDesktopService
	historyFile   string
	notifications map[string]NotificationRequest
	results       map[string][]NotificationResult
}

// NewPersistentDesktopService creates a desktop service with notification history
func NewPersistentDesktopService() *PersistentDesktopService {
	homeDir, _ := os.UserHomeDir()
	historyFile := homeDir + "/.apprise-go-notifications.json"
	
	pds := &PersistentDesktopService{
		InteractiveDesktopService: NewInteractiveDesktopService(),
		historyFile:               historyFile,
		notifications:             make(map[string]NotificationRequest),
		results:                   make(map[string][]NotificationResult),
	}
	
	// Load existing history if available
	_ = pds.loadHistory()
	
	return pds
}

func (pds *PersistentDesktopService) GetServiceID() string {
	return "desktop-persistent"
}

// Send with persistence
func (pds *PersistentDesktopService) Send(ctx context.Context, req NotificationRequest) error {
	// Generate unique ID for this notification
	notificationID := fmt.Sprintf("notif_%d", time.Now().Unix())
	
	// Store notification
	pds.notifications[notificationID] = req
	pds.saveHistory()
	
	// Send notification with ID in group for tracking
	if pds.group == "" {
		pds.group = notificationID
	}
	
	return pds.InteractiveDesktopService.Send(ctx, req)
}

// GetNotificationHistory returns all stored notifications
func (pds *PersistentDesktopService) GetNotificationHistory() map[string]NotificationRequest {
	pds.loadHistory()
	return pds.notifications
}

// GetInteractionHistory returns all interaction results
func (pds *PersistentDesktopService) GetInteractionHistory() map[string][]NotificationResult {
	pds.loadHistory()
	return pds.results
}

func (pds *PersistentDesktopService) saveHistory() error {
	data := struct {
		Notifications map[string]NotificationRequest    `json:"notifications"`
		Results       map[string][]NotificationResult `json:"results"`
	}{
		Notifications: pds.notifications,
		Results:       pds.results,
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(pds.historyFile, jsonData, 0600)
}

func (pds *PersistentDesktopService) loadHistory() error {
	if _, err := os.Stat(pds.historyFile); os.IsNotExist(err) {
		return nil // No history file yet
	}
	
	data, err := os.ReadFile(pds.historyFile)
	if err != nil {
		return err
	}
	
	var historyData struct {
		Notifications map[string]NotificationRequest    `json:"notifications"`
		Results       map[string][]NotificationResult `json:"results"`
	}
	
	if err := json.Unmarshal(data, &historyData); err != nil {
		return err
	}
	
	if historyData.Notifications != nil {
		pds.notifications = historyData.Notifications
	}
	
	if historyData.Results != nil {
		pds.results = historyData.Results
	}
	
	return nil
}