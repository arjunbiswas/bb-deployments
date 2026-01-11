/*
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

package controllers

import (
	"context"
	"fmt"

	buildbarnv1alpha1 "github.com/buildbarn/bb-deployments/operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	schedulerControllerName = "buildbarn-scheduler"
	schedulerFinalizerName  = "buildbarnscheduler.buildbarn.io"
)

// BuildbarnSchedulerReconciler reconciles a BuildbarnScheduler object
type BuildbarnSchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildbarnSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildbarnv1alpha1.BuildbarnScheduler{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnschedulers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BuildbarnSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var scheduler buildbarnv1alpha1.BuildbarnScheduler
	if err := r.Get(ctx, req.NamespacedName, &scheduler); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !scheduler.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(scheduler.ObjectMeta.Finalizers, schedulerFinalizerName) {
			// Remove finalizer
			scheduler.ObjectMeta.Finalizers = removeString(scheduler.ObjectMeta.Finalizers, schedulerFinalizerName)
			if err := r.Update(ctx, &scheduler); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !containsString(scheduler.ObjectMeta.Finalizers, schedulerFinalizerName) {
		scheduler.ObjectMeta.Finalizers = append(scheduler.ObjectMeta.Finalizers, schedulerFinalizerName)
		if err := r.Update(ctx, &scheduler); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &scheduler); err != nil {
		logger.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, &scheduler); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	// Reconcile Ingress
	if err := r.reconcileIngress(ctx, &scheduler); err != nil {
		logger.Error(err, "failed to reconcile ingress")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, &scheduler); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuildbarnSchedulerReconciler) reconcileDeployment(ctx context.Context, scheduler *buildbarnv1alpha1.BuildbarnScheduler) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		replicas := int32(1)
		if scheduler.Spec.Replicas != nil {
			replicas = *scheduler.Spec.Replicas
		}

		image := scheduler.Spec.Image
		if image == "" {
			image = "ghcr.io/buildbarn/bb-scheduler:latest"
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "scheduler",
					"component": scheduler.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "scheduler",
						"component": scheduler.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "scheduler",
							Image: image,
							Args:  []string{fmt.Sprintf("/config/%s.jsonnet", scheduler.Spec.ConfigMapName)},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8982, Protocol: corev1.ProtocolTCP, Name: "client-grpc"},
								{ContainerPort: 8983, Protocol: corev1.ProtocolTCP, Name: "worker-grpc"},
								{ContainerPort: 7982, Protocol: corev1.ProtocolTCP, Name: "http"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "configs",
									MountPath: "/config/",
									ReadOnly:  true,
								},
							},
							Resources: scheduler.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "configs",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: scheduler.Spec.ConfigMapName,
									},
								},
							},
						},
					},
					ImagePullSecrets: scheduler.Spec.ImagePullSecrets,
					NodeSelector:     scheduler.Spec.NodeSelector,
					Tolerations:      scheduler.Spec.Tolerations,
				},
			},
		}

		return ctrl.SetControllerReference(scheduler, deployment, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("deployment reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnSchedulerReconciler) reconcileService(ctx context.Context, scheduler *buildbarnv1alpha1.BuildbarnScheduler) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, service, func() error {
		serviceType := scheduler.Spec.ServiceType
		if serviceType == "" {
			serviceType = corev1.ServiceTypeClusterIP
		}

		service.Spec = corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app":       "scheduler",
				"component": scheduler.Name,
			},
			Ports: []corev1.ServicePort{
				{Port: 8982, Protocol: corev1.ProtocolTCP, Name: "client-grpc"},
				{Port: 8983, Protocol: corev1.ProtocolTCP, Name: "worker-grpc"},
				{Port: 7982, Protocol: corev1.ProtocolTCP, Name: "http"},
			},
		}

		return ctrl.SetControllerReference(scheduler, service, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("service reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnSchedulerReconciler) reconcileIngress(ctx context.Context, scheduler *buildbarnv1alpha1.BuildbarnScheduler) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("bb-scheduler.%s", scheduler.Namespace),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressPath{
							Path:     "/",
							PathType: func() *networkingv1.PathType { p := networkingv1.PathTypePrefix; return &p }(),
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: scheduler.Name,
									Port: networkingv1.ServiceBackendPort{
										Name: "http",
									},
								},
							},
						},
					},
				},
			},
		}

		return ctrl.SetControllerReference(scheduler, ingress, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("ingress reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnSchedulerReconciler) updateStatus(ctx context.Context, scheduler *buildbarnv1alpha1.BuildbarnScheduler) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: scheduler.Name, Namespace: scheduler.Namespace}, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		scheduler.Status.State = "NotFound"
		scheduler.Status.Replicas = 0
		scheduler.Status.ReadyReplicas = 0
	} else {
		scheduler.Status.Replicas = deployment.Status.Replicas
		scheduler.Status.ReadyReplicas = deployment.Status.ReadyReplicas
		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			scheduler.Status.State = "Ready"
		} else {
			scheduler.Status.State = "NotReady"
		}
	}

	return r.Status().Update(ctx, scheduler)
}

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
