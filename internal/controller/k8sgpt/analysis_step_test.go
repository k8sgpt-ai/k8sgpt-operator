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

package k8sgpt

import (
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("AnalysisStep", func() {
	var step *AnalysisStep

	BeforeEach(func() {
		step = &AnalysisStep{
			logger: ctrl.Log.WithName("test-analysis-step"),
		}
	})

	Describe("getResultObjectNamespace", func() {
		Context("when the result has a namespace in the name", func() {
			It("should extract the namespace from the name", func() {
				result := corev1alpha1.ResultSpec{
					Name: "default/my-pod",
					Kind: "Pod",
				}
				Expect(step.getResultObjectNamespace(result)).To(Equal("default"))
			})
		})

		Context("when the result doesn't have a namespace in the name", func() {
			It("should return an empty string", func() {
				result := corev1alpha1.ResultSpec{
					Name: "my-cluster-scoped-resource",
					Kind: "Node",
				}
				Expect(step.getResultObjectNamespace(result)).To(Equal(""))
			})
		})
	})

	Describe("getResultsPerNamespace", func() {
		Context("when given a list of results with different namespaces", func() {
			It("should correctly count results per namespace", func() {
				results := []corev1alpha1.ResultSpec{
					{Name: "default/pod-1", Kind: "Pod"},
					{Name: "default/pod-2", Kind: "Pod"},
					{Name: "kube-system/pod-1", Kind: "Pod"},
					{Name: "cluster-resource", Kind: "Node"},
				}

				counts := step.getResultsPerNamespace(results)
				Expect(counts).To(HaveKeyWithValue("default", 2))
				Expect(counts).To(HaveKeyWithValue("kube-system", 1))
				Expect(counts).To(HaveKeyWithValue("", 1))
			})
		})
	})
})
