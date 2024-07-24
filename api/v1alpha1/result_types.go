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
)

type Failure struct {
	Text      string      `json:"text,omitempty"`
	Sensitive []Sensitive `json:"sensitive,omitempty"`
}

type Sensitive struct {
	Unmasked string `json:"unmasked,omitempty"`
	Masked   string `json:"masked,omitempty"`
}

// ResultSpec defines the desired state of Result
type ResultSpec struct {
	Backend      string    `json:"backend"`
	Kind         string    `json:"kind"`
	Name         string    `json:"name"`
	Error        []Failure `json:"error"`
	Details      string    `json:"details"`
	ParentObject string    `json:"parentObject"`
}

// ResultStatus defines the observed state of Result
type ResultStatus struct {
	LifeCycle string `json:"lifecycle,omitempty"`
	Webhook   string `json:"webhook,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.kind",description="Kind"
//+kubebuilder:printcolumn:name="Backend",type="string",JSONPath=".spec.backend",description="Backend"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age"

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
