# Messaging & Chat

Real-time messaging platforms and chat services

## Services

- **[Discord](#discord)** - Send notifications to Discord channels via webhooks
- **[Slack](#slack)** - Send notifications to Slack channels and users

# Discord Service Documentation

**Service ID:** `discord`

Send notifications to Discord channels via webhooks

## URL Format

```
discord://webhook_id/webhook_token
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `webhook_id` | string | **Yes** | Discord webhook ID |  | `123456789012345678` |
| `webhook_token` | string | **Yes** | Discord webhook token |  | `abcdef123456789` |
| `avatar` | string | No | Custom avatar URL or name |  | `MyBot` |
| `username` | string | No | Custom username for the bot |  | `NotificationBot` |
| `tts` | bool | No | Enable text-to-speech |  | `true` |
| `format` | string | No | Message format (text/markdown) | text | `markdown` |

## Examples

### Example 1: Basic Discord notification

**URL:**
```
discord://123456789012345678/abcdef123456789
```

**Go Code:**
```go
app.Add("discord://123456789012345678/abcdef123456789")
```

### Example 2: Discord with custom avatar and username

**URL:**
```
discord://123456789012345678/abcdef123456789?avatar=MyBot&username=NotificationBot
```

**Go Code:**
```go
app.Add("discord://123456789012345678/abcdef123456789?avatar=MyBot&username=NotificationBot")
```

### Example 3: Discord with markdown formatting

**URL:**
```
discord://123456789012345678/abcdef123456789?format=markdown
```

**Go Code:**
```go
app.Add("discord://123456789012345678/abcdef123456789?format=markdown")
```

## Setup Instructions

1. Go to your Discord server settings
2. Navigate to 'Integrations' → 'Webhooks'
3. Click 'New Webhook' or 'Create Webhook'
4. Configure the webhook name and channel
5. Copy the webhook URL
6. Extract webhook_id and webhook_token from the URL

## Limitations

- Requires webhook permissions in Discord server
- Rate limited by Discord's webhook limits
- Message length limited to 2000 characters

---

# Slack Service Documentation

**Service ID:** `slack`

Send notifications to Slack channels and users

## URL Format

```
slack://TokenA/TokenB/TokenC/Channel
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `token` | string | **Yes** | Slack bot token or webhook URL |  | `xoxb-123-456-789` |
| `channel` | string | **Yes** | Channel name or ID |  | `#general` |
| `username` | string | No | Bot username |  | `NotificationBot` |
| `icon` | string | No | Bot icon emoji or URL |  | `:robot_face:` |
| `format` | string | No | Message format (text/markdown) | text | `` |

## Examples

### Example 1: Slack webhook notification

**URL:**
```
slack://T123456/B123456/xyz789abc/#general
```

**Go Code:**
```go
app.Add("slack://T123456/B123456/xyz789abc/#general")
```

### Example 2: Slack with custom username and icon

**URL:**
```
slack://T123456/B123456/xyz789abc/#alerts?username=AlertBot&icon=:warning:
```

**Go Code:**
```go
app.Add("slack://T123456/B123456/xyz789abc/#alerts?username=AlertBot&icon=:warning:")
```

## Setup Instructions

1. Create a Slack app at https://api.slack.com/apps
2. Add bot token scopes: chat:write, chat:write.public
3. Install the app to your workspace
4. Copy the bot token (starts with xoxb-)
5. Invite the bot to your channels

---

