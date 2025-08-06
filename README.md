# Kubernetes Operators Project

This project contains multiple Kubernetes operators built with Kubebuilder. Each operator is organized 
in its own directory with an `op-` prefix.

## Requirements

### Prerequisites

- **Go** (version 1.21 or later)
- **Docker** (for building container images)
- **kind** (Kubernetes in Docker) - already configured
- **kubectl** (for interacting with Kubernetes clusters)
- **Kubebuilder** (for operator development)

### Development Environment

- Each operator directory follows the naming convention: `op-<operator-name>`
- Operators are built using the Kubebuilder framework
- Local development and testing use kind clusters

#### Setting up the environment

`cp config.example.env config.env` and adjust the values in `config.env` as needed.

```bash
./setup.sh
```

* creates the kind cluster, if necessary
* sets the current context to the cluster
* preinstalls images defined in `scripts/preload-images.sh`

#### Tearing down the environment

It can be helpful to start from a clean slate by deleting the kind cluster.

```bash
./teardown.sh
```

Run `./setup.sh` again to recreate it.

### Getting Started

1. Ensure all prerequisites are installed
2. Navigate to the specific operator directory (e.g., `op-example`)
3. Follow the operator-specific README for build and deployment instructions

### Project Structure

```
op-bed/
├── op-<operator1>/     # First operator
├── op-<operator2>/     # Second operator
├── op-<operator3>/     # Third operator
└── ...                 # Additional operators
```

Each operator directory contains its own Kubebuilder-generated structure with controllers, APIs, and configuration files.

## Observability Stack (Grafana LGTM + Alloy)

This project includes a complete observability stack using Grafana LGTM (Loki, Grafana, Tempo, Mimir) and Alloy for telemetry collection.

### Components

- **Grafana** (port 3000): Unified observability UI with pre-configured datasources
- **Loki** (port 3100): Log aggregation system
- **Tempo** (ports 3200, 4317, 4318): Distributed tracing with OTLP support
- **Mimir** (port 9009): Prometheus-compatible metrics storage
- **Pyroscope** (port 4040): Continuous profiling platform
- **Alloy** (port 12345): Telemetry collector and forwarder

### Running the Stack

1. Start the Kind cluster:
   ```bash
   ./setup.sh
   ```

2. Deploy the observability stack using Tilt:
   ```bash
   tilt up
   ```

3. Access Grafana UI at http://localhost:3000 (anonymous access enabled)

### Integration Requirements

#### For Metrics Collection
- Applications exposing Prometheus metrics will be automatically discovered and scraped by Alloy
- Metrics are forwarded to Mimir and available in Grafana

#### For Tracing (OTLP)
- Send traces to Alloy's OTLP endpoints:
  - gRPC: `localhost:14317`
  - HTTP: `localhost:14318`
- Traces are forwarded to Tempo and available in Grafana

#### For Logging
- Send logs to Alloy's OTLP endpoints (same as tracing)
- Logs are forwarded to Loki and available in Grafana

#### For Continuous Profiling
- Applications must expose pprof endpoints (typically on `/debug/pprof/*`)
- Add these annotations to your pods to enable profiling:
  ```yaml
  metadata:
    annotations:
      pyroscope.io/scrape: "true"
      pyroscope.io/port: "6060"  # Port where pprof endpoints are exposed
  ```
- Profiles are forwarded to Pyroscope and available at http://localhost:4040

### Example Integration

For a Go application with pprof enabled:

```go
import _ "net/http/pprof"

func main() {
    // Start pprof server
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your application code...
}
```

Then annotate your Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        pyroscope.io/scrape: "true"
        pyroscope.io/port: "6060"
    spec:
      containers:
      - name: my-app
        ports:
        - containerPort: 6060
          name: pprof
```