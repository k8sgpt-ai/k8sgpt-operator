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

	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/sinks"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/apps/v1"
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
	Integrations *integrations.Integrations
	SinkClient   *sinks.Client
	K8sGPTClient *kclient.Client
}

// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=k8sgpts/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=results,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources="*",verbs="*"
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
			err := resources.Sync(ctx, r.Client, *k8sgptConfig, resources.DestroyOp)
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
	if client.IgnoreNotFound(err) != nil {
		k8sgptReconcileErrorCount.Inc()
		return r.finishReconcile(err, false)
	}
	err = resources.Sync(ctx, r.Client, *k8sgptConfig, resources.SyncOp)
	if err != nil {
		k8sgptReconcileErrorCount.Inc()
		return r.finishReconcile(err, false)
	}

	if deployment.Status.ReadyReplicas > 0 {

		// Check the version of the deployment image matches the version set in the K8sGPT CR
		imageURI := deployment.Spec.Template.Spec.Containers[0].Image

		image := strings.Split(imageURI, ":")
		imageRepository := image[0]
		imageVersion := image[1]

		// if one of repository or tag is changed, we need to update the deployment
		if imageRepository != k8sgptConfig.Spec.Repository || imageVersion != k8sgptConfig.Spec.Version {
			// Update the deployment image
			deployment.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s:%s",
				imageRepository, k8sgptConfig.Spec.Version)
			err = r.Update(ctx, &deployment)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}

			return r.finishReconcile(nil, false)
		}

		// If the deployment is active, we will query it directly for analysis data
		address, err := kclient.GenerateAddress(ctx, r.Client, k8sgptConfig)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
		// Log address
		fmt.Printf("K8sGPT address: %s\n", address)

		k8sgptClient, err := kclient.NewClient(address)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}

		defer k8sgptClient.Close()

		// Configure the k8sgpt deployment if required
		if k8sgptConfig.Spec.RemoteCache != nil {
			err = k8sgptClient.AddConfig(k8sgptConfig)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
		}
		if k8sgptConfig.Spec.Integrations != nil {
			err = k8sgptClient.AddIntegration(k8sgptConfig)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
		}

		response, err := k8sgptClient.ProcessAnalysis(deployment, k8sgptConfig)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
		// Parse the k8sgpt-deployment response into a list of results
		k8sgptNumberOfResults.Set(float64(len(response.Results)))
		rawResults, err := resources.MapResults(*r.Integrations, response.Results, *k8sgptConfig)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
		// Prior to creating or updating any results we will delete any stale results that
		// no longer are relevent, we can do this by using the resultSpec composed name against
		// the custom resource name
		resultList := &corev1alpha1.ResultList{}
		err = r.List(ctx, resultList)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
		if len(resultList.Items) > 0 {
			// If the result does not exist in the map we will delete it
			for _, result := range resultList.Items {
				fmt.Printf("Checking if %s is still relevant\n", result.Name)
				if _, ok := rawResults[result.Name]; !ok {
					err = r.Delete(ctx, &result)
					if err != nil {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					} else {
						k8sgptNumberOfResultsByType.With(prometheus.Labels{
							"kind": result.Spec.Kind,
							"name": result.Name,
						}).Dec()
					}
				}
			}
		}
		// At this point we are able to loop through our rawResults and create them or update
		// them as needed
		for _, result := range rawResults {
			operation, err := resources.CreateOrUpdateResult(ctx, r.Client, result)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)

			}
			// Update metrics
			if operation == resources.CreatedResult {
				k8sgptNumberOfResultsByType.With(prometheus.Labels{
					"kind": result.Spec.Kind,
					"name": result.Name,
				}).Inc()
			} else if operation == resources.UpdatedResult {
				fmt.Printf("Updated successfully %s \n", result.Name)
			}

		}

		// We emit when result Status is not historical
		// and when user configures a sink for the first time
		latestResultList := &corev1alpha1.ResultList{}
		if err := r.List(ctx, latestResultList); err != nil {
			return r.finishReconcile(err, false)
		}
		if len(latestResultList.Items) == 0 {
			return r.finishReconcile(nil, false)
		}
		sinkEnabled := k8sgptConfig.Spec.Sink != nil && k8sgptConfig.Spec.Sink.Type != "" && k8sgptConfig.Spec.Sink.Endpoint != ""

		var sinkType sinks.ISink
		if sinkEnabled {
			sinkType = sinks.NewSink(k8sgptConfig.Spec.Sink.Type)
			sinkType.Configure(*k8sgptConfig, *r.SinkClient)
		}

		for _, result := range latestResultList.Items {
			var res corev1alpha1.Result
			if err := r.Get(ctx, client.ObjectKey{Namespace: result.Namespace, Name: result.Name}, &res); err != nil {
				return r.finishReconcile(err, false)
			}

			if sinkEnabled {
				if res.Status.LifeCycle != string(resources.NoOpResult) || res.Status.Webhook == "" {
					if err := sinkType.Emit(res.Spec); err != nil {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					}
					res.Status.Webhook = k8sgptConfig.Spec.Sink.Endpoint
				}
			} else {
				// Remove the Webhook status from results
				res.Status.Webhook = ""
			}
			if err := r.Status().Update(ctx, &res); err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
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
		fmt.Printf("Finished Reconciling k8sGPT with error: %s\n", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: interval}, err
	}
	interval := ReconcileSuccessInterval
	if requeueImmediate {
		interval = 0
	}
	fmt.Println("Finished Reconciling k8sGPT")
	return ctrl.Result{Requeue: true, RequeueAfter: interval}, nil
}
