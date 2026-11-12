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
	"sync"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"

	metricspkg "github.com/k8sgpt-ai/k8sgpt-operator/pkg/metrics"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/sinks"
)

const (
	ReconcileErrorInterval   = 10 * time.Second
	ReconcileSuccessInterval = 30 * time.Second
	// aiSecretIndexField is the field index name for spec.ai.secret.name
	aiSecretIndexField = ".spec.ai.secret.name"
)

var (
	k8sgptControllerLog = ctrl.Log.WithName("k8sgpt-controller")
)

// parseInterval parses the interval string into a time.Duration
func parseInterval(interval string) (time.Duration, error) {
	if interval == "" {
		return ReconcileSuccessInterval, nil
	}
	return time.ParseDuration(interval)
}

// K8sGPTReconciler reconciles a K8sGPT object
type K8sGPTReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Integrations        *integrations.Integrations
	SinkClient          *sinks.Client
	MetricsBuilder      *metricspkg.MetricBuilder
	EnableResultLogging bool
	Signal              chan types.InterControllerSignal
	Recorder            record.EventRecorder

	// Reused across reconciles, keyed per K8sGPT resource: see clientFor.
	kclientMu sync.Mutex
	kclients  map[client.ObjectKey]*cachedClient
}

// cachedClient is the connection currently held for one K8sGPT resource, with the address it was
// dialled for so a change can be detected.
type cachedClient struct {
	address string
	client  *kclient.Client
}

// clientFor returns a K8sGPT client for k8sgpt at address, reusing the connection already held for
// that resource while its address is unchanged.
//
// Reconciles requeue every 30s by default, and the previous code dialled a new grpc.ClientConn on
// each pass and dropped the old one without closing it. Each orphaned connection keeps its
// resolver, balancer and transport goroutines alive, so the manager accumulated roughly 1.5
// goroutines per reconcile (~4.3k/day) until it was restarted.
//
// The cache is keyed per resource rather than held in a single slot. One K8sGPTReconciler serves
// every K8sGPT in the cluster and each gets its own service address, so a single slot would thrash
// between them - dialling on every reconcile again, and closing a connection the mutation
// controller may still hold from another resource's last Signal.
//
// For the same reason the connection is not closed at the end of a reconcile: it is handed to the
// mutation controller over Signal and may still be in use there. It is closed only when this
// resource's address changes, or when the resource is deleted (see closeClientFor).
func (r *K8sGPTReconciler) clientFor(k8sgpt *corev1alpha1.K8sGPT, address string) (*kclient.Client, error) {
	key := client.ObjectKeyFromObject(k8sgpt)

	r.kclientMu.Lock()
	defer r.kclientMu.Unlock()

	if cached, ok := r.kclients[key]; ok {
		if cached.address == address {
			return cached.client, nil
		}
		if err := cached.client.Close(); err != nil {
			k8sgptControllerLog.Error(err, "closing K8sGPT client after address change",
				"k8sgpt", key, "address", cached.address)
		}
		delete(r.kclients, key)
	}

	c, err := kclient.NewClient(address)
	if err != nil {
		return nil, err
	}
	if r.kclients == nil {
		r.kclients = make(map[client.ObjectKey]*cachedClient)
	}
	r.kclients[key] = &cachedClient{address: address, client: c}
	return c, nil
}

// closeClientFor releases the connection held for a K8sGPT resource. Called when the resource is
// deleted, so a removed K8sGPT does not leave its connection behind for the life of the manager.
func (r *K8sGPTReconciler) closeClientFor(k8sgpt *corev1alpha1.K8sGPT) {
	key := client.ObjectKeyFromObject(k8sgpt)

	r.kclientMu.Lock()
	defer r.kclientMu.Unlock()

	cached, ok := r.kclients[key]
	if !ok {
		return
	}
	if err := cached.client.Close(); err != nil {
		k8sgptControllerLog.Error(err, "closing K8sGPT client on delete", "k8sgpt", key)
	}
	delete(r.kclients, key)
}

type K8sGPTInstance struct {
	R                *K8sGPTReconciler
	req              ctrl.Request
	Ctx              context.Context
	K8sgptConfig     *corev1alpha1.K8sGPT
	k8sgptDeployment *v1.Deployment
	logger           logr.Logger
	kclient          *kclient.Client
	hasReadyReplicas bool
}

type K8sGPT interface {
	execute(*K8sGPTInstance) (ctrl.Result, error)
	setNext(K8sGPT)
}

// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=results,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources="*",verbs="*"
func (r *K8sGPTReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	instance := K8sGPTInstance{
		R:      r,
		req:    req,
		Ctx:    ctx,
		logger: k8sgptControllerLog,
	}

	initStep := InitStep{}
	finalizerStep := FinalizerStep{}
	configureStep := ConfigureStep{}
	preAnalysisStep := PreAnalysisStep{
		// This passes the channel into the pre-analysis step to flag when connection is ready
		// This in turn is passed to the mutation controller
		Signal: r.Signal,
	}
	analysisStep := AnalysisStep{
		enableResultLogging: r.EnableResultLogging,
		logger:              instance.logger.WithName("analysis"),
	}
	resultStatusStep := ResultStatusStep{}
	calculateRemediationStep := calculateRemediationStep{
		logger: instance.logger.WithName("remediation"),
	}
	initStep.setNext(&finalizerStep)
	finalizerStep.setNext(&configureStep)
	configureStep.setNext(&preAnalysisStep)
	preAnalysisStep.setNext(&analysisStep)
	analysisStep.setNext(&resultStatusStep)
	resultStatusStep.setNext(&calculateRemediationStep)

	return initStep.execute(&instance)

}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sGPTReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Retrieve the metrics
	k8sgptReconcileErrorCount := r.MetricsBuilder.GetCounterVec("k8sgpt_reconcile_error_count")
	k8sgptNumberOfResults := r.MetricsBuilder.GetGaugeVec("k8sgpt_number_of_results")
	k8sgptNumberOfResultsByType := r.MetricsBuilder.GetGaugeVec("k8sgpt_number_of_results_by_type")
	k8sgptNumberOfBackendAICalls := r.MetricsBuilder.GetCounterVec("k8sgpt_number_of_backend_ai_calls")
	k8sgptNumberOfFailedBackendAICalls := r.MetricsBuilder.GetCounterVec("k8sgpt_number_of_failed_backend_ai_calls")

	// Register the metrics
	metrics.Registry.MustRegister(
		k8sgptReconcileErrorCount,
		k8sgptNumberOfResults,
		k8sgptNumberOfResultsByType,
		k8sgptNumberOfBackendAICalls,
		k8sgptNumberOfFailedBackendAICalls,
	)

	// Set up field index for spec.ai.secret.name to efficiently map Secrets to K8sGPT resources
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1alpha1.K8sGPT{}, aiSecretIndexField, func(rawObj client.Object) []string {
		k8sgpt := rawObj.(*corev1alpha1.K8sGPT)
		if k8sgpt.Spec.AI == nil || k8sgpt.Spec.AI.Secret == nil || k8sgpt.Spec.AI.Secret.Name == "" {
			return nil
		}
		return []string{k8sgpt.Spec.AI.Secret.Name}
	}); err != nil {
		return err
	}

	// Setup the controller
	c := ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sGPT{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findK8sGPTsForSecret),
		).
		Complete(r)

	return c
}

// findK8sGPTsForSecret maps a Secret to the K8sGPT resources that reference it.
// This enables the controller to reconcile K8sGPT resources when their AI Secret changes.
func (r *K8sGPTReconciler) findK8sGPTsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret := obj.(*corev1.Secret)

	// Find all K8sGPT resources that reference this Secret
	var k8sgptList corev1alpha1.K8sGPTList
	if err := r.List(ctx, &k8sgptList,
		client.InNamespace(secret.Namespace),
		client.MatchingFields{aiSecretIndexField: secret.Name},
	); err != nil {
		k8sgptControllerLog.Error(err, "failed to list K8sGPT resources for Secret",
			"secret", secret.Name, "namespace", secret.Namespace)
		return nil
	}

	// Create reconcile requests for each K8sGPT resource
	requests := make([]reconcile.Request, len(k8sgptList.Items))
	for i, k8sgpt := range k8sgptList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      k8sgpt.Name,
				Namespace: k8sgpt.Namespace,
			},
		}
		k8sgptControllerLog.Info("enqueuing K8sGPT reconciliation due to Secret change",
			"k8sgpt", k8sgpt.Name, "secret", secret.Name, "namespace", secret.Namespace)
	}

	return requests
}

func (r *K8sGPTReconciler) FinishReconcile(err error, requeueImmediate bool, name string, k8sgpt *corev1alpha1.K8sGPT) (ctrl.Result, error) {
	if err != nil {
		interval := ReconcileErrorInterval
		if requeueImmediate {
			interval = 0
		}
		k8sgptControllerLog.Info("Finished Reconciling k8sGPT with error: %s\n", "error", err.Error())
		reconcileErrorCounter := r.MetricsBuilder.GetCounterVec("k8sgpt_reconcile_error_count")
		if reconcileErrorCounter != nil {
			reconcileErrorCounter.WithLabelValues(name).Inc()
		}
		return ctrl.Result{Requeue: true, RequeueAfter: interval}, err
	}

	// Parse the custom interval if specified
	var interval time.Duration
	if k8sgpt != nil && k8sgpt.Spec.Analysis != nil && k8sgpt.Spec.Analysis.Interval != "" {
		var err error
		interval, err = parseInterval(k8sgpt.Spec.Analysis.Interval)
		if err != nil {
			k8sgptControllerLog.Error(err, "Failed to parse analysis interval, using default")
			interval = ReconcileSuccessInterval
		}
	} else {
		interval = ReconcileSuccessInterval
	}

	if requeueImmediate {
		interval = 0
	}

	k8sgptControllerLog.Info("Finished Reconciling k8sGPT", "RequeueTime", interval)
	return ctrl.Result{Requeue: true, RequeueAfter: interval}, nil
}
