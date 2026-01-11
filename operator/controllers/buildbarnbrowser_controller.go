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
	browserControllerName = "buildbarn-browser"
	browserFinalizerName  = "buildbarnbrowser.buildbarn.io"
)

// BuildbarnBrowserReconciler reconciles a BuildbarnBrowser object
type BuildbarnBrowserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildbarnBrowserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildbarnv1alpha1.BuildbarnBrowser{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnbrowsers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnbrowsers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnbrowsers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BuildbarnBrowserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var browser buildbarnv1alpha1.BuildbarnBrowser
	if err := r.Get(ctx, req.NamespacedName, &browser); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !browser.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(browser.ObjectMeta.Finalizers, browserFinalizerName) {
			browser.ObjectMeta.Finalizers = removeString(browser.ObjectMeta.Finalizers, browserFinalizerName)
			if err := r.Update(ctx, &browser); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !containsString(browser.ObjectMeta.Finalizers, browserFinalizerName) {
		browser.ObjectMeta.Finalizers = append(browser.ObjectMeta.Finalizers, browserFinalizerName)
		if err := r.Update(ctx, &browser); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &browser); err != nil {
		logger.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, &browser); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	// Reconcile Ingress if host is specified
	if browser.Spec.IngressHost != "" {
		if err := r.reconcileIngress(ctx, &browser); err != nil {
			logger.Error(err, "failed to reconcile ingress")
			return ctrl.Result{}, err
		}
	}

	// Update status
	if err := r.updateStatus(ctx, &browser); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuildbarnBrowserReconciler) reconcileDeployment(ctx context.Context, browser *buildbarnv1alpha1.BuildbarnBrowser) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      browser.Name,
			Namespace: browser.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		replicas := int32(3)
		if browser.Spec.Replicas != nil {
			replicas = *browser.Spec.Replicas
		}

		image := browser.Spec.Image
		if image == "" {
			image = "ghcr.io/buildbarn/bb-browser:latest"
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "browser",
					"component": browser.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "browser",
						"component": browser.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "browser",
							Image: image,
							Args:  []string{"/config/browser.jsonnet"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 7984, Protocol: corev1.ProtocolTCP},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "configs", MountPath: "/config/", ReadOnly: true},
							},
							Resources: browser.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{Name: "configs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: browser.Spec.ConfigMapName}}}},
					},
					ImagePullSecrets: browser.Spec.ImagePullSecrets,
					NodeSelector:     browser.Spec.NodeSelector,
					Tolerations:      browser.Spec.Tolerations,
				},
			},
		}

		return ctrl.SetControllerReference(browser, deployment, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("deployment reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnBrowserReconciler) reconcileService(ctx context.Context, browser *buildbarnv1alpha1.BuildbarnBrowser) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      browser.Name,
			Namespace: browser.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, service, func() error {
		serviceType := browser.Spec.ServiceType
		if serviceType == "" {
			serviceType = corev1.ServiceTypeClusterIP
		}

		service.Spec = corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app":       "browser",
				"component": browser.Name,
			},
			Ports: []corev1.ServicePort{
				{Port: 7984, Protocol: corev1.ProtocolTCP, Name: "http"},
			},
		}

		return ctrl.SetControllerReference(browser, service, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("service reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnBrowserReconciler) reconcileIngress(ctx context.Context, browser *buildbarnv1alpha1.BuildbarnBrowser) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      browser.Name,
			Namespace: browser.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		host := browser.Spec.IngressHost
		if host == "" {
			host = fmt.Sprintf("bb-browser.%s", browser.Namespace)
		}

		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressPath{
							Path:     "/",
							PathType: func() *networkingv1.PathType { p := networkingv1.PathTypePrefix; return &p }(),
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: browser.Name,
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

		return ctrl.SetControllerReference(browser, ingress, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("ingress reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnBrowserReconciler) updateStatus(ctx context.Context, browser *buildbarnv1alpha1.BuildbarnBrowser) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: browser.Name, Namespace: browser.Namespace}, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		browser.Status.State = "NotFound"
		browser.Status.Replicas = 0
		browser.Status.ReadyReplicas = 0
	} else {
		browser.Status.Replicas = deployment.Status.Replicas
		browser.Status.ReadyReplicas = deployment.Status.ReadyReplicas
		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			browser.Status.State = "Ready"
		} else {
			browser.Status.State = "NotReady"
		}
	}

	return r.Status().Update(ctx, browser)
}
