# Mandatory Wait Plugin for Argo Rollouts

This plugin provides a mandatory wait functionality for Argo Rollouts that blocks the rollout for a specified duration, similar to a pause step but **without allowing promotion** during the wait period.

## Features

- **Mandatory Blocking**: Unlike regular pause steps, this plugin prevents promotion attempts during the wait period
- **Configurable Duration**: Specify any duration using Go's time format (e.g., "5m", "30s", "1h")
- **State Persistence**: Maintains state across plugin restarts and rollout controller restarts
- **Proper Lifecycle Management**: Handles termination and abort scenarios appropriately

## Configuration

The plugin accepts the following configuration:

```yaml
- plugin:
    name: mandatory-wait
    config:
      duration: "5m"  # Required: Duration to wait (Go duration format)
```

### Duration Format

The duration field accepts any valid Go duration string:
- `"30s"` - 30 seconds
- `"5m"` - 5 minutes
- `"1h"` - 1 hour
- `"2h30m"` - 2 hours and 30 minutes

## Example Rollout

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: example-rollout
spec:
  replicas: 5
  strategy:
    canary:
      steps:
      - setWeight: 20
      - plugin:
          name: mandatory-wait
          config:
            duration: "5m"
      - setWeight: 50
      - plugin:
          name: mandatory-wait
          config:
            duration: "10m"
      - setWeight: 100
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
      - name: example
        image: nginx:1.20
```

## Behavior

### Normal Operation
1. When the plugin starts, it records the current time and begins the mandatory wait period
2. The rollout will remain in "Running" state and cannot be promoted during this time
3. The plugin checks every 15 seconds if the wait period has completed
4. Once the duration has elapsed, the plugin marks the step as successful and allows progression

### Termination
- **During Wait Period**: Termination requests are ignored; the plugin continues waiting
- **After Wait Period**: Termination is allowed and the step completes successfully

### Abort
- Abort requests will immediately complete the step, allowing the rollout to be aborted

## Building

```bash
# Build the plugin
make build-mandatory-wait

# Build for local development
make build-mandatory-wait-local

# Build Docker image
make docker-build

# Run tests
go test ./step-plugin-mandatory-wait/...
```

## Installation

1. Build the plugin binary or Docker image
2. Deploy the plugin as a sidecar or separate deployment in your cluster
3. Configure Argo Rollouts to use the plugin by specifying the plugin's address

## Use Cases

- **Mandatory observation periods**: Ensure monitoring data is collected for a minimum time
- **Compliance requirements**: Meet regulatory requirements for deployment observation periods
- **Staged rollouts**: Force deliberate pauses between rollout stages
- **Integration testing windows**: Allow time for automated integration tests to complete

## Differences from Standard Pause

| Feature | Standard Pause | Mandatory Wait Plugin |
|---------|---------------|----------------------|
| Manual Promotion | ✅ Allowed | ❌ Blocked during wait |
| Automatic Progression | ✅ After duration | ✅ After duration |
| Termination | ✅ Immediate | ❌ Must wait for completion |
| State Persistence | ❌ Basic | ✅ Full state tracking |
| Progress Reporting | ❌ Limited | ✅ Detailed remaining time |