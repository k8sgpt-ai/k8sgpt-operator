package resources

import (
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
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
