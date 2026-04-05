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

func Test_GetAWSConfigMap(t *testing.T) {
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
			AI: &v1alpha1.AISpec{
				Backend: "amazonbedrock",
				RoleArn: "arn:aws:iam::999999999999:role/bedrock-role",
			},
			ExtraOptions: &v1alpha1.ExtraOptionsRef{
				ServiceAccountIRSA: "arn:aws:iam::111111111111:role/irsa-role",
			},
		},
	}

	cm, err := GetAWSConfigMap(config)
	require.NoError(t, err)

	assert.Equal(t, "test-k8sgpt-aws-config", cm.Name)
	assert.Equal(t, "test-namespace", cm.Namespace)
	assert.Contains(t, cm.Data["config"], "arn:aws:iam::111111111111:role/irsa-role")
	assert.Contains(t, cm.Data["config"], "arn:aws:iam::999999999999:role/bedrock-role")
	assert.Contains(t, cm.Data["config"], "[profile source]")
	assert.Contains(t, cm.Data["config"], "[profile cross-account]")
	assert.Contains(t, cm.Data["config"], "source_profile = source")
	assert.Contains(t, cm.Data["config"], "web_identity_token_file = /var/run/secrets/eks.amazonaws.com/serviceaccount/token")
}

func Test_GetDeploymentWithBedrockCrossAccountRole(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	testCases := []struct {
		name              string
		roleArn           string
		irsaArn           string
		expectError       bool
		expectVolume      bool
		expectEnvVars     bool
	}{
		{
			name:          "No roleArn set",
			roleArn:       "",
			irsaArn:       "",
			expectError:   false,
			expectVolume:  false,
			expectEnvVars: false,
		},
		{
			name:          "roleArn set without serviceAccountIRSA returns error",
			roleArn:       "arn:aws:iam::999999999999:role/bedrock-role",
			irsaArn:       "",
			expectError:   true,
			expectVolume:  false,
			expectEnvVars: false,
		},
		{
			name:          "roleArn and serviceAccountIRSA both set",
			roleArn:       "arn:aws:iam::999999999999:role/bedrock-role",
			irsaArn:       "arn:aws:iam::111111111111:role/irsa-role",
			expectError:   false,
			expectVolume:  true,
			expectEnvVars: true,
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
						Backend:   "amazonbedrock",
						Model:     "anthropic.claude-3-sonnet-20240229-v1:0",
						MaxTokens: "2048",
						Topk:      "50",
						Region:    "us-east-1",
						RoleArn:   tc.roleArn,
					},
				},
			}
			if tc.irsaArn != "" {
				config.Spec.ExtraOptions = &v1alpha1.ExtraOptionsRef{
					ServiceAccountIRSA: tc.irsaArn,
				}
			}

			deployment, err := GetDeployment(config, false, fakeClient, "test-sa")
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tc.expectVolume {
				foundVolume := false
				for _, vol := range deployment.Spec.Template.Spec.Volumes {
					if vol.Name == "aws-config" {
						foundVolume = true
						assert.Equal(t, "test-k8sgpt-aws-config", vol.ConfigMap.Name)
						break
					}
				}
				assert.True(t, foundVolume, "aws-config volume not found")

				foundMount := false
				for _, mount := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
					if mount.Name == "aws-config" {
						foundMount = true
						assert.Equal(t, "/root/.aws", mount.MountPath)
						assert.True(t, mount.ReadOnly)
						break
					}
				}
				assert.True(t, foundMount, "aws-config volume mount not found")
			}

			if tc.expectEnvVars {
				envMap := make(map[string]string)
				for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
					envMap[e.Name] = e.Value
				}
				assert.Equal(t, "/root/.aws/config", envMap["AWS_CONFIG_FILE"])
				assert.Equal(t, "cross-account", envMap["AWS_PROFILE"])
			} else {
				for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
					assert.NotEqual(t, "AWS_PROFILE", e.Name)
					assert.NotEqual(t, "AWS_CONFIG_FILE", e.Name)
				}
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
