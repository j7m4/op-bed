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

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/example/op-hello-world/api/v1"
	"github.com/example/op-hello-world/internal/metrics"
	"github.com/example/op-hello-world/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HelloWorldReconciler reconciles a HelloWorld object
type HelloWorldReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.example.com,resources=helloworlds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=helloworlds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=helloworlds/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelloWorld object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *HelloWorldReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("helloworld", req.NamespacedName)

	///////////////////////////////
	// Custom code start
	// This section handles the reconciliation logic for HelloWorld resources

	// Start tracing span
	tracer := tracing.GetTracer("helloworld-controller")
	ctx, span := tracer.Start(ctx, "Reconcile",
		trace.WithAttributes(
			attribute.String("resource.name", req.Name),
			attribute.String("resource.namespace", req.Namespace),
		),
	)
	defer span.End()

	// Start timing the reconciliation
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.ReconcileDuration.WithLabelValues("helloworld").Observe(duration)
		span.SetAttributes(attribute.Float64("reconcile.duration_seconds", duration))
	}()

	// Fetch the HelloWorld instance
	helloworld := &appsv1.HelloWorld{}
	err := r.Get(ctx, req.NamespacedName, helloworld)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("HelloWorld resource not found. Ignoring since object must be deleted")
			metrics.ReconcileTotal.WithLabelValues("helloworld", "resource_deleted").Inc()
			span.SetAttributes(attribute.String("reconcile.result", "resource_deleted"))
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HelloWorld")
		metrics.ReconcileErrors.WithLabelValues("helloworld").Inc()
		metrics.ReconcileTotal.WithLabelValues("helloworld", "error").Inc()
		tracing.RecordError(span, err, "Failed to get HelloWorld resource")
		span.SetStatus(codes.Error, "Failed to get HelloWorld")
		return ctrl.Result{}, err
	}

	// Add resource attributes to span
	span.SetAttributes(
		attribute.String("helloworld.message", helloworld.Spec.Message),
		attribute.String("helloworld.uid", string(helloworld.UID)),
	)

	// Define the desired pod for this HelloWorld resource
	pod := r.podForHelloWorld(helloworld)

	// Set HelloWorld instance as the owner and controller
	if err := controllerutil.SetControllerReference(helloworld, pod, r.Scheme); err != nil {
		tracing.RecordError(span, err, "Failed to set controller reference")
		span.SetStatus(codes.Error, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	// Check if the pod already exists, if not create a new one
	found := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pod", "pod", types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, "message", helloworld.Spec.Message)

		// Create child span for pod creation
		_, createSpan := tracer.Start(ctx, "CreatePod",
			trace.WithAttributes(
				attribute.String("pod.name", pod.Name),
				attribute.String("pod.namespace", pod.Namespace),
			),
		)
		err = r.Create(ctx, pod)
		createSpan.End()

		if err != nil {
			log.Error(err, "Failed to create new Pod", "pod", types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace})
			metrics.PodCreationErrors.WithLabelValues(pod.Namespace).Inc()
			metrics.ReconcileErrors.WithLabelValues("helloworld").Inc()
			metrics.ReconcileTotal.WithLabelValues("helloworld", "error").Inc()
			tracing.RecordError(span, err, "Failed to create pod")
			span.SetStatus(codes.Error, "Failed to create pod")
			return ctrl.Result{}, err
		}
		// Pod created successfully - return and requeue
		metrics.PodCreations.WithLabelValues(pod.Namespace).Inc()
		metrics.ReconcileTotal.WithLabelValues("helloworld", "pod_created").Inc()
		log.Info("Pod created successfully", "pod", types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, "message", helloworld.Spec.Message)
		span.SetAttributes(attribute.String("reconcile.result", "pod_created"))
		span.SetStatus(codes.Ok, "Pod created successfully")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Pod")
		metrics.ReconcileErrors.WithLabelValues("helloworld").Inc()
		metrics.ReconcileTotal.WithLabelValues("helloworld", "error").Inc()
		tracing.RecordError(span, err, "Failed to get pod")
		span.SetStatus(codes.Error, "Failed to get pod")
		return ctrl.Result{}, err
	}

	// Pod already exists - don't update, just log
	log.V(1).Info("Skip reconcile: Pod already exists", "pod", types.NamespacedName{Name: found.Name, Namespace: found.Namespace})
	metrics.ReconcileTotal.WithLabelValues("helloworld", "no_change").Inc()

	// Update HelloWorld resource count metric
	metrics.HelloWorldResources.WithLabelValues(helloworld.Namespace).Set(1)

	span.SetAttributes(attribute.String("reconcile.result", "no_change"))
	span.SetStatus(codes.Ok, "Reconciliation completed")

	// Custom code end
	///////////////////////////////

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelloWorldReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.HelloWorld{}).
		Named("helloworld").
		Complete(r)
}

///////////////////////////////
// Custom code start
// Helper functions for the HelloWorld controller

// podForHelloWorld returns a busybox pod with the same name/namespace as the HelloWorld CR
func (r *HelloWorldReconciler) podForHelloWorld(helloworld *appsv1.HelloWorld) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helloworld.Name + "-pod",
			Namespace: helloworld.Namespace,
			Labels: map[string]string{
				"app":        "helloworld",
				"helloworld": helloworld.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "busybox",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []string{fmt.Sprintf("echo '%s' && sleep 3600", helloworld.Spec.Message)},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			}},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
	return pod
}

// Custom code end
///////////////////////////////
