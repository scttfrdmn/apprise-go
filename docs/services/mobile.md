# Mobile Push Notifications

Mobile push notification services for iOS and Android

## Services

- **[Rich Mobile Push Notifications](#rich-mobile-push)** - Advanced mobile push notifications with rich content for iOS and Android

# Rich Mobile Push Notifications Service Documentation

**Service ID:** `rich-mobile-push`

Advanced mobile push notifications with rich content for iOS and Android

## URL Format

```
rich-mobile-push://platform@device_tokens
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `platform` | string | **Yes** | Target platform (ios/android/both) |  | `both` |
| `device_tokens` | string | **Yes** | Comma-separated device tokens |  | `token1,token2,token3` |
| `priority` | string | No | Notification priority (low/normal/high) | normal | `` |
| `sound` | string | No | Notification sound |  | `alert` |
| `badge` | int | No | App badge count |  | `1` |
| `category` | string | No | Notification category |  | `ALERT` |

## Examples

### Example 1: Rich mobile push for both platforms

**URL:**
```
rich-mobile-push://both@token1,token2,token3?priority=high&sound=alert&badge=1
```

**Go Code:**
```go
app.Add("rich-mobile-push://both@token1,token2,token3?priority=high&sound=alert&badge=1")
```

## Setup Instructions

1. Configure push certificates for iOS (APNS)
2. Configure service account for Android (FCM)
3. Collect device tokens from client applications
4. Test with development tokens first

---

