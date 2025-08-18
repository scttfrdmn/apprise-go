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

