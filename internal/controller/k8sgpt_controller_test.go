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

package controllers

import (
	"errors"
	"time"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TIMEOUT      = time.Second * time.Duration(60)
	POLLINTERVAL = time.Second * time.Duration(1)
	DURATION     = time.Second * time.Duration(10)
)

var _ = Describe("K8sGPT controller", func() {
	SetDefaultEventuallyTimeout(TIMEOUT)
	SetDefaultEventuallyPollingInterval(POLLINTERVAL)
	SetDefaultConsistentlyDuration(DURATION)

	Describe("Create a new Analysis", Label("integration"), func() {
		Context("when the analysis doesn't exist beforehand", func() {
			k8sgpt := corev1alpha1.GetValidProjectResource("my-test", "default")
			nn := types.NamespacedName{
				Namespace: k8sgpt.Namespace,
				Name:      k8sgpt.Name,
			}
			It("Should create a fake secret", func() {
				Expect(k8sClient.Create(ctx, createFakeSecret(k8sgpt.Spec.AI.Secret.Name, "default"))).Should(Succeed())
			})

			It("Should create a k8sgpt serviceAccount", func() {
				Expect(k8sClient.Create(ctx, createServiceAccount("default"))).Should(Succeed())
			})

			It("Should create CR", func() {
				Expect(k8sClient.Create(ctx, &k8sgpt)).Should(Succeed())

			})

			It("Should K8SGPT have a finalizer", func() {
				Eventually(func() error {
					k := corev1alpha1.K8sGPT{}
					err := k8sClient.Get(ctx, nn, &k)
					if err != nil {
						return err
					}

					if len(k.Finalizers) == 0 {
						return errors.New("k8sgpt doesnt have finalizer")
					}

					return nil

				}).Should(BeNil())
			})

			It("Should deploy k8sgpt server in current namespace", func() {
				Eventually(func() error {
					deployment := v1.Deployment{}
					err := k8sClient.Get(ctx, nn, &deployment)
					if err != nil {
						return err
					}

					if *deployment.Spec.Replicas == 0 {
						return errors.New("deployment is not correct")
					}

					return nil

				}).Should(BeNil())
			})
		})

	})
})

func createFakeSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"openai-api-key": []byte("fake-key"),
		},
	}
}

func createServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8sgpt",
			Namespace: namespace,
		},
	}
}
