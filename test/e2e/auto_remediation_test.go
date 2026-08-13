//go:build e2e

package e2e

import (
	"context"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	mutationcontroller "github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/mutation"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

const clusterName = "k8sgpt-remediation-e2e"

// TestBrokenDeploymentIsRemediatedOnKind is deliberately excluded from the
// default test suite. It proves the actual Kubernetes Deployment controller
// can recover from a bad image after the Mutation controller applies the
// policy-gated LLM proposal.
func TestBrokenDeploymentIsRemediatedOnKind(t *testing.T) {
	logf.SetLogger(logr.Discard())
	root := repositoryRoot(t)
	run(t, root, "kind", "delete", "cluster", "--name", clusterName)
	t.Cleanup(func() { run(t, root, "kind", "delete", "cluster", "--name", clusterName) })
	run(t, root, "kind", "create", "cluster", "--name", clusterName, "--wait", "90s")
	run(t, root, "kubectl", "apply", "-f", filepath.Join(root, "config", "crd", "bases"))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	scheme := k8sruntime.NewScheme()
	must(t, clientgoscheme.AddToScheme(scheme))
	must(t, corev1alpha1.AddToScheme(scheme))
	config := ctrl.GetConfigOrDie()
	c, err := client.New(config, client.Options{Scheme: scheme})
	must(t, err)
	must(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "remediation-e2e"}}))
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "remediation-e2e"}, Spec: appsv1.DeploymentSpec{
		Replicas: int32p(1), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "checkout"}},
		Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "checkout"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example.invalid/checkout:missing"}}}},
	}}
	must(t, c.Create(ctx, deployment))
	waitFor(t, ctx, func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment) == nil && deployment.Status.ObservedGeneration == deployment.Generation
	})
	waitFor(t, ctx, func() bool {
		pods := &corev1.PodList{}
		if c.List(ctx, pods, client.InNamespace(deployment.Namespace), client.MatchingLabels{"app": "checkout"}) != nil || len(pods.Items) != 1 {
			return false
		}
		for _, status := range pods.Items[0].Status.ContainerStatuses {
			if status.State.Waiting != nil && (status.State.Waiting.Reason == "ErrImagePull" || status.State.Waiting.Reason == "ImagePullBackOff") {
				return true
			}
		}
		return false
	})
	deployment.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
	origin := deploymentYAMLFromObject(t, deployment)
	proposed := deployment.DeepCopy()
	proposed.Spec.Template.Spec.Containers[0].Image = "registry.k8s.io/pause:3.10"
	proposal := deploymentYAMLFromObject(t, proposed)
	queryClient, closeQuery := queryClient(t, proposal)
	defer closeQuery()
	manager, err := ctrl.NewManager(config, ctrl.Options{Scheme: scheme})
	must(t, err)
	reconciler := &mutationcontroller.MutationReconciler{Client: manager.GetClient(), Scheme: scheme, ServerQueryClient: &queryClient, MetricsBuilder: metrics.InitializeMetrics()}
	must(t, reconciler.SetupWithManager(manager))
	go func() { _ = manager.Start(ctx) }()
	result := &corev1alpha1.Result{ObjectMeta: metav1.ObjectMeta{Name: "checkout-image", Namespace: deployment.Namespace}, Spec: corev1alpha1.ResultSpec{Backend: "stub", Kind: "Deployment", Name: deployment.Namespace + "/" + deployment.Name, Details: "ImagePullBackOff", Error: []corev1alpha1.Failure{{Text: "ImagePullBackOff"}}, TargetRef: &corev1alpha1.ResultTargetReference{APIVersion: "apps/v1", Kind: "Deployment", Namespace: deployment.Namespace, Name: deployment.Name, UID: deployment.UID, ResourceVersion: deployment.ResourceVersion}}}
	must(t, c.Create(ctx, result))
	mutation := &corev1alpha1.Mutation{ObjectMeta: metav1.ObjectMeta{Name: "checkout-image-fix", Namespace: deployment.Namespace}, Spec: corev1alpha1.MutationSpec{ResourceGVK: "apps/v1", ResourceRef: corev1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Namespace: deployment.Namespace, Name: deployment.Name, UID: deployment.UID, ResourceVersion: deployment.ResourceVersion}, ResultRef: corev1.ObjectReference{APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Result", Namespace: result.Namespace, Name: result.Name}, OriginConfiguration: origin}, Status: corev1alpha1.MutationStatus{Phase: corev1alpha1.AutoRemediationPhaseNotStarted}}
	must(t, c.Create(ctx, mutation))
	waitFor(t, ctx, func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment) == nil && deployment.Spec.Template.Spec.Containers[0].Image == "registry.k8s.io/pause:3.10"
	})
	waitFor(t, ctx, func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(deployment), deployment) == nil && deployment.Status.AvailableReplicas == 1
	})
	// A subsequent analyzer pass clears the finding only after the workload is
	// healthy. The controller must then mark the mutation successful.
	must(t, c.Delete(ctx, result))
	waitFor(t, ctx, func() bool {
		return c.Get(ctx, client.ObjectKeyFromObject(mutation), mutation) == nil && mutation.Status.Phase == corev1alpha1.AutoRemediationPhaseSuccessful
	})
}

func deploymentYAMLFromObject(t *testing.T, deployment *appsv1.Deployment) string {
	data, err := yaml.Marshal(deployment)
	must(t, err)
	return string(data)
}
func int32p(v int32) *int32 { return &v }
func waitFor(t *testing.T, ctx context.Context, predicate func() bool) {
	t.Helper()
	for ctx.Err() == nil {
		if predicate() {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal(ctx.Err())
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func run(t *testing.T, dir, command string, arguments ...string) {
	t.Helper()
	output, err := exec.Command(command, arguments...).CombinedOutput()
	if err != nil && !(command == "kind" && len(arguments) > 1 && arguments[0] == "delete") {
		t.Fatalf("%s %v: %v\n%s", command, arguments, err, output)
	}
}
func repositoryRoot(t *testing.T) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

type queryService struct {
	rpc.UnimplementedServerQueryServiceServer
	response string
}

func (s queryService) Query(context.Context, *schemav1.QueryRequest) (*schemav1.QueryResponse, error) {
	return &schemav1.QueryResponse{Response: s.response}, nil
}
func queryClient(t *testing.T, response string) (rpc.ServerQueryServiceClient, func()) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	rpc.RegisterServerQueryServiceServer(server, queryService{response: response})
	go func() { _ = server.Serve(listener) }()
	connection, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	must(t, err)
	return rpc.NewServerQueryServiceClient(connection), func() { _ = connection.Close(); server.Stop(); _ = listener.Close() }
}
