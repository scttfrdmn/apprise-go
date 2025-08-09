# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project follows upstream version tracking with Go-specific port revisions.

**Versioning Strategy**: `{upstream-version}-{port-revision}` (e.g., `1.9.4-1`)

## [1.9.4-1] - 2025-01-09

### Added
- **Version Tracking System**: Implemented upstream version synchronization
  - Version constants and functions in `apprise/version.go`  
  - Dynamic user agent strings across all HTTP services
  - CLI version command with detailed version information
  - Automated upstream version checking script (`scripts/check-upstream.sh`)
- **Desktop Notification Services**: Complete cross-platform desktop notification support
  - Native macOS notifications via `terminal-notifier`
  - Windows system tray notifications via PowerShell
  - Linux notifications via `notify-send`, `zenity`, and `kdialog`
  - Linux DBus notifications with Qt/GLib interface support
  - Platform-specific parameter support (sound, duration, images)
- **Gotify Service**: Self-hosted push notification server support
  - HTTP and HTTPS support with custom priorities
  - Color-coded notifications based on message type
  - Rich metadata support via notification extras
- **Comprehensive Service Implementation**:
  - Slack notification service with webhook and bot API support
  - Telegram Bot API integration with multi-chat support
  - Email (SMTP) service with TLS/STARTTLS support
  - Generic Webhook/JSON service with custom templates
  - Pushover mobile push notification service
  - Microsoft Teams enterprise messaging with MessageCard format
  - Pushbullet cross-platform push notifications
  - Twilio SMS service with rate limiting and phone normalization
- **Advanced Features**:
  - Comprehensive attachment support framework (file, HTTP, memory)
  - Performance benchmarks and optimization analysis
  - Enhanced service registry with proper categorization
  - Rich formatting support (HTML, Markdown, plain text)
  - Notification type-based emoji and color coding
  - Full test coverage matching original Apprise patterns

### Changed
- **User Agent Standardization**: All HTTP services now use dynamic user agents
  - Format: `Apprise-Go/{version} (Go {go-version}; {os}/{arch}) based-on-Apprise/{upstream-version}`
  - Replaces hardcoded "Go-Apprise/1.0" strings across all services
- **Project Metadata**: Updated to reflect upstream tracking strategy
  - README badges and upstream project references
  - Clear comparison table between Python and Go versions
  - Enhanced acknowledgments and contributing guidelines

### Technical Details
- **Upstream Reference**: Based on [Apprise v1.9.4](https://github.com/caronc/apprise/releases/tag/v1.9.4)
- **Port Revision**: `1` (initial implementation)
- **Go Version**: Requires Go 1.21+
- **Platform Support**: macOS, Windows, Linux desktop environments

## [0.1.0] - 2025-08-09 (Legacy)

### Added
- Initial release
- Core notification interface
- Discord service implementation
- Configuration system
- CLI implementation
- Usage examples

[unreleased]: https://github.com/scttfrdmn/apprise-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/apprise-go/releases/tag/v0.1.0