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

// BuildbarnFrontendSpec defines the desired state of BuildbarnFrontend
type BuildbarnFrontendSpec struct {
	// Image is the container image to use for the frontend
	// +optional
	Image string `json:"image,omitempty"`

	// Replicas is the number of frontend replicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// ConfigMapName is the name of the ConfigMap containing the frontend configuration
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// ServiceType is the type of service to create (ClusterIP, LoadBalancer, NodePort)
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// ImagePullSecrets is a list of references to secrets in the same namespace
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Resources defines resource requests and limits for the frontend container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations allow the frontend to be scheduled onto nodes with matching taints
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// BuildbarnFrontendStatus defines the observed state of BuildbarnFrontend
type BuildbarnFrontendStatus struct {
	// State represents the current state of the frontend
	// +optional
	State string `json:"state,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Replicas is the total number of replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Conditions represent the latest available observations of the frontend's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of the BuildbarnFrontend"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas",description="Number of replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Number of ready replicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BuildbarnFrontend is the Schema for the buildbarnfrontends API
type BuildbarnFrontend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildbarnFrontendSpec   `json:"spec,omitempty"`
	Status BuildbarnFrontendStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuildbarnFrontendList contains a list of BuildbarnFrontend
type BuildbarnFrontendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildbarnFrontend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildbarnFrontend{}, &BuildbarnFrontendList{})
}
