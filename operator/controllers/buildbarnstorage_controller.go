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
	storageControllerName = "buildbarn-storage"
	storageFinalizerName  = "buildbarnstorage.buildbarn.io"
)

// BuildbarnStorageReconciler reconciles a BuildbarnStorage object
type BuildbarnStorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildbarnStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildbarnv1alpha1.BuildbarnStorage{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnstorages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=buildbarn.io,resources=buildbarnstorages/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BuildbarnStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var storage buildbarnv1alpha1.BuildbarnStorage
	if err := r.Get(ctx, req.NamespacedName, &storage); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !storage.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(storage.ObjectMeta.Finalizers, storageFinalizerName) {
			storage.ObjectMeta.Finalizers = removeString(storage.ObjectMeta.Finalizers, storageFinalizerName)
			if err := r.Update(ctx, &storage); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !containsString(storage.ObjectMeta.Finalizers, storageFinalizerName) {
		storage.ObjectMeta.Finalizers = append(storage.ObjectMeta.Finalizers, storageFinalizerName)
		if err := r.Update(ctx, &storage); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile StatefulSet
	if err := r.reconcileStatefulSet(ctx, &storage); err != nil {
		logger.Error(err, "failed to reconcile statefulset")
		return ctrl.Result{}, err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, &storage); err != nil {
		logger.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, &storage); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuildbarnStorageReconciler) reconcileStatefulSet(ctx context.Context, storage *buildbarnv1alpha1.BuildbarnStorage) error {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storage.Name,
			Namespace: storage.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, statefulSet, func() error {
		replicas := int32(2)
		if storage.Spec.Replicas != nil {
			replicas = *storage.Spec.Replicas
		}

		image := storage.Spec.Image
		if image == "" {
			image = "ghcr.io/buildbarn/bb-storage:latest"
		}

		statefulSet.Spec = appsv1.StatefulSetSpec{
			ServiceName: storage.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "storage",
					"component": storage.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "storage",
						"component": storage.Name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "volume-init",
							Image: "busybox:1.31.1-uclibc",
							Command: []string{"sh", "-c", "mkdir -m 0700 -p /storage-cas/persistent_state /storage-ac/persistent_state"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cas", MountPath: "/storage-cas"},
								{Name: "ac", MountPath: "/storage-ac"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "storage",
							Image: image,
							Args:  []string{"/config/storage.jsonnet"},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8981, Protocol: corev1.ProtocolTCP},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "configs", MountPath: "/config/", ReadOnly: true},
								{Name: "cas", MountPath: "/storage-cas"},
								{Name: "ac", MountPath: "/storage-ac"},
							},
							Resources: storage.Spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{Name: "configs", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: storage.Spec.ConfigMapName}}}},
					},
					ImagePullSecrets: storage.Spec.ImagePullSecrets,
					NodeSelector:     storage.Spec.NodeSelector,
					Tolerations:      storage.Spec.Tolerations,
				},
			},
		}

		// Add volume claim templates if specified
		if len(storage.Spec.VolumeClaimTemplates) > 0 {
			statefulSet.Spec.VolumeClaimTemplates = storage.Spec.VolumeClaimTemplates
		} else {
			// Default volume claim templates
			statefulSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "cas"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: parseQuantity("33Gi")}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "ac"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: parseQuantity("1Gi")}},
					},
				},
			}
		}

		return ctrl.SetControllerReference(storage, statefulSet, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("statefulset reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnStorageReconciler) reconcileService(ctx context.Context, storage *buildbarnv1alpha1.BuildbarnStorage) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storage.Name,
			Namespace: storage.Namespace,
		},
	}

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Spec = corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector: map[string]string{
				"app":       "storage",
				"component": storage.Name,
			},
			Ports: []corev1.ServicePort{
				{Port: 8981, Protocol: corev1.ProtocolTCP, Name: "grpc"},
			},
		}

		return ctrl.SetControllerReference(storage, service, r.Scheme)
	})

	if err != nil {
		return err
	}

	if op != ctrl.OperationResultNone {
		log.FromContext(ctx).Info("service reconciled", "operation", op)
	}

	return nil
}

func (r *BuildbarnStorageReconciler) updateStatus(ctx context.Context, storage *buildbarnv1alpha1.BuildbarnStorage) error {
	statefulSet := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: storage.Name, Namespace: storage.Namespace}, statefulSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		storage.Status.State = "NotFound"
		storage.Status.Replicas = 0
		storage.Status.ReadyReplicas = 0
	} else {
		storage.Status.Replicas = statefulSet.Status.Replicas
		storage.Status.ReadyReplicas = statefulSet.Status.ReadyReplicas
		if statefulSet.Status.ReadyReplicas == statefulSet.Status.Replicas && statefulSet.Status.Replicas > 0 {
			storage.Status.State = "Ready"
		} else {
			storage.Status.State = "NotReady"
		}
	}

	return r.Status().Update(ctx, storage)
}

func parseQuantity(s string) corev1.ResourceQuantity {
	q, err := corev1.ParseQuantity(s)
	if err != nil {
		// Return a default quantity if parsing fails
		q, _ = corev1.ParseQuantity("1Gi")
	}
	return q
}
