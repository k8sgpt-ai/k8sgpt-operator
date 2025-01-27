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

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("The test cases for the K8sGPT CRDs", func() {
	// Define utility variables for this test suite.
	var (
		ctx       context.Context
		secretRef = SecretRef{
			Name: "k8s-gpt-secret",
			Key:  "k8s-gpt",
		}
		backOff = BackOff{
			Enabled:    false,
			MaxRetries: 5,
		}
		kind       = "K8sGPT"
		baseUrl    = "https://api.k8s-gpt.localhost"
		model      = "345M"
		repository = "ghcr.io/k8sgpt-ai/k8sgpt"
		version    = "v1alpha1"
		language   = "english"
		anonymize  = true
		resource   = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.2"),
				corev1.ResourceMemory: resource.MustParse("156Mi"),
			},
		}

		Namespace = "k8sGPT"

		k8sGPT = K8sGPT{
			TypeMeta: metav1.TypeMeta{
				Kind: kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k8s-gpt",
				Namespace: Namespace,
			},
			Spec: K8sGPTSpec{
				AI: &AISpec{
					Backend:   OpenAI,
					BackOff:   &backOff,
					BaseUrl:   baseUrl,
					Model:     model,
					Enabled:   true,
					Secret:    &secretRef,
					Anonymize: &anonymize,
					Language:  language,
				},
				Version:    version,
				Repository: repository,
				NoCache:    true,
				NodeSelector: map[string]string{
					"nodepool": "management",
				},
				Resources: &resource,
			},
		}

		dontAnonymize = false
		k8sGPT2       = K8sGPT{
			TypeMeta: metav1.TypeMeta{
				Kind: kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k8s-gpt-2",
				Namespace: Namespace,
			},
			Spec: K8sGPTSpec{
				AI: &AISpec{
					Backend:   OpenAI,
					BackOff:   &backOff,
					BaseUrl:   baseUrl,
					Model:     model,
					Secret:    &secretRef,
					Enabled:   false,
					Anonymize: &dontAnonymize,
					Language:  language,
				},
				Repository: repository,
				Version:    version,
				NoCache:    false,
				NodeSelector: map[string]string{
					"nodepool": "management",
				},
			},
		}

		typeNamespace = types.NamespacedName{
			Name:      "k8s-gpt",
			Namespace: Namespace,
		}
	)
	// Setup the test context.
	BeforeEach(func() {
		ctx = context.Background()
	})
	// Define utility functions for this test suite.
	Context("Creating K8sGPT CRDs", func() {
		It("Should create a new K8sGPT CRDs", func() {
			By("Creating a new K8sGPT CRDs")
			// Create a new K8sGPT CRDs.
			Expect(fakeClient.Create(ctx, &k8sGPT)).Should(Succeed())
			Expect(fakeClient.Create(ctx, &k8sGPT2)).Should(Succeed())
		})

		// We can get the k8sGPT CRDs by the name and namespace.
		It("Should get the K8sGPT CRDs by the name and namespace", func() {
			By("Getting the K8sGPT CRDs by the name and namespace")
			// Define the K8sGPT CRDs object.
			k8sGPTObject := K8sGPT{}
			// Get the K8sGPT CRDs by the name and namespace.
			Expect(fakeClient.Get(ctx, typeNamespace, &k8sGPTObject)).Should(Succeed())
			// Check the K8sGPT CRDs object's name and the APIVersion.
			Expect(k8sGPTObject.Name).Should(Equal("k8s-gpt"))
			Expect(k8sGPTObject.APIVersion).Should(Equal(GroupVersion.String()))
			Expect(k8sGPTObject.Spec.AI.Enabled).Should(Equal(true))

			// get K8sGPT CRD by resource name
			Expect(fakeClient.Get(ctx, types.NamespacedName{Name: "k8s-gpt-2", Namespace: Namespace}, &k8sGPTObject)).Should(Succeed())
		})

		// Get K8sGPT by list
		It("Should get the K8sGPT CRDs by list", func() {
			By("Getting the K8sGPT CRDs by list")
			// Define the K8sGPT CRDs object.
			k8sGPTList := K8sGPTList{}
			// Get the K8sGPT CRDs by the name and namespace.
			Expect(fakeClient.List(ctx, &k8sGPTList)).Should(Succeed())
			// check the number of the list should be equal to 2
			Expect(len(k8sGPTList.Items)).Should(Equal(2))
		})

		// Update the K8sGPT CRDs.
		It("Should update the K8sGPT CRDs", func() {
			By("Updating the K8sGPT CRDs")
			// Define the K8sGPT CRDs object.
			k8sGPTObject := K8sGPT{}
			// Get the K8sGPT CRDs by the name and namespace.
			Expect(fakeClient.Get(ctx, typeNamespace, &k8sGPTObject)).Should(Succeed())
			// Update the K8sGPT CRDs.
			k8sGPTObject.Spec.AI.Enabled = false
			Expect(fakeClient.Update(ctx, &k8sGPTObject)).Should(Succeed())
			// check the K8sGPT CRDs should be equal to false
			Expect(k8sGPTObject.Spec.AI.Enabled).Should(Equal(false))
		})

		// Delete the K8sGPT CRDs.
		It("Should delete the K8sGPT CRDs", func() {
			By("Deleting the K8sGPT CRDs")
			// Define the K8sGPT CRDs object.
			Expect(fakeClient.Delete(ctx, &k8sGPT)).Should(Succeed())

			// Check the K8sGPT CRDs by list
			By("Checking the K8sGPT CRDs by list")
			k8sGPTList := K8sGPTList{}
			Expect(fakeClient.List(ctx, &k8sGPTList)).Should(Succeed())
			// check the number of the list should be equal to 1
			Expect(len(k8sGPTList.Items)).Should(Equal(1))
			// check the K8sGPT CRD's name should be equal to "k8s-gpt-2"
			Expect(k8sGPTList.Items[0].Name).Should(Equal("k8s-gpt-2"))
			// remove the K8sGPT CRDs
			Expect(fakeClient.Delete(ctx, &k8sGPT2)).Should(Succeed())
		})
	})
})
