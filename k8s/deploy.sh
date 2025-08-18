#!/bin/bash

# Apprise-Go Kubernetes Deployment Script
# This script deploys Apprise-Go to a Kubernetes cluster

set -e

# Configuration
NAMESPACE="apprise-go"
DOCKER_IMAGE="apprise-go:latest"
KUBECONFIG_PATH=""
DRY_RUN=false
SKIP_BUILD=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy Apprise-Go to Kubernetes cluster

OPTIONS:
    -n, --namespace NAMESPACE    Target namespace (default: apprise-go)
    -i, --image IMAGE           Docker image tag (default: apprise-go:latest)
    -k, --kubeconfig PATH       Path to kubeconfig file
    -d, --dry-run              Perform dry run without applying changes
    -s, --skip-build           Skip Docker image build
    -h, --help                 Show this help message

EXAMPLES:
    $0                         # Deploy with defaults
    $0 -n production           # Deploy to 'production' namespace
    $0 -i apprise-go:v1.2.0    # Deploy specific image version
    $0 -d                      # Dry run to preview changes
    $0 -s                      # Skip building Docker image

EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check Docker (unless skipping build)
    if [ "$SKIP_BUILD" = false ] && ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check kustomize
    if ! command -v kustomize &> /dev/null; then
        log_warning "kustomize not found, using kubectl apply instead"
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

build_image() {
    if [ "$SKIP_BUILD" = true ]; then
        log_info "Skipping Docker image build"
        return
    fi
    
    log_info "Building Docker image: $DOCKER_IMAGE"
    
    # Build from project root
    cd "$(dirname "$0")/.."
    
    if ! docker build -t "$DOCKER_IMAGE" .; then
        log_error "Failed to build Docker image"
        exit 1
    fi
    
    log_success "Docker image built successfully: $DOCKER_IMAGE"
}

create_namespace() {
    log_info "Creating namespace: $NAMESPACE"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Namespace $NAMESPACE already exists"
    else
        kubectl create namespace "$NAMESPACE"
        log_success "Namespace $NAMESPACE created"
    fi
}

update_secrets() {
    log_info "Updating secrets..."
    
    # Generate JWT secret if not exists
    JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
    
    # Update secret with generated values
    kubectl create secret generic apprise-go-secrets \
        --namespace="$NAMESPACE" \
        --from-literal=JWT_SECRET="$JWT_SECRET" \
        --from-literal=ADMIN_PASSWORD="admin123" \
        --from-literal=DATABASE_PASSWORD="db_password123" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Secrets updated"
}

deploy_manifests() {
    log_info "Deploying Kubernetes manifests..."
    
    cd "$(dirname "$0")"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "Performing dry run..."
        kubectl apply --dry-run=client -f .
        log_info "Dry run completed successfully"
        return
    fi
    
    # Apply manifests in order
    local manifests=(
        "namespace.yaml"
        "configmap.yaml"
        "secret.yaml"
        "pvc.yaml"
        "deployment.yaml"
        "service.yaml"
        "hpa.yaml"
        "poddisruptionbudget.yaml"
        "networkpolicy.yaml"
        "ingress.yaml"
        "monitoring.yaml"
    )
    
    for manifest in "${manifests[@]}"; do
        if [ -f "$manifest" ]; then
            log_info "Applying $manifest..."
            kubectl apply -f "$manifest"
        else
            log_warning "Manifest $manifest not found, skipping..."
        fi
    done
    
    log_success "All manifests applied successfully"
}

wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."
    
    if ! kubectl wait --namespace="$NAMESPACE" \
        --for=condition=available \
        --timeout=300s \
        deployment/apprise-go-api; then
        log_error "Deployment failed to become ready within timeout"
        exit 1
    fi
    
    log_success "Deployment is ready"
}

verify_deployment() {
    log_info "Verifying deployment..."
    
    # Check pods
    log_info "Checking pods status:"
    kubectl get pods -n "$NAMESPACE" -l app=apprise-go
    
    # Check services
    log_info "Checking services:"
    kubectl get svc -n "$NAMESPACE"
    
    # Check ingress
    log_info "Checking ingress:"
    kubectl get ingress -n "$NAMESPACE"
    
    # Test health endpoint
    log_info "Testing health endpoint..."
    if kubectl port-forward -n "$NAMESPACE" service/apprise-go-service 8080:8080 &
    then
        PF_PID=$!
        sleep 5
        
        if curl -f http://localhost:8080/health &> /dev/null; then
            log_success "Health check passed"
        else
            log_warning "Health check failed"
        fi
        
        kill $PF_PID 2>/dev/null || true
    fi
    
    log_success "Deployment verification completed"
}

cleanup() {
    log_info "Cleaning up port-forward processes..."
    pkill -f "kubectl port-forward" 2>/dev/null || true
}

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -i|--image)
                DOCKER_IMAGE="$2"
                shift 2
                ;;
            -k|--kubeconfig)
                KUBECONFIG_PATH="$2"
                export KUBECONFIG="$KUBECONFIG_PATH"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -s|--skip-build)
                SKIP_BUILD=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    log_info "Starting Apprise-Go Kubernetes deployment..."
    log_info "Target namespace: $NAMESPACE"
    log_info "Docker image: $DOCKER_IMAGE"
    log_info "Dry run: $DRY_RUN"
    
    check_prerequisites
    build_image
    create_namespace
    update_secrets
    deploy_manifests
    
    if [ "$DRY_RUN" = false ]; then
        wait_for_deployment
        verify_deployment
        
        log_success "Apprise-Go deployment completed successfully!"
        log_info "Access the service:"
        log_info "  kubectl port-forward -n $NAMESPACE service/apprise-go-service 8080:8080"
        log_info "  curl http://localhost:8080/health"
    fi
}

main "$@"