# Apprise Go Usage Guide

This guide covers all the notification services implemented in Apprise Go and their URL formats.

## Quick Start

```go
package main

import "github.com/scttfrdmn/apprise-go/apprise"

func main() {
    app := apprise.New()
    
    // Add services
    app.Add("discord://webhook_id/webhook_token")
    app.Add("slack://TokenA/TokenB/TokenC/general")
    
    // Send notification
    app.Notify("Hello World!", "This is a test notification", apprise.NotifyTypeInfo)
}
```

## Supported Services

### Discord

Discord webhook notifications with rich embeds and custom formatting.

**URL Formats:**
```
discord://webhook_id/webhook_token
discord://avatar@webhook_id/webhook_token
discord://webhook_id/webhook_token?username=MyBot&avatar=https://example.com/avatar.png

# With message flags (v1.9.5+)
discord://webhook_id/webhook_token?flags=4              # Suppress embeds
discord://webhook_id/webhook_token?flags=4096           # Suppress notifications
discord://webhook_id/webhook_token?flags=4100           # Both (4 + 4096)
```

**Query Parameters:**
- `username=string` - Custom bot name
- `avatar=url` - Custom bot avatar URL
- `flags=int` - Discord message flags (bitmask, v1.9.5+)

**Message Flags (v1.9.5+):**
- `4` (SUPPRESS_EMBEDS) - Suppress embeds in the message
- `4096` (SUPPRESS_NOTIFICATIONS) - Do not send push notification
- Combined: `4100` - Both suppress embeds and notifications
- `0` or omit - Normal message (default)

**Features:**
- Rich embeds with titles and colors
- Custom avatar and username
- Notification type-based color coding
- Support for attachments
- Message flags control (suppress embeds, notifications)

### Slack  

Slack notifications via webhooks or bot API with rich formatting.

**URL Formats:**
```
# Webhook mode (3 tokens)
slack://TokenA/TokenB/TokenC
slack://TokenA/TokenB/TokenC/general
slack://TokenA/TokenB/TokenC/@username

# Bot mode (OAuth token)  
slack://bot_token/general
slack://bot_token/@username
slack://TokenA/TokenB/TokenC?username=MyBot&icon_emoji=:ghost:

# With timestamp control (v1.9.5+)
slack://TokenA/TokenB/TokenC/general?timestamp=no
slack://bot_token/general?timestamp=yes
```

**Query Parameters:**
- `username=string` - Custom bot name
- `icon_url=url` - Custom bot avatar URL
- `icon_emoji=:emoji:` - Custom emoji icon (e.g., :ghost:, :rocket:)
- `channel=string` - Override channel (alternative to path)
- `timestamp=yes|no` - Include timestamp in messages (default: yes, v1.9.5+)

**Features:**
- Webhook and bot API support
- Channel and direct message support
- Rich attachments with colors
- Custom bot appearance
- Optional timestamp display control

### Telegram

Telegram Bot API with support for multiple chats and rich formatting.

**URL Formats:**
```
tgram://bot_token/chat_id
telegram://bot_token/chat_id1/chat_id2
tgram://bot_token/@username
tgram://bot_token/chat_id?silent=yes&preview=no&format=html
```

**Query Parameters:**
- `silent=yes/no` - Silent notifications
- `preview=yes/no` - Web page preview  
- `format=html/markdown/markdownv2` - Message formatting
- `thread=123` - Reply to specific thread

**Features:**
- Multiple chat support
- HTML, Markdown, and MarkdownV2 formatting
- Silent notifications
- Thread support
- Emoji indicators by notification type

### Email (SMTP)

Full-featured SMTP email notifications with TLS support.

**URL Formats:**
```
mailto://username:password@smtp.server.com:587/recipient@domain.com
mailtos://username:password@smtp.server.com:465/recipient@domain.com
mailto://user:pass@smtp.gmail.com/to@domain.com?cc=cc@domain.com&bcc=bcc@domain.com
mailto://user:pass@smtp.server.com/to@domain.com?from=sender@domain.com&name=Sender%20Name
```

**Query Parameters:**
- `from=email` - Sender email address
- `name=Name` - Sender display name
- `cc=email` - CC recipients (comma-separated)
- `bcc=email` - BCC recipients (comma-separated)
- `subject=Subject` - Custom subject line
- `skip_verify=yes` - Skip TLS certificate verification
- `no_tls=yes` - Disable STARTTLS

**Features:**
- TLS and STARTTLS support
- HTML and plain text formatting
- CC and BCC support
- Custom sender names
- SMTP authentication

### Webhook/JSON

Generic HTTP webhook notifications with custom templates.

**URL Formats:**
```
webhook://api.example.com/notify
webhooks://api.example.com/notify  (HTTPS)
json://api.example.com/webhook
webhooks://token@api.example.com/notify  (Bearer auth)
webhooks://user:pass@api.example.com/notify  (Basic auth)
webhook://api.example.com/notify?method=PUT&content_type=text/plain
```

**Query Parameters:**
- `method=POST/GET/PUT/PATCH` - HTTP method
- `content_type=application/json` - Content type
- `template={"text":"{{message}}"}` - Custom payload template  
- `header_X-API-Key=secret` - Custom headers

**Features:**
- Multiple content types (JSON, form-encoded, plain text)
- Custom HTTP methods
- Template-based payloads
- Authentication support (Basic, Bearer)
- Custom headers

### Microsoft Teams

Enterprise messaging with rich card formatting and theme colors, including support for Microsoft Power Automate and Workflows.

**URL Formats:**
```
# Modern format (recommended)
msteams://team_name/token_a/token_b/token_c

# Version 3 format
msteams://team_name/token_a/token_b/token_c/token_d

# Legacy format
msteams://token_a/token_b/token_c

# With options
msteams://team_name/token_a/token_b/token_c?image=no

# Power Automate / Workflows format
powerautomate://prod-01.eastus.logic.azure.com/workflows/abc123/triggers/manual/paths/invoke?params
workflows://prod-02.westus.logic.azure.com:443/workflows/def456/triggers/manual/paths/invoke
msflow://prod-03.centralus.logic.azure.com/workflows/ghi789/triggers/manual/paths/invoke
```

**Query Parameters:**
- `image=yes/no` - Include activity images in notifications

**Features:**
- MessageCard format with rich styling
- Color-coded notifications by type
- Activity images for visual context
- Support for all Teams webhook versions
- Markdown text formatting support
- **Power Automate / Workflows integration** (new in v1.9.5-1)
- Microsoft Flow support (legacy)

**Power Automate / Workflows:**

Microsoft Power Automate (formerly Flow) and Workflows use Azure Logic Apps webhooks that accept the same MessageCard format as Teams. Use the `powerautomate://`, `workflows://`, or `msflow://` URL schemes with your Logic Apps webhook URL.

```go
// Power Automate example
app.Add("powerautomate://prod-01.eastus.logic.azure.com/workflows/abc123def456/triggers/manual/paths/invoke?api-version=2016-10-01&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=xyz789")

// Workflows example
app.Add("workflows://prod-02.westus.logic.azure.com:443/workflows/def456/triggers/manual/paths/invoke")

// MS Flow (legacy) example
app.Add("msflow://prod-03.centralus.logic.azure.com/workflows/ghi789/triggers/manual/paths/invoke")

app.Notify("Automation Alert", "Workflow triggered successfully", apprise.NotifyTypeSuccess)
```

**Getting Your Power Automate Webhook URL:**
1. Create a new Flow/Workflow in Power Automate
2. Add a "When a HTTP request is received" trigger
3. Save the Flow to generate the webhook URL
4. Copy the full webhook URL (e.g., `https://prod-01.eastus.logic.azure.com/workflows/.../`)
5. Use it with the `powerautomate://` scheme (remove `https://` and use the scheme instead)

### Mattermost

Open-source team collaboration platform with API v4 support for self-hosted and cloud deployments.

**URL Formats:**
```
# Username/password authentication (HTTP)
mattermost://username:password@mattermost.example.com/general

# Token authentication (HTTPS)
mmosts://token@mattermost.company.com/alerts

# Custom port
mattermost://user:pass@mm.company.com:9000/general

# Multiple channels
mmosts://token@mattermost.example.com/general/alerts/dev-team

# With bot customization
mattermost://token@mm.example.com/general?bot=AlertBot&icon_emoji=:warning:&icon_url=https://example.com/icon.png
```

**Query Parameters:**
- `token=string` - Access token (alternative to URL auth)
- `bot=string` - Custom bot name for message display
- `icon_url=url` - Custom icon URL for bot avatar
- `icon_emoji=:emoji:` - Custom emoji for bot icon

**Features:**
- Mattermost API v4 compliance for broad compatibility
- Multiple authentication methods: username/password and personal access tokens
- Multi-channel messaging in single URL
- Channel name normalization (removes # and @ prefixes)
- Fragment URL parsing for channel references (#channel)
- Bot appearance customization (name, icon URL, emoji)
- Markdown message formatting with emoji support
- Automatic channel ID resolution via API
- Session management for username/password authentication

**Authentication Methods:**
1. **Personal Access Token** (recommended for production)
   ```go
   app.Add("mmosts://sdf2h3jh4k2j3h4k2j3h4k@mattermost.company.com/alerts")
   ```

2. **Username/Password** (for development/testing)
   ```go
   app.Add("mattermost://myuser:mypass@mattermost.example.com/general")
   ```

3. **Token in Query Parameter**
   ```go
   app.Add("mattermost://myuser@mm.example.com/general?token=access_token")
   ```

**Channel Formats:**
- **Simple Names**: `general`, `alerts`, `dev-team`
- **Hash Prefixed**: `#general` (automatically normalized)
- **Direct Messages**: `@username` (automatically normalized)

**Message Formatting:**
- Automatic emoji prefixing based on notification type
- Markdown bold formatting for titles: `**Title**`
- Multi-line support with proper spacing
- Fallback messages for empty notifications

**Example:**
```go
// Send alert to self-hosted Mattermost with custom bot appearance
app.Add("mmosts://token@mattermost.company.com/ops-alerts?bot=MonitoringBot&icon_emoji=:rotating_light:")
```

### PagerDuty

Enterprise incident management with Events API v2 support for both US and EU regions.

**URL Formats:**
```
# Basic integration key
pagerduty://integration_key

# Specify region explicitly  
pagerduty://integration_key@us
pagerduty://integration_key@eu

# With custom source and component
pagerduty://integration_key?source=monitoring&component=api

# Full configuration
pagerduty://integration_key?region=eu&source=server-01&component=database&group=production&class=critical
```

**Query Parameters:**
- `region=us|eu` - PagerDuty region (default: us)
- `source=string` - Alert source identifier (default: apprise-go)
- `component=string` - System component name
- `group=string` - Alert grouping identifier
- `class=string` - Alert classification

**Features:**
- Events API v2 with automatic severity mapping
- US and EU region support
- Custom alert metadata (source, component, group, class)
- Automatic deduplication support
- Title and body included in custom details
- Integration with PagerDuty's incident response workflows

**Example:**
```go
// Send critical database alert to EU region
app.Add("pagerduty://r1234567890abcdef1234567890abcdef@eu?source=db-cluster&component=primary")
```

### Grafana

Grafana Alerting webhook notifications for the leading open-source observability platform.

**URL Formats:**
```
grafana://alerts.example.com/webhook
grafanas://alerts.example.com/webhook
grafana://username:password@alerts.example.com/webhook
grafana://token@alerts.example.com/webhook?method=PUT&max_alerts=100

# With HMAC signature verification
grafana://alerts.example.com/webhook?hmac_secret=yoursecret

# With custom headers
grafana://alerts.example.com/webhook?header_X-Environment=prod
```

**Query Parameters:**
- `method=POST|PUT` - HTTP method (default: POST)
- `max_alerts=N` - Limit number of alerts in payload (default: unlimited)
- `hmac_secret=secret` - Shared secret for HMAC-SHA256 signature generation
- `header_X-Custom=value` - Custom HTTP headers

**Authentication:**
- HTTP Basic Auth: `grafana://user:pass@host/path`
- Bearer Token: `grafana://token@host/path`

**Features:**
- Full Grafana webhook payload format support
- Alert status mapping (firing/resolved)
- Severity levels (info, warning, critical, ok)
- Tag support as alert labels
- HMAC signature for webhook validation
- Custom headers for routing/filtering
- Alert truncation with configurable limits
- Compatible with Grafana v9.0+ Alerting

**Severity Mapping:**
- `NotifyTypeInfo` → severity: "info"
- `NotifyTypeWarning` → severity: "warning"
- `NotifyTypeError` → severity: "critical"
- `NotifyTypeSuccess` → severity: "ok" (alert resolved)

**Example Usage:**
```go
app := apprise.New()
app.Add("grafana://alerts.example.com/api/webhooks/apprise?hmac_secret=secret")

// Send firing alert
app.Notify("High CPU Usage", "CPU at 92%", apprise.NotifyTypeError,
    apprise.WithTags("production", "web-server"))

// Send resolved alert
app.Notify("CPU Normal", "CPU back to normal", apprise.NotifyTypeSuccess)
```

**Grafana Setup:**
1. In Grafana: Alerts & IRM → Alerting → Contact points
2. Click "+ Add contact point"
3. Select "Webhook" integration
4. Enter your apprise-go webhook URL
5. Configure authentication if needed
6. Test the contact point

### Prometheus AlertManager

Prometheus AlertManager webhook notifications for routing and managing alerts from Prometheus monitoring.

**URL Formats:**
```
prometheus://alertmanager.example.com/api/v1/webhook
prometheus://alertmanager.example.com:9093/webhook
prometheusam://alertmanager.example.com/alerts

# With HTTPS
prometheus://alertmanager.example.com:443/alerts
prometheus://alertmanager.example.com/webhook?secure=true

# With options
prometheus://alertmanager.example.com/webhook?send_resolved=false
```

**Query Parameters:**
- `send_resolved=true|false` - Send resolved alerts (default: true)
- `secure=true` - Use HTTPS (default: auto-detect from port 443)

**Features:**
- Full AlertManager webhook payload format (API v4)
- Alert status mapping (firing/resolved)
- Severity mapping (critical, warning, info)
- Tag support as alert labels
- Auto-generated alert fingerprints
- Timestamps in RFC3339 format
- Compatible with Prometheus AlertManager v0.20+

**Severity Mapping:**
- `NotifyTypeError` → severity: "critical" (high priority)
- `NotifyTypeWarning` → severity: "warning"
- `NotifyTypeInfo` → severity: "info"
- `NotifyTypeSuccess` → status: "resolved" (alert cleared)

**Alert Status:**
- `firing` - Active alert (Error, Warning, Info types)
- `resolved` - Cleared alert (Success type)

**Webhook Payload Structure:**
```json
{
  "receiver": "apprise-go",
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {
      "alertname": "apprise_notification",
      "severity": "critical",
      "source": "apprise-go"
    },
    "annotations": {
      "summary": "High CPU Usage",
      "description": "CPU at 95%"
    },
    "startsAt": "2025-01-15T10:30:00Z",
    "fingerprint": "abc123"
  }],
  "version": "4"
}
```

**Example Usage:**
```go
app := apprise.New()
app.Add("prometheus://alertmanager.example.com:9093/webhook")

// Send firing alert
app.Notify("Database Down", "Connection timeout after 30s",
    apprise.NotifyTypeError,
    apprise.WithTags("production", "database"))

// Send resolved alert
app.Notify("Database Restored", "Connection re-established",
    apprise.NotifyTypeSuccess)
```

**AlertManager Setup:**
1. Configure AlertManager `alertmanager.yml`:
```yaml
route:
  receiver: 'apprise-go'

receivers:
  - name: 'apprise-go'
    webhook_configs:
      - url: 'http://your-server:8080/prometheus-webhook'
        send_resolved: true
```

2. In your apprise-go service, set up a webhook receiver:
```go
http.HandleFunc("/prometheus-webhook", func(w http.ResponseWriter, r *http.Request) {
    var payload apprise.PrometheusWebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)

    // Process alerts
    for _, alert := range payload.Alerts {
        fmt.Printf("Alert: %s - %s\n",
            alert.Labels["alertname"],
            alert.Annotations["summary"])
    }

    w.WriteHeader(http.StatusOK)
})
```

**Why Use Go for AlertManager Integration:**
- **Native Prometheus Ecosystem**: Perfect match for Prometheus/Go stack
- **Type-Safe Payloads**: Compile-time validation of webhook structures
- **High Throughput**: Handles thousands of alerts/second efficiently
- **Low Latency**: Sub-millisecond alert processing overhead
- **Kubernetes Native**: Deploy alongside Prometheus in k8s clusters
- **Zero Dependencies**: No external libs needed for webhook handling
- **Memory Efficient**: Minimal footprint compared to Python implementations

### Elasticsearch / OpenSearch

Elasticsearch/OpenSearch log aggregation and search engine notification indexing.

**URL Formats:**
```
elasticsearch://localhost:9200/alerts
elasticsearch://user:pass@es.example.com:9200/notifications
elasticsearch://es.example.com/logs?apikey=abc123
opensearch://opensearch.example.com:9200/apprise
es://localhost:9200/security-alerts

# With HTTPS
elasticsearch://es.example.com/logs?secure=true
elasticsearch://es.example.com/logs?ssl=true
```

**Query Parameters:**
- `apikey=key` - API key authentication (preferred over basic auth)
- `secure=true` - Use HTTPS (default: HTTP)
- `ssl=true` - Use HTTPS (alias for secure)

**Authentication Methods:**
- **Basic Auth**: `elasticsearch://user:pass@host:9200/index`
- **API Key**: `elasticsearch://host/index?apikey=your-api-key` (recommended)

**Features:**
- Document indexing for alert storage
- Searchable notification history
- Tag support for filtering
- Severity level mapping
- Timestamp in RFC3339 format
- Compatible with Elasticsearch 7.x+ and OpenSearch 1.x+
- Works with self-hosted and cloud instances

**Document Structure:**
```json
{
  "@timestamp": "2025-01-15T10:30:00Z",
  "title": "High CPU Usage",
  "message": "CPU at 95%",
  "severity": "error",
  "notify_type": "error",
  "tags": ["production", "web-server"],
  "source": "apprise-go",
  "host": "web-01",
  "environment": "production",
  "application": "myapp"
}
```

**Severity Mapping:**
- `NotifyTypeError` → severity: "error"
- `NotifyTypeWarning` → severity: "warning"
- `NotifyTypeInfo` → severity: "info"
- `NotifyTypeSuccess` → severity: "success"

**Example Usage:**
```go
app := apprise.New()
app.Add("elasticsearch://user:pass@localhost:9200/alerts")

// Index an alert
app.Notify("Database Connection Lost", "PostgreSQL connection timeout after 30s",
    apprise.NotifyTypeError,
    apprise.WithTags("production", "database"))

// Success notification
app.Notify("Database Restored", "Connection re-established",
    apprise.NotifyTypeSuccess)
```

**Elasticsearch Setup:**
1. Create index with mapping (optional but recommended):
```bash
curl -X PUT "localhost:9200/alerts" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "title": { "type": "text" },
      "message": { "type": "text" },
      "severity": { "type": "keyword" },
      "notify_type": { "type": "keyword" },
      "tags": { "type": "keyword" },
      "source": { "type": "keyword" },
      "environment": { "type": "keyword" },
      "application": { "type": "keyword" }
    }
  }
}
'
```

2. Create API key for authentication:
```bash
curl -X POST "localhost:9200/_security/api_key" -H 'Content-Type: application/json' -d'
{
  "name": "apprise-go-notifications",
  "role_descriptors": {
    "apprise_writer": {
      "cluster": ["monitor"],
      "index": [
        {
          "names": ["alerts*"],
          "privileges": ["create_index", "write", "auto_configure"]
        }
      ]
    }
  }
}
'
```

**Query Indexed Alerts:**
```bash
# Get recent alerts
curl "localhost:9200/alerts/_search?q=severity:error&sort=@timestamp:desc"

# Get alerts by tag
curl "localhost:9200/alerts/_search?q=tags:production"

# Complex query with Kibana/OpenSearch Dashboards
GET /alerts/_search
{
  "query": {
    "bool": {
      "must": [
        { "term": { "severity": "error" } },
        { "range": { "@timestamp": { "gte": "now-1h" } } }
      ]
    }
  },
  "sort": [{ "@timestamp": { "order": "desc" } }]
}
```

**Why Use Go for Elasticsearch Integration:**
- **Fast JSON Processing**: Native encoder 2-3x faster than Python
- **Efficient Bulk Operations**: Better connection pooling
- **Type-Safe Documents**: Compile-time validation
- **Memory Efficient**: Lower overhead for high-volume indexing
- **Native HTTP/2**: Better performance for cloud Elasticsearch
- **Zero Dependencies**: No pip packages or version conflicts

### MQTT

MQTT (Message Queuing Telemetry Transport) is the standard protocol for IoT device communication. Perfect for home automation, industrial IoT, edge computing, and real-time telemetry.

**URL Formats:**
```
# Basic MQTT (TCP)
mqtt://broker.example.com/topic/path
mqtt://user:password@broker.example.com:1883/notifications

# Secure MQTT (TLS/SSL)
mqtts://broker.example.com/secure/alerts
mqtts://broker.example.com:8883/home/sensors

# With authentication and QoS
mqtt://user:pass@broker.local/alerts?qos=1&retained=true
mqtt://broker/topic?clientid=my-app-001&qos=2

# With Last Will and Testament
mqtt://broker/topic?will_topic=offline&will_payload=disconnected&will_qos=1&will_retain=true
```

**Query Parameters:**
- `qos=0|1|2` - Quality of Service level (default: 0)
  - 0: At most once delivery (fire and forget)
  - 1: At least once delivery (acknowledged)
  - 2: Exactly once delivery (guaranteed)
- `retained=true|false` - Retain message on broker (default: false)
- `clientid=string` - MQTT client identifier (default: auto-generated as `apprise-go-{timestamp}`)
- `clean=true|false` - Clean session flag (default: true)
- `will_topic=string` - Last Will and Testament topic
- `will_payload=string` - Last Will message content
- `will_qos=0|1|2` - Last Will QoS level (default: 0)
- `will_retain=true|false` - Retain Last Will message (default: false)

**TLS/SSL Parameters (for mqtts://):**
- `insecure=true|false` - Skip TLS certificate verification (default: false)
- `ca_file=/path/to/ca.pem` - CA certificate file path
- `cert_file=/path/to/client.pem` - Client certificate file path
- `key_file=/path/to/client-key.pem` - Client key file path

**Authentication:**
- Username/Password: `mqtt://user:pass@broker/topic`
- Client certificates: Use `cert_file` and `key_file` parameters for mutual TLS

**Features:**
- All 3 QoS levels (0, 1, 2) for delivery guarantees
- TLS/SSL encryption with certificate validation
- Last Will and Testament (LWT) for offline detection
- Topic hierarchy support (e.g., `home/livingroom/temperature`)
- Message retention on broker
- Custom client IDs for persistent sessions
- Auto-reconnection disabled (single-shot notifications)

**Message Format:**
Messages include notification type prefix and optional tags:
```
[INFO] System Update: Version 2.1.0 deployed
[WARN] High Memory: Usage at 85%
[ERROR] Service Down: API not responding [production, critical]
[OK] Deployment Complete: All services running
```

**Example Usage:**
```go
app := apprise.New()

// Basic IoT notification
app.Add("mqtt://broker.local/home/alerts")

// Industrial monitoring with guaranteed delivery
app.Add("mqtt://user:pass@industrial.example.com/plant/sensors?qos=2")

// Home automation with retained messages
app.Add("mqtt://homeassistant.local/notifications?retained=true&qos=1")

// Secure cloud MQTT with Last Will
app.Add("mqtts://mqtt.cloud.com:8883/devices/alerts?will_topic=device/status&will_payload=offline")

// Send notification
app.Notify("Temperature Alert", "Sensor reading: 85°C", apprise.NotifyTypeWarning,
    apprise.WithTags("zone-1", "temperature"))
```

**Use Cases:**
- Home automation notifications (Home Assistant, OpenHAB)
- Industrial IoT monitoring and alerts
- Edge computing event distribution
- Real-time telemetry and sensor data
- Device status updates and health monitoring
- Distributed system notifications

**Popular MQTT Brokers:**
- Mosquitto (open source, lightweight)
- HiveMQ (enterprise, cloud-hosted)
- EMQX (scalable, high-performance)
- AWS IoT Core (managed cloud service)
- Azure IoT Hub (Microsoft cloud service)

**Go Advantages:**
- Native binary protocol support (efficient for IoT)
- Low memory footprint for edge devices
- Fast connection handling
- Built-in TLS/SSL support
- Superior concurrency for multiple topics

### Opsgenie

Atlassian's incident management and alerting service with comprehensive responder and priority management.

**URL Formats:**
```
# Basic API key (defaults to US region)
opsgenie://api_key

# Specify region explicitly
opsgenie://api_key@us
opsgenie://api_key@eu

# With team and user responders
opsgenie://api_key@us/backend-team/user@example.com

# With priority and tags
opsgenie://api_key@us?priority=P1&tags=critical,production

# Full configuration
opsgenie://api_key@eu/oncall-team?priority=P2&teams=devops,backend&entity=web-server&source=monitoring&alias=db-alert
```

**Query Parameters:**
- `region=us|eu` - Opsgenie region (default: us)
- `priority=P1-P5` - Alert priority (P1=Critical, P5=Informational)
- `tags=tag1,tag2` - Comma-separated tags for alert categorization
- `teams=team1,team2` - Additional team responders (comma-separated)
- `alias=string` - Alert alias for deduplication
- `entity=string` - Entity name (server, application, etc.)
- `source=string` - Alert source identifier (default: apprise-go)
- `user=string` - User who created the alert
- `note=string` - Additional note for the alert

**Features:**
- Opsgenie Alerts API v2 compliance
- US and EU region support with appropriate API endpoints
- Multiple responder types: teams (path/query), users (email detection)
- Priority levels P1-P5 with automatic mapping from notification types
- Alert deduplication via alias parameter
- Rich alert metadata: entity, source, tags, notes
- Team and user responder assignment
- Integration with Opsgenie's incident response workflows
- Alert details with notification context

**Responder Detection:**
- **Teams**: Simple names in path or teams query parameter → `{"type": "team", "name": "backend"}`
- **Users**: Email addresses in path → `{"type": "user", "name": "user@example.com"}`
- **Mixed**: Can combine teams and users in single URL

**Priority Mapping:**
- `NotifyTypeError` → P1 (Critical)
- `NotifyTypeWarning` → P2 (High)  
- `NotifyTypeInfo` → P3 (Moderate)
- `NotifyTypeSuccess` → P4 (Low)

**Regional Endpoints:**
- **US Region**: `https://api.opsgenie.com/v2/alerts`
- **EU Region**: `https://api.eu.opsgenie.com/v2/alerts`

**Example:**
```go
// Send P1 alert to EU region with team responders and custom metadata
app.Add("opsgenie://abc123@eu/backend-team/devops?priority=P1&tags=production,database&entity=mysql-cluster&alias=db-performance")
```

### Sentry

Sentry is the leading application monitoring and error tracking platform trusted by 3M+ developers worldwide. Perfect for tracking errors, performance issues, and release health in production applications.

**URL Formats:**
```
# Standard Sentry.io DSN
sentry://public_key@o123456.ingest.sentry.io/789012
sentries://public_key@o123456.ingest.sentry.io/789012

# HTTPS format
https://public_key@sentry.example.com/project_id

# Self-hosted Sentry
sentry://key@sentry.internal.com:8080/project-id
http://key@localhost:9000/123

# Regional Sentry ingestion
sentry://key@o123456.ingest.us.sentry.io/789012
sentry://key@o123456.ingest.eu.sentry.io/789012
```

**DSN Structure:**
```
protocol://public_key@host[:port]/project_id
```

**Components:**
- `protocol`: `sentry`, `sentries` (HTTPS), `http`, or `https`
- `public_key`: Your project's public DSN key (found in project settings)
- `host`: Sentry host (e.g., `o123456.ingest.sentry.io` or self-hosted domain)
- `project_id`: Your Sentry project ID

**Features:**
- Full Sentry envelope format support
- Automatic severity level mapping
- Event ID generation (UUID v4)
- Tag support for categorization
- Extra context metadata
- Self-hosted and cloud (sentry.io) support
- Multi-region ingestion endpoints
- Error tracking with stack traces
- Platform identification (go)

**Severity Mapping:**
- `NotifyTypeInfo` → Sentry level: "info"
- `NotifyTypeSuccess` → Sentry level: "info"
- `NotifyTypeWarning` → Sentry level: "warning"
- `NotifyTypeError` → Sentry level: "error"

**Event Structure:**
All notifications are sent as Sentry events using the envelope format with:
- Unique event ID (UUID v4)
- Timestamp (ISO 8601)
- Platform: "go"
- Logger: "apprise-go"
- Message with title and body
- Tags from notification request
- Extra context metadata

**Finding Your DSN:**
1. Log into [sentry.io](https://sentry.io) or your self-hosted instance
2. Navigate to **Settings** → **Projects** → Select your project
3. Go to **Client Keys (DSN)**
4. Copy the DSN (format: `https://public_key@host/project_id`)
5. Convert to `sentry://` format for apprise-go

**Example Usage:**
```go
app := apprise.New()

// Production error tracking
app.Add("sentry://abc123def456@o123456.ingest.sentry.io/789012")

// Send error event
app.Notify("Database Connection Failed",
    "Unable to connect to PostgreSQL on db.example.com:5432",
    apprise.NotifyTypeError,
    apprise.WithTags("database", "production", "critical"))

// Send warning event
app.Notify("High Memory Usage",
    "Application memory usage at 85% of available",
    apprise.NotifyTypeWarning,
    apprise.WithTags("performance", "memory"))

// Send info event (for non-errors)
app.Notify("Deployment Complete",
    "Version 2.1.0 deployed successfully to production",
    apprise.NotifyTypeInfo,
    apprise.WithTags("deployment", "production"))
```

**Self-Hosted Sentry:**
```go
// Self-hosted on-premises Sentry instance
app.Add("http://my_key@sentry.internal.company.com:8080/project-42")

// With HTTPS
app.Add("https://key123@sentry.example.com/1")
```

**Multi-Region Deployment:**
```go
// US region ingestion
app.Add("sentry://key@o123456.ingest.us.sentry.io/789012")

// EU region ingestion
app.Add("sentry://key@o123456.ingest.eu.sentry.io/789012")
```

**Use Cases:**
- Production error monitoring
- Application crash reporting
- Performance issue detection
- Release health tracking
- User feedback collection
- Integration with CI/CD pipelines
- Real-time error alerts
- Exception aggregation and grouping

**Go Advantages:**
- Native UUID generation (crypto/rand)
- Efficient envelope format handling
- Type-safe event structures
- Compile-time validation
- Fast JSON serialization
- Built-in HTTP client pooling

**Notes:**
- DSNs are safe to keep public (they only allow event submission)
- Events are deduplicated by Sentry based on stack trace and message
- The public key is required; secret key is deprecated and not used
- Notifications appear as "Issues" in Sentry dashboard
- Tags help with filtering and searching in Sentry UI

### Matrix

Decentralized messaging with Client-Server API v3 support for both access token and username/password authentication.

**URL Formats:**
```
# Access token authentication (recommended)
matrix://access_token@matrix.org/!room_id:matrix.org
matrix://access_token@matrix.org/#room_alias:matrix.org

# Username/password authentication
matrix://username:password@matrix.example.com/general
matrix://username:password@homeserver.com/room1/room2

# Token in query parameter
matrix://username@matrix.org/general?token=access_token

# Multiple rooms and options
matrix://token@matrix.org/room1/room2/#room3:matrix.org?msgtype=notice&format=html
```

**Query Parameters:**
- `msgtype=text|notice` - Message type (default: text)
- `format=html` - Enable HTML formatting in messages
- `token=string` - Access token (alternative to URL auth)

**Features:**
- Matrix Client-Server API v3 compliance
- Support for both access token and username/password authentication
- Multiple room targeting in a single URL
- Room ID (!room:server) and alias (#room:server) formats
- Automatic room normalization (simple names become aliases)
- HTML message formatting with proper escaping
- Message and notice types (m.text, m.notice)
- Automatic login and session management
- Support for both public and private homeservers

**Authentication Methods:**
1. **Access Token** (recommended for production)
   ```go
   app.Add("matrix://syt_dXNlcm5hbWU_abcdef123456789@matrix.org/!room:matrix.org")
   ```

2. **Username/Password** (for development/testing)
   ```go
   app.Add("matrix://myuser:mypass@matrix.example.com/general")
   ```

3. **Mixed Authentication** (username with token in query)
   ```go
   app.Add("matrix://myuser@matrix.org/room?token=access_token")
   ```

**Room Formats:**
- **Room ID**: `!AbCdEf123456789:matrix.org` (specific room identifier)
- **Room Alias**: `#general:matrix.org` (human-readable room alias)
- **Simple Name**: `general` (auto-converts to `#general:homeserver`)

**Example:**
```go
// Send critical alert to Matrix operations room with HTML formatting
app.Add("matrix://access_token@company.matrix.org/#ops:company.matrix.org?msgtype=notice&format=html")
```

### Pushover

Mobile push notifications with priority levels and custom sounds.

**URL Formats:**
```
pushover://token@userkey
pover://token@userkey/device1/device2
pushover://token@userkey?priority=1&sound=cosmic
pushover://token@userkey?priority=2&retry=60&expire=3600
```

**Query Parameters:**
- `priority=-2/-1/0/1/2` - Priority level (-2=lowest, 2=emergency)
- `sound=pushover/bike/cosmic` - Notification sound
- `retry=60` - Retry interval for emergency priority (seconds)
- `expire=3600` - Expiration for emergency priority (seconds)

**Features:**
- Priority levels from silent to emergency
- Custom notification sounds
- Device targeting
- Emergency notifications with retry/expire
- Rich formatting with emojis

### Pushbullet

Cross-platform push notifications to devices, emails, and channels.

**URL Formats:**
```
pball://access_token
pushbullet://access_token/device_id
pball://access_token/user@email.com
pball://access_token/#channel_name
pball://access_token?device=device1,device2&email=user@domain.com
```

**Query Parameters:**
- `device=id1,id2` - Target specific devices (comma-separated)
- `email=user@domain.com` - Send to email addresses  
- `channel=channel1,channel2` - Send to channels (comma-separated)

**Features:**
- Multi-device targeting
- Email and channel support
- File attachment support
- Cross-platform compatibility
- Emoji indicators by notification type

### Pushsafer

GDPR-compliant European push notification service for iOS, Android, and Windows 10.

**URL Formats:**
```
pushsafer://privatekey
pushsafer://privatekey/a (all devices)
pushsafer://privatekey/52 (specific device)
pushsafer://privatekey/gs100 (device group)
psafer://privatekey?sound=5&vibration=2&icon=33
pushsafer://key?sound=10&vibration=2&icon=33&color=%23FF0000&priority=2
```

**Query Parameters:**
- `device=id` - Device ID, device group (gs*), or "a" for all devices
- `sound=0-60` - Sound ID (0=silent, 1-60=various sounds)
- `vibration=1-3` - Vibration pattern (1=default, 2=once, 3=twice)
- `icon=1-176` - Icon ID (1-176 icons available)
- `color=#RRGGBB` - Icon color in hex format
- `priority=-2 to 2` - Priority level (-2=lowest, 2=critical/ignores DND)
- `ttl=minutes` - Time to live (0-43200 minutes, ~30 days)

**Features:**
- GDPR-compliant (European servers)
- iOS, Android, Windows 10 support
- Priority levels with Do Not Disturb override
- 176 built-in icons with custom colors
- 60+ notification sounds
- Device groups for targeted notifications
- Image attachments (up to 3 images)
- BBCode formatting support
- Time-to-live for message expiration

**Priority Mapping:**
- `NotifyTypeError` → Priority 2 (Critical - ignores Do Not Disturb)
- `NotifyTypeWarning` → Priority 1 (High)
- `NotifyTypeInfo` → Priority 0 (Normal)
- `NotifyTypeSuccess` → Priority 0 (Normal)

**Example Usage:**
```go
app := apprise.New()
app.Add("pushsafer://a1b2c3d4e5f6/a?sound=5&icon=33&color=%23FF0000")

// Critical error notification
app.Notify("Database Down", "PostgreSQL connection lost",
    apprise.NotifyTypeError) // Priority 2 - ignores DND

// Normal notification to specific device
app.Add("pushsafer://key/52?sound=10")
app.Notify("Backup Complete", "Daily backup finished successfully",
    apprise.NotifyTypeSuccess)

// Device group notification
app.Add("pushsafer://key/gs100?vibration=2")
app.Notify("Server Alert", "CPU usage above 90%",
    apprise.NotifyTypeWarning)
```

**Pushsafer Setup:**
1. Sign up at [https://www.pushsafer.com](https://www.pushsafer.com)
2. Get your Private Key from the dashboard
3. Install Pushsafer app on your devices
4. Devices automatically register to your account
5. Optionally create device groups for targeted notifications

**Device Groups:**
Create device groups in the Pushsafer dashboard to send notifications to specific sets of devices. Use `gs` prefix followed by the group ID: `pushsafer://key/gs100`

**Icon Examples:**
- Icon 1: Information
- Icon 8: Warning triangle
- Icon 10: Error/Stop sign
- Icon 33: Mail/Email
- Icon 50: Database
- Icon 100: Server
- Icon 150: Calendar
- Full list: [https://www.pushsafer.com/en/pushapi_ext#API-I](https://www.pushsafer.com/en/pushapi_ext#API-I)

**Sound Examples:**
- Sound 0: Silent
- Sound 1: Default
- Sound 5: Climb
- Sound 10: Persistent
- Sound 25: Space
- Full list: [https://www.pushsafer.com/en/pushapi_ext#API-S](https://www.pushsafer.com/en/pushapi_ext#API-S)

**Why Use Pushsafer:**
- **GDPR Compliance**: European servers, ideal for EU businesses
- **Privacy Focused**: Data stored in Germany, EU data protection laws
- **Multi-Platform**: Works on iOS, Android, Windows 10
- **Pushover Alternative**: Similar features, better European coverage
- **Rich Customization**: 176 icons, 60 sounds, custom colors
- **Reliable**: Established service with strong European user base

**Why Use Go for Pushsafer:**
- **Fast JSON Encoding**: 2-3x faster than Python for API requests
- **Type-Safe Payloads**: Compile-time validation prevents errors
- **Efficient HTTP Pooling**: Reuses connections for better performance
- **Zero Dependencies**: No pip packages, single binary deployment
- **Memory Efficient**: Lower overhead for high-volume notifications

### Twilio SMS

SMS/MMS messaging with rate limiting and phone number normalization.

**URL Formats:**
```
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543/+15551111111
twilio://ACCOUNT_SID:AUTH_TOKEN@15551234567/15559876543
twilio://ACCOUNT_SID:AUTH_TOKEN@+15551234567/+15559876543?apikey=KEY
```

**Query Parameters:**
- `apikey=KEY` - Optional API key for authentication

**Features:**
- Automatic phone number normalization (E.164 format)
- Rate limiting (0.2 requests/second)
- Multiple recipient support
- US/Canada number auto-formatting
- SMS message length optimization

### Twilio Voice

Voice call notifications with text-to-speech synthesis via Twilio Voice API.

**URL Formats:**
```
# Direct Twilio API
twilio-voice://ACCOUNT_SID:AUTH_TOKEN@api.twilio.com/+15551234567/+15559876543
twilio-voice://ACCOUNT_SID:AUTH_TOKEN@api.twilio.com/+15551234567/+15559876543/+15551111111
twilio-voice://ACCOUNT_SID:AUTH_TOKEN@api.twilio.com/+15551234567?to=+15559876543,+15551111111

# With voice customization
twilio-voice://ACCOUNT_SID:AUTH_TOKEN@api.twilio.com/+15551234567/+15559876543?language=en-US&gender=female
twilio-voice://ACCOUNT_SID:AUTH_TOKEN@api.twilio.com/+15551234567/+15559876543?language=es-ES&gender=male

# Webhook proxy mode (for secure credential management)
twilio-voice://proxy-key@webhook.example.com/twilio-voice?account_sid=ACCOUNT_SID&auth_token=AUTH_TOKEN&from=+15551234567&to=+15559876543
```

**Query Parameters:**
- `language=code` - Voice language (e.g., en-US, es-ES, fr-FR, de-DE) - default: en-US
- `gender=male|female` - Voice gender - default: female
- `to=+1234,+5678` - Comma-separated destination numbers (alternative to path)

**Supported Languages:**
- `en-US` - English (US)
- `en-GB` - English (UK)
- `es-ES` - Spanish (Spain)
- `es-MX` - Spanish (Mexico)
- `fr-FR` - French
- `de-DE` - German
- `it-IT` - Italian
- `ja-JP` - Japanese
- `ko-KR` - Korean
- `pt-BR` - Portuguese (Brazil)
- `ru-RU` - Russian
- `zh-CN` - Chinese (Simplified)
- `zh-TW` - Chinese (Traditional)
- And more...

**Features:**
- Text-to-speech using Twilio's voice synthesis
- Multiple language and voice options
- Automatic TwiML generation
- Multiple recipient support
- Phone number validation (E.164 format)
- Webhook proxy support for secure credentials
- Message cleaning for voice synthesis
- Notification type prefixes (Alert, Warning, Success)
- Direct Twilio API or webhook proxy modes

**TwiML Generation:**
The service automatically generates TwiML (Twilio Markup Language) for voice synthesis:
- Combines title and body into spoken message
- Adds context based on notification type (Alert, Warning, Success)
- Cleans special characters for proper voice synthesis
- Configures voice language and gender

**Example Usage:**
```go
app := apprise.New()

// Basic voice call
app.Add("twilio-voice://AC123:token@api.twilio.com/+15551234567/+15559876543")

// Spanish voice, male
app.Add("twilio-voice://AC123:token@api.twilio.com/+15551234567/+15559876543?language=es-ES&gender=male")

// Multiple recipients via webhook proxy
app.Add("twilio-voice://api-key@webhook.company.com/twilio-voice?account_sid=AC123&auth_token=token&from=+15551234567&to=+15559876543,+15551111111")

// Send critical alert
app.Notify("System Failure", "Database connection lost", apprise.NotifyTypeError)
// Voice will say: "Alert: System Failure. Database connection lost"
```

**Webhook Proxy Mode:**
For production environments, use webhook proxy mode to keep Twilio credentials secure:
```go
// Webhook proxy handles actual Twilio API calls with stored credentials
app.Add("twilio-voice://proxy-api-key@webhook.yourcompany.com/twilio-voice?account_sid=AC123&auth_token=token&from=+15551234567&to=+15559876543")
```

The webhook receives a JSON payload with TwiML and call details, then makes the actual Twilio API call server-side.

**Voice Synthesis Features:**
- Special character replacement (& → "and", < → "less than", etc.)
- Quote handling for natural speech
- Multiple space normalization
- Automatic message trimming
- Context-aware prefixes based on notification type

### Desktop Notifications

Cross-platform desktop notifications using native OS notification systems.

**URL Formats:**
```
# Generic (auto-detects platform)
desktop://

# Platform-specific
macosx://                          # macOS via terminal-notifier
windows://                         # Windows system tray notifications
linux://                           # Linux via notify-send

# Linux-specific DBus notifications
dbus://                            # Auto-detect DBus interface
qt://                              # Force QT interface
kde://                             # KDE desktop environment
glib://                            # GLib interface
gnome://                           # Gnome desktop environment

# With parameters
macosx://?sound=default&image=/path/to/icon.png
windows://?duration=10             # Display for 10 seconds
desktop://?image=/path/to/image.png
```

**Query Parameters:**
- `sound=name` - System sound name (macOS only)
- `duration=seconds` - Display duration in seconds (Windows only, default: 12)
- `image=path` - Path to image file for notification icon

**Platform Requirements:**
- **macOS:** Requires `terminal-notifier` - install with: `brew install terminal-notifier`
- **Windows:** Uses PowerShell and system tray notifications (no extra dependencies)
- **Linux:** Requires one of: `notify-send`, `zenity`, or `kdialog`

**Features:**
- Cross-platform compatibility with native OS integration
- Message length automatically limited to 250 characters
- Platform-specific notification styling and behavior
- Support for custom sounds and images
- Graceful fallbacks when notification tools are unavailable

### Bark

iOS push notification service for sending custom notifications to your iPhone via the Bark app.

**URL Formats:**
```
# HTTP
bark://devicekey@api.day.app/
bark://devicekey@bark.example.com:8080/

# HTTPS (secure)
barks://devicekey@api.day.app/
barks://devicekey@bark.example.com:8443/

# With icon support (v1.9.5+)
bark://devicekey@api.day.app/?icon=https://example.com/icon.png

# With all parameters
bark://devicekey@api.day.app/?icon=https://example.com/icon.png&sound=alarm&badge=5&url=https://example.com&category=news&group=alerts
```

**Query Parameters:**
- `icon=url` - Custom icon URL (v1.9.5+)
- `sound=name` - Sound name (e.g., alarm, bell, chime)
- `badge=int` - Badge count to display on app icon
- `url=url` - URL to open when notification is tapped
- `category=string` - Notification category for grouping
- `group=string` - Notification group for organization

**Features:**
- iOS push notifications via Bark app
- Custom icon URLs (new in v1.9.5)
- Custom sounds and badges
- Action URLs for tappable notifications
- Category and group organization
- Self-hosted server support
- HTTP and HTTPS support

**Example Usage:**
```go
app := apprise.New()

// Basic notification to official Bark server
app.Add("bark://your-device-key@api.day.app/")

// With custom icon
app.Add("bark://your-device-key@api.day.app/?icon=https://example.com/logo.png")

// Self-hosted with HTTPS and custom sound
app.Add("barks://your-device-key@bark.myserver.com/?sound=alarm&badge=3")

// Full featured with action URL
app.Add("bark://your-device-key@api.day.app/?icon=https://example.com/icon.png&sound=bell&badge=1&url=https://example.com/article&category=news")

app.Notify("Alert", "Server CPU usage high!", apprise.NotifyTypeWarning)
```

**Bark Setup:**
1. Install Bark app from iOS App Store
2. Open app and get your device key
3. Use official server (api.day.app) or set up your own Bark server
4. Configure notifications using your device key

**Supported Sounds:**
- alarm, anticipate, bell, birdsong, bloom
- calypso, chime, choo, descent, electronic
- fanfare, glass, gotosleep, healthnotification
- horn, ladder, mailsent, minuet, multiwayinvitation
- newmail, newsflash, noir, paymentsuccess
- shake, sherwoodforest, silence, spell, suspense
- telegraph, tiptoes, typewriters, update

### Gotify

Self-hosted push notification server for sending messages to devices and applications.

**URL Formats:**
```
gotify://hostname:port/app_token
gotifys://secure.example.com/app_token          # HTTPS
gotify://192.168.1.100:8080/ABCDefGHijkL?priority=5
```

**Query Parameters:**
- `priority=0-10` - Message priority level (default: 5)

**Features:**
- Self-hosted push notification solution
- Color-coded messages based on notification type
- Customizable priority levels (0-10)
- HTTP and HTTPS support
- JSON-based API integration
- Supports rich notification metadata via "extras"

### AWS SNS

Amazon Simple Notification Service for enterprise cloud messaging with webhook proxy support.

**URL Formats:**
```
# Via API Gateway webhook endpoint
sns://api.gateway.url/sns-proxy?topic_arn=arn:aws:sns:us-east-1:123456789:my-topic

# Via API Gateway with API key authentication
sns://api-key@api.gateway.url/webhook?topic_arn=arn:aws:sns:eu-west-1:987654321:alerts

# Using topic components instead of full ARN
sns://webhook.example.com/sns?topic=notifications&region=us-west-2&account=123456789

# With custom message attributes and formatting
sns://webhook.url/proxy?topic=alerts&format=json&attr_Environment=production&attr_Service=web-api
```

**Query Parameters:**
- `topic_arn=arn:...` - Full SNS topic ARN (recommended)
- `topic=name` - Topic name (requires `account` parameter)
- `region=us-east-1` - AWS region (default: us-east-1)
- `account=123456789` - AWS account ID (when using topic name)
- `subject=string` - Custom subject line for notifications
- `format=text|json` - Message format (default: text)
- `attr_Key=value` - Custom message attributes (prefix with attr_)
- `test_mode=true` - Use HTTP instead of HTTPS (for testing only)

**Features:**
- Enterprise-grade cloud messaging via Amazon SNS
- Webhook proxy integration for secure API access
- JSON and text message formatting options
- Custom message attributes and metadata
- Subject line customization
- Message size up to 256KB
- Emoji indicators based on notification type
- API key authentication support
- Regional endpoint support

**Authentication Methods:**
1. **API Gateway with API Key** (recommended for production)
   ```go
   app.Add("sns://your-api-key@api.gateway.amazonaws.com/prod/sns?topic_arn=arn:aws:sns:us-east-1:123456789:alerts")
   ```

2. **Custom Webhook Proxy**
   ```go
   app.Add("sns://webhook.yourcompany.com/sns-proxy?topic=notifications&region=us-east-1&account=123456789")
   ```

3. **Direct Integration** (requires AWS SDK setup)
   ```go
   app.Add("sns://your-webhook.com/direct-sns?topic_arn=arn:aws:sns:us-east-1:123456789:topic&format=json")
   ```

**Message Formats:**
- **Text Format** (default): `🔔 Alert Title\n\nAlert details here`
- **JSON Format**: `{"title":"Alert Title","body":"Alert details","type":"warning","emoji":"⚠️","timestamp":"2024-01-15T10:30:00Z"}`

**Message Attributes:**
All notifications include these attributes:
- `NotificationType`: error, warning, info, success
- `Source`: apprise-go
- Custom attributes via `attr_` query parameters

**Integration Notes:**
This service sends webhook requests to your configured endpoint, which should then forward the message to AWS SNS. This approach provides:
- Secure credential management (keys stay on your server)
- Custom authentication and authorization
- Message transformation and routing
- Integration with existing AWS infrastructure

**Example Webhook Payload:**
```json
{
  "topicArn": "arn:aws:sns:us-east-1:123456789:alerts",
  "message": "⚠️ Database Warning\n\nConnection pool at 80% capacity",
  "subject": "Database Warning",
  "region": "us-east-1",
  "messageAttributes": {
    "NotificationType": {"DataType": "String", "StringValue": "warning"},
    "Source": {"DataType": "String", "StringValue": "apprise-go"},
    "Environment": {"DataType": "String", "StringValue": "production"}
  }
}
```

**Example:**
```go
// Send critical alert to SNS via API Gateway with custom attributes
app.Add("sns://api-key@gateway.us-east-1.amazonaws.com/prod/sns?topic_arn=arn:aws:sns:us-east-1:123456789:alerts&format=json&attr_Environment=prod&attr_Team=backend")
```

### AWS SES

Amazon Simple Email Service for enterprise-grade email delivery with template support and rich formatting.

**URL Formats:**
```
# Basic email via webhook proxy
ses://api.gateway.url/ses-proxy?from=alerts@company.com&to=admin@company.com

# Multiple recipients with CC/BCC
ses://webhook.example.com/ses?from=noreply@company.com&to=team@company.com,alerts@company.com&cc=manager@company.com&bcc=audit@company.com

# With API key authentication and custom options
ses://api-key@api.gateway.amazonaws.com/prod/ses?from=Alerts%20Team%20<alerts@company.com>&to=oncall@company.com&subject=Custom%20Subject&region=eu-west-1

# Using SES templates with dynamic data
ses://webhook.url/ses?from=system@company.com&to=user@company.com&template=welcome-email&data_username=john&data_company=Acme%20Corp
```

**Query Parameters:**
- `from=email` - Sender email address (required)
- `name=Name` - Sender display name (optional)
- `to=email1,email2` - Recipient email addresses (required, comma-separated)
- `cc=email1,email2` - CC recipients (optional, comma-separated)
- `bcc=email1,email2` - BCC recipients (optional, comma-separated)
- `reply_to=email` - Reply-to email address (optional)
- `subject=string` - Custom subject line (optional)
- `region=us-east-1` - AWS region (default: us-east-1)
- `template=name` - SES template name for templated emails (optional)
- `data_key=value` - Template data parameters (prefix with data_)
- `test_mode=true` - Use HTTP instead of HTTPS (for testing only)

**Features:**
- Enterprise email delivery via Amazon SES
- HTML and plain text message formatting
- Multiple recipients (TO, CC, BCC)
- Attachment support up to 10MB
- SES template integration with dynamic data
- Custom sender names and reply-to addresses
- Regional endpoint support
- Rich HTML formatting with responsive design
- Emoji indicators based on notification type
- Professional email signatures

**Authentication Methods:**
1. **API Gateway with API Key** (recommended)
   ```go
   app.Add("ses://your-api-key@api.gateway.amazonaws.com/prod/ses?from=alerts@company.com&to=oncall@company.com")
   ```

2. **Custom Webhook Proxy**
   ```go
   app.Add("ses://webhook.yourcompany.com/ses-proxy?from=system@company.com&to=admin@company.com")
   ```

**Message Formatting:**
- **HTML Version**: Professional email template with:
  - Responsive design for mobile compatibility
  - Color-coded headers based on notification type
  - Proper HTML escaping for security
  - Branded footer with timestamp
- **Text Version**: Clean plain text format with:
  - Emoji indicators for notification types
  - Structured layout with clear sections
  - Professional signature

**Template Integration:**
Use SES templates for consistent branding and dynamic content:
```go
app.Add("ses://webhook.url/ses?from=alerts@company.com&to=team@company.com&template=incident-alert&data_severity=critical&data_service=database&data_environment=production")
```

Template data automatically includes:
- `title` - Notification title
- `body` - Notification body  
- `notifyType` - Notification type (info, warning, error, success)
- `timestamp` - ISO 8601 timestamp
- Custom data via `data_` query parameters

**Attachment Support:**
SES supports various attachment types:
```go
app := apprise.New()
app.Add("ses://webhook.url/ses?from=reports@company.com&to=team@company.com")

// Add file attachments
app.AddAttachment("/path/to/report.pdf")
app.AddAttachment("https://example.com/chart.png", "monthly_chart.png")

// Add data attachments
data := []byte("CSV,Data\nvalue1,value2")
app.AddAttachmentData(data, "report.csv", "text/csv")

app.Notify("Monthly Report", "Please find the monthly report attached", apprise.NotifyTypeInfo)
```

**Example Webhook Payload:**
```json
{
  "region": "us-east-1",
  "source": "Alerts Team <alerts@company.com>",
  "destination": {
    "toAddresses": ["oncall@company.com"],
    "ccAddresses": ["manager@company.com"],
    "bccAddresses": ["audit@company.com"]
  },
  "message": {
    "subject": {
      "data": "Database Alert",
      "charset": "UTF-8"
    },
    "body": {
      "html": {
        "data": "<!DOCTYPE html><html>...<h2 style=\"color: #dc3545;\">❌ Database Connection Failed</h2>...",
        "charset": "UTF-8"
      },
      "text": {
        "data": "❌ Database Connection Failed\n\nUnable to connect to primary database server...",
        "charset": "UTF-8"
      }
    }
  },
  "replyToAddresses": ["support@company.com"],
  "attachments": [
    {
      "filename": "error_log.txt",
      "contentType": "text/plain",
      "data": "base64-encoded-content",
      "size": 1024
    }
  ]
}
```

**Integration Notes:**
This service sends webhook requests to your configured endpoint, which should forward the email via AWS SES. This approach provides:
- Secure credential management (AWS keys stay on your server)
- Template customization and branding
- Compliance and audit logging
- Integration with existing SES configurations (reputation management, bounce handling)
- Cost optimization through SES pricing

**Example:**
```go
// Send critical alert with attachments via SES with custom template
app.Add("ses://api-key@gateway.amazonaws.com/prod/ses?from=Critical%20Alerts%20<critical@company.com>&to=oncall@company.com,management@company.com&cc=security@company.com&template=security-incident&data_incident_id=INC-2024-001&data_severity=high")
```

### Google Cloud Pub/Sub

Google Cloud Pub/Sub for scalable real-time messaging with advanced filtering and ordering capabilities.

**URL Formats:**
```
# Basic Pub/Sub via webhook proxy
pubsub://webhook.example.com/pubsub-proxy?project_id=my-project&topic=notifications

# With service account and ordering
pubsub://webhook.example.com/gcp?project_id=company-project&topic=alerts&service_account=sa@project.iam.gserviceaccount.com&ordering_key=region-us

# With API key authentication and custom attributes
pubsub://api-key@api.gateway.googleapis.com/v1/pubsub?project_id=prod-project&topic=events&attr_environment=production&attr_service=api&attr_team=backend

# Full configuration with metadata
pubsub://webhook.url/proxy?project_id=my-project&topic=logs&ordering_key=severity&attr_datacenter=us-east1&attr_version=v1.2.3
```

**Query Parameters:**
- `project_id=string` - Google Cloud Project ID (required)
- `topic=string` - Pub/Sub topic name (required)
- `service_account=email` - Service account for authentication (optional)
- `ordering_key=string` - Key for ordered message delivery (optional)
- `attr_key=value` - Custom message attributes for filtering (prefix with attr_)
- `test_mode=true` - Use HTTP instead of HTTPS (for testing only)

**Features:**
- Google Cloud Pub/Sub integration with structured JSON messaging
- Custom message attributes for subscriber filtering and routing
- Ordered message delivery with configurable ordering keys
- Service account authentication support
- Project-based topic organization
- Rich message metadata with timestamp and version tracking
- Severity-based priority mapping
- Comprehensive attribute-based filtering capabilities

**Authentication Methods:**
1. **Service Account** (recommended for production)
   ```go
   app.Add("pubsub://webhook.url/pubsub?project_id=my-project&topic=alerts&service_account=alerts@my-project.iam.gserviceaccount.com")
   ```

2. **API Key Authentication**
   ```go
   app.Add("pubsub://api-key@webhook.company.com/gcp-proxy?project_id=company-project&topic=notifications")
   ```

3. **Managed Identity** (via webhook proxy)
   ```go
   app.Add("pubsub://webhook.googleapis.com/pubsub-proxy?project_id=my-project&topic=events")
   ```

**Message Structure:**
All messages are published as structured JSON with:
```json
{
  "title": "Alert Title",
  "body": "Alert description and details",
  "type": "warning",
  "timestamp": "2024-01-15T10:30:00Z",
  "source": "apprise-go",
  "version": "1.9.4-2",
  "severity": "WARNING",
  "category": "notification",
  "emoji": "⚠️",
  "color": "#ffc107",
  "priority": "medium",
  "environment": {
    "project": "my-project",
    "topic": "alerts",
    "orderingKey": "region-us"
  }
}
```

**Message Attributes:**
Comprehensive attributes for subscriber filtering:
- `notificationType`: error, warning, info, success
- `severity`: ERROR, WARNING, INFO
- `priority`: HIGH, MEDIUM, LOW, NORMAL
- `alertLevel`: CRITICAL, WARNING, INFO
- `source`: apprise-go
- `version`: Current Apprise-Go version
- `timestamp`: ISO 8601 timestamp
- `topic`: Topic name for routing
- `project`: Project ID for multi-project setups
- `orderingKey`: Ordering key (if specified)
- Custom attributes via `attr_` query parameters

**Ordered Delivery:**
Enable ordered message processing with ordering keys:
```go
// Messages with same ordering key are delivered in order
app.Add("pubsub://webhook.url/pubsub?project_id=my-project&topic=user-events&ordering_key=user-123")
```

**Subscriber Filtering Examples:**
```go
// Subscribers can filter messages by attributes
// Error messages only: attributes.severity = "ERROR"
// High priority only: attributes.priority = "HIGH"
// Specific environment: attributes.environment = "production"
// Custom service: attributes.service = "web-api"
```

**Example Webhook Payload:**
```json
{
  "projectId": "my-project",
  "topicName": "alerts",
  "orderingKey": "region-us",
  "serviceAccount": "alerts@my-project.iam.gserviceaccount.com",
  "message": {
    "data": "{\"title\":\"Database Alert\",\"body\":\"Connection timeout\",\"type\":\"error\",\"severity\":\"ERROR\",\"emoji\":\"❌\"}",
    "messageId": "apprise-1642248600-error",
    "publishTime": "2024-01-15T10:30:00Z"
  },
  "attributes": {
    "notificationType": "error",
    "severity": "ERROR",
    "priority": "HIGH",
    "alertLevel": "CRITICAL",
    "source": "apprise-go",
    "version": "1.9.4-2",
    "timestamp": "2024-01-15T10:30:00Z",
    "topic": "alerts",
    "project": "my-project",
    "environment": "production",
    "service": "database"
  }
}
```

**Integration Notes:**
This service sends webhook requests to your configured endpoint, which should publish messages to Google Cloud Pub/Sub. This approach provides:
- Secure credential management (service account keys stay on your server)
- Custom message transformation and enrichment
- Integration with existing GCP infrastructure
- Advanced Pub/Sub features like dead letter queues and retry policies
- Cost optimization through efficient message batching

**Subscriber Integration:**
Create Pub/Sub subscriptions to process notifications:
```bash
# Create subscription for error messages only
gcloud pubsub subscriptions create error-alerts \
  --topic=alerts \
  --filter='attributes.severity="ERROR"'

# Create subscription for specific service
gcloud pubsub subscriptions create api-events \
  --topic=events \
  --filter='attributes.service="api"'
```

**Example:**
```go
// Send critical system alert with custom routing attributes
app.Add("pubsub://api-key@gateway.googleapis.com/v1/pubsub?project_id=prod-system&topic=critical-alerts&ordering_key=system-health&attr_environment=production&attr_datacenter=us-central1&attr_service=core-api&attr_team=platform")
```

### Ntfy

Simple HTTP push notifications with priority levels, perfect for self-hosted setups and lightweight notifications.

**URL Formats:**
```
# Public ntfy.sh (HTTPS)
ntfys://ntfy.sh/my-topic

# Self-hosted (HTTP)
ntfy://ntfy.example.com:8080/alerts

# With authentication
ntfy://username:password@ntfy.example.com/notifications
ntfys://token@ntfy.sh/alerts

# With priority and tags
ntfy://ntfy.sh/alerts?priority=5&tags=urgent,production

# With advanced features
ntfy://ntfy.sh/alerts?delay=30min&email=admin@example.com&attach=https://example.com/file.pdf
```

**Query Parameters:**
- `priority=1-5` - Message priority (1=min, 3=default, 5=max)
- `tags=tag1,tag2` - Comma-separated tags for message categorization
- `delay=30min` - Delay message delivery (e.g., 30s, 5min, 1h)
- `actions=action1,Label1,url1` - Action buttons (comma-separated)
- `attach=url` - Attachment URL
- `filename=name` - Custom attachment filename
- `click=url` - URL to open when notification is clicked
- `email=address` - Forward notification to email
- `token=string` - Access token (alternative to URL auth)

**Features:**
- Simple HTTP-based push notifications
- Priority levels (1-5) with automatic mapping from notification types
- Tag-based message categorization with emoji support
- Delayed message delivery
- Email forwarding integration
- Attachment support via URLs
- Action buttons for interactive notifications
- Click URLs for notification actions
- Self-hosted and public ntfy.sh support
- Token and username/password authentication

**Priority Mapping:**
- `NotifyTypeInfo` → Priority 3 (Normal)
- `NotifyTypeSuccess` → Priority 3 (Normal)  
- `NotifyTypeWarning` → Priority 4 (High)
- `NotifyTypeError` → Priority 5 (Max)

**Emoji Tags:**
When no custom tags are provided, automatic emoji tags are added:
- ✅ `white_check_mark` for success notifications
- ⚠️ `warning` for warning notifications  
- 🚨 `rotating_light` for error notifications
- ℹ️ `information_source` for info notifications

**Example:**
```go
// Send high-priority alert with custom tags to self-hosted ntfy
app.Add("ntfy://token@ntfy.company.com/alerts?priority=4&tags=production,database&email=oncall@company.com")
```

## Configuration Files

### YAML Format

```yaml
version: 1
urls:
  - url: discord://webhook_id/webhook_token
    tag:
      - team
      - alerts
  - url: mailto://user:pass@smtp.gmail.com/admin@company.com
    tag:
      - admin
  - url: slack://TokenA/TokenB/TokenC/general
    tag:
      - team
```

### Text Format

```
# Team notifications
discord://webhook_id/webhook_token [team,alerts]

# Admin email
mailto://user:pass@smtp.gmail.com/admin@company.com [admin]

# Slack channel
slack://TokenA/TokenB/TokenC/general [team]
```

## Command Line Usage

```bash
# Send simple notification
apprise-cli -t "Hello" -b "World" discord://webhook_id/webhook_token

# Send from config file
echo "Server is down!" | apprise-cli -t "Alert" -c config.yaml

# Send to multiple services with tags
apprise-cli -t "Deploy Success" -b "Version 1.2.3 deployed" --tag production

# Send with different notification types
apprise-cli -t "Error" -b "Database connection failed" -n error

# Send with custom format
apprise-cli -t "Report" -b "<b>Status:</b> OK" --format html
```

## Notification Types

All services support different notification types with appropriate styling:

- `NotifyTypeInfo` (default) - Blue/info styling
- `NotifyTypeSuccess` - Green styling with ✅ emoji  
- `NotifyTypeWarning` - Yellow styling with ⚠️ emoji
- `NotifyTypeError` - Red styling with ❌ emoji

## Error Handling

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token")
app.Add("slack://invalid_tokens")  // This will fail

responses := app.Notify("Test", "Message", apprise.NotifyTypeInfo)

for i, response := range responses {
    if response.Success {
        fmt.Printf("✓ Service %d: Success\n", i+1)
    } else {
        fmt.Printf("✗ Service %d: %v\n", i+1, response.Error)
    }
}
```

## Advanced Features

### Tags

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token", "alerts", "team")
app.Add("mailto://admin@company.com", "admin")

// Send to all services
app.Notify("General", "Message for everyone", apprise.NotifyTypeInfo)

// Send only to admin services  
app.Notify("Admin", "Admin only message", apprise.NotifyTypeWarning,
    apprise.WithTags("admin"))
```

### Custom Timeout

```go
app := apprise.New()
app.SetTimeout(60 * time.Second)  // 60 second timeout
```

### Body Formats

```go
app.Notify("Title", "**Bold** and _italic_ text", apprise.NotifyTypeInfo,
    apprise.WithBodyFormat("markdown"))

app.Notify("Title", "<b>Bold</b> and <i>italic</i> text", apprise.NotifyTypeInfo,
    apprise.WithBodyFormat("html"))
```

### Timezone Support

Control the timezone used for timestamps in notifications. This is useful for ensuring notifications display times in the appropriate timezone for your team or application.

```go
// Send notification with UTC timezone
app.Notify("Server Alert", "CPU usage high", apprise.NotifyTypeWarning,
    apprise.WithTimezone("UTC"))

// Send notification with specific timezone
app.Notify("Meeting Reminder", "Team standup in 15 minutes", apprise.NotifyTypeInfo,
    apprise.WithTimezone("America/New_York"))

// Multiple timezone examples
app.Notify("Title", "Message", apprise.NotifyTypeInfo,
    apprise.WithTimezone("Europe/London"))

app.Notify("Title", "Message", apprise.NotifyTypeInfo,
    apprise.WithTimezone("Asia/Tokyo"))

// Invalid timezone falls back to system local time
app.Notify("Title", "Message", apprise.NotifyTypeInfo,
    apprise.WithTimezone("Invalid/Timezone"))  // Uses time.Local
```

**Supported Timezone Names:**
- IANA timezone database names (e.g., "America/New_York", "Europe/London", "Asia/Tokyo")
- "UTC" for Coordinated Universal Time
- "" (empty string) or invalid names fall back to system local time

**Services Using Timestamps:**
- **Discord**: Uses RFC3339 format (ISO 8601) in embed timestamps
- **Slack**: Uses Unix timestamps in message attachments
- **Other services**: May use timestamps in footers or metadata

**Example with Multiple Services:**
```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token")
app.Add("slack://TokenA/TokenB/TokenC/general")

// All services receive notifications with UTC timestamps
app.Notify("Deploy Complete", "Version 1.2.3 deployed successfully",
    apprise.NotifyTypeSuccess,
    apprise.WithTimezone("UTC"))
```

## Security Best Practices

1. **Never commit tokens to source code** - Use environment variables or config files
2. **Use HTTPS URLs** when possible (`webhooks://`, `mailtos://`, etc.)
3. **Validate webhook URLs** before adding them to prevent SSRF attacks
4. **Use strong passwords** for SMTP authentication
5. **Limit token permissions** to minimum required scope

## Attachment Support

Apprise Go provides comprehensive attachment support for services that support file uploads.

### Basic Attachment Usage

```go
app := apprise.New()
app.Add("discord://webhook_id/webhook_token") // Supports attachments

// Add file attachment
err := app.AddAttachment("/path/to/file.pdf")
if err != nil {
    log.Fatal(err)
}

// Add attachment with custom name
err = app.AddAttachment("/path/to/file.txt", "custom_name.txt")

// Add attachment from URL
err = app.AddAttachment("https://example.com/image.png")

// Add attachment from raw data
data := []byte("Hello, World!")
err = app.AddAttachmentData(data, "hello.txt", "text/plain")

// Send notification with attachments
app.Notify("Title", "Message with attachments", apprise.NotifyTypeInfo)
```

### Attachment Types

**File Attachments:**
```go
// Local file
app.AddAttachment("/path/to/document.pdf")
app.AddAttachment("./relative/path/image.jpg", "custom_name.jpg")
```

**HTTP Attachments:**
```go
// Remote file via HTTP/HTTPS
app.AddAttachment("https://example.com/file.pdf")
app.AddAttachment("http://example.com/image.png", "screenshot.png")
```

**Memory Attachments:**
```go
// Raw data
data := []byte("File content here")
app.AddAttachmentData(data, "filename.txt", "text/plain")

// Data URL (base64 encoded)
app.AddAttachment("data:text/plain;base64,SGVsbG8gV29ybGQ=")
```

### Advanced Attachment Management

```go
app := apprise.New()

// Get attachment manager for advanced operations
mgr := app.GetAttachmentManager()

// Set maximum attachment size (100MB)
mgr.SetMaxSize(100 * 1024 * 1024)

// Set timeout for HTTP attachments
mgr.SetTimeout(60 * time.Second)

// Add multiple attachments
files := []string{
    "/path/to/report.pdf",
    "https://example.com/chart.png",
    "/path/to/data.csv",
}

for _, file := range files {
    if err := app.AddAttachment(file); err != nil {
        log.Printf("Failed to add %s: %v", file, err)
    }
}

// Check attachment info
fmt.Printf("Total attachments: %d\n", app.AttachmentCount())
for _, attachment := range app.GetAttachments() {
    fmt.Printf("- %s (%s, %d bytes)\n", 
        attachment.GetName(), 
        attachment.GetMimeType(), 
        attachment.GetSize())
}

// Send notification
app.Notify("Report", "Please see attached files", apprise.NotifyTypeInfo)

// Clear attachments for next notification
app.ClearAttachments()
```

### Service-Specific Attachment Support

| Service | Attachment Support | Notes |
|---------|-------------------|-------|
| Discord | ✅ Full | Images, documents, up to 8MB |
| Slack | ✅ Full | All file types, size limits apply |
| Telegram | ✅ Full | Photos, documents, audio, video |
| Email (SMTP) | 🚧 Planned | MIME multipart support |
| Matrix | ✅ Full | Media uploads via Matrix API |
| Opsgenie | ❌ Not supported | Alert API doesn't support attachments |
| Pushbullet | ✅ Full | File uploads via API |
| Microsoft Teams | 🚧 Planned | Adaptive cards with attachments |
| Mattermost | ✅ Full | File uploads via API v4 |
| Pushover | ✅ Images | Image attachments only |
| Webhook/JSON | ❌ Not supported | Use base64 encoding in payload |
| Twilio SMS | ❌ Not supported | SMS doesn't support attachments |
| Desktop Notifications | ❌ Not supported | Images via image parameter only |
| Gotify | ❌ Not supported | Text-only notifications |
| Ntfy | ✅ URLs | Attachment support via URLs only |

### Attachment Security

```go
mgr := app.GetAttachmentManager()

// Limit attachment size
mgr.SetMaxSize(10 * 1024 * 1024) // 10MB limit

// Set timeout for HTTP downloads
mgr.SetTimeout(30 * time.Second)

// Validate attachments before sending
for _, attachment := range app.GetAttachments() {
    if !attachment.Exists() {
        log.Printf("Warning: Attachment %s is not accessible", attachment.GetName())
    }
    
    // Check file type
    mimeType := attachment.GetMimeType()
    if !isAllowedMimeType(mimeType) {
        log.Printf("Warning: Attachment %s has restricted type %s", 
            attachment.GetName(), mimeType)
    }
}
```

### Error Handling

```go
// Attachment operations can fail
if err := app.AddAttachment("/nonexistent/file.txt"); err != nil {
    log.Printf("Attachment error: %v", err)
}

// Check attachment availability
for _, attachment := range app.GetAttachments() {
    if !attachment.Exists() {
        log.Printf("Attachment %s is not available", attachment.GetName())
    }
}

// Services may reject attachments
responses := app.Notify("Title", "Message", apprise.NotifyTypeInfo)
for _, response := range responses {
    if !response.Success && response.Error != nil {
        log.Printf("Service %s failed: %v", response.ServiceID, response.Error)
    }
}
```

For more examples, see the `examples/` directory in the repository.