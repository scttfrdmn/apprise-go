# 🚀 Apprise-Go Roadmap

> **Note**: As of v1.9.5-1, this project has transitioned to using GitHub Projects and Issues for roadmap tracking. This document provides a high-level overview. For detailed planning, see our [GitHub Project Board](https://github.com/scttfrdmn/apprise-go/projects) and [Milestones](https://github.com/scttfrdmn/apprise-go/milestones).

## 📊 Current Status (v1.9.5-1)

- **Implemented Services**: **64+** notification services
- **Upstream Tracking**: Synced with Apprise v1.9.5
- **Go Version**: 1.25+
- **Code Quality**: A-grade (Go Report Card)
- **Test Coverage**: Comprehensive service coverage

### ✅ Implemented Services (64+)

<details>
<summary>Click to expand full service list</summary>

**Messaging & Chat**
- Discord, Slack, Telegram, Microsoft Teams, Matrix, Mattermost
- RocketChat, Mastodon, Signal, WhatsApp
- Facebook, Instagram, LinkedIn, Reddit, TikTok, Twitter, YouTube

**Email Services**
- SMTP (Generic), SendGrid, Mailgun, AWS SES, Office365

**Push Notifications**
- Pushover, Pushbullet, APNS, FCM (Firebase), Ntfy, Gotify

**SMS Services**
- Twilio (SMS & Voice), AWS SNS SMS, Nexmo, Plivo
- BulkSMS, ClickSend, MessageBird, TextMagic

**Incident Management**
- PagerDuty, Opsgenie

**Cloud Platforms**
- AWS (SNS, SES, IoT), Azure Service Bus, GCP (Pub/Sub, IoT)

**Monitoring & DevOps**
- Datadog, NewRelic, Jira, GitHub, GitLab, Grafana

**Desktop Notifications**
- Linux (DBus), macOS, Windows
- Advanced/Interactive/Persistent variants

**Automation & Integration**
- HomeAssistant, IFTTT, NodeRED, Zapier
- Generic Webhook/JSON

**And more...**

See [USAGE.md](USAGE.md) for detailed service documentation.

</details>

---

## 🎯 Active Milestones

Track our progress on [GitHub Milestones](https://github.com/scttfrdmn/apprise-go/milestones):

### v1.9.5-1 (Current Release)
**Focus**: Upstream sync with Apprise v1.9.5

Key features from upstream to implement:
- [x] Version tracking update
- [ ] Global timezone support ([#1](https://github.com/scttfrdmn/apprise-go/issues/1))
- [ ] Discord flags enhancement ([#2](https://github.com/scttfrdmn/apprise-go/issues/2))
- [ ] Twilio phone calls ([#3](https://github.com/scttfrdmn/apprise-go/issues/3))
- [ ] Power Automate support ([#4](https://github.com/scttfrdmn/apprise-go/issues/4))
- [ ] Bark service implementation ([#5](https://github.com/scttfrdmn/apprise-go/issues/5))

**Status**: In Development
**Target**: November 2025

### v1.9.5-2 (Next Iteration)
**Focus**: Documentation, testing, and quality improvements

- [ ] Service documentation audit ([#6](https://github.com/scttfrdmn/apprise-go/issues/6))
- [ ] Test coverage improvements ([#7](https://github.com/scttfrdmn/apprise-go/issues/7))
- [ ] Performance benchmarking ([#8](https://github.com/scttfrdmn/apprise-go/issues/8))
- [ ] Upstream sync automation ([#9](https://github.com/scttfrdmn/apprise-go/issues/9))

**Status**: Planned
**Target**: Q1 2026

### v1.10.0 (Future)
**Focus**: Next upstream sync + API enhancements

- Track next upstream Apprise release
- REST API v2 features
- GraphQL support
- Enhanced observability

**Status**: Planning
**Target**: TBD (based on upstream releases)

---

## 📋 Project Tracking

We use GitHub for all project management:

### 🗂️ GitHub Projects
**[Apprise-Go Development](https://github.com/scttfrdmn/apprise-go/projects)**: Main kanban board for tracking all work

### 🏷️ Issue Labels
- **Type**: `bug`, `enhancement`, `documentation`, `technical-debt`, `service-request`
- **Priority**: `priority: critical/high/medium/low`
- **Area**: `area: core/services/cli/api/config/attachments/http/tests/docs/build`
- **Service**: `service: discord/slack/telegram/email/teams/webhook/cloud/other`
- **Status**: `triage`, `needs-info`, `blocked`, `ready`, `in-progress`, `in-review`, `awaiting-merge`
- **Special**: `good first issue`, `help wanted`, `upstream-sync`, `breaking-change`

### 📝 Issue Templates
Use our structured templates for:
- 🐛 [Bug Reports](.github/ISSUE_TEMPLATE/bug_report.yml)
- ✨ [Feature Requests](.github/ISSUE_TEMPLATE/feature_request.yml)
- 🔌 [New Service Requests](.github/ISSUE_TEMPLATE/service_request.yml)
- 📚 [Documentation](.github/ISSUE_TEMPLATE/documentation.yml)
- 🔧 [Technical Debt](.github/ISSUE_TEMPLATE/technical_debt.yml)

---

## 🎯 Strategic Goals

### Quality & Reliability
- Maintain A-grade code quality (Go Report Card)
- >80% test coverage for critical services
- Comprehensive error handling
- Regular security audits

### Performance
- Faster than Python Apprise for all services
- Efficient HTTP connection pooling
- Minimal memory footprint
- Optimized for concurrent notifications

### Developer Experience
- Clear, comprehensive documentation
- Easy-to-use CLI and REST API
- Excellent GoDoc coverage
- Active community engagement

### Upstream Compatibility
- Track upstream Apprise versions
- Maintain API compatibility where possible
- Document Go-specific enhancements
- Contribute improvements back to upstream community

---

## 🤝 Contributing

Want to help? Check out:
- [Good First Issues](https://github.com/scttfrdmn/apprise-go/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
- [Help Wanted](https://github.com/scttfrdmn/apprise-go/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
- [Service Requests](https://github.com/scttfrdmn/apprise-go/issues?q=is%3Aissue+is%3Aopen+label%3Aservice-request)

### Ways to Contribute
- **Implement new services**: Use our [service request template](.github/ISSUE_TEMPLATE/service_request.yml)
- **Improve documentation**: Help document our 64+ services
- **Write tests**: Increase coverage for existing services
- **Fix bugs**: Tackle issues labeled `bug`
- **Performance**: Optimize and benchmark
- **Examples**: Add real-world usage examples

---

## 📊 Success Metrics

### Code Quality
- ✅ Go Report Card A-grade
- ✅ No critical security vulnerabilities
- ✅ Comprehensive test coverage
- ✅ Regular dependency updates

### Community
- Active issue and PR engagement
- Growing adoption in Go ecosystem
- Community service contributions
- Positive upstream collaboration

### Performance
- Documented performance advantages over Python
- Efficient resource usage
- Fast notification delivery
- Scalable for enterprise use

---

## 🔗 Links

- **[GitHub Project Board](https://github.com/scttfrdmn/apprise-go/projects)**: View current work
- **[Milestones](https://github.com/scttfrdmn/apprise-go/milestones)**: Track release progress
- **[Issues](https://github.com/scttfrdmn/apprise-go/issues)**: View all issues
- **[Pull Requests](https://github.com/scttfrdmn/apprise-go/pulls)**: See ongoing work
- **[Releases](https://github.com/scttfrdmn/apprise-go/releases)**: Version history
- **[Upstream Apprise](https://github.com/caronc/apprise)**: Python original

---

*Last Updated: 2025-10-28*
*This roadmap is maintained through GitHub Projects and Issues. Please check there for the most current status.*
