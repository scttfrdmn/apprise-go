# Apprise Go

A Go port of the [Apprise notification library](https://github.com/caronc/apprise). Apprise allows you to send a notification to almost all of the most popular notification services available to us today such as: Telegram, Discord, Slack, Amazon SNS, Gotify, etc.

## Features

- **One notification library to rule them all** - Support for multiple notification services
- **Common and intuitive notification syntax** - Simple, unified API
- **Lightweight** - Minimal dependencies
- **Asynchronous** - Non-blocking notifications
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

- Discord
- (More services coming soon...)

## License

This project is licensed under the BSD-2-Clause License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is a Go port of the original [Apprise](https://github.com/caronc/apprise) library by Chris Caron.