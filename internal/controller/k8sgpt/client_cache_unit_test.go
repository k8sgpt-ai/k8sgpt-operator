/*
Copyright 2024 The K8sGPT Authors.
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
	"testing"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"google.golang.org/grpc/connectivity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func k8sgptResource(namespace, name string) *corev1alpha1.K8sGPT {
	return &corev1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

// One K8sGPTReconciler serves every K8sGPT in the cluster, and each resource has its own service
// address. A single cached client would thrash: each resource's reconcile would evict the other's,
// re-dialling every time and closing a connection the mutation controller may still hold.
func TestClientForKeepsOneConnectionPerResource(t *testing.T) {
	r := &K8sGPTReconciler{}
	a := k8sgptResource("k8sgpt", "instance-a")
	b := k8sgptResource("k8sgpt", "instance-b")

	clientA, err := r.clientFor(a, "instance-a.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor(a): %v", err)
	}
	clientB, err := r.clientFor(b, "instance-b.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor(b): %v", err)
	}
	if clientA == clientB {
		t.Fatal("each K8sGPT resource must get its own client")
	}

	// A second reconcile of a must reuse its connection rather than dial another one.
	again, err := r.clientFor(a, "instance-a.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor(a) second call: %v", err)
	}
	if again != clientA {
		t.Fatal("reconciling the same resource re-dialled instead of reusing the cached client")
	}

	// And b's connection must be untouched by a's reconcile.
	if state := clientB.Conn.GetState(); state == connectivity.Shutdown {
		t.Fatalf("instance-b's connection was closed by instance-a's reconcile (state %v)", state)
	}
}

func TestClientForReplacesTheConnectionWhenTheAddressChanges(t *testing.T) {
	r := &K8sGPTReconciler{}
	a := k8sgptResource("k8sgpt", "instance-a")

	first, err := r.clientFor(a, "old.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor: %v", err)
	}
	second, err := r.clientFor(a, "new.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor after address change: %v", err)
	}

	if second == first {
		t.Fatal("an address change must produce a new client")
	}
	if state := first.Conn.GetState(); state != connectivity.Shutdown {
		t.Fatalf("the superseded connection was not closed (state %v)", state)
	}
}

func TestCloseClientForReleasesTheConnectionAndDropsTheEntry(t *testing.T) {
	r := &K8sGPTReconciler{}
	a := k8sgptResource("k8sgpt", "instance-a")

	c, err := r.clientFor(a, "instance-a.k8sgpt.svc:8080")
	if err != nil {
		t.Fatalf("clientFor: %v", err)
	}

	r.closeClientFor(a)

	if state := c.Conn.GetState(); state != connectivity.Shutdown {
		t.Fatalf("connection not closed on delete (state %v)", state)
	}
	if _, ok := r.kclients[client.ObjectKeyFromObject(a)]; ok {
		t.Fatal("cache entry not dropped on delete")
	}

	// Deleting twice, or deleting something never cached, must not panic.
	r.closeClientFor(a)
	r.closeClientFor(k8sgptResource("k8sgpt", "never-seen"))
}
