package conversions

import (
	"testing"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestResultsToEligibleResourcesHonorsConfiguredResourceKinds(t *testing.T) {
	scheme := testScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "default"}},
	).Build()
	config := &corev1alpha1.K8sGPT{
		Spec: corev1alpha1.K8sGPTSpec{AI: &corev1alpha1.AISpec{
			AutoRemediation: corev1alpha1.AutoRemediation{Resources: []string{"Pod"}},
		}},
	}
	results := &corev1alpha1.ResultList{Items: []corev1alpha1.Result{
		result("pod-result", "Pod", "default/pod"),
		result("deployment-result", "Deployment", "default/deployment"),
	}}

	eligible := ResultsToEligibleResources(config, client, scheme, discardLogger(), results)

	if len(eligible) != 1 {
		t.Fatalf("eligible resources = %d, want 1", len(eligible))
	}
	if eligible[0].ObjectRef.Kind != "Pod" {
		t.Errorf("eligible resource kind = %q, want Pod", eligible[0].ObjectRef.Kind)
	}
}

func TestResultsToEligibleResourcesUsesExactConfiguredKindMatch(t *testing.T) {
	scheme := testScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default"}},
	).Build()
	config := &corev1alpha1.K8sGPT{
		Spec: corev1alpha1.K8sGPTSpec{AI: &corev1alpha1.AISpec{
			AutoRemediation: corev1alpha1.AutoRemediation{Resources: []string{"Pods"}},
		}},
	}

	eligible := ResultsToEligibleResources(config, client, scheme, discardLogger(), &corev1alpha1.ResultList{
		Items: []corev1alpha1.Result{result("pod-result", "Pod", "default/pod")},
	})

	if len(eligible) != 0 {
		t.Fatalf("eligible resources = %d, want 0 for non-exact configured kind", len(eligible))
	}
}

func TestResultsToEligibleResourcesResolvesPodFindingToEnabledDeployment(t *testing.T) {
	scheme := testScheme(t)
	controller := true
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "default"}}
	replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "checkout-abc", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployment.Name, Controller: &controller}}}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "checkout-abc-123", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: replicaSet.Name, Controller: &controller}}}}
	rc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment, replicaSet, pod).Build()
	eligible := ResultsToEligibleResources(remediationConfig("Deployment"), rc, scheme, discardLogger(), &corev1alpha1.ResultList{Items: []corev1alpha1.Result{result("pod-result", "Pod", "default/checkout-abc-123")}})
	if len(eligible) != 1 {
		t.Fatalf("eligible resources = %d, want 1", len(eligible))
	}
	if got := eligible[0].ObjectRef; got.Kind != "Deployment" || got.Name != "checkout" || got.APIVersion != "apps/v1" {
		t.Errorf("resolved target = %#v, want apps/v1 Deployment checkout", got)
	}
}

func TestResultsToEligibleResourcesRejectsMalformedResultNames(t *testing.T) {
	scheme := testScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	config := &corev1alpha1.K8sGPT{
		Spec: corev1alpha1.K8sGPTSpec{AI: &corev1alpha1.AISpec{
			AutoRemediation: corev1alpha1.AutoRemediation{Resources: []string{"Pod"}},
		}},
	}

	for _, name := range []string{"", "pod", "default/pod/extra"} {
		t.Run(name, func(t *testing.T) {
			eligible := ResultsToEligibleResources(config, client, scheme, discardLogger(), &corev1alpha1.ResultList{
				Items: []corev1alpha1.Result{result("pod-result", "Pod", name)},
			})
			if len(eligible) != 0 {
				t.Fatalf("eligible resources = %d, want 0", len(eligible))
			}
		})
	}
}

func TestResultsToEligibleResourcesSkipsUnsupportedConfiguredKinds(t *testing.T) {
	scheme := testScheme(t)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "default"}},
	).Build()
	config := &corev1alpha1.K8sGPT{
		Spec: corev1alpha1.K8sGPTSpec{AI: &corev1alpha1.AISpec{
			AutoRemediation: corev1alpha1.AutoRemediation{Resources: []string{"Service"}},
		}},
	}

	eligible := ResultsToEligibleResources(config, client, scheme, discardLogger(), &corev1alpha1.ResultList{
		Items: []corev1alpha1.Result{result("service-result", "Service", "default/service")},
	})

	if len(eligible) != 0 {
		t.Fatalf("eligible resources = %d, want 0 for unsupported execution kind", len(eligible))
	}
}

func TestResultsToEligibleResourcesResolvesGenericQualifiedTargetRef(t *testing.T) {
	scheme := testScheme(t)
	rc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "settings", Namespace: "default"}},
	).Build()
	config := remediationConfig("v1/ConfigMap")
	item := result("config-result", "ConfigMap", "ignored/legacy")
	item.Spec.TargetRef = &corev1alpha1.ResultTargetReference{
		APIVersion: "v1", Kind: "ConfigMap", Namespace: "default", Name: "settings",
	}

	eligible := ResultsToEligibleResources(config, rc, scheme, discardLogger(), &corev1alpha1.ResultList{Items: []corev1alpha1.Result{item}})
	if len(eligible) != 1 {
		t.Fatalf("eligible resources = %d, want 1", len(eligible))
	}
	if got := eligible[0].ObjectRef.APIVersion; got != "v1" {
		t.Errorf("apiVersion = %q, want v1", got)
	}
	if got := eligible[0].GVK; got != "v1" {
		t.Errorf("GVK = %q, want v1", got)
	}
}

func TestResultsToEligibleResourcesResolvesLegacyResultWithUniqueFullyQualifiedSelector(t *testing.T) {
	scheme := testScheme(t)
	rc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "settings", Namespace: "default"}},
	).Build()
	item := result("config-result", "ConfigMap", "default/settings")
	eligible := ResultsToEligibleResources(remediationConfig("v1/ConfigMap"), rc, scheme, discardLogger(), &corev1alpha1.ResultList{Items: []corev1alpha1.Result{item}})
	if len(eligible) != 1 || eligible[0].ObjectRef.APIVersion != "v1" || eligible[0].ObjectRef.Kind != "ConfigMap" {
		t.Fatalf("legacy fully-qualified ConfigMap target was not resolved: %#v", eligible)
	}
}

func TestResultsToEligibleResourcesRejectsAmbiguousLegacySelectors(t *testing.T) {
	scheme := testScheme(t)
	rc := fake.NewClientBuilder().WithScheme(scheme).Build()
	item := result("config-result", "ConfigMap", "default/settings")
	eligible := ResultsToEligibleResources(remediationConfig("v1/ConfigMap", "example.io/v1/ConfigMap"), rc, scheme, discardLogger(), &corev1alpha1.ResultList{Items: []corev1alpha1.Result{item}})
	if len(eligible) != 0 {
		t.Fatalf("ambiguous legacy target must not be resolved: %#v", eligible)
	}
}

func TestResultsToEligibleResourcesResolvesClusterScopedTargetRef(t *testing.T) {
	scheme := testScheme(t)
	rc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "production"}},
	).Build()
	config := remediationConfig("v1/Namespace")
	item := result("namespace-result", "Namespace", "not/a-valid-target")
	item.Spec.TargetRef = &corev1alpha1.ResultTargetReference{
		APIVersion: "v1", Kind: "Namespace", Name: "production",
	}

	eligible := ResultsToEligibleResources(config, rc, scheme, discardLogger(), &corev1alpha1.ResultList{Items: []corev1alpha1.Result{item}})
	if len(eligible) != 1 {
		t.Fatalf("eligible resources = %d, want 1", len(eligible))
	}
	if got := eligible[0].ObjectRef.Namespace; got != "" {
		t.Errorf("namespace = %q, want cluster-scoped empty namespace", got)
	}
}

func TestAutoRemediationResourceEnabledSelectors(t *testing.T) {
	config := remediationConfig("Deployment", "apps/StatefulSet", "apps/v1/DaemonSet")
	for _, tc := range []struct {
		gvk  schema.GroupVersionKind
		want bool
	}{
		{schema.GroupVersion{Group: "apps", Version: "v1"}.WithKind("Deployment"), true},
		{schema.GroupVersion{Group: "apps", Version: "v1"}.WithKind("StatefulSet"), true},
		{schema.GroupVersion{Group: "apps", Version: "v1"}.WithKind("DaemonSet"), true},
		{schema.GroupVersion{Group: "apps", Version: "v1"}.WithKind("ReplicaSet"), false},
	} {
		if got := AutoRemediationResourceEnabled(config, tc.gvk); got != tc.want {
			t.Errorf("enabled(%s) = %v, want %v", tc.gvk, got, tc.want)
		}
	}
}

func remediationConfig(resources ...string) *corev1alpha1.K8sGPT {
	return &corev1alpha1.K8sGPT{Spec: corev1alpha1.K8sGPTSpec{AI: &corev1alpha1.AISpec{
		AutoRemediation: corev1alpha1.AutoRemediation{Resources: resources},
	}}}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func result(name, kind, resourceName string) corev1alpha1.Result {
	return corev1alpha1.Result{
		TypeMeta:   metav1.TypeMeta{APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Result"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       corev1alpha1.ResultSpec{Kind: kind, Name: resourceName},
	}
}

func discardLogger() logr.Logger {
	return logr.Discard()
}
