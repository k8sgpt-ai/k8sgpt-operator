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
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"
	"time"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"

	metricspkg "github.com/k8sgpt-ai/k8sgpt-operator/pkg/metrics"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/sinks"
)

const (
	ReconcileErrorInterval   = 10 * time.Second
	ReconcileSuccessInterval = 30 * time.Second
)

var (
	k8sgptControllerLog = ctrl.Log.WithName("k8sgpt-controller")
)

// K8sGPTReconciler reconciles a K8sGPT object
type K8sGPTReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Integrations        *integrations.Integrations
	SinkClient          *sinks.Client
	MetricsBuilder      *metricspkg.MetricBuilder
	EnableResultLogging bool
	Signal              chan types.InterControllerSignal
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

	// Setup the controller
	c := ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sGPT{}).
		Complete(r)

	return c
}

func (r *K8sGPTReconciler) FinishReconcile(err error, requeueImmediate bool, name string) (ctrl.Result, error) {
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
	interval := ReconcileSuccessInterval
	if requeueImmediate {
		interval = 0
	}
	k8sgptControllerLog.Info("Finished Reconciling k8sGPT")
	return ctrl.Result{Requeue: true, RequeueAfter: interval}, nil
}
