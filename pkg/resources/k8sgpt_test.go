package resources

import (
	"context"
	"testing"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_GetService(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       "12345",
		},
	}

	// Call GetService
	service, err := GetService(config)
	assert.NoError(t, err)

	// Check the service's properties
	assert.Equal(t, config.Name, service.Name)
	assert.Equal(t, config.Namespace, service.Namespace)
	assert.Equal(t, config.Name, service.OwnerReferences[0].Name)
	assert.Equal(t, int32(8080), service.Spec.Ports[0].Port)
}

func Test_GetServiceAccount(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       "12345",
		},
		Spec: v1alpha1.K8sGPTSpec{
			ImagePullSecrets: []v1alpha1.ImagePullSecrets{
				{Name: "secret1"},
				{Name: "secret2"},
			},
		},
	}

	// Call GetServiceAccount
	serviceAccount, err := GetServiceAccount(config)
	assert.NoError(t, err)

	// Check the service account's properties
	assert.Equal(t, "k8sgpt", serviceAccount.Name)
	assert.Equal(t, config.Namespace, serviceAccount.Namespace)
	assert.Equal(t, "secret1", serviceAccount.ImagePullSecrets[0].Name)
	assert.Equal(t, "secret2", serviceAccount.ImagePullSecrets[1].Name)
}

func Test_GetClusterRoleBinding(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       "12345",
		},
	}

	// Call GetClusterRoleBinding
	clusterRoleBinding, err := GetClusterRoleBinding(config)
	assert.NoError(t, err)

	// Check the cluster role binding's properties
	assert.Equal(t, "k8sgpt", clusterRoleBinding.Name)
	assert.Equal(t, config.UID, clusterRoleBinding.OwnerReferences[0].UID)
	assert.Equal(t, "ServiceAccount", clusterRoleBinding.Subjects[0].Kind)
	assert.Equal(t, "k8sgpt", clusterRoleBinding.Subjects[0].Name)
	assert.Equal(t, config.Namespace, clusterRoleBinding.Subjects[0].Namespace)
	assert.Equal(t, "ClusterRole", clusterRoleBinding.RoleRef.Kind)
	assert.Equal(t, "k8sgpt", clusterRoleBinding.RoleRef.Name)
	assert.Equal(t, "rbac.authorization.k8s.io", clusterRoleBinding.RoleRef.APIGroup)
}

func Test_GetClusterRole(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       "12345",
		},
	}

	// Call GetClusterRole
	clusterRole, err := GetClusterRole(config)
	assert.NoError(t, err)

	// Check the cluster role's properties
	assert.Equal(t, "k8sgpt", clusterRole.Name)
	assert.Equal(t, config.Name, clusterRole.OwnerReferences[0].Name)
	assert.Equal(t, config.UID, clusterRole.OwnerReferences[0].UID)
	assert.Equal(t, config.APIVersion, clusterRole.OwnerReferences[0].APIVersion)
	assert.Contains(t, clusterRole.Rules[0].APIGroups, "*")
	assert.Contains(t, clusterRole.Rules[0].Resources, "*")
	assert.ElementsMatch(t, clusterRole.Rules[0].Verbs, []string{"create", "list", "get", "watch", "delete"})
	assert.Contains(t, clusterRole.Rules[1].APIGroups, "apiextensions.k8s.io")
	assert.Contains(t, clusterRole.Rules[1].Resources, "*")
	assert.Contains(t, clusterRole.Rules[1].Verbs, "*")
}

func Test_GetDeployment(t *testing.T) {
	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			Repository: "repo",
			Version:    "1.0",
			AI: &v1alpha1.AISpec{
				Model:   "model",
				Backend: "azureopenai",
				Secret: &v1alpha1.SecretRef{
					Name: "secret",
					Key:  "key",
				},
				BaseUrl: "http://baseurl",
				Engine:  "engine",
			},
			Kubeconfig: &v1alpha1.SecretRef{
				Name: "kubeconfig",
				Key:  "key",
			},
			RemoteCache: &v1alpha1.RemoteCacheRef{
				Credentials: &v1alpha1.CredentialsRef{
					Name: "credentials",
				},
				Azure: &v1alpha1.AzureBackend{},
			},
			NodeSelector: map[string]string{
				"node": "node1",
			},
		},
	}

	// Call GetDeployment
	deployment, err := GetDeployment(config, true)
	assert.NoError(t, err)

	// Check that the deployment has the expected properties
	assert.Equal(t, "test", deployment.Name)
	assert.Equal(t, "default", deployment.Namespace)
	assert.Equal(t, "repo:1.0", deployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, "model", deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
	assert.Equal(t, "azureopenai", deployment.Spec.Template.Spec.Containers[0].Env[1].Value)
	assert.Equal(t, "secret", deployment.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal(t, "key", deployment.Spec.Template.Spec.Containers[0].Env[4].ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, "credentials", deployment.Spec.Template.Spec.Containers[0].Env[7].ValueFrom.SecretKeyRef.LocalObjectReference.Name)
	assert.Equal(t, "azure_client_secret", deployment.Spec.Template.Spec.Containers[0].Env[7].ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, "kubeconfig", deployment.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal(t, "kubeconfig", deployment.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName)
	assert.Equal(t, "key", deployment.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.Items[0].Key)
	assert.Equal(t, "kubeconfig", deployment.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.Items[0].Path)
	assert.Equal(t, "", deployment.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, false, *deployment.Spec.Template.Spec.AutomountServiceAccountToken)
}

func Test_Sync(t *testing.T) {
	// Create a fake client
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	c := fake.NewFakeClientWithScheme(scheme)

	// Create a K8sGPT object
	config := v1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.K8sGPTSpec{
			Repository: "repo",
			Version:    "1.0",
			AI: &v1alpha1.AISpec{
				Model:   "model",
				Backend: "backend",
			},
		},
	}

	// Call Sync
	err := Sync(context.Background(), c, config, SyncOp)
	assert.NoError(t, err)

	// Check that the expected objects were created
	svcAcc := &v1.ServiceAccount{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "k8sgpt", Namespace: "default"}, svcAcc)
	assert.NoError(t, err)

	clusterRole := &rbacv1.ClusterRole{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "k8sgpt"}, clusterRole)
	assert.NoError(t, err)

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "k8sgpt"}, clusterRoleBinding)
	assert.NoError(t, err)

	svc := &v1.Service{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "test", Namespace: "default"}, svc)
	assert.NoError(t, err)

	deployment := &appsv1.Deployment{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "test", Namespace: "default"}, deployment)
	assert.NoError(t, err)
}

func Test_DeploymentShouldBeSynced(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	//
	// create deployment
	//
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-k8sgpt",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "k8sgpt",
							Image: "ghcr.io/k8sgpt-ai/k8sgpt:v0.3.8",
						},
					},
				},
			},
		},
	}

	// test
	err := doSync(ctx, fakeClient, deployment)
	require.NoError(t, err)

	existDeployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), existDeployment)

	// verify
	require.NoError(t, err)
	assert.NotNil(t, existDeployment)

	//
	// patch deployment
	//
	deploymentUpdated := deployment.DeepCopy()
	updatedImage := "ghcr.io/k8sgpt-ai/k8sgpt:latest"
	deploymentUpdated.Spec.Template.Spec.Containers[0].Image = updatedImage

	// test
	err = doSync(ctx, fakeClient, deploymentUpdated)
	require.NoError(t, err)
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(deployment), existDeployment)
	require.NoError(t, err)

	// verify
	assert.NotNil(t, existDeployment)
	assert.Equal(t, updatedImage, existDeployment.Spec.Template.Spec.Containers[0].Image)
}

func Test_ServiceAccountShouldNotBeSynced(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	//
	// create ServiceAccount
	//
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
		},
		AutomountServiceAccountToken: pointer.Bool(true),
	}

	// test
	err := doSync(ctx, fakeClient, serviceAccount)
	require.NoError(t, err)

	existSA := &v1.ServiceAccount{}
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceAccount), existSA)

	// verify
	require.NoError(t, err)
	assert.NotNil(t, existSA)

	//
	// patch ServiceAccount
	//
	saUpdated := existSA.DeepCopy()
	saUpdated.AutomountServiceAccountToken = nil

	// test
	err = doSync(ctx, fakeClient, saUpdated)
	require.NoError(t, err)
	err = fakeClient.Get(ctx, client.ObjectKeyFromObject(saUpdated), existSA)
	require.NoError(t, err)

	// verify
	assert.NotNil(t, existSA)
	assert.NotNil(t, existSA.AutomountServiceAccountToken)
}
