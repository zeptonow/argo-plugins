# Argo Rollouts Mandatory Pause Plugin

A custom Argo Rollouts Step Plugin that enforces mandatory pauses during rollouts to prevent users from promoting changes too quickly.

## Overview

This plugin addresses the common issue where teams rush through rollout promotions despite having pause steps configured. The mandatory pause plugin:

- ✅ **Enforces minimum wait times** that cannot be bypassed
- ✅ **Configurable pause durations** per step
- ✅ **Persistent state tracking** across rollout controller restarts
- ✅ **Proper cleanup** when rollouts are terminated or aborted
- ✅ **Real-time progress updates** showing remaining pause time

## Features

- **Mandatory Enforcement**: Unlike regular pauses, these cannot be manually promoted early
- **Flexible Duration**: Configure different pause durations for each step
- **State Persistence**: Maintains pause state even if the controller restarts
- **Progress Feedback**: Shows remaining time in rollout status
- **Proper Cleanup**: Handles rollout termination and abortion gracefully

## Building the Plugin

### Prerequisites

- Go 1.21 or later
- Docker (for containerization)
- Kubernetes cluster with Argo Rollouts installed

### Build Commands

```bash
# Build for local testing
make build-local

# Build for Linux (container deployment)
make build

# Build Docker image
make docker-build

# Build everything (clean, deps, build, test)
make all
```

## Deployment

### 1. Build and Push Container Image

```bash
# Build the Docker image
make docker-build

# Tag for your registry
docker tag argo-rollouts-mandatory-pause-plugin:latest your-registry.com/mandatory-pause-plugin:v1.0.0

# Push to registry
docker push your-registry.com/mandatory-pause-plugin:v1.0.0
```

### 2. Update Plugin Configuration

Update the `rollout-cm.yaml` with your image:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-mandatory-pause-plugin
  namespace: argo-rollouts
data:
  plugin: |
    apiVersion: argoproj.io/v1alpha1
    kind: PluginConfig
    metadata:
      name: mandatory-pause
    spec:
      type: Step
      image: your-registry.com/mandatory-pause-plugin:v1.0.0
```

### 3. Apply Configuration

```bash
kubectl apply -f rollout-cm.yaml
```

### 4. Restart Argo Rollouts Controller

```bash
kubectl rollout restart deployment argo-rollouts-controller -n argo-rollouts
```

## Usage

### Basic Usage

Add the plugin step to your rollout specification:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  strategy:
    canary:
      steps:
      - setWeight: 25
      - plugin:
          name: mandatory-pause
          config:
            duration: "5m"  # Mandatory 5-minute pause
      - setWeight: 50
      - plugin:
          name: mandatory-pause
          config:
            duration: "3m"  # Mandatory 3-minute pause
      - setWeight: 100
```

### Configuration Options

| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `duration` | Pause duration (Go duration format) | `2m` | `30s`, `5m`, `1h` |

### Duration Format Examples

- `30s` - 30 seconds
- `2m` - 2 minutes
- `5m30s` - 5 minutes and 30 seconds
- `1h` - 1 hour
- `1h30m` - 1 hour and 30 minutes

## Monitoring

### Check Plugin Status

```bash
# View rollout status
kubectl describe rollout my-app

# Check rollout events
kubectl get events --field-selector involvedObject.name=my-app

# Watch rollout progress
kubectl argo rollouts get rollout my-app --watch
```

### Plugin Logs

```bash
# Check plugin logs (if running as separate pod)
kubectl logs -l app=mandatory-pause-plugin -n argo-rollouts

# Check rollouts controller logs
kubectl logs -l app.kubernetes.io/name=argo-rollouts -n argo-rollouts
```

## Troubleshooting

### Common Issues

1. **Plugin not found**: Ensure the ConfigMap is applied and controller is restarted
2. **Invalid duration**: Check duration format (e.g., `5m`, `30s`)
3. **Permission issues**: Ensure proper RBAC permissions for the plugin

### Debugging Steps

1. Verify plugin registration:
   ```bash
   kubectl get configmap argo-rollouts-mandatory-pause-plugin -n argo-rollouts -o yaml
   ```

2. Check controller logs:
   ```bash
   kubectl logs -l app.kubernetes.io/name=argo-rollouts -n argo-rollouts --tail=100
   ```

3. Validate rollout configuration:
   ```bash
   kubectl describe rollout your-rollout-name
   ```

## Development

### Local Testing

```bash
# Run the plugin locally
make run-local

# Run tests
make test

# Format code
make fmt
```

### Plugin Interface

The plugin implements the Argo Rollouts Step Plugin interface:

- `Init()` - Initialize plugin with rollout configuration
- `Run()` - Execute the mandatory pause logic
- `Terminate()` - Handle rollout termination
- `Abort()` - Handle rollout abortion

## License

This project is licensed under the MIT License.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request