package resources

import (
	"context"
	"os"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// ---------------------------------------------------------------------------
// dynamicRBACEnabled
// ---------------------------------------------------------------------------

func Test_dynamicRBACEnabled_Unset(t *testing.T) {
	os.Unsetenv(dynamicRBACEnvVar)
	assert.True(t, dynamicRBACEnabled(), "should default to true when env var is unset")
}

func Test_dynamicRBACEnabled_False(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "false")
	assert.False(t, dynamicRBACEnabled())
}

func Test_dynamicRBACEnabled_True(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "true")
	assert.True(t, dynamicRBACEnabled())
}

func Test_dynamicRBACEnabled_One(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "1")
	assert.True(t, dynamicRBACEnabled(), "'1' should be treated as true")
}

func Test_dynamicRBACEnabled_Zero(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "0")
	assert.False(t, dynamicRBACEnabled(), "'0' should be treated as false")
}

func Test_dynamicRBACEnabled_InvalidValue(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "notabool")
	assert.True(t, dynamicRBACEnabled(), "unparseable value should default to true")
}

func Test_dynamicRBACEnabled_EmptyString(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "")
	assert.True(t, dynamicRBACEnabled(), "empty string should default to true")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestConfig(namespace string) v1alpha1.K8sGPT {
	return v1alpha1.K8sGPT{
		TypeMeta: metav1.TypeMeta{
			Kind:       "K8sGPT",
			APIVersion: "core.k8sgpt.ai/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: namespace,
			UID:       "test-uid",
		},
		Spec: v1alpha1.K8sGPTSpec{
			Repository:      "ghcr.io/k8sgpt-ai/k8sgpt",
			Version:         "v0.4.1",
			ImagePullPolicy: v1.PullAlways,
			AI: &v1alpha1.AISpec{
				Backend:   "openai",
				Model:     "gpt-4o-mini",
				MaxTokens: "2048",
				Topk:      "50",
			},
		},
	}
}

func newSchemeAndClient(t *testing.T) (client.Client, *runtime.Scheme) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))
	require.NoError(t, rbacv1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	return c, scheme
}

// ---------------------------------------------------------------------------
// Sync – dynamic RBAC enabled (default when env var is unset)
// ---------------------------------------------------------------------------

func Test_Sync_DynamicRBACEnabled_CreatesRBACResources(t *testing.T) {
	os.Unsetenv(dynamicRBACEnvVar) // unset → defaults to true (enabled)

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("test-ns")

	err := Sync(ctx, fakeClient, config, SyncOp)
	require.NoError(t, err)

	expectedSAName := "k8sgpt-test-ns"
	expectedCRName := expectedSAName + "-clusterrole"
	expectedCRBName := expectedCRName + "-binding"

	// ServiceAccount was created
	sa := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: expectedSAName, Namespace: "test-ns"}, sa)
	require.NoError(t, err, "ServiceAccount should be created when dynamic RBAC is enabled")

	// ClusterRole was created
	cr := &rbacv1.ClusterRole{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: expectedCRName}, cr)
	require.NoError(t, err, "ClusterRole should be created when dynamic RBAC is enabled")

	// ClusterRoleBinding was created
	crb := &rbacv1.ClusterRoleBinding{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: expectedCRBName}, crb)
	require.NoError(t, err, "ClusterRoleBinding should be created when dynamic RBAC is enabled")

	// Deployment uses the dynamic SA name
	dep := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep)
	require.NoError(t, err)
	assert.Equal(t, expectedSAName, dep.Spec.Template.Spec.ServiceAccountName)
}

// ---------------------------------------------------------------------------
// Sync – dynamic RBAC disabled (explicit false)
// ---------------------------------------------------------------------------

func Test_Sync_DynamicRBACDisabled_SkipsRBACResources(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "false") // explicit false → disabled

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("test-ns")

	err := Sync(ctx, fakeClient, config, SyncOp)
	require.NoError(t, err)

	// ServiceAccount should NOT be created by the operator
	sa := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "k8sgpt-test-ns", Namespace: "test-ns"}, sa)
	assert.Error(t, err, "ServiceAccount should NOT be created when dynamic RBAC is disabled")

	// ClusterRole should NOT be created
	cr := &rbacv1.ClusterRole{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "k8sgpt-test-ns-clusterrole"}, cr)
	assert.Error(t, err, "ClusterRole should NOT be created when dynamic RBAC is disabled")

	// ClusterRoleBinding should NOT be created
	crb := &rbacv1.ClusterRoleBinding{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "k8sgpt-test-ns-clusterrole-binding"}, crb)
	assert.Error(t, err, "ClusterRoleBinding should NOT be created when dynamic RBAC is disabled")

	// Deployment should still be created with the default SA name
	dep := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep)
	require.NoError(t, err)
	assert.Equal(t, defaultServiceAccountName, dep.Spec.Template.Spec.ServiceAccountName,
		"Deployment should use the default static SA name")
}

// ---------------------------------------------------------------------------
// Sync – destroy path
// ---------------------------------------------------------------------------

func Test_Sync_Destroy_DynamicRBACEnabled(t *testing.T) {
	os.Unsetenv(dynamicRBACEnvVar) // unset → defaults to true

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("test-ns")

	// First sync (create)
	require.NoError(t, Sync(ctx, fakeClient, config, SyncOp))

	// Verify resources exist
	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep))

	// Destroy
	require.NoError(t, Sync(ctx, fakeClient, config, DestroyOp))

	// Verify deployment is gone
	err := fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep)
	assert.Error(t, err, "Deployment should be deleted after DestroyOp")
}

func Test_Sync_Destroy_DynamicRBACDisabled(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "false") // explicit false

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("test-ns")

	// First sync (create)
	require.NoError(t, Sync(ctx, fakeClient, config, SyncOp))

	// Destroy – should only destroy Service + Deployment (no RBAC)
	require.NoError(t, Sync(ctx, fakeClient, config, DestroyOp))

	// Verify deployment is gone
	dep := &appsv1.Deployment{}
	err := fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep)
	assert.Error(t, err, "Deployment should be deleted after DestroyOp")
}

// ---------------------------------------------------------------------------
// GetServiceAccount
// ---------------------------------------------------------------------------

func Test_GetServiceAccount(t *testing.T) {
	config := newTestConfig("test-ns")

	sa, err := GetServiceAccount(config, "my-sa")
	require.NoError(t, err)
	assert.Equal(t, "my-sa", sa.Name)
	assert.Equal(t, "test-ns", sa.Namespace)
	assert.Len(t, sa.OwnerReferences, 1)
	assert.Equal(t, "test-k8sgpt", sa.OwnerReferences[0].Name)
}

func Test_GetServiceAccount_IRSA(t *testing.T) {
	config := newTestConfig("test-ns")
	config.Spec.ExtraOptions = &v1alpha1.ExtraOptionsRef{
		ServiceAccountIRSA: "arn:aws:iam::123456789012:role/test-role",
	}

	sa, err := GetServiceAccount(config, "my-sa")
	require.NoError(t, err)
	assert.Equal(t, "arn:aws:iam::123456789012:role/test-role",
		sa.Annotations["eks.amazonaws.com/role-arn"])
}

// ---------------------------------------------------------------------------
// GetClusterRole
// ---------------------------------------------------------------------------

func Test_GetClusterRole(t *testing.T) {
	config := newTestConfig("test-ns")

	cr, err := GetClusterRole(config, "my-sa")
	require.NoError(t, err)
	assert.Equal(t, "my-sa-clusterrole", cr.Name)
	assert.True(t, len(cr.Rules) > 0, "ClusterRole should have rules")

	// Verify core API group resources
	coreRule := cr.Rules[0]
	assert.Contains(t, coreRule.Resources, "pods")
	assert.Contains(t, coreRule.Resources, "services")
	assert.Contains(t, coreRule.Verbs, "get")
	assert.Contains(t, coreRule.Verbs, "list")
	assert.Contains(t, coreRule.Verbs, "watch")
}

// ---------------------------------------------------------------------------
// GetClusterRoleBinding
// ---------------------------------------------------------------------------

func Test_GetClusterRoleBinding(t *testing.T) {
	config := newTestConfig("test-ns")

	crb, err := GetClusterRoleBinding(config, "my-sa", "my-crb", "my-cr")
	require.NoError(t, err)
	assert.Equal(t, "my-crb", crb.Name)
	assert.Equal(t, "my-cr", crb.RoleRef.Name)
	assert.Equal(t, "ClusterRole", crb.RoleRef.Kind)
	require.Len(t, crb.Subjects, 1)
	assert.Equal(t, "my-sa", crb.Subjects[0].Name)
	assert.Equal(t, "test-ns", crb.Subjects[0].Namespace)
}

// ---------------------------------------------------------------------------
// GetService
// ---------------------------------------------------------------------------

func Test_GetService(t *testing.T) {
	config := newTestConfig("test-ns")

	svc, err := GetService(config)
	require.NoError(t, err)
	assert.Equal(t, config.Name, svc.Name)
	assert.Equal(t, "test-ns", svc.Namespace)
	require.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(8080), svc.Spec.Ports[0].Port)
}

// ---------------------------------------------------------------------------
// Sync – explicit false via env
// ---------------------------------------------------------------------------

func Test_Sync_DynamicRBACExplicitFalse_SkipsRBACResources(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "false") // explicit false → disabled

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("test-ns")

	err := Sync(ctx, fakeClient, config, SyncOp)
	require.NoError(t, err)

	// Same assertions as unset – no RBAC resources created
	sa := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "k8sgpt-test-ns", Namespace: "test-ns"}, sa)
	assert.Error(t, err, "ServiceAccount should NOT be created when dynamic RBAC is explicitly false")

	dep := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "test-ns"}, dep)
	require.NoError(t, err)
	assert.Equal(t, defaultServiceAccountName, dep.Spec.Template.Spec.ServiceAccountName)
}

// ---------------------------------------------------------------------------
// Sync – RBAC ordering: SA before Deployment, CR/CRB after
// ---------------------------------------------------------------------------

func Test_Sync_DynamicRBACEnabled_CreatesAllFiveResources(t *testing.T) {
	os.Unsetenv(dynamicRBACEnvVar) // unset → defaults to true

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("prod")

	err := Sync(ctx, fakeClient, config, SyncOp)
	require.NoError(t, err)

	expectedSAName := "k8sgpt-prod"

	// All 5 resource types should be present
	sa := &v1.ServiceAccount{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: expectedSAName, Namespace: "prod"}, sa))

	svc := &v1.Service{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "prod"}, svc))

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "prod"}, dep))
	assert.Equal(t, expectedSAName, dep.Spec.Template.Spec.ServiceAccountName)

	cr := &rbacv1.ClusterRole{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: expectedSAName + "-clusterrole"}, cr))

	crb := &rbacv1.ClusterRoleBinding{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: expectedSAName + "-clusterrole-binding"}, crb))
}

// ---------------------------------------------------------------------------
// Sync – disabled creates only Service + Deployment
// ---------------------------------------------------------------------------

func Test_Sync_DynamicRBACDisabled_CreatesTwoResources(t *testing.T) {
	t.Setenv(dynamicRBACEnvVar, "false") // explicit false

	fakeClient, _ := newSchemeAndClient(t)
	ctx := context.Background()
	config := newTestConfig("prod")

	err := Sync(ctx, fakeClient, config, SyncOp)
	require.NoError(t, err)

	// Service and Deployment exist
	svc := &v1.Service{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "prod"}, svc))

	dep := &appsv1.Deployment{}
	require.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: "prod"}, dep))
	assert.Equal(t, defaultServiceAccountName, dep.Spec.Template.Spec.ServiceAccountName)

	// No RBAC resources
	sa := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "k8sgpt-prod", Namespace: "prod"}, sa)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func Test_DefaultServiceAccountName(t *testing.T) {
	assert.Equal(t, "k8sgpt", defaultServiceAccountName)
}

func Test_DynamicRBACEnvVar(t *testing.T) {
	assert.Equal(t, "K8SGPT_ENABLE_DYNAMIC_RBAC", dynamicRBACEnvVar)
}
