# Twilio SMS Service Documentation

**Service ID:** `twilio`

Send SMS notifications via Twilio

## URL Format

```
twilio://account_sid:auth_token@from_number/to_number
```

## Parameters

| Name | Type | Required | Description | Default | Example |
|------|------|----------|-------------|---------|----------|
| `account_sid` | string | **Yes** | Twilio Account SID |  | `AC123456789abcdef` |
| `auth_token` | string | **Yes** | Twilio Auth Token |  | `your_auth_token` |
| `from_number` | string | **Yes** | Twilio phone number |  | `+1234567890` |
| `to_number` | string | **Yes** | Recipient phone number |  | `+9876543210` |

## Examples

### Example 1: Twilio SMS notification

**URL:**
```
twilio://AC123456789abcdef:auth_token@+1234567890/+9876543210
```

**Go Code:**
```go
app.Add("twilio://AC123456789abcdef:auth_token@+1234567890/+9876543210")
```

## Setup Instructions

1. Create Twilio account at https://twilio.com
2. Purchase a phone number
3. Get Account SID and Auth Token from console
4. Verify recipient numbers (for trial accounts)

