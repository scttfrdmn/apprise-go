# Twilio Migration Guide

## URL Schema

- **Python:** `twilio://account_sid:auth_token@from_number/to_number`
- **Go:** `twilio://account_sid:auth_token@from_number/to_number`

## Changes

### Parameter

Phone number formatting is normalized

**Before:** `Various formats accepted`

**After:** `E.164 format preferred (+1234567890)`

### Behavior

SMS length limits are enforced

**Before:** `Long messages may fail silently`

**After:** `Messages over 160 chars are split or rejected with error`

## Examples

### Twilio SMS configuration

**Python:**
```python
apprise.Apprise().add("twilio://SID:TOKEN@+1234567890/+0987654321")
```

**Go:**
```go
app.Add("twilio://SID:TOKEN@+1234567890/+0987654321")
```

## Notes

- Phone number validation is stricter
- Better error messages for invalid numbers
- Supports international formatting

