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
