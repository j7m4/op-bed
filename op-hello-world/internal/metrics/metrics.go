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
	"github.com/prometheus/client_golang/prometheus"
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