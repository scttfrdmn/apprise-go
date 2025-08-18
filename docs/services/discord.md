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

