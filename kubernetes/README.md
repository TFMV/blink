# Kubernetes Deployment Guide

This guide explains how to deploy Blink to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster
- kubectl configured to access your cluster
- Docker registry access

## Quick Start

1. Update the `DOCKER_REPO` variable in the Makefile with your Docker repository.
2. Build and push the Docker image:

```bash
make docker-build
make docker-push
```

3. Deploy Blink to your cluster:

```bash
make k8s-deploy
```

## Components

The Kubernetes deployment consists of:

### 1. ConfigMap (`configmap.yaml`)

Contains configuration for Blink:

- Watch path
- Event address
- Include/exclude patterns
- Event types to include/ignore

### 2. Deployment (`deployment.yaml`)

Manages the Blink pods with:

- Resource limits and requests
- Health checks (liveness and readiness probes)
- Volume mounts
- Environment variables from ConfigMap
- Prometheus metrics annotations

### 3. Service (`service.yaml`)

Exposes Blink's API:

- Port 12345 for SSE events
- HTTP endpoints for health checks and metrics

## Configuration

### Resource Limits

```yaml
resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "256Mi"
```

### Health Checks

- Liveness probe: `/health`
- Readiness probe: `/ready`
- Initial delay: 5 seconds
- Period: 10 seconds

### Metrics

Prometheus metrics available at `/metrics`:

- `blink_events_processed_total`
- `blink_events_filtered_total`
- `blink_active_watchers`
- `blink_webhook_latency_seconds`
- `blink_webhook_errors_total`
- `blink_memory_bytes`

## Usage

### Accessing the API

```bash
# Port forward the service
kubectl port-forward service/blink 12345:12345

# Connect to the event stream
curl -N http://localhost:12345/events
```

### Viewing Logs

```bash
# Get pod name
kubectl get pods -l app=blink

# View logs
kubectl logs -f <pod-name>
```

### Updating Configuration

1. Edit the ConfigMap:

```bash
kubectl edit configmap blink-config
```

2. Restart the pods:

```bash
kubectl rollout restart deployment blink
```

## Cleanup

To remove Blink from your cluster:

```bash
make k8s-delete
```

## Troubleshooting

### Common Issues

1. Pod won't start
   - Check events: `kubectl describe pod <pod-name>`
   - Verify resource limits
   - Check image pull policy

2. Health check failures
   - Check pod logs
   - Verify network policies
   - Check resource usage

3. Metrics not appearing
   - Verify Prometheus configuration
   - Check service annotations
   - Test metrics endpoint directly

### Debugging

```bash
# Check pod status
kubectl get pods -l app=blink -o wide

# Check pod events
kubectl describe pod <pod-name>

# Check logs
kubectl logs -f <pod-name>

# Check metrics
kubectl port-forward <pod-name> 12345:12345
curl http://localhost:12345/metrics
```
