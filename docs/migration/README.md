# Migration Guide: Python Apprise to Apprise-Go

This guide helps you migrate from Python Apprise to Apprise-Go.

## Overview

Apprise-Go maintains compatibility with Python Apprise URL formats while providing:
- Better performance and lower memory usage
- Native Go concurrency and error handling
- Enhanced service features and reliability
- Simplified deployment (single binary)

## General Migration Steps

1. Install Apprise-Go: `go get github.com/scttfrdmn/apprise-go/apprise`
2. Replace Python imports with Go imports
3. Adapt Python syntax to Go syntax
4. Update configuration files if needed
5. Test notifications with your existing URLs

## Basic Syntax Comparison

### Python
```python
import apprise

apobj = apprise.Apprise()
apobj.add('discord://webhook_id/webhook_token')
apobj.notify('Hello World', title='Test')
```

### Go
```go
import "github.com/scttfrdmn/apprise-go/apprise"

app := apprise.New()
app.Add("discord://webhook_id/webhook_token")
app.Notify("Test", "Hello World", apprise.NotifyTypeInfo)
```

## Service-Specific Migration

### Slack

**URL Schema:**
- Python: `slack://TokenA/TokenB/TokenC/Channel`
- Go: `slack://TokenA/TokenB/TokenC/Channel`

**Changes:**
- **Parameter**: Channel parameter can now include # prefix
  - Before: `slack://token/token/token/general`
  - After: `slack://token/token/token/general or slack://token/token/token/#general`
- **Behavior**: Bot token validation is more strict
  - Before: `Accepts various token formats`
  - After: `Requires properly formatted bot tokens`
  - **Required Change**

**Migration Examples:**

*Slack webhook notification:*

Python:
```python
apprise.Apprise().add("slack://T123/B456/xyz789/general")
```

Go:
```go
app.Add("slack://T123/B456/xyz789/#general")
```

**Migration Notes:**
- Token format validation is stricter in Go version
- Channel names are normalized consistently

---

### Email

**URL Schema:**
- Python: `mailto://user:pass@server:port/to_email`
- Go: `mailto://user:pass@server:port/to_email`

**Changes:**
- **Parameter**: TLS handling is automatic based on port
  - Before: `?secure=yes for TLS`
  - After: `Automatic TLS detection, use ?tls=false to disable`
- **Behavior**: SMTP authentication is more robust
  - Before: `Basic SMTP auth`
  - After: `Supports PLAIN, LOGIN, and CRAM-MD5`

**Migration Examples:**

*Gmail SMTP configuration:*

Python:
```python
apprise.Apprise().add("mailto://user:pass@smtp.gmail.com:587/to@example.com")
```

Go:
```go
app.Add("mailto://user:pass@smtp.gmail.com:587/to@example.com")
```

**Migration Notes:**
- SMTP connection handling is more reliable
- TLS negotiation is automatic
- Better error reporting for authentication failures

---

### Twilio

**URL Schema:**
- Python: `twilio://account_sid:auth_token@from_number/to_number`
- Go: `twilio://account_sid:auth_token@from_number/to_number`

**Changes:**
- **Parameter**: Phone number formatting is normalized
  - Before: `Various formats accepted`
  - After: `E.164 format preferred (+1234567890)`
- **Behavior**: SMS length limits are enforced
  - Before: `Long messages may fail silently`
  - After: `Messages over 160 chars are split or rejected with error`

**Migration Examples:**

*Twilio SMS configuration:*

Python:
```python
apprise.Apprise().add("twilio://SID:TOKEN@+1234567890/+0987654321")
```

Go:
```go
app.Add("twilio://SID:TOKEN@+1234567890/+0987654321")
```

**Migration Notes:**
- Phone number validation is stricter
- Better error messages for invalid numbers
- Supports international formatting

---

### Desktop

**URL Schema:**
- Python: `desktop://`
- Go: `desktop://`

**Changes:**
- **Behavior**: Cross-platform notification handling improved
  - Before: `Platform-specific implementations`
  - After: `Unified API with platform-specific optimizations`
- **Parameter**: New advanced desktop notification options
  - Before: `desktop://`
  - After: `desktop-advanced:// for enhanced features`

**Migration Examples:**

*Basic desktop notification:*

Python:
```python
apprise.Apprise().add("desktop://")
```

Go:
```go
app.Add("desktop://")
```

*Advanced desktop notification (Go only):*

Python:
```python
N/A
```

Go:
```go
app.Add("desktop-advanced://?sound=alert&timeout=10")
```

**Migration Notes:**
- Go version includes advanced desktop notification features
- Better cross-platform compatibility
- Interactive notifications available on supported platforms

---

### Discord

**URL Schema:**
- Python: `discord://webhook_id/webhook_token`
- Go: `discord://webhook_id/webhook_token`

**Changes:**
- **Parameter**: Avatar parameter syntax changed
  - Before: `?avatar=https://example.com/image.png`
  - After: `?avatar=MyBot (name) or ?avatar=https://example.com/image.png (URL)`
- **Behavior**: Default message format is now text instead of markdown
  - Before: `markdown by default`
  - After: `text by default, use ?format=markdown for markdown`

**Migration Examples:**

*Basic Discord notification:*

Python:
```python
apprise.Apprise().add("discord://123456789/abcdef123456789")
```

Go:
```go
app := apprise.New(); app.Add("discord://123456789/abcdef123456789")
```

*Discord with custom formatting:*

Python:
```python
apprise.Apprise().add("discord://123456789/abcdef123456789?format=markdown")
```

Go:
```go
app.Add("discord://123456789/abcdef123456789?format=markdown")
```

**Migration Notes:**
- URL format is identical between Python and Go versions
- Parameter handling is consistent
- Error handling uses Go idioms instead of Python exceptions

---

## Common Migration Patterns

### Error Handling

**Python:**
```python
try:
    apobj.notify('message')
except Exception as e:
    print(f'Error: {e}')
```

**Go:**
```go
responses := app.Notify("Title", "Message", apprise.NotifyTypeInfo)
for _, response := range responses {
    if response.Error != nil {
        fmt.Printf("Error: %v\n", response.Error)
    }
}
```

### Batch Notifications

**Python:**
```python
apobj.add(['discord://...', 'slack://...'])
apobj.notify('message')
```

**Go:**
```go
urls := []string{"discord://...", "slack://..."}
for _, url := range urls {
    app.Add(url)
}
app.Notify("Title", "Message", apprise.NotifyTypeInfo)
```

### Configuration Files

**Python:**
```python
config = apprise.AppriseConfig()
config.add('path/to/config.yml')
apobj.add(config)
```

**Go:**
```go
config := apprise.NewConfigLoader(app)
config.AddFromFile("path/to/config.yml")
config.ApplyToApprise()
```

## Advanced Features in Apprise-Go

The Go version includes several enhancements not available in Python:

- **Rich Mobile Push**: Enhanced push notifications with actions and rich content
- **Advanced Desktop**: Interactive desktop notifications with callbacks
- **Configuration Templating**: Environment-based configuration with templates
- **Batch Processing**: Optimized batch notification handling
- **Concurrent Delivery**: Native Go concurrency for faster delivery
- **Better Error Handling**: Detailed error responses per service

## Troubleshooting

### URL Format Issues
- Ensure special characters are properly URL-encoded
- Check that tokens and credentials are valid
- Verify service-specific parameter requirements

### Service Differences
- Some services have stricter validation in Go version
- Error messages are more descriptive
- Rate limiting behavior may differ slightly

### Performance Considerations
- Go version typically uses less memory
- Concurrent notifications are faster
- Startup time is significantly reduced

