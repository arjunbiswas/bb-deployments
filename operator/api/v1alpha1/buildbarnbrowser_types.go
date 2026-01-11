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

// BuildbarnBrowserSpec defines the desired state of BuildbarnBrowser
type BuildbarnBrowserSpec struct {
	// Image is the container image to use for the browser
	// +optional
	Image string `json:"image,omitempty"`

	// Replicas is the number of browser replicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// ConfigMapName is the name of the ConfigMap containing the browser configuration
	// +optional
	ConfigMapName string `json:"configMapName,omitempty"`

	// ServiceType is the type of service to create (ClusterIP, LoadBalancer, NodePort)
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// IngressHost is the hostname for the ingress (optional)
	// +optional
	IngressHost string `json:"ingressHost,omitempty"`

	// ImagePullSecrets is a list of references to secrets in the same namespace
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Resources defines resource requests and limits for the browser container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations allow the browser to be scheduled onto nodes with matching taints
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// BuildbarnBrowserStatus defines the observed state of BuildbarnBrowser
type BuildbarnBrowserStatus struct {
	// State represents the current state of the browser
	// +optional
	State string `json:"state,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Replicas is the total number of replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Conditions represent the latest available observations of the browser's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State of the BuildbarnBrowser"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas",description="Number of replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Number of ready replicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BuildbarnBrowser is the Schema for the buildbarnbrowsers API
type BuildbarnBrowser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildbarnBrowserSpec   `json:"spec,omitempty"`
	Status BuildbarnBrowserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuildbarnBrowserList contains a list of BuildbarnBrowser
type BuildbarnBrowserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildbarnBrowser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildbarnBrowser{}, &BuildbarnBrowserList{})
}
