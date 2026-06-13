package resources

import (
	"context"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_DeploymentShouldBeSynced(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	//
	// create deployment
	//
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-k8sgpt",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "k8sgpt",
							Image: "ghcr.io/k8sgpt-ai/k8sgpt:v0.3.8",
						},
					},
				},
			},
		},
	}

	// test
	err := doSync(ctx, fakeClient, deployment)
	require.NoError(t, err)

	existDeployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), existDeployment)

	// verify
	require.NoError(t, err)
	assert.NotNil(t, existDeployment)

	//
	// patch deployment
	//
	deploymentUpdated := deployment.DeepCopy()
	updatedImage := "ghcr.io/k8sgpt-ai/k8sgpt:latest"
	deploymentUpdated.Spec.Template.Spec.Containers[0].Image = updatedImage

	// test
	err = doSync(ctx, fakeClient, deploymentUpdated)
	require.NoError(t, err)
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), existDeployment)
	require.NoError(t, err)

	// verify
	assert.NotNil(t, existDeployment)
	assert.Equal(t, updatedImage, existDeployment.Spec.Template.Spec.Containers[0].Image)
}

func Test_ServiceAccountShouldNotBeSynced(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	//
	// create ServiceAccount
	//
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
		},
		AutomountServiceAccountToken: pointer.Bool(true),
	}

	// test
	err := doSync(ctx, fakeClient, serviceAccount)
	require.NoError(t, err)

	existSA := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceAccount), existSA)

	// verify
	require.NoError(t, err)
	assert.NotNil(t, existSA)

	//
	// patch ServiceAccount
	//
	saUpdated := existSA.DeepCopy()
	saUpdated.AutomountServiceAccountToken = nil

	// test
	err = doSync(ctx, fakeClient, saUpdated)
	require.NoError(t, err)
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(saUpdated), existSA)
	require.NoError(t, err)

	// verify
	assert.NotNil(t, existSA)
	assert.NotNil(t, existSA.AutomountServiceAccountToken)
}
func Test_GetDeploymentWithKubeconfigAndIRSA(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))

	// Create a fake client with a secret for testing
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-kubeconfig-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"kubeconfig": []byte("test-kubeconfig-content"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	// Test cases
	testCases := []struct {
		name                   string
		backend                string
		hasKubeconfig          bool
		hasIRSA                bool
		expectedServiceAccount string
	}{
		{
			name:                   "No kubeconfig, no IRSA",
			backend:                "openai",
			hasKubeconfig:          false,
			hasIRSA:                false,
			expectedServiceAccount: "test-sa",
		},
		{
			name:                   "With kubeconfig, no IRSA",
			backend:                "openai",
			hasKubeconfig:          true,
			hasIRSA:                false,
			expectedServiceAccount: "",
		},
		{
			name:                   "With kubeconfig, with IRSA, non-Bedrock backend",
			backend:                "openai",
			hasKubeconfig:          true,
			hasIRSA:                true,
			expectedServiceAccount: "",
		},
		{
			name:                   "With kubeconfig, with IRSA, Bedrock backend",
			backend:                "amazonbedrock",
			hasKubeconfig:          true,
			hasIRSA:                true,
			expectedServiceAccount: "test-sa",
		},
		{
			name:                   "With kubeconfig, no IRSA, Bedrock backend",
			backend:                "amazonbedrock",
			hasKubeconfig:          true,
			hasIRSA:                false,
			expectedServiceAccount: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a K8sGPT config for testing
			config := v1alpha1.K8sGPT{
				TypeMeta: metav1.TypeMeta{
					Kind:       "K8sGPT",
					APIVersion: "core.k8sgpt.ai/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-k8sgpt",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
				Spec: v1alpha1.K8sGPTSpec{
					Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
					Version:         "v0.4.1",
					ImagePullPolicy: v1.PullAlways,
					AI: &v1alpha1.AISpec{
						Backend:   tc.backend,
						Model:     "gpt-4o-mini",
						MaxTokens: "2048",
						Topk:      "50",
						Region: func() string {
							if tc.backend == "amazonbedrock" {
								return "us-east-1"
							}
							return ""
						}(),
					},
				},
			}

			// Add kubeconfig if needed
			if tc.hasKubeconfig {
				config.Spec.Kubeconfig = &v1alpha1.SecretRef{
					Name: "test-kubeconfig-secret",
					Key:  "kubeconfig",
				}
			}

			// Add IRSA if needed
			if tc.hasIRSA {
				config.Spec.ExtraOptions = &v1alpha1.ExtraOptionsRef{
					ServiceAccountIRSA: "arn:aws:iam::123456789012:role/test-role",
				}
			}

			// Call GetDeployment
			deployment, err := GetDeployment(config, tc.hasKubeconfig, fakeClient, "test-sa")
			require.NoError(t, err)

			// Verify service account setting
			assert.Equal(t, tc.expectedServiceAccount, deployment.Spec.Template.Spec.ServiceAccountName)

			// Verify kubeconfig volume mount if kubeconfig is specified
			if tc.hasKubeconfig {
				// Check for kubeconfig volume
				foundVolume := false
				for _, vol := range deployment.Spec.Template.Spec.Volumes {
					if vol.Name == "kubeconfig" {
						foundVolume = true
						assert.Equal(t, "test-kubeconfig-secret", vol.Secret.SecretName)
						break
					}
				}
				assert.True(t, foundVolume, "Kubeconfig volume not found")

				// Check for kubeconfig volume mount
				foundMount := false
				for _, mount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
					if mount.Name == "kubeconfig" {
						foundMount = true
						break
					}
				}
				assert.True(t, foundMount, "Kubeconfig volume mount not found")

				// Check for kubeconfig arg
				foundArg := false
				for _, arg := range deployment.Spec.Template.Spec.Containers[0].Args {
					if arg == "--kubeconfig=/tmp/test-k8sgpt/kubeconfig" {
						foundArg = true
						break
					}
				}
				assert.True(t, foundArg, "Kubeconfig arg not found")
			}
		})
	}
}

func Test_GetDeploymentWithFilters(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	testCases := []struct {
		name         string
		filters      []string
		expectedArgs []string
	}{
		{
			name:         "No filters specified",
			filters:      []string{},
			expectedArgs: []string{"serve"},
		},
		{
			name:         "Single filter",
			filters:      []string{"Pod"},
			expectedArgs: []string{"serve"},
		},
		{
			name:         "Multiple filters including Deployment",
			filters:      []string{"Pod", "Deployment", "Service"},
			expectedArgs: []string{"serve"},
		},
		{
			name:         "All common filters",
			filters:      []string{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress"},
			expectedArgs: []string{"serve"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := v1alpha1.K8sGPT{
				TypeMeta: metav1.TypeMeta{
					Kind:       "K8sGPT",
					APIVersion: "core.k8sgpt.ai/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-k8sgpt",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
				Spec: v1alpha1.K8sGPTSpec{
					Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
					Version:         "v0.4.1",
					ImagePullPolicy: v1.PullAlways,
					Filters:         tc.filters,
					AI: &v1alpha1.AISpec{
						Backend:   "openai",
						Model:     "gpt-4o-mini",
						MaxTokens: "2048",
						Topk:      "50",
					},
				},
			}

			deployment, err := GetDeployment(config, false, fakeClient, "test-sa")
			require.NoError(t, err)

			// Verify the args contain the expected values
			// Note: Filters are now passed via the gRPC API when analysis is requested,
			// not as command-line arguments to the serve command.
			assert.Equal(t, tc.expectedArgs, deployment.Spec.Template.Spec.Containers[0].Args,
				"Expected args to match for filter configuration: %v", tc.filters)
		})
	}
}

func findEnvVar(envs []v1.EnvVar, name string) *v1.EnvVar {
	for _, e := range envs {
		if e.Name == name {
			return &e
		}
	}
	return nil
}

func Test_GetDeploymentWithAzureAPIType(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	testCases := []struct {
		name             string
		backend          string
		azureAPIType     string
		expectError      bool
		expectedErrorMsg string
		expectedEnvValue string
	}{
		{
			name:             "AzureAPIType AZURE with azureopenai",
			backend:          "azureopenai",
			azureAPIType:     "AZURE",
			expectedEnvValue: "AZURE",
		},
		{
			name:             "AzureAPIType AZURE_AD with azureopenai",
			backend:          "azureopenai",
			azureAPIType:     "AZURE_AD",
			expectedEnvValue: "AZURE_AD",
		},
		{
			name:             "AzureAPIType CLOUDFLARE_AZURE with azureopenai",
			backend:          "azureopenai",
			azureAPIType:     "CLOUDFLARE_AZURE",
			expectedEnvValue: "CLOUDFLARE_AZURE",
		},
		{
			name:             "AzureAPIType with non-azure backend returns error",
			backend:          "openai",
			azureAPIType:     "AZURE",
			expectError:      true,
			expectedErrorMsg: "azureAPIType is supported only by azureopenai provider",
		},
		{
			name:         "Empty AzureAPIType with azureopenai is no-op",
			backend:      "azureopenai",
			azureAPIType: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := v1alpha1.K8sGPT{
				TypeMeta: metav1.TypeMeta{
					Kind:       "K8sGPT",
					APIVersion: "core.k8sgpt.ai/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-k8sgpt",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
				Spec: v1alpha1.K8sGPTSpec{
					Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
					Version:         "v0.4.1",
					ImagePullPolicy: v1.PullAlways,
					AI: &v1alpha1.AISpec{
						Backend:      tc.backend,
						Model:        "gpt-4o-mini",
						MaxTokens:    "2048",
						Topk:         "50",
						AzureAPIType: tc.azureAPIType,
					},
				},
			}

			deployment, err := GetDeployment(config, false, fakeClient, "test-sa")

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				return
			}

			require.NoError(t, err)
			envVar := findEnvVar(deployment.Spec.Template.Spec.Containers[0].Env, "K8SGPT_AZURE_API_TYPE")
			if tc.expectedEnvValue != "" {
				require.NotNil(t, envVar, "Expected K8SGPT_AZURE_API_TYPE env var to be set")
				assert.Equal(t, tc.expectedEnvValue, envVar.Value)
			} else {
				assert.Nil(t, envVar, "Expected K8SGPT_AZURE_API_TYPE env var to NOT be set")
			}
		})
	}
}

func Test_GetDeploymentWithCustomHeaders(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	testCases := []struct {
		name             string
		backend          string
		customHeaders    string
		expectedEnvValue string
	}{
		{
			name:             "CustomHeaders with openai",
			backend:          "openai",
			customHeaders:    "X-Custom:val1",
			expectedEnvValue: "X-Custom:val1",
		},
		{
			name:             "CustomHeaders with azureopenai",
			backend:          "azureopenai",
			customHeaders:    "Key1:Val1,Key2:Val2",
			expectedEnvValue: "Key1:Val1,Key2:Val2",
		},
		{
			name:          "Empty CustomHeaders is no-op",
			backend:       "openai",
			customHeaders: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := v1alpha1.K8sGPT{
				TypeMeta: metav1.TypeMeta{
					Kind:       "K8sGPT",
					APIVersion: "core.k8sgpt.ai/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-k8sgpt",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
				Spec: v1alpha1.K8sGPTSpec{
					Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
					Version:         "v0.4.1",
					ImagePullPolicy: v1.PullAlways,
					AI: &v1alpha1.AISpec{
						Backend:       tc.backend,
						Model:         "gpt-4o-mini",
						MaxTokens:     "2048",
						Topk:          "50",
						CustomHeaders: tc.customHeaders,
					},
				},
			}

			deployment, err := GetDeployment(config, false, fakeClient, "test-sa")
			require.NoError(t, err)

			envVar := findEnvVar(deployment.Spec.Template.Spec.Containers[0].Env, "K8SGPT_CUSTOM_HEADERS")
			if tc.expectedEnvValue != "" {
				require.NotNil(t, envVar, "Expected K8SGPT_CUSTOM_HEADERS env var to be set")
				assert.Equal(t, tc.expectedEnvValue, envVar.Value)
			} else {
				assert.Nil(t, envVar, "Expected K8SGPT_CUSTOM_HEADERS env var to NOT be set")
			}
		})
	}
}

func Test_GetDeploymentWithAzureAPITypeAndCustomHeaders(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	testCases := []struct {
		name             string
		backend          string
		azureAPIType     string
		customHeaders    string
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:          "Both set with azureopenai",
			backend:       "azureopenai",
			azureAPIType:  "AZURE_AD",
			customHeaders: "X-Key:val",
		},
		{
			name:             "AzureAPIType with wrong backend errors before CustomHeaders",
			backend:          "openai",
			azureAPIType:     "AZURE",
			customHeaders:    "X-Key:val",
			expectError:      true,
			expectedErrorMsg: "azureAPIType is supported only by azureopenai provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := v1alpha1.K8sGPT{
				TypeMeta: metav1.TypeMeta{
					Kind:       "K8sGPT",
					APIVersion: "core.k8sgpt.ai/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-k8sgpt",
					Namespace: "test-namespace",
					UID:       "test-uid",
				},
				Spec: v1alpha1.K8sGPTSpec{
					Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
					Version:         "v0.4.1",
					ImagePullPolicy: v1.PullAlways,
					AI: &v1alpha1.AISpec{
						Backend:       tc.backend,
						Model:         "gpt-4o-mini",
						MaxTokens:     "2048",
						Topk:          "50",
						AzureAPIType:  tc.azureAPIType,
						CustomHeaders: tc.customHeaders,
					},
				},
			}

			deployment, err := GetDeployment(config, false, fakeClient, "test-sa")

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				return
			}

			require.NoError(t, err)

			apiTypeEnv := findEnvVar(deployment.Spec.Template.Spec.Containers[0].Env, "K8SGPT_AZURE_API_TYPE")
			require.NotNil(t, apiTypeEnv, "Expected K8SGPT_AZURE_API_TYPE env var to be set")
			assert.Equal(t, tc.azureAPIType, apiTypeEnv.Value)

			headersEnv := findEnvVar(deployment.Spec.Template.Spec.Containers[0].Env, "K8SGPT_CUSTOM_HEADERS")
			require.NotNil(t, headersEnv, "Expected K8SGPT_CUSTOM_HEADERS env var to be set")
			assert.Equal(t, tc.customHeaders, headersEnv.Value)
		})
	}
}
