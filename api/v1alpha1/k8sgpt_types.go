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

type Backstage struct {
	Enabled bool `json:"enabled,omitempty"`
}

type SecretRef struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}

type ExtraOptionsRef struct {
	Backstage *Backstage `json:"backstage,omitempty"`
}

type CredentialsRef struct {
	Name string `json:"name,omitempty"`
}

type RemoteCacheRef struct {
	Credentials *CredentialsRef `json:"credentials,omitempty"`
	GCS         *GCSBackend     `json:"gcs,omitempty"`
	S3          *S3Backend      `json:"s3,omitempty"`
	Azure       *AzureBackend   `json:"azure,omitempty"`
}

type S3Backend struct {
	BucketName string `json:"bucketName,omitempty"`
	Region     string `json:"region,omitempty"`
}

type AzureBackend struct {
	StorageAccount string `json:"storageAccount,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
}

type GCSBackend struct {
	BucketName string `json:"bucketName,omitempty"`
	Region     string `json:"region,omitempty"`
	ProjectId  string `json:"projectId,omitempty"`
}

type WebhookRef struct {
	// +kubebuilder:validation:Enum=slack;mattermost
	Type     string     `json:"type,omitempty"`
	Endpoint string     `json:"webhook,omitempty"`
	Channel  string     `json:"channel,omitempty"`
	UserName string     `json:"username,omitempty"`
	IconURL  string     `json:"icon_url,omitempty"`
	Secret   *SecretRef `json:"secret,omitempty"`
}

type BackOff struct {
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`
	// +kubebuilder:default:=5
	MaxRetries int `json:"maxRetries"`
}

type AISpec struct {
	// +kubebuilder:default:=openai
	// +kubebuilder:validation:Enum=openai;localai;azureopenai;amazonbedrock;cohere;amazonsagemaker
	Backend string   `json:"backend"`
	BackOff *BackOff `json:"backOff,omitempty"`
	BaseUrl string   `json:"baseUrl,omitempty"`
	// +kubebuilder:default:=gpt-3.5-turbo
	Model   string     `json:"model,omitempty"`
	Engine  string     `json:"engine,omitempty"`
	Secret  *SecretRef `json:"secret,omitempty"`
	Enabled bool       `json:"enabled,omitempty"`
	// +kubebuilder:default:=true
	Anonymize bool `json:"anonymized,omitempty"`
	// +kubebuilder:default:=english
	Language string `json:"language,omitempty"`
}

type Trivy struct {
	Enabled     bool   `json:"enabled,omitempty"`
	SkipInstall bool   `json:"skipInstall,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}
type Integrations struct {
	Trivy *Trivy `json:"trivy,omitempty"`
}

type ImagePullSecrets struct {
	Name string `json:"name,omitempty"`
}

// K8sGPTSpec defines the desired state of K8sGPT
type K8sGPTSpec struct {
	Version string `json:"version,omitempty"`
	// +kubebuilder:default:=ghcr.io/k8sgpt-ai/k8sgpt
	Repository       string             `json:"repository,omitempty"`
	ImagePullSecrets []ImagePullSecrets `json:"imagePullSecrets,omitempty"`
	NoCache          bool               `json:"noCache,omitempty"`
	Filters          []string           `json:"filters,omitempty"`
	ExtraOptions     *ExtraOptionsRef   `json:"extraOptions,omitempty"`
	Sink             *WebhookRef        `json:"sink,omitempty"`
	AI               *AISpec            `json:"ai,omitempty"`
	RemoteCache      *RemoteCacheRef    `json:"remoteCache,omitempty"`
	Integrations     *Integrations      `json:"integrations,omitempty"`
	NodeSelector     map[string]string  `json:"nodeSelector,omitempty"`
	TargetNamespace  string             `json:"targetNamespace,omitempty"`
	// Define the kubeconfig the Deployment must use.
	// If empty, the Deployment will use the ServiceAccount provided by Kubernetes itself.
	Kubeconfig *SecretRef `json:"kubeconfig,omitempty"`
}

const (
	OpenAI          = "openai"
	AzureOpenAI     = "azureopenai"
	LocalAI         = "localai"
	AmazonBedrock   = "amazonbedrock"
	AmazonSageMaker = "AmazonSageMaker"
	Cohere          = "cohere"
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
