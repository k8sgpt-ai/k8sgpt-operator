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
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	mgr       ctrl.Manager
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	timeout := 3 * time.Minute
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing:    true,
		ControlPlaneStartTimeout: timeout,
		ControlPlaneStopTimeout:  timeout,
		AttachControlPlaneOutput: false,
	}

	var cfg *rest.Config
	var err error
	// this is a channel to signal when the test environment is ready
	done := make(chan interface{})
	go func() {
		// this will block until the test environment is ready
		defer GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	// wait for the test environment to be ready
	Eventually(done).WithContext(ctx).WithTimeout(timeout).Should(BeClosed())
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	Expect(corev1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(k8sscheme.AddToScheme(scheme)).To(Succeed())
	Expect(apiv1.AddToScheme(scheme)).To(Succeed())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("Creating controller manager")
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Port:               8443,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mgr).ToNot(BeNil())

	kc, err := kclient.NewClient("localhost:50051")
	Expect(err).ToNot(HaveOccurred())

	kcs := map[string]*kclient.Client{
		"localhost:50051": kc,
	}

	By("Creatng the controllers")
	k8sGPTController := &K8sGPTReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		K8sGPTClient:  kc,
		k8sGPTClients: kcs,
	}
	Expect(k8sGPTController.SetupWithManager(mgr)).To(Succeed())

	go func() {
		defer GinkgoRecover()
		ctrl.Log.Info("Starting the manager")
		Expect(mgr.Start(ctrl.SetupSignalHandler())).To(Succeed())
	}()

	crd := &apiv1.CustomResourceDefinition{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "k8sgpts.core.k8sgpt.ai"}, crd)).To(Succeed())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "results.core.k8sgpt.ai"}, crd)).To(Succeed())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	// Here we need to exit directly because the testEnv.Stop() may hang forever in some cases.
	if err != nil {
		os.Exit(1)
	}
})
