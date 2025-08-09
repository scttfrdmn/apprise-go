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

For more examples, see the `examples/` directory in the repository.