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

package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/k8sgpt"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/mutation"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/shared"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/metrics"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/sinks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

// parseNamespaceList splits a comma-separated list of namespaces into a slice,
// trimming whitespace and dropping empty entries.
func parseNamespaceList(list string) []string {
	var namespaces []string
	for _, ns := range strings.Split(list, ",") {
		ns = strings.TrimSpace(ns)
		if ns != "" {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces
}

// secretCacheOptions returns the manager cache options for the given Secret
// namespaces. With an empty list it returns the zero value, i.e. the default
// cluster-wide Secrets cache, so behaviour is unchanged unless the operator is
// explicitly configured. Scoping the Secrets informer lets the operator run
// with a ServiceAccount that cannot list Secrets cluster-wide: without a scope
// the Secrets informer (needed by the K8sGPT controller's Secret watch) never
// syncs and the reconcile hangs (see k8sgpt-ai/k8sgpt-operator#876).
func secretCacheOptions(namespaces []string) cache.Options {
	if len(namespaces) == 0 {
		return cache.Options{}
	}
	scoped := make(map[string]cache.Config, len(namespaces))
	for _, ns := range namespaces {
		scoped[ns] = cache.Config{}
	}
	return cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {Namespaces: scoped},
		},
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableResultLogging bool
	var secretCacheNamespaces string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableResultLogging, "enable-result-logging", false, "Whether to enable results logging")
	flag.StringVar(&secretCacheNamespaces, "secret-cache-namespaces", "",
		"Comma-separated list of namespaces whose Secrets the operator caches and watches, "+
			"typically the namespaces containing K8sGPT resources and their AI/sink Secrets. "+
			"Set this on installs where the operator ServiceAccount cannot list Secrets "+
			"cluster-wide; when unset, Secrets are cached cluster-wide (default). "+
			"Note: a K8sGPT resource in a namespace outside this list still reconciles "+
			"(Secret reads go through the uncached API reader), but Secret rotations in it "+
			"are not detected via watch.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	if os.Getenv("LOCAL_MODE") != "" {
		setupLog.Info("Running in local mode")
		min := 7000
		max := 8000
		metricsAddr = fmt.Sprintf(":%d", rand.Intn(max-min+1)+min)
		probeAddr = fmt.Sprintf(":%d", rand.Intn(max-min+1)+min)
		setupLog.Info(fmt.Sprintf("Metrics address: %s", metricsAddr))
		setupLog.Info(fmt.Sprintf("Probe address: %s", probeAddr))
	}
	secretNamespaces := parseNamespaceList(secretCacheNamespaces)
	if len(secretNamespaces) > 0 {
		setupLog.Info("Scoping the Secrets cache and watch to specific namespaces",
			"namespaces", secretNamespaces)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache:  secretCacheOptions(secretNamespaces),
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ea9c19f7.k8sgpt.ai",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	integration, err := integrations.NewIntegrations(mgr.GetClient(), context.Background())
	if err != nil {
		setupLog.Error(err, "unable to create REST client to initialise Integrations")
		os.Exit(1)
	}

	timeout, exists := os.LookupEnv("OPERATOR_SINK_WEBHOOK_TIMEOUT_SECONDS")
	if !exists {
		timeout = "35s"
	}

	sinkTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		setupLog.Error(err, "unable to read webhook timeout value")
		os.Exit(1)
	}
	sinkClient := sinks.NewClient(sinkTimeout)

	metricsBuilder := metrics.InitializeMetrics()

	// This channel allows us to indicate when K8sGPT deployment is ready for active comms
	// This is a necessity for the mutation system to work
	ready := make(chan types.InterControllerSignal, 10)

	// Start a goroutine to listen for signals and set the shared ServerQueryClient
	go func() {
		for signal := range ready {
			setupLog.Info("Received signal from K8sGPT controller, setting shared client")
			if signal.K8sGPTClient != nil {
				client := signal.K8sGPTClient.GetServerQueryServiceClient()
				shared.SetServerQueryClient(&client)
				shared.SetRemoteBackend(signal.Backend)
				shared.SetK8sGPTConfig(signal.K8sGPT)
				setupLog.Info("Shared ServerQueryClient set successfully", "backend", signal.Backend)
			}
		}
	}()

	if err = (&mutation.MutationReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		MetricsBuilder: metricsBuilder,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Mutation")
		os.Exit(1)
	}

	if err = (&k8sgpt.K8sGPTReconciler{
		Client:              mgr.GetClient(),
		APIReader:           mgr.GetAPIReader(),
		Scheme:              mgr.GetScheme(),
		Signal:              ready,
		Integrations:        integration,
		SinkClient:          sinkClient,
		MetricsBuilder:      metricsBuilder,
		EnableResultLogging: enableResultLogging,
		Recorder:            mgr.GetEventRecorderFor("k8sgpt-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "K8sGPT")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
