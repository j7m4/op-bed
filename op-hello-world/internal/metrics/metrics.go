/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileTotal is a counter for total reconciliations per controller and result
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helloworld_reconcile_total",
			Help: "Total number of reconciliations per controller",
		},
		[]string{"controller", "result"},
	)

	// ReconcileErrors is a counter for reconciliation errors per controller
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helloworld_reconcile_errors_total",
			Help: "Total number of reconciliation errors per controller",
		},
		[]string{"controller"},
	)

	// ReconcileDuration is a histogram for reconciliation durations
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helloworld_reconcile_duration_seconds",
			Help:    "Duration of reconciliations per controller",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	// HelloWorldResources is a gauge for the number of HelloWorld resources
	HelloWorldResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helloworld_resources",
			Help: "Number of HelloWorld resources",
		},
		[]string{"namespace"},
	)

	// PodCreations is a counter for successful pod creations
	PodCreations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helloworld_pod_creations_total",
			Help: "Total number of successful pod creations",
		},
		[]string{"namespace"},
	)

	// PodCreationErrors is a counter for pod creation errors
	PodCreationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helloworld_pod_creation_errors_total",
			Help: "Total number of pod creation errors",
		},
		[]string{"namespace"},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileErrors,
		ReconcileDuration,
		HelloWorldResources,
		PodCreations,
		PodCreationErrors,
	)
}

// InitMetrics initializes OpenTelemetry metrics
func InitMetrics(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Get OTLP endpoint from environment or use default
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	// Create OTLP metrics exporter
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP metrics exporter: %w", err)
	}

	// Create resource with service information
	resource := sdkresource.NewWithAttributes(
		"", // Empty schema URL to avoid conflicts
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "1.0.0"),
		attribute.String("environment", getEnvironment()),
	)

	// Create metrics provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(resource),
	)

	// Set global metrics provider
	otel.SetMeterProvider(meterProvider)

	// Return shutdown function
	return meterProvider.Shutdown, nil
}

// GetMeter returns a meter for the given component
func GetMeter(component string) metric.Meter {
	return otel.GetMeterProvider().Meter(
		"github.com/example/op-hello-world",
		metric.WithInstrumentationVersion("1.0.0"),
		metric.WithInstrumentationAttributes(
			attribute.String("component", component),
		),
	)
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return env
}
