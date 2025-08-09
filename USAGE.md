# Apprise Go Usage Guide

This guide covers all the notification services implemented in Apprise Go and their URL formats.

## Quick Start

```go
package main

import "github.com/scttfrdmn/apprise-go/apprise"

func main() {
    app := apprise.New()
    
    // Add services
    app.Add("discord://webhook_id/webhook_token")
    app.Add("slack://TokenA/TokenB/TokenC/general")
    
    // Send notification
    app.Notify("Hello World!", "This is a test notification", apprise.NotifyTypeInfo)
}
```

## Supported Services

### Discord

Discord webhook notifications with rich embeds and custom formatting.

**URL Formats:**
```
discord://webhook_id/webhook_token
discord://avatar@webhook_id/webhook_token
discord://webhook_id/webhook_token?username=MyBot&avatar=https://example.com/avatar.png
```

**Features:**
- Rich embeds with titles and colors
- Custom avatar and username
- Notification type-based color coding
- Support for attachments

### Slack  

Slack notifications via webhooks or bot API with rich formatting.

**URL Formats:**
```
# Webhook mode (3 tokens)
slack://TokenA/TokenB/TokenC
slack://TokenA/TokenB/TokenC/general
slack://TokenA/TokenB/TokenC/@username

# Bot mode (OAuth token)  
slack://bot_token/general
slack://bot_token/@username
slack://TokenA/TokenB/TokenC?username=MyBot&icon_emoji=:ghost:
```

**Features:**
- Webhook and bot API support
- Channel and direct message support
- Rich attachments with colors
- Custom bot appearance

### Telegram

Telegram Bot API with support for multiple chats and rich formatting.

**URL Formats:**
```
tgram://bot_token/chat_id
telegram://bot_token/chat_id1/chat_id2
tgram://bot_token/@username
tgram://bot_token/chat_id?silent=yes&preview=no&format=html
```

**Query Parameters:**
- `silent=yes/no` - Silent notifications
- `preview=yes/no` - Web page preview  
- `format=html/markdown/markdownv2` - Message formatting
- `thread=123` - Reply to specific thread

**Features:**
- Multiple chat support
- HTML, Markdown, and MarkdownV2 formatting
- Silent notifications
- Thread support
- Emoji indicators by notification type

### Email (SMTP)

Full-featured SMTP email notifications with TLS support.

**URL Formats:**
```
mailto://username:password@smtp.server.com:587/recipient@domain.com
mailtos://username:password@smtp.server.com:465/recipient@domain.com
mailto://user:pass@smtp.gmail.com/to@domain.com?cc=cc@domain.com&bcc=bcc@domain.com
mailto://user:pass@smtp.server.com/to@domain.com?from=sender@domain.com&name=Sender%20Name
```

**Query Parameters:**
- `from=email` - Sender email address
- `name=Name` - Sender display name
- `cc=email` - CC recipients (comma-separated)
- `bcc=email` - BCC recipients (comma-separated)
- `subject=Subject` - Custom subject line
- `skip_verify=yes` - Skip TLS certificate verification
- `no_tls=yes` - Disable STARTTLS

**Features:**
- TLS and STARTTLS support
- HTML and plain text formatting
- CC and BCC support
- Custom sender names
- SMTP authentication

### Webhook/JSON

Generic HTTP webhook notifications with custom templates.

**URL Formats:**
```
webhook://api.example.com/notify
webhooks://api.example.com/notify  (HTTPS)
json://api.example.com/webhook
webhooks://token@api.example.com/notify  (Bearer auth)
webhooks://user:pass@api.example.com/notify  (Basic auth)
webhook://api.example.com/notify?method=PUT&content_type=text/plain
```

**Query Parameters:**
- `method=POST/GET/PUT/PATCH` - HTTP method
- `content_type=application/json` - Content type
- `template={"text":"{{message}}"}` - Custom payload template  
- `header_X-API-Key=secret` - Custom headers

**Features:**
- Multiple content types (JSON, form-encoded, plain text)
- Custom HTTP methods
- Template-based payloads
- Authentication support (Basic, Bearer)
- Custom headers

### Microsoft Teams

Enterprise messaging with rich card formatting and theme colors.

**URL Formats:**
```
# Modern format (recommended)
msteams://team_name/token_a/token_b/token_c

# Version 3 format
msteams://team_name/token_a/token_b/token_c/token_d

# Legacy format
msteams://token_a/token_b/token_c

# With options
msteams://team_name/token_a/token_b/token_c?image=no
```

**Query Parameters:**
- `image=yes/no` - Include activity images in notifications

**Features:**
- MessageCard format with rich styling
- Color-coded notifications by type
- Activity images for visual context
- Support for all Teams webhook versions
- Markdown text formatting support

### Pushover

Mobile push notifications with priority levels and custom sounds.

**URL Formats:**
```
pushover://token@userkey
pover://token@userkey/device1/device2
pushover://token@userkey?priority=1&sound=cosmic
pushover://token@userkey?priority=2&retry=60&expire=3600
```

**Query Parameters:**
- `priority=-2/-1/0/1/2` - Priority level (-2=lowest, 2=emergency)
- `sound=pushover/bike/cosmic` - Notification sound
- `retry=60` - Retry interval for emergency priority (seconds)
- `expire=3600` - Expiration for emergency priority (seconds)

**Features:**
- Priority levels from silent to emergency
- Custom notification sounds
- Device targeting
- Emergency notifications with retry/expire
- Rich formatting with emojis

### Pushbullet

Cross-platform push notifications to devices, emails, and channels.

**URL Formats:**
```
pball://access_token
pushbullet://access_token/device_id
pball://access_token/user@email.com
pball://access_token/#channel_name
pball://access_token?device=device1,device2&email=user@domain.com
```

**Query Parameters:**
- `device=id1,id2` - Target specific devices (comma-separated)
- `email=user@domain.com` - Send to email addresses  
- `channel=channel1,channel2` - Send to channels (comma-separated)

**Features:**
- Multi-device targeting
- Email and channel support
- File attachment support
- Cross-platform compatibility
- Emoji indicators by notification type

### Twilio SMS

SMS/MMS messaging with rate limiting and phone number normalization.

**URL Formats:**
```
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543/+15551111111
twilio://ACCOUNT_SID:AUTH_TOKEN@15551234567/15559876543
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543?apikey=KEY
```

**Query Parameters:**
- `apikey=KEY` - Optional API key for authentication

**Features:**
- Automatic phone number normalization (E.164 format)
- Rate limiting (0.2 requests/second)
- Multiple recipient support
- US/Canada number auto-formatting
- SMS message length optimization

### Desktop Notifications

Cross-platform desktop notifications using native OS notification systems.

**URL Formats:**
```
# Generic (auto-detects platform)
desktop://

# Platform-specific
macosx://                          # macOS via terminal-notifier
windows://                         # Windows system tray notifications
linux://                           # Linux via notify-send

# Linux-specific DBus notifications
dbus://                            # Auto-detect DBus interface
qt://                              # Force QT interface
kde://                             # KDE desktop environment
glib://                            # GLib interface
gnome://                           # Gnome desktop environment

# With parameters
macosx://?sound=default&image=/path/to/icon.png
windows://?duration=10             # Display for 10 seconds
desktop://?image=/path/to/image.png
```

**Query Parameters:**
- `sound=name` - System sound name (macOS only)
- `duration=seconds` - Display duration in seconds (Windows only, default: 12)
- `image=path` - Path to image file for notification icon

**Platform Requirements:**
- **macOS:** Requires `terminal-notifier` - install with: `brew install terminal-notifier`
- **Windows:** Uses PowerShell and system tray notifications (no extra dependencies)
- **Linux:** Requires one of: `notify-send`, `zenity`, or `kdialog`

**Features:**
- Cross-platform compatibility with native OS integration
- Message length automatically limited to 250 characters
- Platform-specific notification styling and behavior
- Support for custom sounds and images
- Graceful fallbacks when notification tools are unavailable

### Gotify

Self-hosted push notification server for sending messages to devices and applications.

**URL Formats:**
```
gotify://hostname:port/app_token
gotifys://secure.example.com/app_token          # HTTPS
gotify://192.168.1.100:8080/ABCDefGHijkL?priority=5
```

**Query Parameters:**
- `priority=0-10` - Message priority level (default: 5)

**Features:**
- Self-hosted push notification solution
- Color-coded messages based on notification type
- Customizable priority levels (0-10)
- HTTP and HTTPS support
- JSON-based API integration
- Supports rich notification metadata via "extras"

## Configuration Files

### YAML Format

```yaml
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
  - url: mailto://user:pass@smtp.gmail.com/admin@company.com
    tag:
      - admin
  - url: slack://TokenA/TokenB/TokenC/general
    tag:
      - team
```

### Text Format

```
# Team notifications
discord://webhook_id/webhook_token [team,alerts]

# Admin email
mailto://user:pass@smtp.gmail.com/admin@company.com [admin]

# Slack channel
slack://TokenA/TokenB/TokenC/general [team]
```

## Command Line Usage

```bash
# Send simple notification
apprise-cli -t "Hello" -b "World" discord://webhook_id/webhook_token

# Send from config file
echo "Server is down!" | apprise-cli -t "Alert" -c config.yaml

# Send to multiple services with tags
apprise-cli -t "Deploy Success" -b "Version 1.2.3 deployed" --tag production

# Send with different notification types
apprise-cli -t "Error" -b "Database connection failed" -n error

# Send with custom format
apprise-cli -t "Report" -b "<b>Status:</b> OK" --format html
```

## Notification Types

All services support different notification types with appropriate styling:

- `NotifyTypeInfo` (default) - Blue/info styling
- `NotifyTypeSuccess` - Green styling with ✅ emoji  
- `NotifyTypeWarning` - Yellow styling with ⚠️ emoji
- `NotifyTypeError` - Red styling with ❌ emoji

## Error Handling

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token")
app.Add("slack://invalid_tokens")  // This will fail

responses := app.Notify("Test", "Message", apprise.NotifyTypeInfo)

for i, response := range responses {
    if response.Success {
        fmt.Printf("✓ Service %d: Success\n", i+1)
    } else {
        fmt.Printf("✗ Service %d: %v\n", i+1, response.Error)
    }
}
```

## Advanced Features

### Tags

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token", "alerts", "team")
app.Add("mailto://admin@company.com", "admin")

// Send to all services
app.Notify("General", "Message for everyone", apprise.NotifyTypeInfo)

// Send only to admin services  
app.Notify("Admin", "Admin only message", apprise.NotifyTypeWarning,
    apprise.WithTags("admin"))
```

### Custom Timeout

```go
app := apprise.New()
app.SetTimeout(60 * time.Second)  // 60 second timeout
```

### Body Formats

```go
app.Notify("Title", "**Bold** and _italic_ text", apprise.NotifyTypeInfo,
    apprise.WithBodyFormat("markdown"))

app.Notify("Title", "<b>Bold</b> and <i>italic</i> text", apprise.NotifyTypeInfo,
    apprise.WithBodyFormat("html"))
```

## Security Best Practices

1. **Never commit tokens to source code** - Use environment variables or config files
2. **Use HTTPS URLs** when possible (`webhooks://`, `mailtos://`, etc.)
3. **Validate webhook URLs** before adding them to prevent SSRF attacks
4. **Use strong passwords** for SMTP authentication
5. **Limit token permissions** to minimum required scope

## Attachment Support

Apprise Go provides comprehensive attachment support for services that support file uploads.

### Basic Attachment Usage

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token") // Supports attachments

// Add file attachment
err := app.AddAttachment("/path/to/file.pdf")
if err != nil {
    log.Fatal(err)
}

// Add attachment with custom name
err = app.AddAttachment("/path/to/file.txt", "custom_name.txt")

// Add attachment from URL
err = app.AddAttachment("https://example.com/image.png")

// Add attachment from raw data
data := []byte("Hello, World!")
err = app.AddAttachmentData(data, "hello.txt", "text/plain")

// Send notification with attachments
app.Notify("Title", "Message with attachments", apprise.NotifyTypeInfo)
```

### Attachment Types

**File Attachments:**
```go
// Local file
app.AddAttachment("/path/to/document.pdf")
app.AddAttachment("./relative/path/image.jpg", "custom_name.jpg")
```

**HTTP Attachments:**
```go
// Remote file via HTTP/HTTPS
app.AddAttachment("https://example.com/file.pdf")
app.AddAttachment("http://example.com/image.png", "screenshot.png")
```

**Memory Attachments:**
```go
// Raw data
data := []byte("File content here")
app.AddAttachmentData(data, "filename.txt", "text/plain")

// Data URL (base64 encoded)
app.AddAttachment("data:text/plain;base64,SGVsbG8gV29ybGQ=")
```

### Advanced Attachment Management

```go
app := apprise.New()

// Get attachment manager for advanced operations
mgr := app.GetAttachmentManager()

// Set maximum attachment size (100MB)
mgr.SetMaxSize(100 * 1024 * 1024)

// Set timeout for HTTP attachments
mgr.SetTimeout(60 * time.Second)

// Add multiple attachments
files := []string{
    "/path/to/report.pdf",
    "https://example.com/chart.png",
    "/path/to/data.csv",
}

for _, file := range files {
    if err := app.AddAttachment(file); err != nil {
        log.Printf("Failed to add %s: %v", file, err)
    }
}

// Check attachment info
fmt.Printf("Total attachments: %d\n", app.AttachmentCount())
for _, attachment := range app.GetAttachments() {
    fmt.Printf("- %s (%s, %d bytes)\n", 
        attachment.GetName(), 
        attachment.GetMimeType(), 
        attachment.GetSize())
}

// Send notification
app.Notify("Report", "Please see attached files", apprise.NotifyTypeInfo)

// Clear attachments for next notification
app.ClearAttachments()
```

### Service-Specific Attachment Support

| Service | Attachment Support | Notes |
|---------|-------------------|-------|
| Discord | ✅ Full | Images, documents, up to 8MB |
| Slack | ✅ Full | All file types, size limits apply |
| Telegram | ✅ Full | Photos, documents, audio, video |
| Email (SMTP) | 🚧 Planned | MIME multipart support |
| Pushbullet | ✅ Full | File uploads via API |
| Microsoft Teams | 🚧 Planned | Adaptive cards with attachments |
| Pushover | ✅ Images | Image attachments only |
| Webhook/JSON | ❌ Not supported | Use base64 encoding in payload |
| Twilio SMS | ❌ Not supported | SMS doesn't support attachments |
| Desktop Notifications | ❌ Not supported | Images via image parameter only |
| Gotify | ❌ Not supported | Text-only notifications |

### Attachment Security

```go
mgr := app.GetAttachmentManager()

// Limit attachment size
mgr.SetMaxSize(10 * 1024 * 1024) // 10MB limit

// Set timeout for HTTP downloads
mgr.SetTimeout(30 * time.Second)

// Validate attachments before sending
for _, attachment := range app.GetAttachments() {
    if !attachment.Exists() {
        log.Printf("Warning: Attachment %s is not accessible", attachment.GetName())
    }
    
    // Check file type
    mimeType := attachment.GetMimeType()
    if !isAllowedMimeType(mimeType) {
        log.Printf("Warning: Attachment %s has restricted type %s", 
            attachment.GetName(), mimeType)
    }
}
```

### Error Handling

```go
// Attachment operations can fail
if err := app.AddAttachment("/nonexistent/file.txt"); err != nil {
    log.Printf("Attachment error: %v", err)
}

// Check attachment availability
for _, attachment := range app.GetAttachments() {
    if !attachment.Exists() {
        log.Printf("Attachment %s is not available", attachment.GetName())
    }
}

// Services may reject attachments
responses := app.Notify("Title", "Message", apprise.NotifyTypeInfo)
for _, response := range responses {
    if !response.Success && response.Error != nil {
        log.Printf("Service %s failed: %v", response.ServiceID, response.Error)
    }
}
```

For more examples, see the `examples/` directory in the repository.