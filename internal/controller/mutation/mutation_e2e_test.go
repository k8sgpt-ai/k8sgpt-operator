package mutation

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	ctrl "sigs.k8s.io/controller-runtime"
)

// TestMutationReconcilerRepairsBrokenDeployment is a deterministic system test:
// an API-server-backed Deployment and Result enter the actual Mutation
// reconciler, a gRPC K8sGPT stub returns the proposed manifest, and the second
// reconcile performs the dry-run, policy validation, and persistent JSON patch.
func TestMutationReconcilerRepairsBrokenDeployment(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	env := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")}}
	config, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}); err != nil {
		t.Fatal(err)
	}

	deployment := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "shop"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "checkout"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "checkout"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.example/checkout:missing"}}}},
		},
	}
	if err := c.Create(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
		t.Fatal(err)
	}
	deployment.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
	origin := mustYAML(t, deployment)
	proposed := deployment.DeepCopy()
	proposed.Spec.Template.Spec.Containers[0].Image = "registry.example/checkout:fixed"
	proposal := mustYAML(t, proposed)

	result := &corev1alpha1.Result{ObjectMeta: metav1.ObjectMeta{Name: "checkout-image-pull", Namespace: "shop"}, Spec: corev1alpha1.ResultSpec{
		Backend: "stub", Kind: "Deployment", Name: "shop/checkout", Details: "ImagePullBackOff: image does not exist", Error: []corev1alpha1.Failure{{Text: "ImagePullBackOff"}},
		TargetRef: &corev1alpha1.ResultTargetReference{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "shop", Name: "checkout", UID: deployment.UID, ResourceVersion: deployment.ResourceVersion},
	}}
	if err := c.Create(ctx, result); err != nil {
		t.Fatal(err)
	}
	mutation := &corev1alpha1.Mutation{ObjectMeta: metav1.ObjectMeta{Name: "checkout-image-fix", Namespace: "shop"}, Spec: corev1alpha1.MutationSpec{
		ResourceGVK: "apps/v1", ResourceRef: corev1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "shop", Name: "checkout", UID: deployment.UID, ResourceVersion: deployment.ResourceVersion},
		ResultRef: corev1.ObjectReference{APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Result", Namespace: "shop", Name: result.Name}, OriginConfiguration: origin,
	}, Status: corev1alpha1.MutationStatus{Phase: corev1alpha1.AutoRemediationPhaseNotStarted}}
	if err := c.Create(ctx, mutation); err != nil {
		t.Fatal(err)
	}

	queryClient, closeGRPC := remediationQueryClient(t, proposal)
	defer closeGRPC()
	reconciler := &MutationReconciler{Client: c, Scheme: scheme, ServerQueryClient: &queryClient, logger: logr.Discard()}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(mutation)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("proposal reconciliation: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(mutation), mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Status.Phase != corev1alpha1.AutoRemediationPhaseInProgress {
		t.Fatalf("proposal phase = %v, want in-progress", mutation.Status.Phase)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("execution reconciliation: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Spec.Containers[0].Image; got != "registry.example/checkout:fixed" {
		t.Fatalf("deployment image = %q, want repaired image", got)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(mutation), mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Status.Phase != corev1alpha1.AutoRemediationPhaseCompleted {
		t.Fatalf("mutation phase = %v, want completed", mutation.Status.Phase)
	}
	// Model the next analyzer pass: once the original finding is absent, the
	// existing verification contract records a successful remediation.
	deployment.Status.ObservedGeneration = deployment.Generation
	deployment.Status.Replicas = 1
	deployment.Status.ReadyReplicas = 1
	deployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, deployment); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, result); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("verification reconciliation: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(mutation), mutation); err != nil {
		t.Fatal(err)
	}
	if mutation.Status.Phase != corev1alpha1.AutoRemediationPhaseSuccessful {
		t.Fatalf("verification phase = %v, want successful", mutation.Status.Phase)
	}
}

type queryStub struct {
	rpc.UnimplementedServerQueryServiceServer
	response string
}

func (s queryStub) Query(context.Context, *schemav1.QueryRequest) (*schemav1.QueryResponse, error) {
	return &schemav1.QueryResponse{Response: s.response}, nil
}

func remediationQueryClient(t *testing.T, response string) (rpc.ServerQueryServiceClient, func()) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	rpc.RegisterServerQueryServiceServer(server, queryStub{response: response})
	go func() { _ = server.Serve(listener) }()
	connection, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	return rpc.NewServerQueryServiceClient(connection), func() { _ = connection.Close(); server.Stop(); _ = listener.Close() }
}

func mustYAML(t *testing.T, object any) string {
	t.Helper()
	data, err := yaml.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure typed objects are converted in exactly the form received from the
	// gRPC suggestion path, including identity/concurrency metadata.
	parsed := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(data, &parsed.Object); err != nil {
		t.Fatal(err)
	}
	data, err = yaml.Marshal(parsed.Object)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
