# Apprise Go

A Go port of the [Apprise notification library](https://github.com/caronc/apprise). Apprise allows you to send a notification to almost all of the most popular notification services available to us today such as: Telegram, Discord, Slack, Amazon SNS, Gotify, etc.

## Features

- **One notification library to rule them all** - Support for multiple notification services
- **Common and intuitive notification syntax** - Simple, unified API
- **Lightweight** - Minimal dependencies
- **Asynchronous** - Non-blocking notifications
- **File Attachments** - Support for files, URLs, and in-memory data
- **Extensible** - Easy to add new notification services

## Installation

```bash
go get github.com/scttfrdmn/apprise-go
```

## Usage

```go
package main

import (
    "github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
    // Create a new Apprise instance
    app := apprise.New()
    
    // Add notification services
    app.Add("discord://webhook_id/webhook_token")
    
    // Send a notification
    app.Notify("Hello World!", "This is a test notification")
}
```

## Supported Services

### ✅ Fully Implemented
- **Discord** - Webhook notifications with rich embeds
- **Slack** - Webhook and bot API support
- **Telegram** - Bot API with multiple chat support
- **Email (SMTP)** - Full SMTP support with TLS/STARTTLS
- **Webhook/JSON** - Generic HTTP webhooks with custom templates
- **Microsoft Teams** - Enterprise messaging with adaptive cards
- **Pushover** - Mobile push notifications with priority levels
- **Pushbullet** - Cross-platform push notifications
- **Twilio SMS** - SMS/MMS messaging with rate limiting

### 🚧 Coming Soon
- AWS SNS
- Gotify
- Matrix
- Signal
- WhatsApp Business
- And many more...

## License

This project is licensed under the BSD-2-Clause License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is a Go port of the original [Apprise](https://github.com/caronc/apprise) library by Chris Caron.