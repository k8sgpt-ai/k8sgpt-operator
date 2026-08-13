package conversions

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/remediation/policy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestResourceToExecutionAppliesOnlyPolicyApprovedPatch(t *testing.T) {
	ctx := context.Background()
	live := testDeployment(t, "registry.example/checkout:broken")
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(live).Build()
	live = getDeployment(t, ctx, c)
	origin := marshalObject(t, live)
	proposed := live.DeepCopy()
	setDeploymentImage(t, proposed, "registry.example/checkout:fixed")
	mutation := testMutation(origin)
	if err := c.Create(ctx, &mutation); err != nil {
		t.Fatal(err)
	}

	result, err := ResourceToExecution(ObjectExecutionConfig{Ctx: ctx, Rc: c, Obj: proposed, Mutation: mutation, Log: logr.Discard()})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter == 0 {
		var rejected corev1alpha1.Mutation
		if err := c.Get(ctx, client.ObjectKeyFromObject(&mutation), &rejected); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("successful execution must requeue for verification; mutation = %#v", rejected.Status)
	}
	updated := getDeployment(t, ctx, c)
	image := deploymentImage(t, updated)
	if image != "registry.example/checkout:fixed" {
		t.Fatalf("image = %q, want fixed image", image)
	}
	var updatedMutation corev1alpha1.Mutation
	if err := c.Get(ctx, client.ObjectKeyFromObject(&mutation), &updatedMutation); err != nil {
		t.Fatal(err)
	}
	if updatedMutation.Status.Phase != corev1alpha1.AutoRemediationPhaseCompleted {
		t.Fatalf("mutation phase = %v, want completed", updatedMutation.Status.Phase)
	}
	if updatedMutation.Status.PolicyDecision != "approved" || len(updatedMutation.Status.ChangedPaths) != 1 || updatedMutation.Status.AppliedAt == nil {
		t.Fatalf("missing policy audit evidence: %#v", updatedMutation.Status)
	}
}

func TestResourceToExecutionAbortsForbiddenChangeWithoutMutatingTarget(t *testing.T) {
	ctx := context.Background()
	live := testDeployment(t, "registry.example/checkout:broken")
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(live).Build()
	live = getDeployment(t, ctx, c)
	origin := marshalObject(t, live)
	proposed := live.DeepCopy()
	setDeploymentImage(t, proposed, "registry.example/checkout:fixed")
	if err := unstructured.SetNestedField(proposed.Object, int64(3), "spec", "replicas"); err != nil {
		t.Fatal(err)
	}
	mutation := testMutation(origin)
	if err := c.Create(ctx, &mutation); err != nil {
		t.Fatal(err)
	}

	if _, err := ResourceToExecution(ObjectExecutionConfig{Ctx: ctx, Rc: c, Obj: proposed, Mutation: mutation, Log: logr.Discard()}); err != nil {
		t.Fatal(err)
	}
	updated := getDeployment(t, ctx, c)
	if image := deploymentImage(t, updated); image != "registry.example/checkout:broken" {
		t.Fatalf("forbidden proposal changed image to %q", image)
	}
	var updatedMutation corev1alpha1.Mutation
	if err := c.Get(ctx, client.ObjectKeyFromObject(&mutation), &updatedMutation); err != nil {
		t.Fatal(err)
	}
	if updatedMutation.Status.Phase != corev1alpha1.AutoRemediationAborted || !strings.Contains(updatedMutation.Status.Message, "Policy rejected") {
		t.Fatalf("mutation was not rejected: %#v", updatedMutation.Status)
	}
	if updatedMutation.Status.PolicyDecision != "rejected" || len(updatedMutation.Status.ChangedPaths) != 2 {
		t.Fatalf("rejection audit evidence missing: %#v", updatedMutation.Status)
	}
}

func TestResourceToExecutionAppliesExplicitConfigMapPathOptIn(t *testing.T) {
	ctx := context.Background()
	live, err := policy.ParseYAMLProposal([]byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: shop
  annotations:
    core.k8sgpt.ai/auto-remediation-allowed-paths: /data/log-level
data:
  log-level: info
`))
	if err != nil {
		t.Fatal(err)
	}
	live.SetUID(types.UID("settings-uid"))
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(live).Build()
	live = getObject(t, ctx, c, "v1", "ConfigMap", "shop", "settings")
	origin := marshalObject(t, live)
	proposed := live.DeepCopy()
	if err := unstructured.SetNestedField(proposed.Object, "debug", "data", "log-level"); err != nil {
		t.Fatal(err)
	}
	mutation := testMutation(origin)
	if err := c.Create(ctx, &mutation); err != nil {
		t.Fatal(err)
	}
	if _, err := ResourceToExecution(ObjectExecutionConfig{Ctx: ctx, Rc: c, Obj: proposed, Mutation: mutation, Log: logr.Discard()}); err != nil {
		t.Fatal(err)
	}
	updated := getObject(t, ctx, c, "v1", "ConfigMap", "shop", "settings")
	value, _, err := unstructured.NestedString(updated.Object, "data", "log-level")
	if err != nil || value != "debug" {
		t.Fatalf("ConfigMap log-level = %q, err = %v", value, err)
	}
}

func testDeployment(t *testing.T, image string) *unstructured.Unstructured {
	t.Helper()
	object, err := policy.ParseYAMLProposal([]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
  namespace: shop
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: app
        image: ` + image + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	object.SetUID(types.UID("checkout-uid"))
	return object
}

func deploymentImage(t *testing.T, object *unstructured.Unstructured) string {
	t.Helper()
	containers, found, err := unstructured.NestedSlice(object.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("containers = %#v, found = %v, err = %v", containers, found, err)
	}
	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("container = %#v", containers[0])
	}
	image, ok := container["image"].(string)
	if !ok {
		t.Fatalf("image = %#v", container["image"])
	}
	return image
}

func setDeploymentImage(t *testing.T, object *unstructured.Unstructured, image string) {
	t.Helper()
	containers, found, err := unstructured.NestedSlice(object.Object, "spec", "template", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("containers = %#v, found = %v, err = %v", containers, found, err)
	}
	container := containers[0].(map[string]any)
	container["image"] = image
	containers[0] = container
	if err := unstructured.SetNestedSlice(object.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		t.Fatal(err)
	}
}

func getDeployment(t *testing.T, ctx context.Context, c client.Client) *unstructured.Unstructured {
	t.Helper()
	return getObject(t, ctx, c, "apps/v1", "Deployment", "shop", "checkout")
}

func getObject(t *testing.T, ctx context.Context, c client.Client, apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	t.Helper()
	object := &unstructured.Unstructured{}
	object.SetAPIVersion(apiVersion)
	object.SetKind(kind)
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, object); err != nil {
		t.Fatal(err)
	}
	return object
}

func marshalObject(t *testing.T, object *unstructured.Unstructured) string {
	t.Helper()
	data, err := yaml.Marshal(object.Object)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func testMutation(origin string) corev1alpha1.Mutation {
	return corev1alpha1.Mutation{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout-fix", Namespace: "shop"},
		Spec:       corev1alpha1.MutationSpec{OriginConfiguration: origin},
	}
}
