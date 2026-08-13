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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Failure struct {
	Text      string      `json:"text,omitempty"`
	Sensitive []Sensitive `json:"sensitive,omitempty"`
}

type Sensitive struct {
	Unmasked string `json:"unmasked,omitempty"`
	Masked   string `json:"masked,omitempty"`
}

// Enum for Phase
type AutoRemediationPhase int

// The auto remediation phase will begin in a not-started phase, after an evaluation has been made to remediate the resource
// This is decoupled from the executor of the phase which will put it into in-progress.
// Completion will be based on the resource stablising and the result being asked for deletion
// Upon which the AutoRemediation will be checked. It is not expected Results with a completed phase will be kept, only in circumstances of DR or restart.
const (
	AutoRemediationPhaseNotStarted AutoRemediationPhase = iota
	AutoRemediationPhaseInProgress
	AutoRemediationPhaseCompleted
	AutoRemediationPhaseSuccessful
	AutoRemediationPending
	AutoRemediationAborted = 5
)

type AutoRemediationStatus struct {
	Phase AutoRemediationPhase `json:"phase,omitempty"`
}

// ResultTargetReference identifies the exact Kubernetes object that produced a
// result. All fields are optional while analyzers migrate from the legacy
// Kind/Name representation, but auto-remediation must require a complete,
// matching reference before mutating an object.
//
// This intentionally contains only object identity and concurrency fields. It
// is not a Kubernetes ObjectReference because references such as FieldPath and
// Controller are not meaningful for an analyzer finding.
type ResultTargetReference struct {
	APIVersion      string    `json:"apiVersion,omitempty"`
	Kind            string    `json:"kind,omitempty"`
	Namespace       string    `json:"namespace,omitempty"`
	Name            string    `json:"name,omitempty"`
	UID             types.UID `json:"uid,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
}

// ResultSpec defines the desired state of Result
type ResultSpec struct {
	Backend               string                `json:"backend"`
	AutoRemediationStatus AutoRemediationStatus `json:"autoRemediationStatus"`
	Kind                  string                `json:"kind"`
	Name                  string                `json:"name"`
	Error                 []Failure             `json:"error"`
	Details               string                `json:"details"`
	ParentObject          string                `json:"parentObject"`
	// TargetRef is the optional exact identity of the resource that produced this result.
	// Kind and Name remain for backwards compatibility with existing analyzers.
	TargetRef *ResultTargetReference `json:"targetRef,omitempty"`
}

// ResultStatus defines the observed state of Result
type ResultStatus struct {
	LifeCycle   string `json:"lifecycle,omitempty"`
	Webhook     string `json:"webhook,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.kind",description="Kind"
// +kubebuilder:printcolumn:name="Backend",type="string",JSONPath=".spec.backend",description="Backend"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age"
// Result is the Schema for the results API
type Result struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResultSpec   `json:"spec,omitempty"`
	Status ResultStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ResultList contains a list of Result
type ResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Result `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Result{}, &ResultList{})
}
