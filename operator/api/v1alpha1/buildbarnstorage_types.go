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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildbarnStorageSpec defines the desired state of BuildbarnStorage
type BuildbarnStorageSpec struct {
	// Image is the container image to use for the storage
	// +optional
	Image string `json:"image,omitempty"`

	// Replicas is the number of storage replicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// ConfigMapName is the name of the ConfigMap containing the storage configuration
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// ImagePullSecrets is a list of references to secrets in the same namespace
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Resources defines resource requests and limits for the storage container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations allow the storage to be scheduled onto nodes with matching taints
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// VolumeClaimTemplates defines persistent volume claims for the StatefulSet
	// +optional
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
}

// BuildbarnStorageStatus defines the observed state of BuildbarnStorage
type BuildbarnStorageStatus struct {
	// State represents the current state of the storage
	// +optional
	State string `json:"state,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Replicas is the total number of replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Conditions represent the latest available observations of the storage's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of the BuildbarnStorage"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas",description="Number of replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Number of ready replicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BuildbarnStorage is the Schema for the buildbarnstorages API
type BuildbarnStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildbarnStorageSpec   `json:"spec,omitempty"`
	Status BuildbarnStorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuildbarnStorageList contains a list of BuildbarnStorage
type BuildbarnStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildbarnStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildbarnStorage{}, &BuildbarnStorageList{})
}
