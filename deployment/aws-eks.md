# AWS EKS Deployment Guide

Deploy Apprise-Go on Amazon Elastic Kubernetes Service (EKS) with enterprise-grade scalability and security.

## Architecture Overview

```
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│   Application   │   │   Network       │   │   Data          │
│   Load Balancer │   │   Load Balancer │   │   Storage       │
└─────────────────┘   └─────────────────┘   └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│      EKS        │   │      VPC        │   │      EBS        │
│   Kubernetes    │   │   Multi-AZ      │   │   Persistent    │
│    Cluster      │   │   Subnets       │   │    Volumes      │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

## Prerequisites

### Required Tools
```bash
# Install AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip && sudo ./aws/install

# Install eksctl
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install kubectl /usr/local/bin/

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### AWS Configuration
```bash
# Configure AWS credentials
aws configure
# AWS Access Key ID: YOUR_ACCESS_KEY
# AWS Secret Access Key: YOUR_SECRET_KEY
# Default region: us-west-2
# Default output format: json

# Verify configuration
aws sts get-caller-identity
```

## Cluster Setup

### Create EKS Cluster
```yaml
# cluster-config.yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: apprise-go-cluster
  region: us-west-2
  version: "1.28"

iam:
  withOIDC: true

managedNodeGroups:
  - name: apprise-workers
    instanceType: t3.medium
    minSize: 2
    maxSize: 10
    desiredCapacity: 3
    volumeSize: 20
    volumeType: gp3
    amiFamily: AmazonLinux2
    iam:
      attachPolicyARNs:
        - arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy
        - arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy
        - arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly
        - arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy
    tags:
      Environment: production
      Application: apprise-go

addons:
  - name: vpc-cni
    version: latest
  - name: coredns
    version: latest
  - name: kube-proxy
    version: latest
  - name: aws-ebs-csi-driver
    version: latest

cloudWatch:
  clusterLogging:
    enableTypes: ["*"]
```

```bash
# Create cluster
eksctl create cluster -f cluster-config.yaml

# Update kubeconfig
aws eks update-kubeconfig --region us-west-2 --name apprise-go-cluster

# Verify cluster
kubectl get nodes
kubectl get pods --all-namespaces
```

### Install Cluster Add-ons

#### AWS Load Balancer Controller
```bash
# Create IAM role for service account
eksctl create iamserviceaccount \
  --cluster=apprise-go-cluster \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --role-name=AmazonEKSLoadBalancerControllerRole \
  --attach-policy-arn=arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess \
  --approve

# Install with Helm
helm repo add eks https://aws.github.io/eks-charts
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=apprise-go-cluster \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller
```

#### EBS CSI Driver
```bash
# Create IAM role for EBS CSI driver
eksctl create iamserviceaccount \
  --cluster=apprise-go-cluster \
  --namespace=kube-system \
  --name=ebs-csi-controller-sa \
  --role-name=AmazonEKS_EBS_CSI_DriverRole \
  --attach-policy-arn=arn:aws:iam::aws:policy/service-role/Amazon_EBS_CSI_DriverPolicy \
  --approve

# Enable EBS CSI add-on
aws eks create-addon \
  --cluster-name apprise-go-cluster \
  --addon-name aws-ebs-csi-driver \
  --service-account-role-arn arn:aws:iam::ACCOUNT-ID:role/AmazonEKS_EBS_CSI_DriverRole
```

#### Cluster Autoscaler
```bash
# Create IAM policy
cat > cluster-autoscaler-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:DescribeLaunchConfigurations",
                "autoscaling:DescribeTags",
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup",
                "ec2:DescribeLaunchTemplateVersions"
            ],
            "Resource": "*"
        }
    ]
}
EOF

aws iam create-policy \
  --policy-name AmazonEKSClusterAutoscalerPolicy \
  --policy-document file://cluster-autoscaler-policy.json

# Create service account
eksctl create iamserviceaccount \
  --cluster=apprise-go-cluster \
  --namespace=kube-system \
  --name=cluster-autoscaler \
  --attach-policy-arn=arn:aws:iam::ACCOUNT-ID:policy/AmazonEKSClusterAutoscalerPolicy \
  --approve

# Deploy cluster autoscaler
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml

# Configure cluster autoscaler
kubectl -n kube-system annotate deployment.apps/cluster-autoscaler \
  cluster-autoscaler.kubernetes.io/safe-to-evict="false"

kubectl -n kube-system edit deployment.apps/cluster-autoscaler
# Add: --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/apprise-go-cluster
```

## Storage Configuration

### EBS Storage Classes
```yaml
# storage-classes.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: apprise-ssd
provisioner: ebs.csi.aws.com
volumeBindingMode: WaitForFirstConsumer
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: apprise-backup
provisioner: ebs.csi.aws.com
volumeBindingMode: WaitForFirstConsumer
parameters:
  type: st1
  encrypted: "true"
allowVolumeExpansion: true
```

### Persistent Volume Configuration
```yaml
# Update k8s/pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: apprise-go-data
  namespace: apprise-go
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: apprise-ssd
  resources:
    requests:
      storage: 20Gi
```

## Application Deployment

### Secrets Management with AWS Secrets Manager
```bash
# Create secrets in AWS Secrets Manager
aws secretsmanager create-secret \
  --name "apprise-go/jwt-secret" \
  --description "JWT secret for Apprise-Go" \
  --secret-string "$(openssl rand -base64 32)"

aws secretsmanager create-secret \
  --name "apprise-go/service-configs" \
  --description "Service configuration secrets" \
  --secret-string '{
    "discord_webhook": "https://discord.com/api/webhooks/...",
    "slack_token": "xoxb-...",
    "twilio_sid": "AC...",
    "twilio_token": "..."
  }'
```

### Install Secrets Store CSI Driver
```bash
# Install Secrets Store CSI driver
helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
helm install csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver \
  --namespace kube-system \
  --set syncSecret.enabled=true

# Install AWS provider
kubectl apply -f https://raw.githubusercontent.com/aws/secrets-store-csi-driver-provider-aws/main/deployment/aws-provider-installer.yaml
```

### Configure Secret Provider
```yaml
# aws-secret-provider.yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: apprise-go-secrets
  namespace: apprise-go
spec:
  provider: aws
  parameters:
    objects: |
      - objectName: "apprise-go/jwt-secret"
        objectType: "secretsmanager"
        jmesPath:
          - path: "."
            objectAlias: "jwt-secret"
      - objectName: "apprise-go/service-configs"
        objectType: "secretsmanager"
        jmesPath:
          - path: "discord_webhook"
            objectAlias: "discord-webhook"
          - path: "slack_token"
            objectAlias: "slack-token"
  secretObjects:
  - secretName: apprise-go-secrets
    type: Opaque
    data:
    - objectName: jwt-secret
      key: jwt-secret
    - objectName: discord-webhook
      key: discord-webhook
    - objectName: slack-token
      key: slack-token
```

### Update Deployment for AWS
```yaml
# k8s/aws-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apprise-go-api
  namespace: apprise-go
spec:
  replicas: 3
  template:
    spec:
      serviceAccountName: apprise-go-service-account
      containers:
      - name: apprise-go
        image: apprise-go:latest
        ports:
        - containerPort: 8080
        env:
        - name: AWS_REGION
          value: us-west-2
        volumeMounts:
        - name: secrets-store
          mountPath: "/mnt/secrets-store"
          readOnly: true
        - name: config
          mountPath: /app/config
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi
      volumes:
      - name: secrets-store
        csi:
          driver: secrets-store.csi.k8s.io
          readOnly: true
          volumeAttributes:
            secretProviderClass: apprise-go-secrets
      - name: config
        configMap:
          name: apprise-go-config
      nodeSelector:
        kubernetes.io/os: linux
```

### Deploy Application
```bash
# Apply AWS-specific configurations
kubectl apply -f k8s/aws-deployment.yaml
kubectl apply -f aws-secret-provider.yaml

# Apply standard manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml

# Verify deployment
kubectl get pods -n apprise-go
kubectl logs -n apprise-go deployment/apprise-go-api
```

## Load Balancing & Ingress

### Application Load Balancer (ALB)
```yaml
# alb-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apprise-go-alb
  namespace: apprise-go
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /health
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS": 443}]'
    alb.ingress.kubernetes.io/ssl-redirect: '443'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-west-2:ACCOUNT:certificate/CERT-ID
    alb.ingress.kubernetes.io/tags: Environment=production,Application=apprise-go
spec:
  rules:
  - host: apprise.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: apprise-go-service
            port:
              number: 8080
```

### Network Load Balancer (NLB) for High Performance
```yaml
# nlb-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: apprise-go-nlb
  namespace: apprise-go
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
  selector:
    app: apprise-go
```

## Monitoring & Observability

### CloudWatch Container Insights
```bash
# Install CloudWatch agent
curl https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/quickstart/cwagent-fluentd-quickstart.yaml | sed "s/{{cluster_name}}/apprise-go-cluster/;s/{{region_name}}/us-west-2/" | kubectl apply -f -
```

### Prometheus & Grafana Setup
```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --values prometheus-values.yaml
```

```yaml
# prometheus-values.yaml
grafana:
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: nlb
  adminPassword: "secure-admin-password"
  
prometheus:
  prometheusSpec:
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: apprise-ssd
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
```

### Application Performance Monitoring (APM)
```yaml
# Deploy AWS X-Ray daemon
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: xray-daemon
  namespace: apprise-go
spec:
  selector:
    matchLabels:
      app: xray-daemon
  template:
    metadata:
      labels:
        app: xray-daemon
    spec:
      serviceAccountName: xray-daemon
      containers:
      - name: xray-daemon
        image: amazon/aws-xray-daemon:latest
        command:
        - /usr/bin/xray
        - -o
        - -n us-west-2
        ports:
        - containerPort: 2000
          protocol: UDP
        resources:
          limits:
            memory: 256Mi
          requests:
            memory: 32Mi
```

## Security Configuration

### IAM Roles and Service Accounts
```bash
# Create IAM role for Apprise-Go service account
eksctl create iamserviceaccount \
  --cluster=apprise-go-cluster \
  --namespace=apprise-go \
  --name=apprise-go-service-account \
  --role-name=AppriseGoServiceRole \
  --attach-policy-arn=arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy \
  --attach-policy-arn=arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess \
  --approve
```

### Network Security
```yaml
# VPC Security Group rules (via eksctl or AWS CLI)
# Allow inbound HTTPS (443) from internet
# Allow inbound HTTP (80) from internet (for redirects)
# Allow internal cluster communication
# Deny all other inbound traffic
```

### Pod Security Standards
```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: apprise-go-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  runAsUser:
    rule: MustRunAsNonRoot
  seLinux:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - configMap
  - secret
  - persistentVolumeClaim
  - csi
```

## Auto-scaling Configuration

### Horizontal Pod Autoscaler
```yaml
# hpa-aws.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: apprise-go-hpa
  namespace: apprise-go
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
  - type: External
    external:
      metric:
        name: sqs_queue_depth
        selector:
          matchLabels:
            queue_name: notification-queue
      target:
        type: AverageValue
        averageValue: "5"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

### Vertical Pod Autoscaler
```bash
# Install VPA
git clone https://github.com/kubernetes/autoscaler.git
cd autoscaler/vertical-pod-autoscaler/
./hack/vpa-install.sh

# Apply VPA configuration
kubectl apply -f - << EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: apprise-go-vpa
  namespace: apprise-go
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: apprise-go-api
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: apprise-go
      maxAllowed:
        cpu: 2000m
        memory: 2Gi
      minAllowed:
        cpu: 100m
        memory: 128Mi
EOF
```

## Backup & Disaster Recovery

### Automated EBS Snapshots
```bash
# Create backup policy
aws dlm create-lifecycle-policy \
  --execution-role-arn arn:aws:iam::ACCOUNT:role/AWSDataLifecycleManagerDefaultRole \
  --description "Apprise-Go EBS backup policy" \
  --state ENABLED \
  --policy-details '{
    "PolicyType": "EBS_SNAPSHOT_MANAGEMENT",
    "ResourceTypes": ["VOLUME"],
    "TargetTags": [
      {
        "Key": "Application",
        "Value": "apprise-go"
      }
    ],
    "Schedules": [
      {
        "Name": "DailyBackups",
        "CopyTags": true,
        "TagsToAdd": [
          {
            "Key": "BackupType",
            "Value": "Automated"
          }
        ],
        "CreateRule": {
          "Interval": 24,
          "IntervalUnit": "HOURS",
          "Times": ["03:00"]
        },
        "RetainRule": {
          "Count": 7
        }
      }
    ]
  }'
```

### Cross-Region Backup
```bash
# Create cross-region backup using AWS Backup
aws backup create-backup-vault \
  --backup-vault-name apprise-go-backup-vault \
  --region us-east-1

# Create backup plan
aws backup create-backup-plan \
  --backup-plan '{
    "BackupPlanName": "apprise-go-backup-plan",
    "Rules": [
      {
        "RuleName": "DailyBackups",
        "TargetBackupVault": "apprise-go-backup-vault",
        "ScheduleExpression": "cron(0 5 ? * * *)",
        "StartWindowMinutes": 480,
        "Lifecycle": {
          "DeleteAfterDays": 30
        },
        "CopyActions": [
          {
            "DestinationBackupVaultArn": "arn:aws:backup:us-east-1:ACCOUNT:backup-vault:apprise-go-backup-vault",
            "Lifecycle": {
              "DeleteAfterDays": 30
            }
          }
        ]
      }
    ]
  }'
```

## Cost Optimization

### Resource Right-sizing
```bash
# Use AWS Compute Optimizer recommendations
aws compute-optimizer get-ec2-instance-recommendations \
  --filters name=Finding,values=Underprovisioned,Overprovisioned

# Analyze EKS usage
kubectl top nodes
kubectl top pods --all-namespaces
```

### Spot Instances for Development
```yaml
# Add to cluster-config.yaml for dev cluster
managedNodeGroups:
  - name: apprise-spot-workers
    instanceTypes: 
      - t3.medium
      - t3.large
      - m5.large
    spot: true
    minSize: 1
    maxSize: 10
    desiredCapacity: 2
    tags:
      Environment: development
```

### Reserved Instances
```bash
# Purchase Reserved Instances for production
aws ec2 purchase-reserved-instances-offering \
  --reserved-instances-offering-id OFFERING-ID \
  --instance-count 3
```

## Maintenance & Updates

### EKS Cluster Updates
```bash
# Update control plane
eksctl update cluster --name apprise-go-cluster --version 1.29

# Update managed node groups
eksctl update nodegroup --cluster apprise-go-cluster --name apprise-workers

# Update add-ons
aws eks update-addon --cluster-name apprise-go-cluster --addon-name vpc-cni --addon-version latest
```

### Application Updates
```bash
# Rolling update
kubectl set image deployment/apprise-go-api apprise-go=apprise-go:v1.2.0 -n apprise-go

# Monitor rollout
kubectl rollout status deployment/apprise-go-api -n apprise-go

# Rollback if needed
kubectl rollout undo deployment/apprise-go-api -n apprise-go
```

## Troubleshooting

### Common EKS Issues

**Pods stuck in Pending:**
```bash
# Check node capacity
kubectl describe nodes

# Check PV/PVC status
kubectl get pv,pvc -n apprise-go

# Check node group scaling
aws eks describe-nodegroup --cluster-name apprise-go-cluster --nodegroup-name apprise-workers
```

**Load Balancer issues:**
```bash
# Check ALB controller logs
kubectl logs -n kube-system deployment/aws-load-balancer-controller

# Check ingress status
kubectl describe ingress -n apprise-go

# Check security groups
aws ec2 describe-security-groups --group-names k8s-elb-*
```

**High costs:**
```bash
# Check unused resources
kubectl get pods --all-namespaces --field-selector=status.phase=Failed

# Review CloudWatch costs
aws ce get-cost-and-usage --time-period Start=2024-01-01,End=2024-01-31 --granularity MONTHLY --metrics BlendedCost --group-by Type=DIMENSION,Key=SERVICE
```

## Production Checklist

- [ ] EKS cluster created with managed node groups
- [ ] AWS Load Balancer Controller installed
- [ ] EBS CSI driver configured
- [ ] Cluster Autoscaler deployed
- [ ] Application deployed with HPA
- [ ] Monitoring stack (Prometheus/Grafana) installed
- [ ] CloudWatch Container Insights enabled
- [ ] Secrets managed via AWS Secrets Manager
- [ ] Backup policies configured
- [ ] Security policies applied
- [ ] Network policies configured
- [ ] SSL/TLS certificates installed
- [ ] Cost monitoring alerts set up
- [ ] Disaster recovery plan documented

## Support Resources

- **AWS EKS Documentation**: https://docs.aws.amazon.com/eks/
- **eksctl Documentation**: https://eksctl.io/
- **Kubernetes on AWS**: https://aws.amazon.com/kubernetes/
- **EKS Best Practices**: https://aws.github.io/aws-eks-best-practices/

For AWS-specific support and optimization, consider AWS Enterprise Support or AWS Professional Services.