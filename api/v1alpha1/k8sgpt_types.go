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

type Backstage struct {
	Enabled bool `json:"enabled,omitempty"`
}

type SecretRef struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}

type ExtraOptionsRef struct {
	Backstage          *Backstage `json:"backstage,omitempty"`
	ServiceAccountIRSA string     `json:"serviceAccountIRSA,omitempty"`
}

type CredentialsRef struct {
	Name string `json:"name,omitempty"`
}

type RemoteCacheRef struct {
	Credentials *CredentialsRef   `json:"credentials,omitempty"`
	GCS         *GCSBackend       `json:"gcs,omitempty"`
	S3          *S3Backend        `json:"s3,omitempty"`
	Azure       *AzureBackend     `json:"azure,omitempty"`
	Interplex   *InterplexBackend `json:"interplex,omitempty"`
}

type InterplexBackend struct {
	Endpoint string `json:"endpoint,omitempty"`
}

type S3Backend struct {
	BucketName string `json:"bucketName,omitempty"`
	Region     string `json:"region,omitempty"`
}

type AzureBackend struct {
	StorageAccount string `json:"storageAccount,omitempty"`
	ContainerName  string `json:"containerName,omitempty"`
}

type Connection struct {
	Url  string `json:"url,omitempty"`
	Port int    `json:"port,omitempty"`
}

type CustomAnalyzer struct {
	Name       string      `json:"name,omitempty"`
	Connection *Connection `json:"connection,omitempty"`
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
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled"`
	// +kubebuilder:default:=5
	MaxRetries int `json:"maxRetries"`
}

type GitOpsConfiguration struct {
	GitHubOrganisationName string `json:"gitHubOrganisationName"`
	// Reference to the secret holding the GitHub token
	Secret *SecretRef `json:"secret,omitempty"`
}

type AutoRemediation struct {
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled"`

	GitOpsConfiguration *GitOpsConfiguration `json:"gitOpsConfiguration"`
	// Defaults to 10%
	// +kubebuilder:default="90"
	SimilarityRequirement string `json:"similarityRequirement"`
	// Support Pod, Deployment, Service and Ingress
	// +kubebuilder:default:={"Pod","Deployment","Service","Ingress"}
	Resources []string `json:"resources"`
}

type AISpec struct {
	AutoRemediation AutoRemediation `json:"autoRemediation,omitempty"`
	// +kubebuilder:default:=openai
	// +kubebuilder:validation:Enum=ibmwatsonxai;openai;localai;azureopenai;amazonbedrock;cohere;amazonsagemaker;google;googlevertexai;customrest
	Backend string   `json:"backend"`
	BackOff *BackOff `json:"backOff,omitempty"`
	BaseUrl string   `json:"baseUrl,omitempty"`
	Region  string   `json:"region,omitempty"`
	// +kubebuilder:default:=gpt-4o-mini
	Model   string     `json:"model,omitempty"`
	Engine  string     `json:"engine,omitempty"`
	Secret  *SecretRef `json:"secret,omitempty"`
	Enabled bool       `json:"enabled,omitempty"`
	// +kubebuilder:default:=true
	Anonymize *bool `json:"anonymized,omitempty"`
	// +kubebuilder:default:=english
	Language      string `json:"language,omitempty"`
	ProxyEndpoint string `json:"proxyEndpoint,omitempty"`
	ProviderId    string `json:"providerId,omitempty"`
	// +kubebuilder:default:="2048"
	MaxTokens string `json:"maxTokens,omitempty"`
	// +kubebuilder:default:="50"
	Topk string `json:"topk,omitempty"`
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

type AnalysisConfig struct {
	// Interval is the time between analysis runs
	// +kubebuilder:validation:Pattern=`^[0-9]+[smh]$`
	Interval string `json:"interval,omitempty"`
}

// K8sGPTSpec defines the desired state of K8sGPT
type K8sGPTSpec struct {
	Version string `json:"version,omitempty"`
	// +kubebuilder:default:=ghcr.io/k8sgpt-ai/k8sgpt
	Repository       string                       `json:"repository,omitempty"`
	ImagePullPolicy  corev1.PullPolicy            `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []ImagePullSecrets           `json:"imagePullSecrets,omitempty"`
	Resources        *corev1.ResourceRequirements `json:"resources,omitempty"`
	NoCache          bool                         `json:"noCache,omitempty"`
	CustomAnalyzers  []CustomAnalyzer             `json:"customAnalyzers,omitempty"`
	Filters          []string                     `json:"filters,omitempty"`
	ExtraOptions     *ExtraOptionsRef             `json:"extraOptions,omitempty"`
	Sink             *WebhookRef                  `json:"sink,omitempty"`
	AI               *AISpec                      `json:"ai,omitempty"`
	RemoteCache      *RemoteCacheRef              `json:"remoteCache,omitempty"`
	Integrations     *Integrations                `json:"integrations,omitempty"`
	NodeSelector     map[string]string            `json:"nodeSelector,omitempty"`
	TargetNamespace  string                       `json:"targetNamespace,omitempty"`
	Analysis         *AnalysisConfig              `json:"analysis,omitempty"`
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
	Google          = "google"
	GoogleVertexAI  = "googlevertexai"
	IBMWatsonxAI    = "ibmwatsonxai"
)

// K8sGPTStatus defines the observed state of K8sGPT
// show the current backend used
// +kubebuilder:printcolumn:name="Backend",type="string",JSONPath=".spec.ai.backend",description="The current backend used"
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
