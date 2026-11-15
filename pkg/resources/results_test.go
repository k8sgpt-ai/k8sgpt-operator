package resources

import (
	"context"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHashResultContentIncludesExactTargetReference(t *testing.T) {
	result := v1alpha1.ResultSpec{
		Kind: "Deployment",
		Name: "default/web",
		TargetRef: &v1alpha1.ResultTargetReference{
			APIVersion:      "apps/v1",
			Kind:            "Deployment",
			Namespace:       "default",
			Name:            "web",
			UID:             types.UID("target-uid"),
			ResourceVersion: "1",
		},
	}

	firstHash := hashResultContent(result)
	result.TargetRef.ResourceVersion = "2"

	require.NotEqual(t, firstHash, hashResultContent(result))
}

func TestMapResultsKeepsDifferentKindsWithSameResourceName(t *testing.T) {
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{Name: "k8sgpt", Namespace: "k8sgpt"},
		Spec: v1alpha1.K8sGPTSpec{
			Filters: []string{"Deployment", "Service"},
			AI:      &v1alpha1.AISpec{Backend: "openai"},
		},
	}
	results := []v1alpha1.ResultSpec{
		{Kind: "Deployment", Name: "shop/checkout", Error: []v1alpha1.Failure{{Text: "deployment failure"}}},
		{Kind: "Service", Name: "shop/checkout", Error: []v1alpha1.Failure{{Text: "service failure"}}},
	}

	mapped, err := MapResults(integrations.Integrations{}, results, config)
	require.NoError(t, err)
	require.Len(t, mapped, 2)

	var kinds []string
	var names []string
	for _, result := range mapped {
		kinds = append(kinds, result.Spec.Kind)
		names = append(names, result.Name)
	}
	require.ElementsMatch(t, []string{"Deployment", "Service"}, kinds)
	require.NotEqual(t, names[0], names[1])
}

func TestMapResultsCreatesOneResultForEveryRequestedFilter(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{Name: "k8sgpt", Namespace: "k8sgpt"},
		Spec: v1alpha1.K8sGPTSpec{
			Filters: []string{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress"},
			AI:      &v1alpha1.AISpec{Backend: "openai"},
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&config).
		Build()

	// All analyzers use "<namespace>/<name>" as their Result.Name. Use the same
	// resource name for every kind to reproduce the collision from issue #752.
	results := make([]v1alpha1.ResultSpec, 0, len(config.Spec.Filters))
	for _, kind := range config.Spec.Filters {
		results = append(results, v1alpha1.ResultSpec{
			Kind:  kind,
			Name:  "shop/checkout",
			Error: []v1alpha1.Failure{{Text: kind + " failure"}},
		})
	}

	mapped, err := MapResults(integrations.Integrations{}, results, config)
	require.NoError(t, err)
	require.Len(t, mapped, len(results))

	for _, result := range mapped {
		_, err := CreateOrUpdateResult(context.Background(), k8sClient, result)
		require.NoError(t, err)
	}

	var resultList v1alpha1.ResultList
	require.NoError(t, k8sClient.List(context.Background(), &resultList, client.InNamespace(config.Namespace)))
	require.Len(t, resultList.Items, len(results))

	var kinds []string
	for _, result := range resultList.Items {
		kinds = append(kinds, result.Spec.Kind)
	}
	require.ElementsMatch(t, config.Spec.Filters, kinds)
}
