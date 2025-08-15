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

### Mattermost

Open-source team collaboration platform with API v4 support for self-hosted and cloud deployments.

**URL Formats:**
```
# Username/password authentication (HTTP)
mattermost://username:password@mattermost.example.com/general

# Token authentication (HTTPS)
mmosts://token@mattermost.company.com/alerts

# Custom port
mattermost://user:pass@mm.company.com:9000/general

# Multiple channels
mmosts://token@mattermost.example.com/general/alerts/dev-team

# With bot customization
mattermost://token@mm.example.com/general?bot=AlertBot&icon_emoji=:warning:&icon_url=https://example.com/icon.png
```

**Query Parameters:**
- `token=string` - Access token (alternative to URL auth)
- `bot=string` - Custom bot name for message display
- `icon_url=url` - Custom icon URL for bot avatar
- `icon_emoji=:emoji:` - Custom emoji for bot icon

**Features:**
- Mattermost API v4 compliance for broad compatibility
- Multiple authentication methods: username/password and personal access tokens
- Multi-channel messaging in single URL
- Channel name normalization (removes # and @ prefixes)
- Fragment URL parsing for channel references (#channel)
- Bot appearance customization (name, icon URL, emoji)
- Markdown message formatting with emoji support
- Automatic channel ID resolution via API
- Session management for username/password authentication

**Authentication Methods:**
1. **Personal Access Token** (recommended for production)
   ```go
   app.Add("mmosts://sdf2h3jh4k2j3h4k2j3h4k@mattermost.company.com/alerts")
   ```

2. **Username/Password** (for development/testing)
   ```go
   app.Add("mattermost://myuser:mypass@mattermost.example.com/general")
   ```

3. **Token in Query Parameter**
   ```go
   app.Add("mattermost://myuser@mm.example.com/general?token=access_token")
   ```

**Channel Formats:**
- **Simple Names**: `general`, `alerts`, `dev-team`
- **Hash Prefixed**: `#general` (automatically normalized)
- **Direct Messages**: `@username` (automatically normalized)

**Message Formatting:**
- Automatic emoji prefixing based on notification type
- Markdown bold formatting for titles: `**Title**`
- Multi-line support with proper spacing
- Fallback messages for empty notifications

**Example:**
```go
// Send alert to self-hosted Mattermost with custom bot appearance
app.Add("mmosts://token@mattermost.company.com/ops-alerts?bot=MonitoringBot&icon_emoji=:rotating_light:")
```

### PagerDuty

Enterprise incident management with Events API v2 support for both US and EU regions.

**URL Formats:**
```
# Basic integration key
pagerduty://integration_key

# Specify region explicitly  
pagerduty://integration_key@us
pagerduty://integration_key@eu

# With custom source and component
pagerduty://integration_key?source=monitoring&component=api

# Full configuration
pagerduty://integration_key?region=eu&source=server-01&component=database&group=production&class=critical
```

**Query Parameters:**
- `region=us|eu` - PagerDuty region (default: us)
- `source=string` - Alert source identifier (default: apprise-go)
- `component=string` - System component name
- `group=string` - Alert grouping identifier
- `class=string` - Alert classification

**Features:**
- Events API v2 with automatic severity mapping
- US and EU region support
- Custom alert metadata (source, component, group, class)
- Automatic deduplication support
- Title and body included in custom details
- Integration with PagerDuty's incident response workflows

**Example:**
```go
// Send critical database alert to EU region
app.Add("pagerduty://r1234567890abcdef1234567890abcdef@eu?source=db-cluster&component=primary")
```

### Opsgenie

Atlassian's incident management and alerting service with comprehensive responder and priority management.

**URL Formats:**
```
# Basic API key (defaults to US region)
opsgenie://api_key

# Specify region explicitly
opsgenie://api_key@us
opsgenie://api_key@eu

# With team and user responders
opsgenie://api_key@us/backend-team/user@example.com

# With priority and tags
opsgenie://api_key@us?priority=P1&tags=critical,production

# Full configuration
opsgenie://api_key@eu/oncall-team?priority=P2&teams=devops,backend&entity=web-server&source=monitoring&alias=db-alert
```

**Query Parameters:**
- `region=us|eu` - Opsgenie region (default: us)
- `priority=P1-P5` - Alert priority (P1=Critical, P5=Informational)
- `tags=tag1,tag2` - Comma-separated tags for alert categorization
- `teams=team1,team2` - Additional team responders (comma-separated)
- `alias=string` - Alert alias for deduplication
- `entity=string` - Entity name (server, application, etc.)
- `source=string` - Alert source identifier (default: apprise-go)
- `user=string` - User who created the alert
- `note=string` - Additional note for the alert

**Features:**
- Opsgenie Alerts API v2 compliance
- US and EU region support with appropriate API endpoints
- Multiple responder types: teams (path/query), users (email detection)
- Priority levels P1-P5 with automatic mapping from notification types
- Alert deduplication via alias parameter
- Rich alert metadata: entity, source, tags, notes
- Team and user responder assignment
- Integration with Opsgenie's incident response workflows
- Alert details with notification context

**Responder Detection:**
- **Teams**: Simple names in path or teams query parameter → `{"type": "team", "name": "backend"}`
- **Users**: Email addresses in path → `{"type": "user", "name": "user@example.com"}`
- **Mixed**: Can combine teams and users in single URL

**Priority Mapping:**
- `NotifyTypeError` → P1 (Critical)
- `NotifyTypeWarning` → P2 (High)  
- `NotifyTypeInfo` → P3 (Moderate)
- `NotifyTypeSuccess` → P4 (Low)

**Regional Endpoints:**
- **US Region**: `https://api.opsgenie.com/v2/alerts`
- **EU Region**: `https://api.eu.opsgenie.com/v2/alerts`

**Example:**
```go
// Send P1 alert to EU region with team responders and custom metadata
app.Add("opsgenie://abc123@eu/backend-team/devops?priority=P1&tags=production,database&entity=mysql-cluster&alias=db-performance")
```

### Matrix

Decentralized messaging with Client-Server API v3 support for both access token and username/password authentication.

**URL Formats:**
```
# Access token authentication (recommended)
matrix://access_token@matrix.org/!room_id:matrix.org
matrix://access_token@matrix.org/#room_alias:matrix.org

# Username/password authentication
matrix://username:password@matrix.example.com/general
matrix://username:password@homeserver.com/room1/room2

# Token in query parameter
matrix://username@matrix.org/general?token=access_token

# Multiple rooms and options
matrix://token@matrix.org/room1/room2/#room3:matrix.org?msgtype=notice&format=html
```

**Query Parameters:**
- `msgtype=text|notice` - Message type (default: text)
- `format=html` - Enable HTML formatting in messages
- `token=string` - Access token (alternative to URL auth)

**Features:**
- Matrix Client-Server API v3 compliance
- Support for both access token and username/password authentication
- Multiple room targeting in a single URL
- Room ID (!room:server) and alias (#room:server) formats
- Automatic room normalization (simple names become aliases)
- HTML message formatting with proper escaping
- Message and notice types (m.text, m.notice)
- Automatic login and session management
- Support for both public and private homeservers

**Authentication Methods:**
1. **Access Token** (recommended for production)
   ```go
   app.Add("matrix://syt_dXNlcm5hbWU_abcdef123456789@matrix.org/!room:matrix.org")
   ```

2. **Username/Password** (for development/testing)
   ```go
   app.Add("matrix://myuser:mypass@matrix.example.com/general")
   ```

3. **Mixed Authentication** (username with token in query)
   ```go
   app.Add("matrix://myuser@matrix.org/room?token=access_token")
   ```

**Room Formats:**
- **Room ID**: `!AbCdEf123456789:matrix.org` (specific room identifier)
- **Room Alias**: `#general:matrix.org` (human-readable room alias)
- **Simple Name**: `general` (auto-converts to `#general:homeserver`)

**Example:**
```go
// Send critical alert to Matrix operations room with HTML formatting
app.Add("matrix://access_token@company.matrix.org/#ops:company.matrix.org?msgtype=notice&format=html")
```

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

### AWS SNS

Amazon Simple Notification Service for enterprise cloud messaging with webhook proxy support.

**URL Formats:**
```
# Via API Gateway webhook endpoint
sns://api.gateway.url/sns-proxy?topic_arn=arn:aws:sns:us-east-1:123456789:my-topic

# Via API Gateway with API key authentication
sns://api-key@api.gateway.url/webhook?topic_arn=arn:aws:sns:eu-west-1:987654321:alerts

# Using topic components instead of full ARN
sns://webhook.example.com/sns?topic=notifications&region=us-west-2&account=123456789

# With custom message attributes and formatting
sns://webhook.url/proxy?topic=alerts&format=json&attr_Environment=production&attr_Service=web-api
```

**Query Parameters:**
- `topic_arn=arn:...` - Full SNS topic ARN (recommended)
- `topic=name` - Topic name (requires `account` parameter)
- `region=us-east-1` - AWS region (default: us-east-1)
- `account=123456789` - AWS account ID (when using topic name)
- `subject=string` - Custom subject line for notifications
- `format=text|json` - Message format (default: text)
- `attr_Key=value` - Custom message attributes (prefix with attr_)
- `test_mode=true` - Use HTTP instead of HTTPS (for testing only)

**Features:**
- Enterprise-grade cloud messaging via Amazon SNS
- Webhook proxy integration for secure API access
- JSON and text message formatting options
- Custom message attributes and metadata
- Subject line customization
- Message size up to 256KB
- Emoji indicators based on notification type
- API key authentication support
- Regional endpoint support

**Authentication Methods:**
1. **API Gateway with API Key** (recommended for production)
   ```go
   app.Add("sns://your-api-key@api.gateway.amazonaws.com/prod/sns?topic_arn=arn:aws:sns:us-east-1:123456789:alerts")
   ```

2. **Custom Webhook Proxy**
   ```go
   app.Add("sns://webhook.yourcompany.com/sns-proxy?topic=notifications&region=us-east-1&account=123456789")
   ```

3. **Direct Integration** (requires AWS SDK setup)
   ```go
   app.Add("sns://your-webhook.com/direct-sns?topic_arn=arn:aws:sns:us-east-1:123456789:topic&format=json")
   ```

**Message Formats:**
- **Text Format** (default): `🔔 Alert Title\n\nAlert details here`
- **JSON Format**: `{"title":"Alert Title","body":"Alert details","type":"warning","emoji":"⚠️","timestamp":"2024-01-15T10:30:00Z"}`

**Message Attributes:**
All notifications include these attributes:
- `NotificationType`: error, warning, info, success
- `Source`: apprise-go
- Custom attributes via `attr_` query parameters

**Integration Notes:**
This service sends webhook requests to your configured endpoint, which should then forward the message to AWS SNS. This approach provides:
- Secure credential management (keys stay on your server)
- Custom authentication and authorization
- Message transformation and routing
- Integration with existing AWS infrastructure

**Example Webhook Payload:**
```json
{
  "topicArn": "arn:aws:sns:us-east-1:123456789:alerts",
  "message": "⚠️ Database Warning\n\nConnection pool at 80% capacity",
  "subject": "Database Warning",
  "region": "us-east-1",
  "messageAttributes": {
    "NotificationType": {"DataType": "String", "StringValue": "warning"},
    "Source": {"DataType": "String", "StringValue": "apprise-go"},
    "Environment": {"DataType": "String", "StringValue": "production"}
  }
}
```

**Example:**
```go
// Send critical alert to SNS via API Gateway with custom attributes
app.Add("sns://api-key@gateway.us-east-1.amazonaws.com/prod/sns?topic_arn=arn:aws:sns:us-east-1:123456789:alerts&format=json&attr_Environment=prod&attr_Team=backend")
```

### AWS SES

Amazon Simple Email Service for enterprise-grade email delivery with template support and rich formatting.

**URL Formats:**
```
# Basic email via webhook proxy
ses://api.gateway.url/ses-proxy?from=alerts@company.com&to=admin@company.com

# Multiple recipients with CC/BCC
ses://webhook.example.com/ses?from=noreply@company.com&to=team@company.com,alerts@company.com&cc=manager@company.com&bcc=audit@company.com

# With API key authentication and custom options
ses://api-key@api.gateway.amazonaws.com/prod/ses?from=Alerts%20Team%20<alerts@company.com>&to=oncall@company.com&subject=Custom%20Subject&region=eu-west-1

# Using SES templates with dynamic data
ses://webhook.url/ses?from=system@company.com&to=user@company.com&template=welcome-email&data_username=john&data_company=Acme%20Corp
```

**Query Parameters:**
- `from=email` - Sender email address (required)
- `name=Name` - Sender display name (optional)
- `to=email1,email2` - Recipient email addresses (required, comma-separated)
- `cc=email1,email2` - CC recipients (optional, comma-separated)
- `bcc=email1,email2` - BCC recipients (optional, comma-separated)
- `reply_to=email` - Reply-to email address (optional)
- `subject=string` - Custom subject line (optional)
- `region=us-east-1` - AWS region (default: us-east-1)
- `template=name` - SES template name for templated emails (optional)
- `data_key=value` - Template data parameters (prefix with data_)
- `test_mode=true` - Use HTTP instead of HTTPS (for testing only)

**Features:**
- Enterprise email delivery via Amazon SES
- HTML and plain text message formatting
- Multiple recipients (TO, CC, BCC)
- Attachment support up to 10MB
- SES template integration with dynamic data
- Custom sender names and reply-to addresses
- Regional endpoint support
- Rich HTML formatting with responsive design
- Emoji indicators based on notification type
- Professional email signatures

**Authentication Methods:**
1. **API Gateway with API Key** (recommended)
   ```go
   app.Add("ses://your-api-key@api.gateway.amazonaws.com/prod/ses?from=alerts@company.com&to=oncall@company.com")
   ```

2. **Custom Webhook Proxy**
   ```go
   app.Add("ses://webhook.yourcompany.com/ses-proxy?from=system@company.com&to=admin@company.com")
   ```

**Message Formatting:**
- **HTML Version**: Professional email template with:
  - Responsive design for mobile compatibility
  - Color-coded headers based on notification type
  - Proper HTML escaping for security
  - Branded footer with timestamp
- **Text Version**: Clean plain text format with:
  - Emoji indicators for notification types
  - Structured layout with clear sections
  - Professional signature

**Template Integration:**
Use SES templates for consistent branding and dynamic content:
```go
app.Add("ses://webhook.url/ses?from=alerts@company.com&to=team@company.com&template=incident-alert&data_severity=critical&data_service=database&data_environment=production")
```

Template data automatically includes:
- `title` - Notification title
- `body` - Notification body  
- `notifyType` - Notification type (info, warning, error, success)
- `timestamp` - ISO 8601 timestamp
- Custom data via `data_` query parameters

**Attachment Support:**
SES supports various attachment types:
```go
app := apprise.New()
app.Add("ses://webhook.url/ses?from=reports@company.com&to=team@company.com")

// Add file attachments
app.AddAttachment("/path/to/report.pdf")
app.AddAttachment("https://example.com/chart.png", "monthly_chart.png")

// Add data attachments
data := []byte("CSV,Data\nvalue1,value2")
app.AddAttachmentData(data, "report.csv", "text/csv")

app.Notify("Monthly Report", "Please find the monthly report attached", apprise.NotifyTypeInfo)
```

**Example Webhook Payload:**
```json
{
  "region": "us-east-1",
  "source": "Alerts Team <alerts@company.com>",
  "destination": {
    "toAddresses": ["oncall@company.com"],
    "ccAddresses": ["manager@company.com"],
    "bccAddresses": ["audit@company.com"]
  },
  "message": {
    "subject": {
      "data": "Database Alert",
      "charset": "UTF-8"
    },
    "body": {
      "html": {
        "data": "<!DOCTYPE html><html>...<h2 style=\"color: #dc3545;\">❌ Database Connection Failed</h2>...",
        "charset": "UTF-8"
      },
      "text": {
        "data": "❌ Database Connection Failed\n\nUnable to connect to primary database server...",
        "charset": "UTF-8"
      }
    }
  },
  "replyToAddresses": ["support@company.com"],
  "attachments": [
    {
      "filename": "error_log.txt",
      "contentType": "text/plain",
      "data": "base64-encoded-content",
      "size": 1024
    }
  ]
}
```

**Integration Notes:**
This service sends webhook requests to your configured endpoint, which should forward the email via AWS SES. This approach provides:
- Secure credential management (AWS keys stay on your server)
- Template customization and branding
- Compliance and audit logging
- Integration with existing SES configurations (reputation management, bounce handling)
- Cost optimization through SES pricing

**Example:**
```go
// Send critical alert with attachments via SES with custom template
app.Add("ses://api-key@gateway.amazonaws.com/prod/ses?from=Critical%20Alerts%20<critical@company.com>&to=oncall@company.com,management@company.com&cc=security@company.com&template=security-incident&data_incident_id=INC-2024-001&data_severity=high")
```

### Ntfy

Simple HTTP push notifications with priority levels, perfect for self-hosted setups and lightweight notifications.

**URL Formats:**
```
# Public ntfy.sh (HTTPS)
ntfys://ntfy.sh/my-topic

# Self-hosted (HTTP)
ntfy://ntfy.example.com:8080/alerts

# With authentication
ntfy://username:password@ntfy.example.com/notifications
ntfys://token@ntfy.sh/alerts

# With priority and tags
ntfy://ntfy.sh/alerts?priority=5&tags=urgent,production

# With advanced features
ntfy://ntfy.sh/alerts?delay=30min&email=admin@example.com&attach=https://example.com/file.pdf
```

**Query Parameters:**
- `priority=1-5` - Message priority (1=min, 3=default, 5=max)
- `tags=tag1,tag2` - Comma-separated tags for message categorization
- `delay=30min` - Delay message delivery (e.g., 30s, 5min, 1h)
- `actions=action1,Label1,url1` - Action buttons (comma-separated)
- `attach=url` - Attachment URL
- `filename=name` - Custom attachment filename
- `click=url` - URL to open when notification is clicked
- `email=address` - Forward notification to email
- `token=string` - Access token (alternative to URL auth)

**Features:**
- Simple HTTP-based push notifications
- Priority levels (1-5) with automatic mapping from notification types
- Tag-based message categorization with emoji support
- Delayed message delivery
- Email forwarding integration
- Attachment support via URLs
- Action buttons for interactive notifications
- Click URLs for notification actions
- Self-hosted and public ntfy.sh support
- Token and username/password authentication

**Priority Mapping:**
- `NotifyTypeInfo` → Priority 3 (Normal)
- `NotifyTypeSuccess` → Priority 3 (Normal)  
- `NotifyTypeWarning` → Priority 4 (High)
- `NotifyTypeError` → Priority 5 (Max)

**Emoji Tags:**
When no custom tags are provided, automatic emoji tags are added:
- ✅ `white_check_mark` for success notifications
- ⚠️ `warning` for warning notifications  
- 🚨 `rotating_light` for error notifications
- ℹ️ `information_source` for info notifications

**Example:**
```go
// Send high-priority alert with custom tags to self-hosted ntfy
app.Add("ntfy://token@ntfy.company.com/alerts?priority=4&tags=production,database&email=oncall@company.com")
```

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
| Matrix | ✅ Full | Media uploads via Matrix API |
| Opsgenie | ❌ Not supported | Alert API doesn't support attachments |
| Pushbullet | ✅ Full | File uploads via API |
| Microsoft Teams | 🚧 Planned | Adaptive cards with attachments |
| Mattermost | ✅ Full | File uploads via API v4 |
| Pushover | ✅ Images | Image attachments only |
| Webhook/JSON | ❌ Not supported | Use base64 encoding in payload |
| Twilio SMS | ❌ Not supported | SMS doesn't support attachments |
| Desktop Notifications | ❌ Not supported | Images via image parameter only |
| Gotify | ❌ Not supported | Text-only notifications |
| Ntfy | ✅ URLs | Attachment support via URLs only |

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