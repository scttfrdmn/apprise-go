# Email Migration Guide

## URL Schema

- **Python:** `mailto://user:pass@server:port/to_email`
- **Go:** `mailto://user:pass@server:port/to_email`

## Changes

### Parameter

TLS handling is automatic based on port

**Before:** `?secure=yes for TLS`

**After:** `Automatic TLS detection, use ?tls=false to disable`

### Behavior

SMTP authentication is more robust

**Before:** `Basic SMTP auth`

**After:** `Supports PLAIN, LOGIN, and CRAM-MD5`

## Examples

### Gmail SMTP configuration

**Python:**
```python
apprise.Apprise().add("mailto://user:pass@smtp.gmail.com:587/to@example.com")
```

**Go:**
```go
app.Add("mailto://user:pass@smtp.gmail.com:587/to@example.com")
```

## Notes

- SMTP connection handling is more reliable
- TLS negotiation is automatic
- Better error reporting for authentication failures

