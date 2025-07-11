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

package mutation

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	t "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

var _ = Describe("Mutation Controller", func() {
	Context("When reconciling a resource with targetConfiguration not set", func() {
		const resourceName = "test-mutation-no-targetconfig"

		ctx := context.Background()

		typeNamespacedName := t.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		mutation := &corev1alpha1.Mutation{}
		reconciler := &MutationReconciler{}
		BeforeEach(func() {
			By("creating the custom resource for the Kind Mutation")
			err := reconciler.Get(ctx, typeNamespacedName, mutation)
			if err != nil && errors.IsNotFound(err) {
				resource := &corev1alpha1.Mutation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: corev1alpha1.MutationSpec{
						ResourceRef: v1.ObjectReference{
							Kind:      "Service",
							Name:      "my-service",
							Namespace: "default",
						},
						OriginConfiguration: `
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: my-app
  type: LoadBalancer
`,
						TargetConfiguration: "", // Empty targetConfiguration
					},
				}
				Expect(reconciler.Client.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			reconciler := &MutationReconciler{}
			resource := &corev1alpha1.Mutation{}
			err := reconciler.Client.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Mutation")
			Expect(reconciler.Client.Delete(ctx, resource)).To(Succeed())
		})
		It("should requeue the resource and not update the status", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MutationReconciler{
				Client:            reconciler.Client,
				Scheme:            reconciler.Client.Scheme(),
				ServerQueryClient: nil, // Simulate client not ready
				RemoteBackend:     "test-backend",
			}
			// The controller should now requeue if ServerQueryClient is nil, without blocking or using a signal.
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Assertions
			Expect(result.RequeueAfter).To(Equal(30 * time.Second)) // util.ErrorRequeueTime

			// Fetch the updated Mutation object
			updatedMutation := &corev1alpha1.Mutation{}
			err = reconciler.Client.Get(ctx, typeNamespacedName, updatedMutation)
			Expect(err).NotTo(HaveOccurred())

			// Verify the status phase remains unchanged (still InProgress)
			Expect(updatedMutation.Status.Phase).To(Equal(corev1alpha1.AutoRemediationPhaseInProgress))
		})
	})
})
