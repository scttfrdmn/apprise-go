# Slack Migration Guide

## URL Schema

- **Python:** `slack://TokenA/TokenB/TokenC/Channel`
- **Go:** `slack://TokenA/TokenB/TokenC/Channel`

## Changes

### Parameter

Channel parameter can now include # prefix

**Before:** `slack://token/token/token/general`

**After:** `slack://token/token/token/general or slack://token/token/token/#general`

### Behavior ⚠️ **Required**

Bot token validation is more strict

**Before:** `Accepts various token formats`

**After:** `Requires properly formatted bot tokens`

## Examples

### Slack webhook notification

**Python:**
```python
apprise.Apprise().add("slack://T123/B456/xyz789/general")
```

**Go:**
```go
app.Add("slack://T123/B456/xyz789/#general")
```

## Notes

- Token format validation is stricter in Go version
- Channel names are normalized consistently

