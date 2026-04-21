<div align="center">
  <img src="docs/apprise-go.png" alt="Apprise Go Logo" width="200"/>
  
  # Apprise Go
</div>

[![Go Version](https://img.shields.io/github/go-mod/go-version/scttfrdmn/apprise-go)](https://golang.org/)
[![Go Reference](https://pkg.go.dev/badge/github.com/scttfrdmn/apprise-go.svg)](https://pkg.go.dev/github.com/scttfrdmn/apprise-go)
[![License](https://img.shields.io/github/license/scttfrdmn/apprise-go)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/scttfrdmn/apprise-go)](https://github.com/scttfrdmn/apprise-go/releases)

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/apprise-go)](https://goreportcard.com/report/github.com/scttfrdmn/apprise-go)
[![codecov](https://codecov.io/gh/scttfrdmn/apprise-go/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/apprise-go)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen)](https://github.com/scttfrdmn/apprise-go/security)
[![Build Status](https://img.shields.io/github/actions/workflow/status/scttfrdmn/apprise-go/ci.yml?branch=main)](https://github.com/scttfrdmn/apprise-go/actions)
[![GitHub Issues](https://img.shields.io/github/issues/scttfrdmn/apprise-go)](https://github.com/scttfrdmn/apprise-go/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/scttfrdmn/apprise-go)](https://github.com/scttfrdmn/apprise-go/pulls)

A Go port of the [Apprise notification library](https://github.com/caronc/apprise) by [Chris Caron](https://github.com/caronc). Apprise allows you to send a notification to almost all of the most popular notification services available to us today such as: Telegram, Discord, Slack, Amazon SNS, Gotify, etc.

> **Upstream Project**: This is a Go implementation inspired by the original [Apprise v1.9.5](https://github.com/caronc/apprise/releases/tag/v1.9.5) (⭐ 14,186) Python library. While maintaining API compatibility and feature parity, this Go version offers improved performance, static compilation, and native cross-platform support.
>
> **Version Strategy**: This project tracks the upstream version with a Go-specific suffix (e.g., `1.9.5-2` tracks upstream `1.9.5` with Go port revision `2`).
>
> **Current Status**: **86 services implemented** (76% upstream parity) - See [USAGE.md](USAGE.md) for complete service list

## Features

- **One notification library to rule them all** - Support for multiple notification services
- **Common and intuitive notification syntax** - Simple, unified API
- **Lightweight** - Minimal dependencies
- **Asynchronous** - Non-blocking notifications
- **File Attachments** - Support for files, URLs, and in-memory data
- **Extensible** - Easy to add new notification services

## Installation

### Go Library
```bash
go get github.com/scttfrdmn/apprise-go
```

### Pre-built Binaries
Download from [Releases](https://github.com/scttfrdmn/apprise-go/releases) for your platform.

### Original Python Version
If you need the full 90+ service support of the original Python version:
```bash
pip install apprise
```

**Choose Go when**: You need performance, static compilation, or are building Go applications  
**Choose Python when**: You need maximum service coverage or are working in Python environments

## Performance

Apprise-Go is designed for high-performance notification delivery:

- **Single notification**: ~880 ns/op with minimal memory allocation
- **Concurrent notifications**: Excellent parallel performance with goroutine safety
- **Service registry**: Very fast service creation at ~25 ns/op
- **Attachment handling**: Efficient with constant-time performance
- **Memory footprint**: Low allocation patterns, scales linearly

See [BENCHMARKS.md](BENCHMARKS.md) for detailed performance analysis and benchmarking tools.

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
    app.Notify("Hello World!", "This is a test notification", apprise.NotifyTypeInfo)

    // Send with timezone support (new in v1.9.5)
    app.Notify("Server Alert", "CPU usage high", apprise.NotifyTypeWarning,
        apprise.WithTimezone("America/New_York"))
}
```

## Supported Services

### ✅ 86 Services Fully Implemented (76% Upstream Parity)

**DevOps & Observability:**
- **Prometheus AlertManager** - Kubernetes monitoring webhook receiver (API v4)
- **Grafana** - Monitoring alert webhooks
- **Sentry** - Error tracking and application monitoring
- **Elasticsearch/OpenSearch** - Log aggregation and document indexing
- **Datadog** - Cloud monitoring and analytics
- **New Relic** - Application performance monitoring

**Team Collaboration:**
- **Discord** - Webhook notifications with rich embeds
- **Slack** - Webhook and bot API support
- **Microsoft Teams** - Enterprise messaging with adaptive cards (includes Power Automate / Workflows support)
- **Mattermost** - Open-source team collaboration with API v4
- **Rocket.Chat** - Self-hosted team collaboration
- **Matrix** - Decentralized messaging with Client-Server API v3
- **Lark/Feishu** - ByteDance enterprise platform (500M+ users)
- **DingTalk** - Alibaba enterprise platform (500M+ users)

**Incident Management:**
- **PagerDuty** - Events API v2 (US/EU regions)
- **Opsgenie** - Atlassian incident management (US/EU regions)

**Messaging & Communication:**
- **Telegram** - Bot API with multiple chat support
- **Email (SMTP)** - Full SMTP support with TLS/STARTTLS
- **SendGrid** - Cloud email delivery
- **Mailgun** - Email API service
- **Twilio SMS** - SMS/MMS messaging
- **Twilio Voice** - Voice call notifications with TTS

**Push Notifications:**
- **Pushover** - Mobile push with priority levels
- **Pushbullet** - Cross-platform push
- **Pushsafer** - GDPR-compliant European push (176 icons, 60 sounds)
- **OneSignal** - Mobile push for iOS/Android
- **APNS** - Apple Push Notification Service
- **FCM** - Firebase Cloud Messaging

**Desktop Notifications:**
- **Desktop Notifications** - Cross-platform (macOS, Windows, Linux)
- **Gotify** - Self-hosted push
- **Ntfy** - Simple HTTP push
- **Bark** - iOS push notifications

**IoT & Messaging:**
- **MQTT/MQTTS** - IoT messaging protocol with QoS and TLS

**Webhooks & Automation:**
- **Webhook/JSON** - Generic HTTP webhooks
- **IFTTT** - Webhook automation
- **Zapier** - Workflow automation
- **Home Assistant** - Smart home integration
- **Node-RED** - Flow-based automation

**Team Chat & Collaboration:**
- **Zulip** - Open-source team messaging (streams + topics)
- **Cisco Webex** - Enterprise team messaging with markdown support
- **Revolt** - Open-source Discord alternative

**Self-Hosted & Media:**
- **Synology Chat** - Synology NAS integrated chat
- **Kodi** - Media center notifications via JSON-RPC

**Push & Alerting:**
- **Join** - Android push notifications with multi-device support
- **SIGNL4** - Mobile on-call alerting for operations teams
- **SimplePush** - Lightweight self-hostable push notifications

**Social Media:**
- **Reddit** - Subreddit posting
- **Mastodon** - Fediverse social network
- **Facebook** - Page posting
- **Instagram** - Photo sharing
- **Twitter** - Tweet posting
- **YouTube** - Video management
- **TikTok** - Video posting
- **LinkedIn** - Professional network

**Cloud Services:**
- **AWS SNS** - Amazon Simple Notification Service
- **AWS SES** - Amazon Simple Email Service
- **GCP Pub/Sub** - Google Cloud messaging
- **Azure Service Bus** - Microsoft cloud messaging

**Development Platforms:**
- **GitHub** - Repository notifications
- **GitLab** - DevOps platform notifications
- **Jira** - Issue tracking and project management

**SMS Providers:**
- **BulkSMS** - International SMS
- **ClickSend** - Multi-channel messaging
- **MessageBird** - SMS/Voice API
- **Nexmo/Vonage** - Communications API
- **Plivo** - Voice and SMS API
- **TextMagic** - SMS marketing

**And 25+ more services** - See [USAGE.md](USAGE.md) for the complete list with examples

### 🚧 Coming Soon (Tier 3 Priority)
- Signal Messenger
- WhatsApp Business API
- AWS IoT Core / GCP IoT Core
- Amazon Polly (text-to-speech)
- Additional SMS providers
- And more...

## License

This project is licensed under the BSD-2-Clause License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is a Go port of the original [Apprise](https://github.com/caronc/apprise) library by [Chris Caron](https://github.com/caronc).

### Upstream Project

- **Original Apprise**: https://github.com/caronc/apprise
- **Version Reference**: v1.9.5 (Latest as of October 2025)
- **Language**: Python
- **Stars**: ⭐ 14,186+
- **License**: BSD-2-Clause

### Version Tracking Strategy

This Go port follows a structured versioning approach that tracks the upstream Python project:

- **Format**: `{upstream-version}-{port-revision}`
- **Example**: `1.9.4-1` means Go port revision `1` based on upstream Apprise `v1.9.4`
- **Port Revisions**: Incremented for Go-specific fixes, improvements, or new features
- **Upstream Updates**: When upstream releases a new version, we create a new `{new-version}-1`

**Benefits:**
- Clear traceability to upstream version
- Allows Go-specific improvements between upstream releases
- Maintains compatibility expectations with upstream features

**Maintenance:**
```bash
# Check for upstream updates
./scripts/check-upstream.sh

# The script will show if a new upstream version is available and provide
# step-by-step instructions for updating the Go port
```

### Differences from Original

| Feature | Original Python | This Go Port |
|---------|----------------|--------------|
| **Language** | Python 3.6+ | Go 1.25+ |
| **Current Version** | v1.9.5 | v1.9.5-2 |
| **Deployment** | pip install + dependencies | Single static binary |
| **Performance** | ~1ms per notification | ~0.88ms per notification |
| **Memory Usage** | ~50MB runtime | ~10MB runtime |
| **Concurrency** | AsyncIO (single-threaded) | Native goroutines (multi-core) |
| **Attachments** | Basic file support | Advanced multi-source framework |
| **CLI Tool** | `apprise` command | `apprise-cli` binary |
| **Configuration** | YAML/Text files | YAML/Text files ✅ |
| **Services** | 113+ services | 86 services (76% parity) |
| **Type Safety** | Runtime validation | Compile-time validation |

**Go Port Advantages:**
- **Performance**: ~14% faster with 80% less memory usage
- **Static Compilation**: Single binary deployment with no external dependencies  
- **Cross-Platform**: Native compilation for multiple architectures (ARM64, AMD64, etc.)
- **Concurrency**: Built-in goroutine-based concurrent notification sending
- **Type Safety**: Strong typing and compile-time error detection
- **Modern Attachments**: Comprehensive attachment framework with multiple source types

### Contributing Back

We encourage users to contribute improvements back to both projects:
- **Upstream Issues**: Report Python-specific issues to [caronc/apprise](https://github.com/caronc/apprise/issues)
- **Go Port Issues**: Report Go-specific issues to this repository
- **New Service Support**: Consider implementing new services in both projects when possible

Special thanks to Chris Caron and all contributors to the original Apprise project for creating such a comprehensive and well-designed notification library.