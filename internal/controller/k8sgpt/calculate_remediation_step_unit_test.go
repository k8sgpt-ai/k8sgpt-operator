package k8sgpt

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestMutationNameIsStableAndBounded(t *testing.T) {
	ref := corev1.ObjectReference{
		APIVersion: "apps/v1", Kind: "Deployment", Namespace: "shop", Name: "checkout", UID: types.UID("deployment-uid"),
	}
	name := mutationName(strings.Repeat("a", 80), ref)
	if len(name) > 63 {
		t.Fatalf("mutation name length = %d, want <= 63", len(name))
	}
	if name != mutationName(strings.Repeat("a", 80), ref) {
		t.Fatal("mutation name must be stable")
	}
	other := ref
	other.UID = types.UID("other-uid")
	if name == mutationName(strings.Repeat("a", 80), other) {
		t.Fatal("different targets must not share a mutation name")
	}
}
