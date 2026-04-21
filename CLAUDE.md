# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run tests for the core library only
go test ./apprise/...

# Run a single test
go test ./apprise/... -run TestDiscordService

# Run tests with verbose output
go test ./apprise/... -v -run TestDiscordService

# Run tests with coverage
go test ./apprise/... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Build CLI tools
go build ./cmd/apprise-cli/
go build ./cmd/apprise-api/

# Run the API server
go run ./cmd/apprise-api/

# Lint (uses go vet)
go vet ./...

# Security scan
gosec ./...

# Check for upstream version updates
./scripts/check-upstream.sh
```

## Architecture

This is a Go port of [Apprise v1.9.5](https://github.com/caronc/apprise), a unified notification library. The Go port version format is `{upstream-version}-{port-revision}` (e.g., `1.9.5-2`).

### Packages

- **`apprise/`** — Core library. Contains all service implementations, the `Service` interface, `Apprise` manager, attachment system, HTTP pool, scheduler, and config loading.
- **`api/`** — REST API server (gorilla/mux). Handlers for notifications, scheduling, auth (JWT), config, and a web dashboard.
- **`cmd/apprise-cli/`** — Cobra-based CLI for sending notifications from the terminal.
- **`cmd/apprise-api/`** — Wraps `api/` into a runnable server binary.
- **`cmd/apprise-docs/`** — Generates service documentation.
- **`cmd/apprise-migrate/`** — Config migration tool.

### Service Interface

Every notification service implements:

```go
type Service interface {
    GetServiceID() string
    GetDefaultPort() int
    ParseURL(*url.URL) error
    Send(context.Context, NotificationRequest) error
    TestURL(string) error
    SupportsAttachments() bool
    GetMaxBodyLength() int
}
```

Services are registered in `apprise/services.go` via `CreateService(serviceID)` and `GetSupportedServices()`. Adding a new service requires: (1) a `<service>.go` implementation file, (2) a `<service>_test.go` test file, and (3) registration in both `CreateService` and `GetSupportedServices` in `services.go`.

### HTTP Connection Pooling

Services use shared HTTP clients from `apprise/http_pool.go`. There are three pool profiles — call the appropriate getter in your service's constructor:
- `GetDefaultHTTPClient(name)` — General webhooks (30s timeout)
- `GetWebhookHTTPClient(name)` — Fast webhook services (15s timeout)
- `GetCloudHTTPClient(name)` — Cloud APIs with longer timeouts (60s)

### Notification Flow

`app.Add(url)` → `ParseURL()` configures the service → `app.Notify(title, body, type)` → `NotifyAll()` → concurrent goroutines call `Service.Send(ctx, req)`.

### Attachments

`apprise/attachment.go` provides `AttachmentManager` with support for file paths, HTTP URLs, and in-memory data. Services check `SupportsAttachments()` before processing `req.AttachmentMgr`.

### Scheduler

`apprise/scheduler.go` + 6 related files implement a cron-based SQLite-backed scheduler. The `api/handlers_scheduler.go` exposes it over REST.

### Test Pattern

Tests use `net/http/httptest` to create mock servers. Look at `apprise/discord_test.go` or `apprise/slack_test.go` as canonical examples for new service tests.

## Upstream Parity

Current status: **86/113 services** (~76%). See `UPSTREAM_SYNC.md` for the sync strategy and `ROADMAP.md` for planned services. When implementing a new service, consult the upstream Python implementation at `https://github.com/caronc/apprise/tree/master/Apprise_cli/plugins` for URL schema and behavior reference.

Services not yet implemented include: Google Chat, Guilded, WebPush/Vapid, Line, WeChat, VictorOps/Splunk, Nextcloud Talk, Viber, SparkPost, Brevo, Resend, Bluesky, and additional regional SMS providers (46elks, Sinch, MSG91, Octopush).
