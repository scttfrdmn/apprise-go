# Apprise-Go Monitoring Stack

Comprehensive monitoring and observability for Apprise-Go using Prometheus and Grafana.

## Overview

This monitoring stack provides:

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and alerting dashboards
- **Alert Rules** - Proactive monitoring and SLA tracking
- **Pre-built Dashboards** - Ready-to-use monitoring views

## Quick Start

### Docker Compose

```bash
# Start the full monitoring stack
cd monitoring
docker-compose up -d

# Access services
echo "Grafana: http://localhost:3000 (admin/admin123)"
echo "Prometheus: http://localhost:9090"
echo "Apprise API: http://localhost:8080"
```

### Kubernetes

```bash
# Deploy monitoring components
kubectl apply -f ../k8s/monitoring.yaml

# Port forward to access services
kubectl port-forward -n apprise-go service/grafana 3000:3000
kubectl port-forward -n apprise-go service/prometheus 9090:9090
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│  Apprise-Go     │───▶│   Prometheus    │───▶│    Grafana      │
│  /metrics       │    │   (Storage)     │    │  (Dashboards)   │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         │                        ▼                        │
         │              ┌─────────────────┐                │
         │              │                 │                │
         └─────────────▶│  Alert Manager  │◀───────────────┘
                        │  (Notifications)│
                        │                 │
                        └─────────────────┘
```

## Available Metrics

### Notification Metrics
- `apprise_notifications_total` - Total notifications sent by service and status
- `apprise_notifications_failed_total` - Failed notifications by service and error type
- `apprise_notification_duration_seconds` - Notification processing time
- `apprise_notification_batch_size` - Size of notification batches

### HTTP API Metrics
- `apprise_http_requests_total` - Total HTTP requests by method and endpoint
- `apprise_http_requests_failed_total` - Failed HTTP requests
- `apprise_http_request_duration_seconds` - HTTP request latency

### System Metrics
- `apprise_active_connections` - Number of active connections
- `apprise_services_configured_total` - Number of configured notification services
- `apprise_notification_queue_size` - Size of notification queue
- `apprise_memory_usage_bytes` - Memory usage in bytes
- `apprise_goroutines_total` - Number of active goroutines

## Dashboards

### 1. Overview Dashboard
- **File**: `grafana/dashboards/apprise-overview.json`
- **Purpose**: High-level system health and performance
- **Metrics**: Success rates, response times, service counts, memory usage

### 2. Service Metrics Dashboard
- **File**: `grafana/dashboards/apprise-services.json`
- **Purpose**: Per-service detailed metrics
- **Features**: Service comparison, error rates, response time percentiles

### 3. Infrastructure Dashboard
- **File**: `grafana/dashboards/apprise-infrastructure.json`
- **Purpose**: System resource monitoring
- **Metrics**: Memory, goroutines, HTTP metrics, queue sizes

### 4. Alerts & SLA Dashboard
- **File**: `grafana/dashboards/apprise-alerts.json`
- **Purpose**: SLA tracking and alerting overview
- **Features**: Success rate tracking, error analysis, health summary

## Alert Rules

### Critical Alerts
- **AppriseServiceDown** - Service is unresponsive (30s)
- **AppriseHighErrorRate** - Error rate > 5% (2m)
- **AppriseSuccessRateLow** - Success rate < 95% (5m)

### Warning Alerts
- **AppriseHighResponseTime** - 95th percentile > 500ms (5m)
- **AppriseHighMemoryUsage** - Memory usage > 1GB (10m)
- **AppriseHighGoroutineCount** - Goroutines > 1000 (5m)
- **AppriseQueueBacklog** - Queue size > 100 (2m)

### Service-Specific Alerts
- **AppriseServiceHighErrorRate** - Per-service error rate > 10% (5m)
- **AppriseNoRecentNotifications** - No activity for 30m (info)

## Configuration

### Prometheus Configuration
```yaml
# prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'apprise-go'
    static_configs:
      - targets: ['apprise-api:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### Grafana Datasources
```yaml
# grafana/datasources/prometheus.yaml
datasources:
  - name: Prometheus
    type: prometheus
    url: http://prometheus:9090
    isDefault: true
```

## Customization

### Adding Custom Metrics

1. **Extend MetricsManager** in `apprise/metrics.go`:
```go
// Add new metric
customCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Name:      "custom_metric_total",
        Help:      "Custom metric description",
    },
    []string{"label1", "label2"},
)
```

2. **Record Metrics** in your code:
```go
app.GetMetrics().RecordCustomMetric("value1", "value2")
```

3. **Create Dashboard Panels** using the new metric:
```json
{
  "targets": [{
    "expr": "rate(apprise_custom_metric_total[5m])",
    "legendFormat": "{{label1}} - {{label2}}"
  }]
}
```

### Custom Alert Rules

Create new rules in `prometheus/apprise-alerts.yml`:

```yaml
- alert: CustomAlert
  expr: apprise_custom_metric_total > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Custom alert triggered"
    description: "Custom metric value: {{ $value }}"
```

## Troubleshooting

### Common Issues

**Metrics not appearing in Grafana:**
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Verify metric endpoint
curl http://localhost:8080/metrics
```

**High memory usage:**
```bash
# Check Prometheus retention settings
# Adjust in prometheus.yml:
storage:
  tsdb:
    retention.time: 7d  # Reduce retention
    retention.size: 10GB
```

**Dashboard import issues:**
1. Copy dashboard JSON to Grafana UI
2. Or use provisioning in `grafana/dashboards/`
3. Verify datasource name matches

### Performance Tuning

**Prometheus Settings:**
```yaml
# Reduce scrape frequency for large deployments
global:
  scrape_interval: 30s     # Default: 15s
  evaluation_interval: 30s # Default: 15s

# Optimize storage
storage:
  tsdb:
    retention.time: 15d
    retention.size: 50GB
```

**Grafana Settings:**
```yaml
# Optimize query performance
query_timeout: 60s
max_concurrent_queries: 20
```

## Monitoring Best Practices

### 1. SLA Definition
- **Success Rate**: > 95%
- **Response Time**: 95th percentile < 500ms
- **Availability**: > 99.9%

### 2. Alert Fatigue Prevention
- Use appropriate thresholds
- Implement alert suppression during maintenance
- Group related alerts
- Set up escalation policies

### 3. Dashboard Design
- Use consistent time ranges
- Group related metrics
- Include context and documentation
- Optimize for mobile viewing

### 4. Metric Collection
- Use appropriate metric types (counter, histogram, gauge)
- Add meaningful labels
- Avoid high-cardinality labels
- Document metric purposes

## Advanced Features

### Multi-Environment Setup

```yaml
# Different configs per environment
environments:
  production:
    retention: 30d
    scrape_interval: 10s
  staging:
    retention: 7d
    scrape_interval: 30s
```

### Federated Prometheus

```yaml
# prometheus-federation.yml
scrape_configs:
  - job_name: 'federate'
    scrape_interval: 15s
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{job=~"apprise.*"}'
    static_configs:
      - targets: ['prometheus-1:9090', 'prometheus-2:9090']
```

### External Alert Integration

Connect to external systems:
- **Slack**: Webhook notifications
- **PagerDuty**: Incident management  
- **Email**: SMTP alerting
- **SMS**: Twilio integration

## Support and Documentation

- **Prometheus**: https://prometheus.io/docs/
- **Grafana**: https://grafana.com/docs/
- **Metrics Best Practices**: https://prometheus.io/docs/practices/naming/
- **Dashboard Examples**: https://grafana.com/grafana/dashboards/

For Apprise-Go specific questions, see the main project documentation.