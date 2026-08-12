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
	"context"
	"errors"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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

	Describe("analysis error status", func() {
		var (
			ctx            context.Context
			scheme         *runtime.Scheme
			k8sgpt         *corev1alpha1.K8sGPT
			namespacedName types.NamespacedName
		)

		BeforeEach(func() {
			ctx = context.Background()
			scheme = runtime.NewScheme()
			Expect(corev1alpha1.AddToScheme(scheme)).To(Succeed())
			k8sgpt = &corev1alpha1.K8sGPT{}
			k8sgpt.Name = "test-k8sgpt"
			k8sgpt.Namespace = "default"
			namespacedName = types.NamespacedName{Name: k8sgpt.Name, Namespace: k8sgpt.Namespace}
		})

		It("stores the latest Analyze error on status", func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&corev1alpha1.K8sGPT{}).
				WithObjects(k8sgpt).
				Build()

			Expect(setAnalysisErrorStatus(ctx, k8sClient, k8sgpt, errors.New("failed while calling AI provider openai: unexpected EOF"))).To(Succeed())

			updated := &corev1alpha1.K8sGPT{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.LastAnalysisError).To(ContainSubstring("unexpected EOF"))
			Expect(updated.Status.LastAnalysisErrorTime).NotTo(BeNil())
		})

		It("does not update status for an unchanged Analyze error", func() {
			const analysisError = "failed while calling AI provider openai: unexpected EOF"
			k8sgpt.Status.LastAnalysisError = analysisError
			originalTime := metav1.Now()
			k8sgpt.Status.LastAnalysisErrorTime = &originalTime
			statusUpdateCalls := 0
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&corev1alpha1.K8sGPT{}).
				WithObjects(k8sgpt).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
						statusUpdateCalls++
						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			step.recordAnalysisError(&K8sGPTInstance{
				R:            &K8sGPTReconciler{Client: k8sClient},
				Ctx:          ctx,
				K8sgptConfig: k8sgpt,
				logger:       ctrl.Log.WithName("test-analysis-error-status"),
			}, errors.New(analysisError))
			Expect(statusUpdateCalls).To(Equal(0))
			Expect(k8sgpt.Status.LastAnalysisErrorTime).To(Equal(&originalTime))
		})

		It("clears the stored Analyze error after recovery", func() {
			k8sgpt.Status.LastAnalysisError = "failed while calling AI provider openai: unexpected EOF"
			now := metav1.Now()
			k8sgpt.Status.LastAnalysisErrorTime = &now
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&corev1alpha1.K8sGPT{}).
				WithObjects(k8sgpt).
				Build()

			Expect(clearAnalysisErrorStatus(ctx, k8sClient, k8sgpt)).To(Succeed())

			updated := &corev1alpha1.K8sGPT{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.LastAnalysisError).To(BeEmpty())
			Expect(updated.Status.LastAnalysisErrorTime).To(BeNil())
		})
	})
})
