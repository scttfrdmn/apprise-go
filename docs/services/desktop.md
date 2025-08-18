# Desktop Notifications

Desktop notification systems for local alerts

## Services

- **[Advanced Desktop Notifications](#desktop-advanced)** - Enhanced desktop notifications with actions, timeouts, and custom UI

# Advanced Desktop Notifications Service Documentation

**Service ID:** `desktop-advanced`

Enhanced desktop notifications with actions, timeouts, and custom UI

## URL Format

```
desktop-advanced://
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `sound` | string | No | Notification sound | default | `` |
| `timeout` | int | No | Timeout in seconds | 5 | `` |
| `category` | string | No | Notification category |  | `ALERT` |
| `action1` | string | No | First action (id:title:url) |  | `view:View Details:https://example.com` |
| `action2` | string | No | Second action |  | `dismiss:Dismiss` |

## Examples

### Example 1: Advanced desktop notification with actions

**URL:**
```
desktop-advanced://?sound=alert&timeout=10&action1=view:View:https://example.com&action2=dismiss:Dismiss
```

**Go Code:**
```go
app.Add("desktop-advanced://?sound=alert&timeout=10&action1=view:View:https://example.com&action2=dismiss:Dismiss")
```

## Setup Instructions

1. No setup required for basic functionality
2. On Linux: Install notify-send or similar
3. On macOS: Uses built-in notification center
4. On Windows: Uses Windows toast notifications

---

