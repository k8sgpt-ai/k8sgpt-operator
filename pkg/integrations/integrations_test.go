package integrations

import (
	"context"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_NewIntegrations(t *testing.T) {
	// Create a fake client
	cl := fake.NewClientBuilder().Build()

	// Call the function with the fake client and a context
	integrations, err := NewIntegrations(cl, context.Background())

	// Check that the function returned a non-nil Integrations object and no error
	assert.NotNil(t, integrations)
	assert.NoError(t, err)
}

func Test_BackstageLabel(t *testing.T) {
	testKind := &unstructured.Unstructured{}
	testKind.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "test-kind"})
	testKind.SetName("test")
	testKind.SetNamespace("default")
	testKind.SetLabels(map[string]string{backstageLabelKey: "test-value"})

	// Create a fake client
	cl := fake.NewClientBuilder().WithObjects(testKind).Build()
	// Create a fake RESTMapper
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
	restMapper.Add(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "test-kind"}, meta.RESTScopeNamespace)

	// Create an Integrations object with the fake client and a context
	i := &Integrations{
		client:     cl,
		ctx:        context.Background(),
		restMapper: restMapper,
	}

	testCases := []struct {
		name     string
		result   v1alpha1.ResultSpec
		expected map[string]string
	}{
		{
			name: "non-existing kind",
			result: v1alpha1.ResultSpec{
				Name: "default/test",
			},
			expected: map[string]string{},
		},
		{
			name: "existing kind and name with slash",
			result: v1alpha1.ResultSpec{
				Name: "",
				Kind: "test-kind",
			},
			expected: map[string]string{},
		},
		{
			name: "existing kind and name with slash",
			result: v1alpha1.ResultSpec{
				Name: "default/test",
				Kind: "test-kind",
			},
			expected: map[string]string{backstageLabelKey: "test-value"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := i.BackstageLabel(tc.result)
			assert.Equal(t, tc.expected, labels)
		})
	}
}
func Test_BackstageLabelNotExist(t *testing.T) {
	testKind := &unstructured.Unstructured{}
	testKind.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "test-kind"})
	testKind.SetName("test")
	testKind.SetNamespace("default")

	// Create a fake client
	cl := fake.NewClientBuilder().WithObjects(testKind).Build()
	// Create a fake RESTMapper
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
	restMapper.Add(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "test-kind"}, meta.RESTScopeNamespace)

	// Create an Integrations object with the fake client and a context
	i := &Integrations{
		client:     cl,
		ctx:        context.Background(),
		restMapper: restMapper,
	}

	labels := i.BackstageLabel(v1alpha1.ResultSpec{
		Name: "default/test",
		Kind: "test-kind",
	})
	assert.Equal(t, map[string]string{"backstage.io/kubernetes-id": ""}, labels)
}
