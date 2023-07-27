package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
