# Apprise-Go Documentation

Welcome to the comprehensive documentation for Apprise-Go, a high-performance notification delivery library for Go.

## 📚 Documentation Sections

### Service Documentation
- [**Complete Service Reference**](services/README.md) - All supported notification services
- [**Service Categories**](services/) - Services organized by category
- [**API Reference**](services/api.json) - Machine-readable service documentation

### Category Documentation
- [MESSAGING](services/messaging.md)
- [EMAIL](services/email.md)
- [SMS](services/sms.md)
- [MOBILE](services/mobile.md)
- [DESKTOP](services/desktop.md)
- [SOCIAL](services/social.md)

### Individual Service Documentation
- [discord](services/discord.md)
- [slack](services/slack.md)
- [email](services/email.md)
- [twilio](services/twilio.md)
- [rich-mobile-push](services/rich-mobile-push.md)
- [desktop-advanced](services/desktop-advanced.md)

### Migration Documentation
- [**Migration Guide**](migration/README.md) - Complete guide for migrating from Python Apprise
- [discord Migration](migration/discord.md)
- [slack Migration](migration/slack.md)
- [email Migration](migration/email.md)
- [twilio Migration](migration/twilio.md)

## 🚀 Quick Start

### Installation
```bash
go get github.com/scttfrdmn/apprise-go/apprise
```

### Basic Usage
```go
package main

import (
    "github.com/scttfrdmn/apprise-go/apprise"
)

func main() {
    // Create Apprise instance
    app := apprise.New()
    
    // Add notification services
    app.Add("discord://webhook_id/webhook_token")
    app.Add("slack://token/token/token/#general")
    
    // Send notification
    responses := app.Notify("Hello", "This is a test message", apprise.NotifyTypeInfo)
    
    // Check responses
    for _, response := range responses {
        if response.Error != nil {
            println("Error:", response.Error.Error())
        } else {
            println("Success:", response.Service)
        }
    }
}
```

## 🎯 Key Features

- **60+ Notification Services** - Discord, Slack, Email, SMS, Push notifications, and more
- **High Performance** - Native Go concurrency and optimized delivery
- **Configuration Templating** - Environment-based configuration with templates  
- **Rich Mobile Push** - Advanced push notifications with actions and rich content
- **Migration Tools** - Easy migration from Python Apprise
- **Enterprise Ready** - Docker containers, Kubernetes manifests, and monitoring

## 📖 Documentation Tools

The documentation is generated using custom tools:

- `cmd/apprise-docs` - Generate service documentation
- `cmd/apprise-migrate` - Migration assistance and validation
- `scripts/generate-docs.sh` - Complete documentation generation

To regenerate documentation:
```bash
./scripts/generate-docs.sh
```

## 🤝 Contributing

See the main repository for contribution guidelines and development information.

## 📄 License

This project is licensed under the MIT License.
