/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package resources

import (
	"context"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	r1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// enum create or destroy
type CreateOrDestroy int

const (
	Create CreateOrDestroy = iota
	Destroy
	DeploymentName = "k8sgpt-deployment"
)

// Create service for K8sGPT
func GetService(config v1alpha1.K8sGPT) (*v1.Service, error) {

	// Create service
	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: config.Spec.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": DeploymentName,
			},
			Ports: []v1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	return &service, nil
}

// Create Service Account for K8sGPT and bind it to K8sGPT role
func GetServiceAccount(config v1alpha1.K8sGPT) (*v1.ServiceAccount, error) {

	// Create service account
	serviceAccount := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: config.Spec.Namespace,
		},
	}

	return &serviceAccount, nil
}

// Create cluster role binding for K8sGPT
func GetClusterRoleBinding(config v1alpha1.K8sGPT) (*r1.ClusterRoleBinding, error) {

	// Create cluster role binding
	clusterRoleBinding := r1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
					APIVersion: config.APIVersion,
				},
			},
		},
		Subjects: []r1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "k8sgpt",
				Namespace: config.Spec.Namespace,
			},
		},
		RoleRef: r1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "k8sgpt",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return &clusterRoleBinding, nil
}

// Create ClusterRole for K8sGPT with cluster read all
func GetClusterRole(config v1alpha1.K8sGPT) (*r1.ClusterRole, error) {

	// Create cluster role
	clusterRole := r1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8sgpt",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
					APIVersion: config.APIVersion,
				},
			},
		},
		Rules: []r1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	return &clusterRole, nil
}

// Create deployment with the latest K8sGPT image
func GetDeployment(config v1alpha1.K8sGPT) (*appsv1.Deployment, error) {

	// Create deployment
	replicas := int32(1)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: config.Spec.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
					APIVersion: config.APIVersion,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": DeploymentName,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": DeploymentName,
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "k8sgpt",
					Containers: []v1.Container{
						{
							Name:  "k8sgpt",
							Image: "ghcr.io/k8sgpt-ai/k8sgpt:" + config.Spec.Version,
							Args: []string{
								"serve",
							},
							Env: []v1.EnvVar{
								{
									Name: "K8SGPT_PASSWORD",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: config.Spec.Secret.Name,
											},
											Key: config.Spec.Secret.Key,
										},
									},
								},
								{
									Name:  "K8SGPT_MODEL",
									Value: config.Spec.Model,
								},
								{
									Name:  "K8SGPT_BACKEND",
									Value: config.Spec.Backend,
								},
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("1"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("0.2"),
									v1.ResourceMemory: resource.MustParse("156Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
	return &deployment, nil
}

func Sync(ctx context.Context, c client.Client,
	config v1alpha1.K8sGPT, i CreateOrDestroy) error {

	var objs []client.Object

	svc, err := GetService(config)
	if err != nil {
		return err
	}

	objs = append(objs, svc)

	svcAcc, err := GetServiceAccount(config)
	if err != nil {
		return err
	}

	objs = append(objs, svcAcc)

	clusterRole, err := GetClusterRole(config)
	if err != nil {
		return err
	}

	objs = append(objs, clusterRole)

	clusterRoleBinding, err := GetClusterRoleBinding(config)
	if err != nil {
		return err
	}

	objs = append(objs, clusterRoleBinding)

	deployment, err := GetDeployment(config)
	if err != nil {
		return err
	}

	objs = append(objs, deployment)

	// for each object, create or destroy
	for _, obj := range objs {
		switch i {
		case Create:
			err := c.Create(ctx, obj)
			if err != nil {
				// If the object already exists, ignore the error
				if !errors.IsAlreadyExists(err) {
					return err
				}
			}
		case Destroy:
			err := c.Delete(ctx, obj)
			if err != nil {
				// if the object is not found, ignore the error
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}

	return nil
}
