# Kubernetes Deployment Guide

This directory contains Kubernetes manifests and deployment tools for Apprise-Go.

## Quick Start

### Prerequisites

- Kubernetes cluster (1.19+)
- kubectl configured
- Docker (for building images)
- 10GB+ available storage

### Deploy

```bash
# Build and deploy with defaults
./k8s/deploy.sh

# Deploy to specific namespace
./k8s/deploy.sh -n production

# Deploy specific image version
./k8s/deploy.sh -i apprise-go:v1.2.0

# Dry run to preview changes
./k8s/deploy.sh -d
```

## Architecture

### Components

- **API Server**: Main application with REST API
- **Prometheus**: Metrics collection
- **Grafana**: Monitoring dashboards
- **Persistent Storage**: SQLite database and logs

### Scaling

- **HPA**: Auto-scales 3-20 pods based on CPU/memory/requests
- **PDB**: Ensures minimum 2 pods during disruptions
- **Resource Limits**: CPU: 500m, Memory: 512Mi per pod

### Security

- **NetworkPolicy**: Restricts pod-to-pod communication
- **RBAC**: Minimal service account permissions
- **Secrets**: Encrypted storage for credentials
- **Non-root**: Containers run as non-root user

## Manifest Files

| File | Description |
|------|-------------|
| `namespace.yaml` | Namespace and ServiceAccount |
| `configmap.yaml` | Application and Prometheus configuration |
| `secret.yaml` | Secrets for JWT, passwords, service credentials |
| `deployment.yaml` | Main application deployment |
| `service.yaml` | Service definitions |
| `ingress.yaml` | External access configuration |
| `hpa.yaml` | Horizontal Pod Autoscaler |
| `pvc.yaml` | Persistent Volume Claims |
| `monitoring.yaml` | Prometheus and Grafana components |
| `networkpolicy.yaml` | Network security policies |
| `poddisruptionbudget.yaml` | Disruption budget for availability |
| `kustomization.yaml` | Kustomize configuration |

## Configuration

### Environment Variables

```yaml
# Application
APPRISE_CONFIG_FILE: "/etc/apprise/app.yaml"
JWT_SECRET: "<from secret>"
ADMIN_PASSWORD: "<from secret>"

# Service Credentials (examples)
DISCORD_WEBHOOK: "https://discord.com/api/webhooks/..."
SLACK_TOKEN: "xoxb-your-slack-bot-token"
TWILIO_ACCOUNT_SID: "your-twilio-account-sid"
SENDGRID_API_KEY: "your-sendgrid-api-key"
```

### Secrets Management

1. **Update secrets before deployment:**
```bash
# Edit k8s/secret.yaml
# Replace base64 encoded values with your credentials
```

2. **Runtime secret updates:**
```bash
# Update JWT secret
kubectl create secret generic apprise-go-secrets \
  --namespace=apprise-go \
  --from-literal=JWT_SECRET="$(openssl rand -base64 32)" \
  --dry-run=client -o yaml | kubectl apply -f -

# Update service credentials
kubectl create secret generic apprise-go-service-secrets \
  --namespace=apprise-go \
  --from-literal=DISCORD_WEBHOOK="https://..." \
  --from-literal=SLACK_TOKEN="xoxb-..." \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Storage Classes

Update `storageClassName` in PVC manifests:

```yaml
# For AWS EKS
storageClassName: gp2

# For Google GKE
storageClassName: standard

# For Azure AKS
storageClassName: default
```

### Ingress Configuration

Update ingress hosts in `ingress.yaml`:

```yaml
spec:
  tls:
  - hosts:
    - apprise-go.your-domain.com
    - api.apprise-go.your-domain.com
  rules:
  - host: apprise-go.your-domain.com
    # ...
```

## Monitoring

### Prometheus Metrics

Access Prometheus at: `http://prometheus.apprise-go.your-domain.com`

Available metrics:
- `apprise_notifications_total` - Total notifications sent
- `apprise_notifications_failed_total` - Failed notifications
- `apprise_response_duration_seconds` - Response times
- `apprise_active_connections` - Active connections

### Grafana Dashboards

Access Grafana at: `http://grafana.apprise-go.your-domain.com`

Default credentials: `admin/admin123` (change after first login)

Pre-configured dashboards:
- **Apprise Overview**: High-level metrics and health
- **Service Performance**: Per-service notification metrics  
- **Infrastructure**: Pod resource usage and scaling

## Troubleshooting

### Common Issues

**Pods stuck in Pending**
```bash
# Check resource availability
kubectl describe node
kubectl get pv,pvc -n apprise-go

# Check pod events
kubectl describe pod -l app=apprise-go -n apprise-go
```

**Service unavailable**
```bash
# Check pod status
kubectl get pods -n apprise-go
kubectl logs -l app=apprise-go -n apprise-go

# Test internal connectivity
kubectl exec -it deployment/apprise-go-api -n apprise-go -- curl localhost:8080/health
```

**Ingress not working**
```bash
# Check ingress controller
kubectl get pods -n ingress-nginx

# Verify ingress configuration
kubectl describe ingress apprise-go-ingress -n apprise-go

# Check cert-manager (if using TLS)
kubectl get certificates -n apprise-go
```

### Debug Commands

```bash
# Pod logs
kubectl logs -f deployment/apprise-go-api -n apprise-go

# Execute into pod
kubectl exec -it deployment/apprise-go-api -n apprise-go -- /bin/sh

# Port forward for local testing
kubectl port-forward service/apprise-go-service -n apprise-go 8080:8080

# Check resource usage
kubectl top pods -n apprise-go
kubectl top nodes
```

### Performance Tuning

**High CPU/Memory usage:**
1. Increase resource limits in `deployment.yaml`
2. Scale up replicas: `kubectl scale deployment apprise-go-api --replicas=5 -n apprise-go`
3. Adjust HPA thresholds in `hpa.yaml`

**Slow response times:**
1. Enable connection pooling
2. Increase `read_timeout` and `write_timeout`
3. Add Redis cache for frequent requests

**Database lock issues:**
1. Switch to PostgreSQL for multi-pod deployments
2. Use external database service
3. Implement connection pooling

## Maintenance

### Updates

```bash
# Update image version
kubectl set image deployment/apprise-go-api apprise-go=apprise-go:v1.2.0 -n apprise-go

# Rolling restart
kubectl rollout restart deployment/apprise-go-api -n apprise-go

# Check rollout status
kubectl rollout status deployment/apprise-go-api -n apprise-go
```

### Backup

```bash
# Backup persistent data
kubectl exec -it deployment/apprise-go-api -n apprise-go -- tar -czf - /app/data | gzip > apprise-backup.tar.gz

# Backup configuration
kubectl get configmap,secret -n apprise-go -o yaml > apprise-config-backup.yaml
```

### Cleanup

```bash
# Remove all resources
kubectl delete namespace apprise-go

# Or use the deployment script
./k8s/deploy.sh --cleanup
```

## Production Considerations

### Security Hardening

1. **Use non-root containers** ✓ (already configured)
2. **Enable Pod Security Standards**
3. **Implement network segmentation** ✓ (NetworkPolicy configured)
4. **Regular security scanning**
5. **Rotate secrets regularly**

### High Availability

1. **Multi-zone deployment**: Update node affinity rules
2. **External database**: Replace SQLite with PostgreSQL/MySQL
3. **Load balancer**: Configure external load balancer
4. **Backup strategy**: Implement automated backups

### Resource Planning

| Component | CPU Request | CPU Limit | Memory Request | Memory Limit |
|-----------|-------------|-----------|----------------|--------------|
| API Pod | 100m | 500m | 128Mi | 512Mi |
| Prometheus | 100m | 500m | 256Mi | 1Gi |
| Grafana | 100m | 500m | 128Mi | 512Mi |

**Storage Requirements:**
- Application data: 5GB per deployment
- Prometheus metrics: 10GB (15 day retention)
- Grafana data: 5GB
- **Total**: ~20GB minimum

### Cost Optimization

1. **Right-size resources** based on actual usage
2. **Use spot instances** for non-critical workloads
3. **Enable cluster autoscaling**
4. **Monitor unused resources**
5. **Implement resource quotas**