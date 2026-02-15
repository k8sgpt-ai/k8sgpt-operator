package resources

import (
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_GetDeploymentWithDefaultVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test case: version not specified (empty string)
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
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "", // Empty version - should default to "latest"
			AI: &v1alpha1.AISpec{
				Backend: "openai",
				Model:   "gpt-4",
			},
		},
	}

	// Call GetDeployment
	deployment, err := GetDeployment(config, false, fakeClient, "test-sa")
	require.NoError(t, err)

	// Verify the image uses "latest" tag when version is not specified
	expectedImage := "ghcr.io/k8sgpt-ai/k8sgpt:latest"
	actualImage := deployment.Spec.Template.Spec.Containers[0].Image
	assert.Equal(t, expectedImage, actualImage, "Expected image to use 'latest' tag when version is not specified")
}

func Test_GetDeploymentWithExplicitVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test case: version explicitly specified
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
			Repository: "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:    "v0.4.27",
			AI: &v1alpha1.AISpec{
				Backend: "openai",
				Model:   "gpt-4",
			},
		},
	}

	// Call GetDeployment
	deployment, err := GetDeployment(config, false, fakeClient, "test-sa")
	require.NoError(t, err)

	// Verify the image uses the specified version
	expectedImage := "ghcr.io/k8sgpt-ai/k8sgpt:v0.4.27"
	actualImage := deployment.Spec.Template.Spec.Containers[0].Image
	assert.Equal(t, expectedImage, actualImage, "Expected image to use the specified version tag")
}
