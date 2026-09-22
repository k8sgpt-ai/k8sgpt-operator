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

func TestCalculateSecretChecksum_Deterministic(t *testing.T) {
	// Test that the same secret data always produces the same checksum
	// regardless of map iteration order
	secret1 := &v1.Secret{
		Data: map[string][]byte{
			"password": []byte("my-secret-key"),
			"username": []byte("admin"),
			"token":    []byte("abc123"),
		},
	}

	secret2 := &v1.Secret{
		Data: map[string][]byte{
			"token":    []byte("abc123"),
			"username": []byte("admin"),
			"password": []byte("my-secret-key"),
		},
	}

	checksum1, err := calculateSecretChecksum(secret1)
	require.NoError(t, err)

	checksum2, err := calculateSecretChecksum(secret2)
	require.NoError(t, err)

	assert.Equal(t, checksum1, checksum2, "checksums should be identical regardless of map iteration order")
	assert.Len(t, checksum1, 64, "SHA-256 hex digest should be 64 characters")
}

func TestCalculateSecretChecksum_DifferentData(t *testing.T) {
	// Test that different secret data produces different checksums
	secret1 := &v1.Secret{
		Data: map[string][]byte{
			"password": []byte("old-password"),
		},
	}

	secret2 := &v1.Secret{
		Data: map[string][]byte{
			"password": []byte("new-password"),
		},
	}

	checksum1, err := calculateSecretChecksum(secret1)
	require.NoError(t, err)

	checksum2, err := calculateSecretChecksum(secret2)
	require.NoError(t, err)

	assert.NotEqual(t, checksum1, checksum2, "checksums should differ when secret data changes")
}

func TestCalculateSecretChecksum_MetadataDoesNotMatter(t *testing.T) {
	// Test that metadata changes don't affect the checksum
	// This documents WHY we use checksum instead of resourceVersion
	secret1 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          map[string]string{"team": "platform"},
			Annotations:     map[string]string{"owner": "alice"},
			ResourceVersion: "12345",
		},
		Data: map[string][]byte{
			"password": []byte("same-secret"),
		},
	}

	secret2 := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          map[string]string{"team": "sre"},
			Annotations:     map[string]string{"owner": "bob"},
			ResourceVersion: "67890",
		},
		Data: map[string][]byte{
			"password": []byte("same-secret"),
		},
	}

	checksum1, err := calculateSecretChecksum(secret1)
	require.NoError(t, err)

	checksum2, err := calculateSecretChecksum(secret2)
	require.NoError(t, err)

	assert.Equal(t, checksum1, checksum2, "checksums should be identical when only metadata differs")
}

func TestCalculateSecretChecksum_EmptyAndNil(t *testing.T) {
	tests := []struct {
		name   string
		secret *v1.Secret
	}{
		{
			name:   "nil secret",
			secret: nil,
		},
		{
			name: "empty secret data",
			secret: &v1.Secret{
				Data: map[string][]byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, err := calculateSecretChecksum(tt.secret)
			require.NoError(t, err)
			assert.Empty(t, checksum, "checksum should be empty for nil/empty secret")
		})
	}
}

func TestCalculateSecretChecksum_CollisionResistance(t *testing.T) {
	// This test guards against delimiter-collision bugs.
	// Under a naive key:value\n serialization, these two secrets produce identical
	// bytes ("a:x\nb:y\n"), so their hashes would collide.
	// The JSON-based implementation must produce distinct checksums.
	secret1 := &v1.Secret{
		Data: map[string][]byte{
			"a": []byte("x\nb:y"),
		},
	}

	secret2 := &v1.Secret{
		Data: map[string][]byte{
			"a": []byte("x"),
			"b": []byte("y"),
		},
	}

	checksum1, err := calculateSecretChecksum(secret1)
	require.NoError(t, err)

	checksum2, err := calculateSecretChecksum(secret2)
	require.NoError(t, err)

	assert.NotEqual(t, checksum1, checksum2,
		"distinct secret data must not produce the same checksum (delimiter-collision guard)")
}

func TestGetDeployment_WithSecretChecksum(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("my-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Secret: &v1alpha1.SecretRef{
					Name: "test-secret",
					Key:  "password",
				},
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
		},
	}

	deployment, err := GetDeployment(config, false, fakeClient, "default")
	require.NoError(t, err)

	// Verify checksum annotation exists
	checksum, exists := deployment.Spec.Template.Annotations[aiSecretChecksumAnnotation]
	assert.True(t, exists, "checksum annotation should exist")
	assert.NotEmpty(t, checksum, "checksum should not be empty")
	assert.Len(t, checksum, 64, "SHA-256 hex digest should be 64 characters")
}

func TestGetDeployment_SecretNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Secret: &v1alpha1.SecretRef{
					Name: "nonexistent-secret",
					Key:  "password",
				},
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
		},
	}

	// GetDeployment should fail when Secret doesn't exist
	_, err := GetDeployment(config, false, fakeClient, "default")
	assert.Error(t, err, "GetDeployment should fail when referenced Secret doesn't exist")
	assert.Contains(t, err.Error(), "failed to get AI Secret")
}

func TestGetDeployment_WithoutSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				// No secret reference - valid for some backends
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
		},
	}

	deployment, err := GetDeployment(config, false, fakeClient, "default")
	require.NoError(t, err)

	// Verify no checksum annotation when no secret is referenced
	_, exists := deployment.Spec.Template.Annotations[aiSecretChecksumAnnotation]
	assert.False(t, exists, "checksum annotation should not exist when no secret is referenced")
}

func TestGetDeployment_PreservesPodAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("my-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Secret: &v1alpha1.SecretRef{
					Name: "test-secret",
					Key:  "password",
				},
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
			PodAnnotations: map[string]string{
				"custom-annotation":    "custom-value",
				"prometheus.io/scrape": "true",
			},
		},
	}

	deployment, err := GetDeployment(config, false, fakeClient, "default")
	require.NoError(t, err)

	// Verify both user annotations and checksum annotation exist
	assert.Equal(t, "custom-value", deployment.Spec.Template.Annotations["custom-annotation"])
	assert.Equal(t, "true", deployment.Spec.Template.Annotations["prometheus.io/scrape"])

	checksum, exists := deployment.Spec.Template.Annotations[aiSecretChecksumAnnotation]
	assert.True(t, exists, "checksum annotation should exist alongside user annotations")
	assert.NotEmpty(t, checksum)
}

func TestGetDeployment_OperatorOwnsChecksumAnnotation(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("my-api-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Secret: &v1alpha1.SecretRef{
					Name: "test-secret",
					Key:  "password",
				},
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
			PodAnnotations: map[string]string{
				// User tries to set the operator-controlled annotation
				aiSecretChecksumAnnotation: "fake-user-value",
			},
		},
	}

	deployment, err := GetDeployment(config, false, fakeClient, "default")
	require.NoError(t, err)

	// Verify operator overwrites user-provided value
	actualChecksum := deployment.Spec.Template.Annotations[aiSecretChecksumAnnotation]
	assert.NotEqual(t, "fake-user-value", actualChecksum, "operator should overwrite user-provided checksum annotation")
	assert.Len(t, actualChecksum, 64, "operator should set correct SHA-256 checksum")
}

func TestSync_DestroyOp_SucceedsWhenSecretAlreadyDeleted(t *testing.T) {
	// Regression test for the DestroyOp ordering fix.
	// Before the fix, GetDeployment() was called unconditionally and returned an error
	// when the AI Secret was missing, blocking finalizer/resource cleanup on deletion.
	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()

	// Config references a Secret that does not exist in the fake client.
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Backend: "openai",
				Secret: &v1alpha1.SecretRef{
					Name: "already-deleted-secret",
					Key:  "password",
				},
			},
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.3.8",
		},
	}

	err := Sync(ctx, fakeClient, config, DestroyOp)
	assert.NoError(t, err, "DestroyOp must succeed even when the referenced AI Secret no longer exists")
}
