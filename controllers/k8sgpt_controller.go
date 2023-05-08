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
	"context"
	"fmt"
	"strings"
	"time"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	k8sgptclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	FinalizerName            = "k8sgpt.ai/finalizer"
	ReconcileErrorInterval   = 10 * time.Second
	ReconcileSuccessInterval = 30 * time.Second
)

var (
	// Metrics
	// k8sgptReconcileErrorCount is a metric for the number of errors during reconcile
	k8sgptReconcileErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "k8sgpt_reconcile_error_count",
		Help: "The total number of errors during reconcile",
	})
	// k8sgptNumberOfResults is a metric for the number of results
	k8sgptNumberOfResults = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "k8sgpt_number_of_results",
		Help: "The total number of results",
	})
	// k8sgptNumberOfResultsByType is a metric for the number of results by type
	k8sgptNumberOfResultsByType = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8sgpt_number_of_results_by_type",
		Help: "The total number of results by type",
	}, []string{"kind", "name"})
)

// K8sGPTReconciler reconciles a K8sGPT object
type K8sGPTReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	K8sGPTClient *k8sgptclient.Client
}

// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=results,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources="*",verbs=get;list;watch;create;update;patch;delete

func (r *K8sGPTReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Look up the instance for this reconcile request
	k8sgptConfig := &corev1alpha1.K8sGPT{}
	err := r.Get(ctx, req.NamespacedName, k8sgptConfig)
	if err != nil {
		// Error reading the object - requeue the request.
		k8sgptReconcileErrorCount.Inc()
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finaliser if there isn't one
	if k8sgptConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(k8sgptConfig.GetFinalizers(), FinalizerName) {
			controllerutil.AddFinalizer(k8sgptConfig, FinalizerName)
			if err := r.Update(ctx, k8sgptConfig); err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(k8sgptConfig.GetFinalizers(), FinalizerName) {

			// Delete any external resources associated with the instance
			err := resources.Sync(ctx, r.Client, *k8sgptConfig, resources.Destroy)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
			controllerutil.RemoveFinalizer(k8sgptConfig, FinalizerName)
			if err := r.Update(ctx, k8sgptConfig); err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
		}
		// Stop reconciliation as the item is being deleted
		return r.finishReconcile(nil, false)
	}

	// Check and see if the instance is new or has a K8sGPT deployment in flight
	deployment := v1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Namespace: k8sgptConfig.Namespace,
		Name: "k8sgpt-deployment"}, &deployment)
	if err != nil {

		err = resources.Sync(ctx, r.Client, *k8sgptConfig, resources.Create)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
	}

	// If the deployment is active, we will query it directly for analysis data
	if deployment.Status.ReadyReplicas > 0 {
		// Get the K8sGPT client
		response, err := r.K8sGPTClient.ProcessAnalysis(deployment, k8sgptConfig)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}

		// Create results from the analysis data
		k8sgptNumberOfResults.Set(float64(len(response.Results)))
		for _, resultSpec := range response.Results {
			name := strings.ReplaceAll(resultSpec.Name, "-", "")
			name = strings.ReplaceAll(name, "/", "")
			result := corev1alpha1.Result{
				Spec: resultSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: k8sgptConfig.Namespace,
				},
			}
			// Update metrics
			k8sgptNumberOfResultsByType.With(prometheus.Labels{
				"kind": resultSpec.Kind,
				"name": resultSpec.Name,
			}).Inc()

			err = r.Create(ctx, &result)
			if err != nil {
				// if the result already exists, we will update it
				if errors.IsAlreadyExists(err) {

					result.ResourceVersion = k8sgptConfig.GetResourceVersion()
					err = r.Update(ctx, &result)
					if err != nil {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					}
				} else {
					return r.finishReconcile(err, false)
				}
			}
		}

	}

	return r.finishReconcile(nil, false)
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sGPTReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c := ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sGPT{}).
		Complete(r)

	metrics.Registry.MustRegister(k8sgptReconcileErrorCount, k8sgptNumberOfResults, k8sgptNumberOfResultsByType)

	return c
}

func (r *K8sGPTReconciler) finishReconcile(err error, requeueImmediate bool) (ctrl.Result, error) {
	if err != nil {
		interval := ReconcileErrorInterval
		if requeueImmediate {
			interval = 0
		}
		fmt.Printf("Finished Reconciling K8sGPT with error: %s\n", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: interval}, err
	}
	interval := ReconcileSuccessInterval
	if requeueImmediate {
		interval = 0
	}
	fmt.Println("Finished Reconciling K8sGPT")
	return ctrl.Result{Requeue: true, RequeueAfter: interval}, nil
}
