# Email (SMTP) Service Documentation

**Service ID:** `email`

Send email notifications via SMTP servers

## URL Format

```
mailto://user:pass@server:port/to_email
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `user` | string | **Yes** | SMTP username |  | `myemail@gmail.com` |
| `pass` | string | **Yes** | SMTP password or app password |  | `mypassword` |
| `server` | string | **Yes** | SMTP server hostname |  | `smtp.gmail.com` |
| `port` | int | No | SMTP port | 587 | `587` |
| `to` | string | **Yes** | Recipient email address |  | `recipient@example.com` |
| `from` | string | No | Sender name |  | `Notification System` |
| `name` | string | No | Sender display name |  | `MyApp Notifications` |

## Examples

### Example 1: Gmail SMTP email notification

**URL:**
```
mailto://myemail@gmail.com:mypassword@smtp.gmail.com:587/recipient@example.com
```

**Go Code:**
```go
app.Add("mailto://myemail@gmail.com:mypassword@smtp.gmail.com:587/recipient@example.com")
```

### Example 2: Custom SMTP email with sender name

**URL:**
```
mailto://user:pass@mail.company.com:587/admin@company.com?name=Alert+System
```

**Go Code:**
```go
app.Add("mailto://user:pass@mail.company.com:587/admin@company.com?name=Alert+System")
```

## Setup Instructions

1. Configure your SMTP server credentials
2. For Gmail: Enable 2FA and generate an app password
3. For other providers: Use provided SMTP settings
4. Test connectivity to SMTP server

## Limitations

- Requires SMTP server access
- May be rate-limited by email provider
- Credentials stored in configuration

