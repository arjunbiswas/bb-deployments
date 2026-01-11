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

	buildbarnv1alpha1 "github.com/buildbarn/bb-deployments/operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	frontendControllerName = "buildbarn-frontend"
	frontendFinalizerName  = "buildbarnfrontend.buildbarn.io"
)

// BuildbarnFrontendReconciler reconciles a BuildbarnFrontend object
type BuildbarnFrontendReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildbarnFrontendReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildbarnv1alpha1.BuildbarnFrontend{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnfrontends,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnfrontends/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnfrontends/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BuildbarnFrontendReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var frontend buildbarnv1alpha1.BuildbarnFrontend
	if err := r.Get(ctx, req.NamespacedName, &frontend); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !frontend.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(frontend.ObjectMeta.Finalizers, frontendFinalizerName) {
			frontend.ObjectMeta.Finalizers = removeString(frontend.ObjectMeta.Finalizers, frontendFinalizerName)
			if err := r.Update(ctx, &frontend); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !containsString(frontend.ObjectMeta.Finalizers, frontendFinalizerName) {
		frontend.ObjectMeta.Finalizers = append(frontend.ObjectMeta.Finalizers, frontendFinalizerName)
		if err := r.Update(ctx, &frontend); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &frontend); err != nil {
		logger.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, &frontend); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, &frontend); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuildbarnFrontendReconciler) reconcileDeployment(ctx context.Context, frontend *buildbarnv1alpha1.BuildbarnFrontend) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontend.Name,
			Namespace: frontend.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		replicas := int32(3)
		if frontend.Spec.Replicas != nil {
			replicas = *frontend.Spec.Replicas
		}

		image := frontend.Spec.Image
		if image == "" {
			image = "ghcr.io/buildbarn/bb-storage:latest"
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "frontend",
					"component": frontend.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "frontend",
						"component": frontend.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "storage",
							Image: image,
							Args:  []string{"/config/frontend.jsonnet"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8980, Protocol: corev1.ProtocolTCP},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "configs", MountPath: "/config/", ReadOnly: true},
							},
							Resources: frontend.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{Name: "configs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: frontend.Spec.ConfigMapName}}}},
					},
					ImagePullSecrets: frontend.Spec.ImagePullSecrets,
					NodeSelector:     frontend.Spec.NodeSelector,
					Tolerations:      frontend.Spec.Tolerations,
				},
			},
		}

		return ctrl.SetControllerReference(frontend, deployment, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("deployment reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnFrontendReconciler) reconcileService(ctx context.Context, frontend *buildbarnv1alpha1.BuildbarnFrontend) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      frontend.Name,
			Namespace: frontend.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, service, func() error {
		serviceType := frontend.Spec.ServiceType
		if serviceType == "" {
			serviceType = corev1.ServiceTypeLoadBalancer
		}

		service.Spec = corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app":       "frontend",
				"component": frontend.Name,
			},
			Ports: []corev1.ServicePort{
				{Port: 8980, Protocol: corev1.ProtocolTCP, Name: "grpc"},
			},
		}

		return ctrl.SetControllerReference(frontend, service, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("service reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnFrontendReconciler) updateStatus(ctx context.Context, frontend *buildbarnv1alpha1.BuildbarnFrontend) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: frontend.Name, Namespace: frontend.Namespace}, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		frontend.Status.State = "NotFound"
		frontend.Status.Replicas = 0
		frontend.Status.ReadyReplicas = 0
	} else {
		frontend.Status.Replicas = deployment.Status.Replicas
		frontend.Status.ReadyReplicas = deployment.Status.ReadyReplicas
		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			frontend.Status.State = "Ready"
		} else {
			frontend.Status.State = "NotReady"
		}
	}

	return r.Status().Update(ctx, frontend)
}
