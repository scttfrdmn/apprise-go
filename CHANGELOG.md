# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Slack notification service with webhook and bot API support
- Telegram Bot API integration with multi-chat support
- Email (SMTP) service with TLS/STARTTLS support
- Generic Webhook/JSON service with custom templates
- Pushover mobile push notification service
- Microsoft Teams enterprise messaging with MessageCard format
- Pushbullet cross-platform push notifications
- Twilio SMS service with rate limiting and phone normalization
- Enhanced service registry with proper categorization
- Comprehensive URL format support for all services
- Rich formatting support (HTML, Markdown, plain text)
- Notification type-based emoji and color coding

## [0.1.0] - 2025-08-09

### Added
- Initial release
- Core notification interface
- Discord service implementation
- Configuration system
- CLI implementation
- Usage examples

[unreleased]: https://github.com/scttfrdmn/apprise-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/apprise-go/releases/tag/v0.1.0