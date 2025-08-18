# Apprise-Go Enterprise Deployment Guide

Complete enterprise deployment documentation for production environments.

## Deployment Options

| Method | Complexity | Scalability | Use Case |
|--------|------------|-------------|----------|
| [Docker Compose](#docker-compose) | Low | Small-Medium | Development, Small Teams |
| [Kubernetes](#kubernetes) | Medium | High | Production, Large Scale |
| [AWS EKS](#aws-eks) | Medium | Very High | Cloud-Native AWS |
| [Google GKE](#google-gke) | Medium | Very High | Cloud-Native GCP |
| [Azure AKS](#azure-aks) | Medium | Very High | Cloud-Native Azure |
| [Bare Metal](#bare-metal) | High | Variable | On-Premises |

## Quick Start

### Production-Ready Docker Compose
```bash
# Clone and deploy
git clone https://github.com/username/apprise-go
cd apprise-go
docker-compose up -d

# Verify deployment
curl http://localhost:8080/health
```

### Production Kubernetes
```bash
# Apply manifests
kubectl apply -f k8s/

# Check status
kubectl get pods -n apprise-go
```

## Deployment Architectures

### Small Scale (< 1000 notifications/day)
```
┌─────────────────┐
│   Apprise-Go    │
│   Single Pod    │
│                 │
└─────────────────┘
```

### Medium Scale (< 100K notifications/day)  
```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ Apprise-Go  │   │ Apprise-Go  │   │ Apprise-Go  │
│   Pod 1     │   │   Pod 2     │   │   Pod 3     │
└─────────────┘   └─────────────┘   └─────────────┘
         │                 │                 │
         └─────────────────┼─────────────────┘
                           │
                  ┌─────────────────┐
                  │  Load Balancer  │
                  └─────────────────┘
```

### Large Scale (> 1M notifications/day)
```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └─────────┬───────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼───────┐     ┌───────▼───────┐     ┌───────▼───────┐
│  API Gateway  │     │  API Gateway  │     │  API Gateway  │
│   Region 1    │     │   Region 2    │     │   Region 3    │
└───────┬───────┘     └───────┬───────┘     └───────┬───────┘
        │                     │                     │
┌───────▼───────┐     ┌───────▼───────┐     ┌───────▼───────┐
│ Apprise-Go    │     │ Apprise-Go    │     │ Apprise-Go    │
│   Cluster     │     │   Cluster     │     │   Cluster     │
│ (Auto-Scale)  │     │ (Auto-Scale)  │     │ (Auto-Scale)  │
└───────────────┘     └───────────────┘     └───────────────┘
```

## Docker Compose Deployment

### Production Configuration
```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  apprise-api:
    image: apprise-go:${VERSION:-latest}
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - APPRISE_PORT=8080
      - APPRISE_HOST=0.0.0.0
      - APPRISE_LOG_LEVEL=info
      - APPRISE_MAX_REQUESTS_PER_MINUTE=1000
    volumes:
      - ./configs:/app/configs:ro
      - apprise-data:/app/data
      - /etc/ssl/certs:/etc/ssl/certs:ro
    networks:
      - apprise-network
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    networks:
      - apprise-network
    depends_on:
      - apprise-api

volumes:
  apprise-data:
    driver: local

networks:
  apprise-network:
    driver: bridge
```

### SSL/TLS Configuration
```nginx
# nginx/nginx.conf
server {
    listen 443 ssl http2;
    server_name apprise.company.com;
    
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    
    location / {
        proxy_pass http://apprise-api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Kubernetes Deployment

### Prerequisites
- Kubernetes 1.19+
- kubectl configured
- 4GB+ available memory
- Persistent storage provisioner

### Quick Deploy
```bash
# Create namespace
kubectl create namespace apprise-go

# Deploy with Kustomize
kubectl apply -k k8s/

# Or deploy individual manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### Production Hardening
```yaml
# k8s/production-patches.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apprise-go-api
spec:
  replicas: 3
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
      containers:
      - name: apprise-go
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi
```

### Auto-scaling Configuration
```yaml
# k8s/hpa-production.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: apprise-go-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: apprise-go-api
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Cloud Platform Deployments

Each cloud platform guide includes:
- Platform-specific setup
- Managed service integration
- Auto-scaling configuration
- Security best practices
- Cost optimization
- Monitoring setup

📚 **Detailed Guides:**
- [AWS EKS Deployment](aws-eks.md)
- [Google GKE Deployment](google-gke.md) 
- [Azure AKS Deployment](azure-aks.md)

## Configuration Management

### Environment-Specific Configs

#### Development
```yaml
# configs/development.yaml
server:
  host: "0.0.0.0"
  port: 8080
logging:
  level: debug
rate_limiting:
  enabled: false
authentication:
  enabled: false
```

#### Production
```yaml
# configs/production.yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
logging:
  level: info
  format: json
rate_limiting:
  enabled: true
  requests_per_minute: 1000
authentication:
  enabled: true
  jwt_secret: "${JWT_SECRET}"
```

### Secret Management

#### Kubernetes Secrets
```bash
# Create secrets
kubectl create secret generic apprise-secrets \
  --from-literal=jwt-secret="$(openssl rand -base64 32)" \
  --from-literal=webhook-secret="$(openssl rand -base64 16)"

# Use in deployment
env:
- name: JWT_SECRET
  valueFrom:
    secretKeyRef:
      name: apprise-secrets
      key: jwt-secret
```

#### HashiCorp Vault Integration
```bash
# Install Vault injector
helm install vault hashicorp/vault

# Configure secret injection
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "apprise"
  vault.hashicorp.com/agent-inject-secret-config: "secret/apprise/config"
```

## Security Hardening

### Network Security
```yaml
# Network policies
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: apprise-network-policy
spec:
  podSelector:
    matchLabels:
      app: apprise-go
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443
```

### Pod Security Standards
```yaml
# Pod security context
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: apprise-go
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      readOnlyRootFilesystem: true
```

## Monitoring & Observability

### Prometheus Integration
```yaml
# ServiceMonitor for Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: apprise-go
spec:
  selector:
    matchLabels:
      app: apprise-go
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### Grafana Dashboards
```bash
# Import pre-built dashboards
kubectl apply -f monitoring/grafana/dashboards/
```

### Centralized Logging
```yaml
# Fluentd configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
data:
  fluent.conf: |
    <match apprise.go.**>
      @type elasticsearch
      host elasticsearch
      port 9200
      index_name apprise-go
    </match>
```

## Backup & Disaster Recovery

### Database Backups
```bash
#!/bin/bash
# backup-apprise.sh
kubectl exec -n apprise-go deployment/apprise-go-api -- \
  sqlite3 /app/data/apprise.db ".backup /tmp/backup.db"
kubectl cp apprise-go/apprise-go-api-xxx:/tmp/backup.db ./backups/apprise-$(date +%Y%m%d).db
```

### Configuration Backups
```bash
# Backup configurations
kubectl get configmap -n apprise-go -o yaml > configs-backup.yaml
kubectl get secret -n apprise-go -o yaml > secrets-backup.yaml
```

### Disaster Recovery Plan
1. **Backup Strategy**: Automated daily backups
2. **Recovery Testing**: Monthly recovery drills
3. **Multi-Region**: Cross-region replication
4. **Documentation**: Step-by-step recovery procedures

## Performance Optimization

### Resource Tuning
```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m      # Adjust based on load
    memory: 512Mi  # Monitor actual usage
```

### JVM Tuning (if applicable)
```bash
# Go-specific optimizations
export GOGC=100          # GC target percentage
export GOMEMLIMIT=512MiB # Memory limit
```

### Connection Pool Optimization
```go
// HTTP client tuning
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableKeepAlives:   false,
    },
}
```

## Load Testing & Capacity Planning

### Benchmark Results
```bash
# Run performance tests
cd benchmarks
./run_benchmarks.sh -t all

# Capacity planning
./run_benchmarks.sh -config stress -duration 10m
```

### Scaling Guidelines
| Metric | Small | Medium | Large | Enterprise |
|--------|-------|--------|-------|------------|
| Notifications/day | <1K | 1K-100K | 100K-1M | >1M |
| Replicas | 1-2 | 2-5 | 5-20 | 20+ |
| CPU (cores) | 0.5 | 1-2 | 2-8 | 8+ |
| Memory (GB) | 0.5 | 1-2 | 2-8 | 8+ |

## Troubleshooting

### Common Issues

**Pod CrashLoopBackOff:**
```bash
# Check logs
kubectl logs -n apprise-go deployment/apprise-go-api --previous

# Check resource constraints
kubectl describe pod -n apprise-go -l app=apprise-go
```

**High Memory Usage:**
```bash
# Get memory profile
kubectl port-forward -n apprise-go service/apprise-go-service 6060:8080
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Database Connection Issues:**
```bash
# Check database connectivity
kubectl exec -n apprise-go deployment/apprise-go-api -- \
  sqlite3 /app/data/apprise.db ".tables"
```

### Health Checks
```bash
# Application health
curl -f http://apprise.company.com/health

# Metrics endpoint
curl http://apprise.company.com/metrics

# Kubernetes health
kubectl get pods -n apprise-go
kubectl get events -n apprise-go --sort-by='.lastTimestamp'
```

## Maintenance Procedures

### Rolling Updates
```bash
# Update image
kubectl set image -n apprise-go deployment/apprise-go-api \
  apprise-go=apprise-go:v1.2.0

# Monitor rollout
kubectl rollout status -n apprise-go deployment/apprise-go-api

# Rollback if needed
kubectl rollout undo -n apprise-go deployment/apprise-go-api
```

### Certificate Rotation
```bash
# Update TLS certificates
kubectl create secret tls apprise-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Log Rotation
```bash
# Configure log rotation
apiVersion: v1
kind: ConfigMap
metadata:
  name: logrotate-config
data:
  logrotate.conf: |
    /app/logs/*.log {
        daily
        rotate 7
        compress
        missingok
    }
```

## Support & Documentation

### Getting Help
- **Issues**: GitHub Issues for bug reports
- **Discussions**: GitHub Discussions for questions
- **Documentation**: Complete guides in `/docs`
- **Examples**: Sample configurations in `/examples`

### Professional Support
For enterprise support:
- 24/7 monitoring and alerting
- Custom integration development
- Performance optimization consulting
- Training and onboarding

---

📚 **Next Steps**: Choose your deployment method and follow the detailed guides in this directory.