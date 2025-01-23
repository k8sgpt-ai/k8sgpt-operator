/*
Copyright 2023 K8sGPT Contributors.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MutationSpec defines the desired state of Mutation.
type MutationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	SimilarityScore     string                 `json:"similarityScore,omitempty"`
	ResourceGVK         string                 `json:"resourceGVK,omitempty"`
	ResourceRef         corev1.ObjectReference `json:"resource,omitempty"`
	ResultRef           corev1.ObjectReference `json:"result,omitempty"`
	OriginConfiguration string                 `json:"originConfiguration,omitempty"`
	TargetConfiguration string                 `json:"targetConfiguration,omitempty"`
}

// MutationStatus defines the observed state of Mutation.
type MutationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Phase   AutoRemediationPhase `json:"phase,omitempty"`
	Message string               `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

// Display in wide format the autoremediationphase status and similarity score
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.message",description="Updates of the autoremediation phase"
// +kubebuilder:printcolumn:name="Similarity Score",type="string",JSONPath=".spec.similarityScore",description="The similarity score of the autoremediation"
// Mutation is the Schema for the mutations API.
type Mutation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MutationSpec   `json:"spec,omitempty"`
	Status            MutationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MutationList contains a list of Mutation.
type MutationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mutation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mutation{}, &MutationList{})
}
