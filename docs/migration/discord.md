# Discord Migration Guide

## URL Schema

- **Python:** `discord://webhook_id/webhook_token`
- **Go:** `discord://webhook_id/webhook_token`

## Changes

### Parameter

Avatar parameter syntax changed

**Before:** `?avatar=https://example.com/image.png`

**After:** `?avatar=MyBot (name) or ?avatar=https://example.com/image.png (URL)`

### Behavior

Default message format is now text instead of markdown

**Before:** `markdown by default`

**After:** `text by default, use ?format=markdown for markdown`

## Examples

### Basic Discord notification

**Python:**
```python
apprise.Apprise().add("discord://123456789/abcdef123456789")
```

**Go:**
```go
app := apprise.New(); app.Add("discord://123456789/abcdef123456789")
```

### Discord with custom formatting

**Python:**
```python
apprise.Apprise().add("discord://123456789/abcdef123456789?format=markdown")
```

**Go:**
```go
app.Add("discord://123456789/abcdef123456789?format=markdown")
```

## Notes

- URL format is identical between Python and Go versions
- Parameter handling is consistent
- Error handling uses Go idioms instead of Python exceptions

