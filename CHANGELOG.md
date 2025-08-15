# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project follows upstream version tracking with Go-specific port revisions.

**Versioning Strategy**: `{upstream-version}-{port-revision}` (e.g., `1.9.4-1`)

## [1.9.4-3] - 2025-01-15

### Added - Phase 2 Cloud & Performance Expansion 🚀
- **Multi-Cloud Platform Support**:
  - **AWS SNS**: Amazon Simple Notification Service via webhook proxy with JSON/text formats
    - Regional endpoint support and custom message attributes
    - Structured JSON messaging with severity mapping and environment context
  - **AWS SES**: Amazon Simple Email Service for enterprise email delivery
    - HTML/text email templates with multiple recipient support (TO/CC/BCC)
    - Professional email formatting with attachment support integration
  - **Azure Service Bus**: Enterprise messaging with queue/topic patterns
    - SAS authentication and managed identity support
    - Connection string and individual parameter configuration
  - **Google Cloud Pub/Sub**: Real-time messaging with ordered delivery
    - Service account authentication and attribute filtering
    - Project-based organization with custom message attributes

- **Complete Attachment Support**:
  - **SMTP Email Attachments**: Full MIME multipart functionality
    - Base64 encoding with RFC 2045 compliant line wrapping
    - Inline vs attachment disposition based on content type
    - Support for multiple attachments with proper boundary generation
  - **Microsoft Teams Attachments**: Adaptive Cards with rich content
    - Image attachments with data URLs for inline display
    - Dual-mode operation (MessageCard vs message+attachments)
    - File attachment support with proper MIME type handling

- **HTTP Connection Pooling**: Enterprise-grade performance infrastructure
  - Configurable connection pools with specialized configurations:
    - Default: General purpose (30s timeout, 100 idle connections)
    - Cloud: Optimized for cloud APIs (60s timeout, 200 idle connections)  
    - Webhook: Fast webhooks (15s timeout, 50 idle connections)
  - HTTP/2 support with automatic protocol negotiation
  - Thread-safe implementation with proper resource management
  - Applied to all HTTP-based services for improved scalability

### Enhanced
- **Email Service**: Updated `SupportsAttachments()` to return true with full MIME multipart support
- **Microsoft Teams Service**: Updated `SupportsAttachments()` to return true with Adaptive Card support
- **All HTTP Services**: Migrated to use optimized connection pooling for better performance

### Technical Improvements
- Comprehensive test coverage for all new services and attachment functionality
- Performance benchmarks demonstrating connection pooling benefits
- Proper error handling and timeout configuration across all cloud services
- Enhanced documentation with usage examples and configuration guides

## [1.9.4-2] - 2025-01-15

### Added - Phase 1 Service Expansion Complete 🎉
- **Enterprise Incident Management Services**:
  - **PagerDuty**: Complete Events API v2 implementation with US/EU region support
    - Integration key authentication with automatic region endpoint selection
    - Custom alert metadata: source, component, group, class parameters
    - Severity mapping from notification types to PagerDuty priority levels
    - Alert deduplication and custom details support
  - **Opsgenie**: Atlassian incident management and alerting service
    - Alerts API v2 with US/EU regional endpoint support
    - Multiple responder types: teams and users with automatic email detection
    - Priority levels P1-P5 with automatic mapping from notification types
    - Rich alert metadata: entity, source, tags, alias, notes
    - GenieKey authentication with proper API headers

- **Team Collaboration Services**:
  - **Matrix**: Decentralized messaging with Client-Server API v3
    - Multiple authentication methods: access tokens and username/password
    - Multi-room support with automatic room normalization
    - Room formats: IDs (!room:server), aliases (#room:server), simple names
    - HTML message formatting with security escaping
    - Fragment URL parsing for room aliases (#room:server)
    - Automatic login and session management
  - **Mattermost**: Open-source team collaboration with API v4
    - Personal access token and username/password authentication
    - Multi-channel messaging in single URL
    - Channel name normalization with # and @ prefix handling
    - Bot appearance customization (name, icon URL, emoji)
    - Automatic channel ID resolution via Mattermost API
    - Session management and fragment URL parsing

- **Developer/Self-Hosted Services**:
  - **Ntfy**: Simple HTTP push notifications with priority levels
    - Public ntfy.sh and self-hosted instance support
    - Priority levels (1-5) with automatic mapping from notification types
    - Advanced features: tags, delayed delivery, email forwarding
    - Interactive features: action buttons, click URLs, attachment URLs
    - Emoji tag support with automatic fallbacks
    - Token and username/password authentication

### Enhanced
- **Service Registry**: Added 5 new service registrations with proper URL schemes
- **Documentation**: Comprehensive updates to README.md and USAGE.md
  - Complete service documentation with URL formats and examples
  - Service comparison table updates
  - Attachment support matrix updates
- **Test Coverage**: Added 50+ test functions with 200+ test cases across new services
- **Version System**: Updated to version 1.9.4-2 reflecting port revision increment

### Technical Details
- **Code Additions**: 2,500+ lines of production code across 5 new services
- **Service Count**: Increased from 13 to 18 services (+38% expansion)
- **Enterprise Focus**: Targeted DevOps market with incident management and team collaboration
- **Quality Assurance**: Comprehensive error handling, URL validation, and network resilience
- **Performance**: Maintained existing performance characteristics with concurrent processing

### Breaking Changes
None - All changes are additive and backward compatible.

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