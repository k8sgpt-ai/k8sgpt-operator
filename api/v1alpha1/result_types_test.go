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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("The test cases for the K8sGPT CRDs result types", func() {
	var (
		ctx          context.Context
		Namespace    = "k8sGPT"
		Kind         = "ResultRef"
		Unmasked     = "This is unmasked"
		Masked       = "This is masked"
		Text         = "This is a failure"
		Details      = "This is a result"
		ParentObject = "k8s-gpt"
		Name         = "result"

		sensitive = Sensitive{
			Unmasked: Unmasked,
			Masked:   Masked,
		}

		// Implement a instance of Failure type
		failure = Failure{
			Text:      Text,
			Sensitive: []Sensitive{sensitive},
		}

		// Implement a instance of ResultSpec type
		result = Result{
			TypeMeta: metav1.TypeMeta{
				Kind: Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      Name,
				Namespace: Namespace,
			},
			Spec: ResultSpec{
				Kind:         Kind,
				Name:         Name,
				Error:        []Failure{failure},
				Details:      Details,
				ParentObject: ParentObject,
			},
		}
		// Create a Namespace object
		typeNamespace = types.NamespacedName{
			Name:      Name,
			Namespace: Namespace,
		}
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("Create a new ResultRef object", func() {
		It("Should create a new ResultRef object", func() {
			Expect(fakeClient.Create(ctx, &result)).Should(Succeed())
		})
	})

	// Get the ResultRef object
	Context("Get the ResultRef object", func() {
		It("Should get the ResultRef object", func() {
			Expect(fakeClient.Get(ctx, typeNamespace, &result)).Should(Succeed())
		})
		// Check the ResultRef object's filed values
		It("Should check the ResultRef object's filed values", func() {
			Expect(result.Spec.Kind).Should(Equal(Kind))
			Expect(result.Spec.Name).Should(Equal(Name))
			Expect(result.Spec.Error[0].Text).Should(Equal(Text))
			Expect(result.Spec.Details).Should(Equal(Details))
			Expect(result.Spec.ParentObject).Should(Equal(ParentObject))
		})
	})
	// Update the ResultRef object
	Context("Update the ResultRef object", func() {
		It("Should update the ResultRef object", func() {
			result.Spec.Details = "This is a new result"
			Expect(fakeClient.Update(ctx, &result)).Should(Succeed())
		})
		// Check the ResultRef object's filed values
		It("Should check the ResultRef object's filed values", func() {
			Expect(result.Spec.Kind).Should(Equal(Kind))
			Expect(result.Spec.Name).Should(Equal(Name))
			Expect(result.Spec.Error[0].Text).Should(Equal(Text))
			Expect(result.Spec.Details).Should(Equal("This is a new result"))
			Expect(result.Spec.ParentObject).Should(Equal(ParentObject))
		})
	})
	// Get the ResultRef object by list
	Context("Get the ResultRef object by list", func() {
		It("Should get the ResultRef object by list", func() {
			resultList := ResultList{}
			Expect(fakeClient.List(ctx, &resultList)).Should(Succeed())
		})
		// Check the length of ResultRef object list
		It("Should check the length of ResultRef object list", func() {
			Expect(len(result.Spec.Error)).Should(Equal(1))
		})
	})
	// delete the ResultRef object
	Context("Delete the ResultRef object", func() {
		It("Should delete the ResultRef object", func() {
			Expect(fakeClient.Delete(ctx, &result)).Should(Succeed())
		})
		// Check the ResultRef object has been deleted
		It("Should check the ResultRef object has been deleted", func() {
			Expect(fakeClient.Get(ctx, typeNamespace, &result)).ShouldNot(Succeed())
		})
	})
})
