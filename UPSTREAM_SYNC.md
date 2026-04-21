# Upstream Sync Strategy

> **Status**: Tracking upstream Apprise v1.9.5

This document outlines our strategy for maintaining parity with the upstream Python Apprise project while developing Go-specific enhancements.

## Current Status

- **Upstream Version**: v1.9.5 (September 30, 2024)
- **Our Version**: v1.9.5-3
- **Port Revision**: 3
- **Services**: 86 implemented / 113+ in upstream (~76% parity)

## Version Tracking Strategy

### Version Format

We use the format: `{upstream-version}-{port-revision}`

**Examples:**
- `1.9.5-1` = Port revision 1 based on upstream v1.9.5
- `1.9.5-2` = Port revision 2 (improvements/additions while still tracking v1.9.5)
- `1.10.0-1` = Port revision 1 for next upstream version

### When to Increment

**Port Revision** (`-X`):
- Go-specific bug fixes
- New service implementations
- Performance improvements
- Documentation updates
- Feature additions from current upstream version
- No breaking changes

**Upstream Version** (e.g., `1.9.5` → `1.10.0`):
- When syncing to a new upstream release
- Reset port revision to `-1`
- Update upstream tracking references

## Release Plan: Catching Up to v1.9.5

We're implementing upstream parity across multiple port revisions:

### v1.9.5-1 (Current) - Upstream Sync Features ✅
**Focus**: Core v1.9.5 features
**Target**: November 2025

- [x] Version tracking update
- [ ] Global timezone support ([#1](https://github.com/scttfrdmn/apprise-go/issues/1))
- [ ] Discord flags enhancement ([#2](https://github.com/scttfrdmn/apprise-go/issues/2))
- [ ] Twilio phone calls ([#3](https://github.com/scttfrdmn/apprise-go/issues/3))
- [ ] Power Automate support ([#4](https://github.com/scttfrdmn/apprise-go/issues/4))
- [ ] Bark service ([#5](https://github.com/scttfrdmn/apprise-go/issues/5))
- [ ] Slack timestamp parameter ([#11](https://github.com/scttfrdmn/apprise-go/issues/11))

**Status**: 6 features (1 done, 5 pending)

### v1.9.5-2 - Documentation & Quality 📚
**Focus**: Documentation and testing improvements
**Target**: December 2025 / Q1 2026

- [ ] Service documentation audit ([#6](https://github.com/scttfrdmn/apprise-go/issues/6))
- [ ] Test coverage improvements ([#7](https://github.com/scttfrdmn/apprise-go/issues/7))
- [ ] Performance benchmarking ([#8](https://github.com/scttfrdmn/apprise-go/issues/8))
- [ ] Upstream sync automation ([#9](https://github.com/scttfrdmn/apprise-go/issues/9))

**Status**: 4 quality initiatives

### v1.9.5-3 - v1.9.4 Services (High Priority) 🚀
**Focus**: Critical v1.9.4 services
**Target**: Q1 2026

New services from v1.9.4:
- [ ] Lark - Enterprise communication ([#13](https://github.com/scttfrdmn/apprise-go/issues/13))
- [ ] SMPP - SMS gateway protocol ([#14](https://github.com/scttfrdmn/apprise-go/issues/14))
- [ ] Vapid/WebPush - Browser push ([#15](https://github.com/scttfrdmn/apprise-go/issues/15))

**Status**: 3 high-priority services

### v1.9.5-4 - v1.9.4 Services (Medium Priority) 📦
**Focus**: Additional v1.9.4 services
**Target**: Q1-Q2 2026

- [ ] Spike.sh - Alert management
- [ ] SIGNL4 - Mobile alerting
- [ ] SendPulse - Marketing automation

**Status**: 3 medium-priority services

### v1.9.5-5 - v1.9.4 Services (Complete) ✨
**Focus**: Remaining v1.9.4 services
**Target**: Q2 2026

- [ ] Spug Push
- [ ] QQ Push
- [ ] Clickatell

**Status**: 3 lower-priority services

## Upstream Feature Tracking

### v1.9.5 Features (September 2024)

| Feature | Status | Issue | Priority | Target |
|---------|--------|-------|----------|--------|
| Global timezone (tz=) | 🔄 Planned | #1 | Medium | v1.9.5-1 |
| Discord flags | 🔄 Planned | #2 | Low | v1.9.5-1 |
| Twilio phone calls | 🔄 Planned | #3 | Medium | v1.9.5-1 |
| Power Automate URLs | 🔄 Planned | #4 | Low | v1.9.5-1 |
| Bark icon field | 🔄 Planned | #5 | Low | v1.9.5-1 |
| Slack timestamp param | 🔄 Planned | #11 | Low | v1.9.5-1 |

### v1.9.4 Services (August 2024)

| Service | Status | Issue | Priority | Target |
|---------|--------|-------|----------|--------|
| Lark | 🔄 Planned | #13 | High | v1.9.5-3 |
| SMPP | 🔄 Planned | #14 | High | v1.9.5-3 |
| Vapid/WebPush | 🔄 Planned | #15 | High | v1.9.5-3 |
| Spike.sh | 📋 Tracked | #12 | Medium | v1.9.5-4 |
| SIGNL4 | 📋 Tracked | #12 | Medium | v1.9.5-4 |
| SendPulse | 📋 Tracked | #12 | Medium | v1.9.5-4 |
| Spug Push | 📋 Tracked | #12 | Low | v1.9.5-5 |
| QQ Push | 📋 Tracked | #12 | Low | v1.9.5-5 |
| Clickatell | 📋 Tracked | #12 | Low | v1.9.5-5 |

## Service Coverage Analysis

### Implemented (64 services)

<details>
<summary>Click to expand service list</summary>

**Messaging & Chat (14)**
Discord, Slack, Telegram, Microsoft Teams, Matrix, Mattermost, RocketChat, Mastodon, Signal, WhatsApp, Facebook, Instagram, LinkedIn, Reddit

**Email (5)**
SMTP, SendGrid, Mailgun, AWS SES, Office365

**Push Notifications (6)**
Pushover, Pushbullet, APNS, FCM, Ntfy, Gotify

**SMS (9)**
Twilio (SMS & Voice), AWS SNS SMS, Nexmo, Plivo, BulkSMS, ClickSend, MessageBird, TextMagic

**Incident Management (2)**
PagerDuty, Opsgenie

**Cloud (6)**
AWS (SNS, SES, IoT), Azure Service Bus, GCP (Pub/Sub, IoT)

**Monitoring (6)**
Datadog, NewRelic, Jira, GitHub, GitLab, Grafana

**Desktop (3)**
Linux DBus, macOS, Windows

**Automation (4)**
HomeAssistant, IFTTT, NodeRED, Zapier

**Social Media (4)**
TikTok, Twitter, YouTube, Mastodon

**Other (5)**
Generic Webhook/JSON, Mock (testing), Polly (text-to-speech)

</details>

### Missing from Upstream (45+ services)

<details>
<summary>High-priority missing services</summary>

**Enterprise & Business**
- Webex Teams
- Google Chat
- Zulip
- Guilded
- Revolt

**Regional/International**
- DingTalk (China)
- WeCom Bot (China)
- Line (Japan/Asia)
- Feishu/Lark (China/Asia) - planned v1.9.5-3
- QQ Push (China) - planned v1.9.5-5

**Monitoring & Alerting**
- Splunk/VictorOps
- Grafana (separate from current)
- AlertManager (Prometheus)

**Mobile Push**
- OneSignal
- Pushy
- PushSafer
- Kumulos

**SMS/Communication**
- SMPP (gateway protocol) - planned v1.9.5-3
- Sinch
- Seven
- MSG91
- D7 Networks

**Specialized**
- MQTT
- Syslog/RSyslog
- KODI/XBMC
- Synology
- LaMetric Time

**Email Services**
- Resend
- Postmark
- SparkPost
- SMTP2Go

**Web/Browser**
- Vapid/WebPush - planned v1.9.5-3
- SimplePush
- Popcorn Notify

</details>

## Checking for Upstream Updates

### Automated Check

Run the upstream check script:

```bash
./scripts/check-upstream.sh
```

This will:
- Check current version against upstream
- Show latest upstream release
- Provide update instructions if needed

### Manual Check

1. Visit https://github.com/caronc/apprise/releases
2. Compare with our `VERSION` file
3. Review release notes for new features/services
4. Create issues for new features to port

## Porting Process

### When a New Upstream Version is Released

1. **Assess the Release**
   ```bash
   # Check for updates
   ./scripts/check-upstream.sh

   # Review release notes
   open https://github.com/caronc/apprise/releases/tag/vX.Y.Z
   ```

2. **Create Tracking Issues**
   - Use issue templates for new services
   - Label with `upstream-sync`
   - Assign to appropriate milestone
   - Document upstream PR/commit references

3. **Update Version Tracking**
   ```bash
   # Update VERSION file
   echo "X.Y.Z-1" > VERSION

   # Update version constants in apprise/version.go
   # Update go.mod comments
   # Update README.md references
   ```

4. **Prioritize Features**
   - High: Core features, popular services
   - Medium: Niche services, enhancements
   - Low: Rare services, cosmetic changes

5. **Implement Across Multiple Releases**
   - Use port revisions (-1, -2, -3, etc.)
   - Group related features together
   - Balance new features with quality improvements

### Porting a New Service

1. **Research**
   - Read upstream Python implementation
   - Study service API documentation
   - Test API with curl/Postman

2. **Design**
   - Design URL scheme matching upstream
   - Plan configuration options
   - Consider Go-specific improvements

3. **Implement**
   - Follow existing service patterns
   - Use appropriate HTTP client pool
   - Implement proper error handling

4. **Test**
   - Write comprehensive unit tests
   - Test with real service (if possible)
   - Add to CI test suite

5. **Document**
   - Add GoDoc comments
   - Update USAGE.md with examples
   - Add to service registry

## Go-Specific Enhancements

We can add Go-specific features that don't exist in upstream:

### Already Implemented
- ✅ HTTP connection pooling with optimized configs
- ✅ Advanced attachment framework
- ✅ REST API server with authentication
- ✅ Prometheus metrics
- ✅ Type-safe configuration

### Planned Enhancements
- GraphQL API (v1.10.0)
- OpenTelemetry tracing (v1.10.0)
- Enhanced observability (v1.9.5-2)
- Performance optimizations

## Contributing Upstream

When we implement improvements that could benefit the Python version:

1. Open an issue on upstream repo describing the enhancement
2. If accepted, implement in Python (or collaborate with upstream maintainer)
3. Reference our Go implementation as proof of concept
4. Maintain compatibility with both versions

## Divergence Policy

We may diverge from upstream when:

### Acceptable Divergence
- ✅ Performance optimizations specific to Go
- ✅ Type safety improvements
- ✅ Better error handling
- ✅ Go ecosystem integration
- ✅ Additional features that don't break compatibility

### Avoid Divergence
- ❌ Breaking URL format changes
- ❌ Removing features that upstream has
- ❌ Incompatible configuration formats
- ❌ Different service behavior without good reason

## Communication

### Upstream References
- Always link to upstream issues/PRs in our issues
- Credit upstream contributors
- Keep CHANGELOG in sync with upstream changes
- Maintain "Upstream Project" section in README

### Community
- Encourage users to contribute to both projects
- Direct Python-specific issues to upstream
- Share learnings and improvements

## Success Metrics

### Coverage Targets
- **v1.9.5 Feature Parity**: 100% (by v1.9.5-1)
- **v1.9.4 Service Parity**: 100% (by v1.9.5-5)
- **Overall Service Parity**: 70%+ (ongoing)

### Quality Targets
- A-grade code quality
- >80% test coverage for all services
- Performance better than or equal to upstream
- Comprehensive documentation

---

## Quick Reference

### Useful Links
- **Upstream Releases**: https://github.com/caronc/apprise/releases
- **Upstream Wiki**: https://github.com/caronc/apprise/wiki
- **Our Milestones**: https://github.com/scttfrdmn/apprise-go/milestones
- **Upstream Sync Issues**: https://github.com/scttfrdmn/apprise-go/labels/upstream-sync

### Key Files
- `VERSION` - Current version
- `apprise/version.go` - Version constants
- `CHANGELOG.md` - Version history
- `scripts/check-upstream.sh` - Version checker

---

*Last Updated: 2025-10-28*
*Tracking: Apprise v1.9.5*
