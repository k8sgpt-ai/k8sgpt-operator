package resources

import (
	"context"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_MapResults(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			AI: &v1alpha1.AISpec{
				Backend: "backend",
			},
			ExtraOptions: &v1alpha1.ExtraOptionsRef{
				Backstage: &v1alpha1.Backstage{
					Enabled: true,
				},
			},
		},
	}

	// Create a ResultSpec slice
	resultsSpec := []v1alpha1.ResultSpec{
		{
			Name: "result-1",
		},
		{
			Name: "result-2",
		},
	}

	c := fake.NewFakeClient()

	// Create an Integrations object
	i, err := integrations.NewIntegrations(c, context.Background())
	assert.NoError(t, err)

	// Call MapResults
	results, err := MapResults(*i, resultsSpec, config)
	assert.NoError(t, err)

	// Check that the expected results were returned
	assert.Equal(t, 2, len(results))
	assert.Contains(t, results, "result1")
	assert.Contains(t, results, "result2")
}

func Test_CreateOrUpdateResult(t *testing.T) {
	// Create a fake client
	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.Result{})
	cl := fake.NewFakeClientWithScheme(s)

	// Create a new result object
	res := &v1alpha1.Result{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"test": "test",
			},
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.ResultSpec{
			Error: []v1alpha1.Failure{
				{
					Text: "error",
				},
			},
		},
	}

	// Test creating a new result
	op, err := CreateOrUpdateResult(context.Background(), cl, *res)
	assert.NoError(t, err)
	assert.Equal(t, CreatedResult, op)
}