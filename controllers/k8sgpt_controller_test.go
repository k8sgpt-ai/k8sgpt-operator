/*
Copyright 2023 K8sGPT Contributors.

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
	"context"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("K8sGPT controller suit test", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	Context("Getting k8sgpt CRDs", func() {
		It("Should get error when getting k8sgpt CRDs", func() {
			By("Getting k8sgpt CRDs")
			k8sgpt := &corev1alpha1.K8sGPT{}
			namespace := "default"
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, k8sgpt)).Should(HaveOccurred())
		})
	})

	Context("Creating k8sgpt CRDs", func() {
		It("Should not create k8sgpt CRDs by invalid backend", func() {
			k8sgpt := &corev1alpha1.K8sGPT{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: corev1alpha1.K8sGPTSpec{
					EnableAI: true,
					NoCache:  true,
					Backend:  "gpt2",
					Filters:  []string{"gpt2"},
				},
			}
			Expect(k8sClient.Create(ctx, k8sgpt)).Should(HaveOccurred())
		})

		It("Should create k8sgpt CRDs by valid backend", func() {
			k8sgpt := &corev1alpha1.K8sGPT{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: corev1alpha1.K8sGPTSpec{
					EnableAI: true,
					NoCache:  true,
					Backend:  "openai",
					Filters:  []string{"openai"},
				},
			}
			Expect(k8sClient.Create(ctx, k8sgpt)).Should(Succeed())
		})
	})

	Context("Getting k8sgpt CRDs", func() {
		It("Should get k8sgpt CRDs", func() {
			By("Getting k8sgpt CRDs")
			k8sgpt := &corev1alpha1.K8sGPT{}
			namespace := "default"
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, k8sgpt)).Should(Succeed())
		})
	})

	Context("Updating k8sgpt CRDs", func() {
		It("Should update k8sgpt CRDs", func() {
			By("Updating k8sgpt CRDs")
			k8sgpt := &corev1alpha1.K8sGPT{}
			namespace := "default"
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, k8sgpt)).Should(Succeed())
			k8sgpt.Spec.EnableAI = false
			Expect(k8sClient.Update(ctx, k8sgpt)).Should(Succeed())
		})
	})

	Context("Getting k8sgpt CRDs", func() {
		It("Should get k8sgpt CRDs with the latest value", func() {
			By("Getting k8sgpt CRDs")
			k8sgpt := &corev1alpha1.K8sGPT{}
			namespace := "default"
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, k8sgpt)).Should(Succeed())
			Expect(k8sgpt.Spec.EnableAI).Should(BeFalse())
		})
	})

	Context("Deleting k8sgpt CRDs", func() {
		It("Should delete k8sgpt CRDs", func() {
			By("Deleting k8sgpt CRDs")
			k8sgpt := &corev1alpha1.K8sGPT{}
			namespace := "default"
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, k8sgpt)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, k8sgpt)).Should(Succeed())
		})
	})
})
