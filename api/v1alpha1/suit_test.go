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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runtime "k8s.io/apimachinery/pkg/runtime"
	cgScheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// Entry
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sgpt CRDs API Suite")
}

// Initial fake client for every cases
var fakeClient client.Client

var _ = BeforeSuite(func() {
	By("Bootstrapping test environment")

	// Initial the scheme
	scheme := runtime.NewScheme()
	// For the standard k8s types
	_ = cgScheme.AddToScheme(scheme))
	err := AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	// Initial fake client
	fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
})
