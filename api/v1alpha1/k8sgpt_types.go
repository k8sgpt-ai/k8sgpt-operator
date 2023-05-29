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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type SecretRef struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}

type CredentialsRef struct {
	Name            string `json:"name,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_acess_key,omitempty"`
}

type RemoteCacheRef struct {
	Credentials *CredentialsRef `json:"credentials,omitempty"`
	BucketName  string          `json:"bucketName,omitempty"`
	Region      string          `json:"region,omitempty"`
}

// K8sGPTSpec defines the desired state of K8sGPT
type K8sGPTSpec struct {
	// +kubebuilder:default:=openai
	// +kubebuilder:validation:Enum=openai;localai;azureopenai
	Backend `json:"backend"`
	BaseUrl string `json:"baseUrl,omitempty"`
	// +kubebuilder:default:=gpt-3.5-turbo
	Model       string          `json:"model,omitempty"`
	Engine      string          `json:"engine,omitempty"`
	Secret      *SecretRef      `json:"secret,omitempty"`
	Version     string          `json:"version,omitempty"`
	EnableAI    bool            `json:"enableAI,omitempty"`
	NoCache     bool            `json:"noCache,omitempty"`
	Filters     []string        `json:"filters,omitempty"`
	RemoteCache *RemoteCacheRef `json:"remoteCache,omitempty"`
}

type Backend string

const (
	OpenAI      Backend = "openai"
	AzureOpenAI Backend = "azureopenai"
	LocalAI     Backend = "localai"
)

// K8sGPTStatus defines the observed state of K8sGPT
type K8sGPTStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// K8sGPT is the Schema for the k8sgpts API
type K8sGPT struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sGPTSpec   `json:"spec,omitempty"`
	Status K8sGPTStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sGPTList contains a list of K8sGPT
type K8sGPTList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sGPT `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sGPT{}, &K8sGPTList{})
}
