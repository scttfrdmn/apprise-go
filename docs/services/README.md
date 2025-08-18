# Apprise-Go Service Documentation

This document provides comprehensive documentation for all notification services supported by Apprise-Go.

## Table of Contents

- [Email Services](#email-services)
- [SMS & Text Messaging](#sms-&-text-messaging)
- [Desktop Notifications](#desktop-notifications)
- [Cloud Services](#cloud-services)
- [IoT & Automation](#iot-&-automation)
- [Messaging & Chat](#messaging-&-chat)
- [Social Media](#social-media)
- [Mobile Push Notifications](#mobile-push-notifications)
- [Instant Messaging](#instant-messaging)
- [Push Notification Services](#push-notification-services)
- [DevOps & Monitoring](#devops-&-monitoring)
- [Voice & Audio](#voice-&-audio)
- [Webhooks & APIs](#webhooks-&-apis)

## Desktop Notifications

Desktop notification systems for local alerts

### Services

- [Advanced Desktop Notifications](#desktop-advanced) - Enhanced desktop notifications with actions, timeouts, and custom UI

### Advanced Desktop Notifications

**Service ID:** `desktop-advanced`

Enhanced desktop notifications with actions, timeouts, and custom UI

**URL Format:**
```
desktop-advanced://
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `sound` | string | No | Notification sound | default | `` |
| `timeout` | int | No | Timeout in seconds | 5 | `` |
| `category` | string | No | Notification category |  | `ALERT` |
| `action1` | string | No | First action (id:title:url) |  | `view:View Details:https://example.com` |
| `action2` | string | No | Second action |  | `dismiss:Dismiss` |

**Examples:**

*Advanced desktop notification with actions:*
```go
app.Add("desktop-advanced://?sound=alert&timeout=10&action1=view:View:https://example.com&action2=dismiss:Dismiss")
```

**Setup:**

1. No setup required for basic functionality
2. On Linux: Install notify-send or similar
3. On macOS: Uses built-in notification center
4. On Windows: Uses Windows toast notifications

---

## Cloud Services

Cloud platform notification and messaging services

### Services


## IoT & Automation

Internet of Things and home automation platforms

### Services


## Messaging & Chat

Real-time messaging platforms and chat services

### Services

- [Discord](#discord) - Send notifications to Discord channels via webhooks
- [Slack](#slack) - Send notifications to Slack channels and users

### Discord

**Service ID:** `discord`

Send notifications to Discord channels via webhooks

**URL Format:**
```
discord://webhook_id/webhook_token
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `webhook_id` | string | **Yes** | Discord webhook ID |  | `123456789012345678` |
| `webhook_token` | string | **Yes** | Discord webhook token |  | `abcdef123456789` |
| `avatar` | string | No | Custom avatar URL or name |  | `MyBot` |
| `username` | string | No | Custom username for the bot |  | `NotificationBot` |
| `tts` | bool | No | Enable text-to-speech |  | `true` |
| `format` | string | No | Message format (text/markdown) | text | `markdown` |

**Examples:**

*Basic Discord notification:*
```go
app.Add("discord://123456789012345678/abcdef123456789")
```

*Discord with custom avatar and username:*
```go
app.Add("discord://123456789012345678/abcdef123456789?avatar=MyBot&username=NotificationBot")
```

*Discord with markdown formatting:*
```go
app.Add("discord://123456789012345678/abcdef123456789?format=markdown")
```

**Setup:**

1. Go to your Discord server settings
2. Navigate to 'Integrations' → 'Webhooks'
3. Click 'New Webhook' or 'Create Webhook'
4. Configure the webhook name and channel
5. Copy the webhook URL
6. Extract webhook_id and webhook_token from the URL

**Limitations:**

- Requires webhook permissions in Discord server
- Rate limited by Discord's webhook limits
- Message length limited to 2000 characters

---

### Slack

**Service ID:** `slack`

Send notifications to Slack channels and users

**URL Format:**
```
slack://TokenA/TokenB/TokenC/Channel
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `token` | string | **Yes** | Slack bot token or webhook URL |  | `xoxb-123-456-789` |
| `channel` | string | **Yes** | Channel name or ID |  | `#general` |
| `username` | string | No | Bot username |  | `NotificationBot` |
| `icon` | string | No | Bot icon emoji or URL |  | `:robot_face:` |
| `format` | string | No | Message format (text/markdown) | text | `` |

**Examples:**

*Slack webhook notification:*
```go
app.Add("slack://T123456/B123456/xyz789abc/#general")
```

*Slack with custom username and icon:*
```go
app.Add("slack://T123456/B123456/xyz789abc/#alerts?username=AlertBot&icon=:warning:")
```

**Setup:**

1. Create a Slack app at https://api.slack.com/apps
2. Add bot token scopes: chat:write, chat:write.public
3. Install the app to your workspace
4. Copy the bot token (starts with xoxb-)
5. Invite the bot to your channels

---

## Social Media

Social media platforms for public notifications

### Services


## Mobile Push Notifications

Mobile push notification services for iOS and Android

### Services

- [Rich Mobile Push Notifications](#rich-mobile-push) - Advanced mobile push notifications with rich content for iOS and Android

### Rich Mobile Push Notifications

**Service ID:** `rich-mobile-push`

Advanced mobile push notifications with rich content for iOS and Android

**URL Format:**
```
rich-mobile-push://platform@device_tokens
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `platform` | string | **Yes** | Target platform (ios/android/both) |  | `both` |
| `device_tokens` | string | **Yes** | Comma-separated device tokens |  | `token1,token2,token3` |
| `priority` | string | No | Notification priority (low/normal/high) | normal | `` |
| `sound` | string | No | Notification sound |  | `alert` |
| `badge` | int | No | App badge count |  | `1` |
| `category` | string | No | Notification category |  | `ALERT` |

**Examples:**

*Rich mobile push for both platforms:*
```go
app.Add("rich-mobile-push://both@token1,token2,token3?priority=high&sound=alert&badge=1")
```

**Setup:**

1. Configure push certificates for iOS (APNS)
2. Configure service account for Android (FCM)
3. Collect device tokens from client applications
4. Test with development tokens first

---

## Instant Messaging

Instant messaging and secure communication platforms

### Services


## Push Notification Services

Push notification platforms and services

### Services


## DevOps & Monitoring

Development operations and system monitoring platforms

### Services


## Voice & Audio

Voice calling and audio notification services

### Services


## Webhooks & APIs

Generic webhook and API notification endpoints

### Services


## Email Services

Email notification services and providers

### Services

- [Email (SMTP)](#email) - Send email notifications via SMTP servers
- [SendGrid Email](#sendgrid) - Send email notifications via SendGrid API

### Email (SMTP)

**Service ID:** `email`

Send email notifications via SMTP servers

**URL Format:**
```
mailto://user:pass@server:port/to_email
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `user` | string | **Yes** | SMTP username |  | `myemail@gmail.com` |
| `pass` | string | **Yes** | SMTP password or app password |  | `mypassword` |
| `server` | string | **Yes** | SMTP server hostname |  | `smtp.gmail.com` |
| `port` | int | No | SMTP port | 587 | `587` |
| `to` | string | **Yes** | Recipient email address |  | `recipient@example.com` |
| `from` | string | No | Sender name |  | `Notification System` |
| `name` | string | No | Sender display name |  | `MyApp Notifications` |

**Examples:**

*Gmail SMTP email notification:*
```go
app.Add("mailto://myemail@gmail.com:mypassword@smtp.gmail.com:587/recipient@example.com")
```

*Custom SMTP email with sender name:*
```go
app.Add("mailto://user:pass@mail.company.com:587/admin@company.com?name=Alert+System")
```

**Setup:**

1. Configure your SMTP server credentials
2. For Gmail: Enable 2FA and generate an app password
3. For other providers: Use provided SMTP settings
4. Test connectivity to SMTP server

**Limitations:**

- Requires SMTP server access
- May be rate-limited by email provider
- Credentials stored in configuration

---

### SendGrid Email

**Service ID:** `sendgrid`

Send email notifications via SendGrid API

**URL Format:**
```
sendgrid://api_key@from_email/to_email
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `api_key` | string | **Yes** | SendGrid API key |  | `SG.abc123...` |
| `from_email` | string | **Yes** | Verified sender email |  | `noreply@myapp.com` |
| `to_email` | string | **Yes** | Recipient email address |  | `user@example.com` |
| `name` | string | No | Sender name |  | `MyApp Notifications` |

**Examples:**

*SendGrid email notification:*
```go
app.Add("sendgrid://SG.abc123@noreply@myapp.com/user@example.com")
```

**Setup:**

1. Create SendGrid account at https://sendgrid.com
2. Verify your sender identity
3. Generate an API key with Mail Send permissions
4. Configure your from address

---

## SMS & Text Messaging

SMS and text messaging services

### Services

- [Twilio SMS](#twilio) - Send SMS notifications via Twilio

### Twilio SMS

**Service ID:** `twilio`

Send SMS notifications via Twilio

**URL Format:**
```
twilio://account_sid:auth_token@from_number/to_number
```

**Parameters:**

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `account_sid` | string | **Yes** | Twilio Account SID |  | `AC123456789abcdef` |
| `auth_token` | string | **Yes** | Twilio Auth Token |  | `your_auth_token` |
| `from_number` | string | **Yes** | Twilio phone number |  | `+1234567890` |
| `to_number` | string | **Yes** | Recipient phone number |  | `+9876543210` |

**Examples:**

*Twilio SMS notification:*
```go
app.Add("twilio://AC123456789abcdef:auth_token@+1234567890/+9876543210")
```

**Setup:**

1. Create Twilio account at https://twilio.com
2. Purchase a phone number
3. Get Account SID and Auth Token from console
4. Verify recipient numbers (for trial accounts)

---

