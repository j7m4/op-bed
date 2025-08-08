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
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch

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

	// Ensure pull secret exists in the namespace
	if err := r.ensurePullSecret(ctx, req.Namespace); err != nil {
		log.Error(err, "Failed to ensure pull secret")
		metrics.ReconcileErrors.WithLabelValues("helloworld").Inc()
		tracing.RecordError(span, err, "Failed to ensure pull secret")
		// Continue even if pull secret fails - pod might use public images
	}

	// Define the desired pod for this HelloWorld resource
	pod := r.podForHelloWorld(helloworld)

	// Set HelloWorld instance as the owner and controller
	if err := controllerutil.SetControllerReference(helloworld, pod, r.Scheme); err != nil {
		tracing.RecordError(span, err, "Failed to set controller reference")
		span.SetStatus(codes.Error, "Failed to set controller reference")

		// Update status to Failed
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "OwnerReferenceFailed", "Failed to set owner reference")
		r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionFalse, "Error", err.Error())
		r.updateStatus(ctx, helloworld, appsv1.PhaseFailed, "", fmt.Sprintf("Failed to set owner reference: %v", err))

		return ctrl.Result{}, err
	}

	// Check if the pod already exists, if not create a new one
	found := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Pod", "pod", types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, "message", helloworld.Spec.Message)

		// Update status to Pending before creating pod
		r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionTrue, "CreatingPod", "Creating pod for HelloWorld resource")
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "PodNotReady", "Pod is being created")
		if err := r.updateStatus(ctx, helloworld, appsv1.PhasePending, "", "Creating pod"); err != nil {
			log.Error(err, "Failed to update status")
		}

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

			// Update status to Failed
			r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "PodCreationFailed", "Failed to create pod")
			r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionFalse, "Error", err.Error())
			r.setCondition(helloworld, appsv1.TypeDegraded, metav1.ConditionTrue, "PodCreationError", fmt.Sprintf("Pod creation failed: %v", err))
			r.updateStatus(ctx, helloworld, appsv1.PhaseFailed, "", fmt.Sprintf("Failed to create pod: %v", err))

			return ctrl.Result{}, err
		}
		// Pod created successfully - update status and requeue
		metrics.PodCreations.WithLabelValues(pod.Namespace).Inc()
		metrics.ReconcileTotal.WithLabelValues("helloworld", "pod_created").Inc()
		log.Info("Pod created successfully", "pod", types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, "message", helloworld.Spec.Message)

		// Update status to Running
		r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionTrue, "PodCreated", "Pod has been created successfully")
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "PodStarting", "Pod is starting up")
		if err := r.updateStatus(ctx, helloworld, appsv1.PhaseRunning, pod.Name, "Pod created successfully"); err != nil {
			log.Error(err, "Failed to update status")
		}

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

	// Pod already exists - check its status and update accordingly
	log.V(1).Info("Skip reconcile: Pod already exists", "pod", types.NamespacedName{Name: found.Name, Namespace: found.Namespace})
	metrics.ReconcileTotal.WithLabelValues("helloworld", "no_change").Inc()

	// Update status based on pod phase
	podPhase := string(found.Status.Phase)
	switch found.Status.Phase {
	case corev1.PodRunning:
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionTrue, "PodRunning", "Pod is running successfully")
		r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionFalse, "Stable", "Resource is stable")
		if err := r.updateStatus(ctx, helloworld, appsv1.PhaseRunning, found.Name, fmt.Sprintf("Pod is %s", podPhase)); err != nil {
			log.Error(err, "Failed to update status")
		}
	case corev1.PodPending:
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "PodPending", "Pod is pending")
		r.setCondition(helloworld, appsv1.TypeProgressing, metav1.ConditionTrue, "PodStarting", "Pod is starting up")
		if err := r.updateStatus(ctx, helloworld, appsv1.PhasePending, found.Name, fmt.Sprintf("Pod is %s", podPhase)); err != nil {
			log.Error(err, "Failed to update status")
		}
	case corev1.PodFailed:
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionFalse, "PodFailed", "Pod has failed")
		r.setCondition(helloworld, appsv1.TypeDegraded, metav1.ConditionTrue, "PodFailure", "Pod is in failed state")
		if err := r.updateStatus(ctx, helloworld, appsv1.PhaseFailed, found.Name, fmt.Sprintf("Pod is %s", podPhase)); err != nil {
			log.Error(err, "Failed to update status")
		}
	default:
		r.setCondition(helloworld, appsv1.TypeReady, metav1.ConditionUnknown, "PodStatusUnknown", fmt.Sprintf("Pod is in %s state", podPhase))
		if err := r.updateStatus(ctx, helloworld, appsv1.PhaseUnknown, found.Name, fmt.Sprintf("Pod is %s", podPhase)); err != nil {
			log.Error(err, "Failed to update status")
		}
	}

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

// updateStatus updates the status of a HelloWorld resource
func (r *HelloWorldReconciler) updateStatus(ctx context.Context, helloworld *appsv1.HelloWorld, phase string, podName string, message string) error {
	// Update status fields
	helloworld.Status.Phase = phase
	helloworld.Status.PodName = podName
	helloworld.Status.Message = message
	now := metav1.Now()
	helloworld.Status.LastUpdateTime = &now
	helloworld.Status.ObservedGeneration = helloworld.Generation

	// Update status in Kubernetes
	if err := r.Status().Update(ctx, helloworld); err != nil {
		return fmt.Errorf("failed to update HelloWorld status: %w", err)
	}
	return nil
}

// setCondition sets a condition on the HelloWorld status
func (r *HelloWorldReconciler) setCondition(helloworld *appsv1.HelloWorld, conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
		ObservedGeneration: helloworld.Generation,
	}

	// Find existing condition and update it, or append if not found
	found := false
	for i, c := range helloworld.Status.Conditions {
		if c.Type == conditionType {
			// Only update if status changed
			if c.Status != status {
				helloworld.Status.Conditions[i] = condition
			} else {
				// Update message and reason even if status didn't change
				helloworld.Status.Conditions[i].Message = message
				helloworld.Status.Conditions[i].Reason = reason
				helloworld.Status.Conditions[i].ObservedGeneration = helloworld.Generation
			}
			found = true
			break
		}
	}
	if !found {
		helloworld.Status.Conditions = append(helloworld.Status.Conditions, condition)
	}
}

// ensurePullSecret ensures the ghcr-login secret exists in the target namespace by copying it from the default namespace
func (r *HelloWorldReconciler) ensurePullSecret(ctx context.Context, namespace string) error {
	log := logf.FromContext(ctx)
	secretName := "ghcr-login"

	// Check if secret already exists in target namespace
	targetSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, targetSecret)
	if err == nil {
		// Secret already exists
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Get the secret from default namespace
	sourceSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, sourceSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Pull secret not found in default namespace, skipping copy", "secret", secretName)
			return nil
		}
		return fmt.Errorf("failed to get source secret: %w", err)
	}

	// Create a copy of the secret for the target namespace
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	// Create the secret in the target namespace
	err = r.Create(ctx, newSecret)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pull secret in namespace %s: %w", namespace, err)
	}

	log.Info("Pull secret copied to namespace", "secret", secretName, "namespace", namespace)
	return nil
}

// Custom code end
///////////////////////////////
