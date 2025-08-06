# HelloWorld Operator Observability

The HelloWorld operator has been instrumented with comprehensive observability features including metrics, logs, and traces.

## Metrics

### Prometheus Metrics

The operator exposes the following custom metrics:

- `helloworld_reconcile_total` - Total number of reconciliations per controller with result labels
- `helloworld_reconcile_errors_total` - Total number of reconciliation errors
- `helloworld_reconcile_duration_seconds` - Histogram of reconciliation durations
- `helloworld_resources` - Gauge tracking number of HelloWorld resources by namespace
- `helloworld_pod_creations_total` - Counter for successful pod creations
- `helloworld_pod_creation_errors_total` - Counter for pod creation errors

### Accessing Metrics

The operator exposes metrics on port 8443 at the `/metrics` endpoint. The ServiceMonitor is configured to enable Prometheus scraping.

## Logging

### Structured Logging

The operator uses structured logging with the following enhancements:

- Resource context included in all log messages
- Verbosity levels for different log priorities
- Consistent key-value pairs for easier filtering
- Namespace and name included in log context

### Log Levels

- Info level: Important state changes (pod creation, successful reconciliation)
- Debug level (V=1): Routine operations (resource already exists)
- Error level: Failures and errors

## Tracing

### OpenTelemetry Integration

The operator includes OpenTelemetry tracing with:

- Span creation for each reconciliation
- Child spans for significant operations (pod creation)
- Error recording with context
- Resource attributes for filtering

### Configuration

Set the following environment variables to configure tracing:

- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (default: localhost:4317)
- `ENVIRONMENT` - Environment name (default: development)

## Grafana Dashboard

A pre-configured Grafana dashboard is available at `config/grafana/helloworld-dashboard.json` with panels for:

- Reconciliation rate by result
- Error rate monitoring
- Reconciliation duration percentiles
- Resource count by namespace
- Pod creation rate

## Testing with Grafana LGTM Stack

To test the observability features:

1. Deploy the operator with metrics and ServiceMonitor enabled
2. Ensure Prometheus is scraping the metrics endpoint
3. Import the Grafana dashboard
4. Configure the OTLP endpoint for tracing
5. Create some HelloWorld resources to generate metrics and traces

## Example Deployment

```bash
# Deploy with observability features
make deploy IMG=<your-registry>/op-hello-world:tag

# Set OTLP endpoint for tracing
kubectl set env deployment/op-hello-world-controller-manager -n op-hello-world-system OTEL_EXPORTER_OTLP_ENDPOINT=<otlp-endpoint>:4317
```