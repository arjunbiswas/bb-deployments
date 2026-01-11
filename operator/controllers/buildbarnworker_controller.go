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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	workerControllerName = "buildbarn-worker"
	workerFinalizerName  = "buildbarnworker.buildbarn.io"
)

// BuildbarnWorkerReconciler reconciles a BuildbarnWorker object
type BuildbarnWorkerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildbarnWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildbarnv1alpha1.BuildbarnWorker{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnworkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnworkers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnworkers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BuildbarnWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var worker buildbarnv1alpha1.BuildbarnWorker
	if err := r.Get(ctx, req.NamespacedName, &worker); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !worker.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(worker.ObjectMeta.Finalizers, workerFinalizerName) {
			worker.ObjectMeta.Finalizers = removeString(worker.ObjectMeta.Finalizers, workerFinalizerName)
			if err := r.Update(ctx, &worker); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !containsString(worker.ObjectMeta.Finalizers, workerFinalizerName) {
		worker.ObjectMeta.Finalizers = append(worker.ObjectMeta.Finalizers, workerFinalizerName)
		if err := r.Update(ctx, &worker); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &worker); err != nil {
		logger.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, &worker); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuildbarnWorkerReconciler) reconcileDeployment(ctx context.Context, worker *buildbarnv1alpha1.BuildbarnWorker) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      worker.Name,
			Namespace: worker.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		replicas := int32(8)
		if worker.Spec.Replicas != nil {
			replicas = *worker.Spec.Replicas
		}

		image := worker.Spec.Image
		if image == "" {
			image = "ghcr.io/buildbarn/bb-worker:latest"
		}

		runnerImage := worker.Spec.RunnerImage
		if runnerImage == "" {
			runnerImage = "ghcr.io/catthehacker/ubuntu:act-22.04"
		}

		runnerInstallerImage := worker.Spec.RunnerInstallerImage
		if runnerInstallerImage == "" {
			runnerInstallerImage = "ghcr.io/buildbarn/bb-runner-installer:latest"
		}

		instance := worker.Spec.Instance
		if instance == "" {
			instance = "ubuntu22-04"
		}

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "worker",
					"instance":  instance,
					"component": worker.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "worker",
						"instance":  instance,
						"component": worker.Name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "bb-runner-installer",
							Image: runnerInstallerImage,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "empty", MountPath: "/bb/"},
							},
						},
						{
							Name:  "volume-init",
							Image: "busybox:1.31.1-uclibc",
							Command: []string{"sh", "-c", "mkdir -pm 0777 /worker/build && mkdir -pm 0700 /worker/cache && chmod 0777 /worker"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "worker", MountPath: "/worker"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "worker",
							Image: image,
							Args:  []string{fmt.Sprintf("/config/worker-%s.jsonnet", instance)},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "configs", MountPath: "/config/", ReadOnly: true},
								{Name: "worker", MountPath: "/worker"},
							},
							Env: []corev1.EnvVar{
								{Name: "NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
								{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							},
							Resources: worker.Spec.Resources,
						},
						{
							Name:    "runner",
							Image:   runnerImage,
							Command: []string{"/bb/tini", "-v", "--", "/bb/bb_runner", fmt.Sprintf("/config/runner-%s.jsonnet", instance)},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                func() *int64 { u := int64(65534); return &u }(),
								AllowPrivilegeEscalation: func() *bool { b := false; return &b }(),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "configs", MountPath: "/config/", ReadOnly: true},
								{Name: "worker", MountPath: "/worker"},
								{Name: "empty", MountPath: "/bb", ReadOnly: true},
							},
							Resources: worker.Spec.RunnerResources,
						},
					},
					Volumes: []corev1.Volume{
						{Name: "empty", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						{Name: "configs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: worker.Spec.ConfigMapName}}}},
						{Name: "worker", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					},
					ImagePullSecrets: worker.Spec.ImagePullSecrets,
					NodeSelector:     worker.Spec.NodeSelector,
					Tolerations:       worker.Spec.Tolerations,
				},
			},
		}

		return ctrl.SetControllerReference(worker, deployment, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("deployment reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnWorkerReconciler) updateStatus(ctx context.Context, worker *buildbarnv1alpha1.BuildbarnWorker) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: worker.Name, Namespace: worker.Namespace}, deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		worker.Status.State = "NotFound"
		worker.Status.Replicas = 0
		worker.Status.ReadyReplicas = 0
	} else {
		worker.Status.Replicas = deployment.Status.Replicas
		worker.Status.ReadyReplicas = deployment.Status.ReadyReplicas
		if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
			worker.Status.State = "Ready"
		} else {
			worker.Status.State = "NotReady"
		}
	}

	return r.Status().Update(ctx, worker)
}
