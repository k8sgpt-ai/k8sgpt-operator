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
	"net"
	"os"
	"strings"
	"time"

	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	K8sGPTClient *kclient.Client
	// This is a map of clients for each deployment
	k8sGPTClients map[string]*kclient.Client
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

	if deployment.Status.ReadyReplicas > 0 {

		// Check the version of the deployment image matches the version set in the K8sGPT CR
		imageURI := deployment.Spec.Template.Spec.Containers[0].Image
		imageVersion := strings.Split(imageURI, ":")[1]
		if imageVersion != k8sgptConfig.Spec.Version {
			// Update the deployment image
			deployment.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s:%s",
				strings.Split(imageURI, ":")[0], k8sgptConfig.Spec.Version)
			err = r.Update(ctx, &deployment)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}

			return r.finishReconcile(nil, false)
		}

		// If the deployment is active, we will query it directly for analysis data
		if _, ok := r.k8sGPTClients[k8sgptConfig.Name]; !ok {
			// Create a new client
			var address string
			if os.Getenv("LOCAL_MODE") != "" {
				address = "localhost:8080"
			} else {
				// Get k8sgpt-deployment service pod ip
				podList := &corev1.PodList{}
				listOpts := []client.ListOption{
					client.InNamespace(k8sgptConfig.Namespace),
					client.MatchingLabels{"app": "k8sgpt-deployment"},
				}
				err := r.List(ctx, podList, listOpts...)
				if err != nil {
					k8sgptReconcileErrorCount.Inc()
					return r.finishReconcile(err, false)
				}
				if len(podList.Items) == 0 {
					k8sgptReconcileErrorCount.Inc()
					return r.finishReconcile(fmt.Errorf("no pods found for k8sgpt-deployment"), false)
				}
				address = fmt.Sprintf("%s:8080", podList.Items[0].Status.PodIP)
			}

			fmt.Printf("Creating new client for %s\n", address)
			// Test if the port is open
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}

			fmt.Printf("Connection established between %s and localhost with time out of %d seconds.\n", address, int64(1))
			fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())
			fmt.Printf("Local Address : %s \n", conn.LocalAddr().String())

			k8sgptClient, err := kclient.NewClient(address)
			if err != nil {
				k8sgptReconcileErrorCount.Inc()
				return r.finishReconcile(err, false)
			}
			r.k8sGPTClients[k8sgptConfig.Name] = k8sgptClient
		}

		response, err := r.k8sGPTClients[k8sgptConfig.Name].ProcessAnalysis(deployment, k8sgptConfig)
		if err != nil {
			k8sgptReconcileErrorCount.Inc()
			return r.finishReconcile(err, false)
		}
		// Parse the k8sgpt-deployment response into a list of results
		k8sgptNumberOfResults.Set(float64(len(response.Results)))
		rawResults := make(map[string]corev1alpha1.Result)
		for _, resultSpec := range response.Results {
			resultSpec.Backend = k8sgptConfig.Spec.Backend
			name := strings.ReplaceAll(resultSpec.Name, "-", "")
			name = strings.ReplaceAll(name, "/", "")
			result := corev1alpha1.Result{
				Spec: resultSpec,
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: k8sgptConfig.Namespace,
				},
			}
			if k8sgptConfig.Spec.Backstage == true {
				labelKey := "backstage.io/kubernetes-id"
				namespace, resourceName, _ := strings.Cut(resultSpec.Name, "/")
				m, err := cmdutil.NewFactory(genericclioptions.NewConfigFlags(true)).ToRESTMapper()
				if err != nil {
					k8sgptReconcileErrorCount.Inc()
					return r.finishReconcile(err, false)
				}

				gvr, err := m.ResourceFor(schema.GroupVersionResource{
					Resource: resultSpec.Kind,
				})
				if err != nil {
					k8sgptReconcileErrorCount.Inc()
					return r.finishReconcile(err, false)
				}

				obj := &unstructured.Unstructured{}
				gvk := schema.GroupVersionKind{
					Group:   gvr.Group,
					Kind:    resultSpec.Kind,
					Version: gvr.Version,
				}

				obj.SetGroupVersionKind(gvk)

				// Retrieve the resource using the client
				err = r.Client.Get(ctx, client.ObjectKey{Name: resourceName, Namespace: namespace}, obj)
				if err != nil {
					if errors.IsNotFound(err) {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					} else {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					}
				}
				labels := obj.GetLabels()
				if value, exists := labels[labelKey]; exists {
					// Assign the same label key/value to result CR
					result.ObjectMeta.Labels = map[string]string{labelKey: value}
				} else {
					// too verbose?
					fmt.Printf("Label key '%s' does not exist in %s resource: %s\n", labelKey, resultSpec.Kind, obj.GetName())
				}
			}
			rawResults[name] = result
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
			// Check if the result already exists
			var existingResult corev1alpha1.Result
			err = r.Get(ctx, client.ObjectKey{Namespace: k8sgptConfig.Namespace,
				Name: result.Name}, &existingResult)
			if err != nil {
				// if the result doesn't exist, we will create it
				if errors.IsNotFound(err) {
					err = r.Create(ctx, &result)
					if err != nil {
						k8sgptReconcileErrorCount.Inc()
						return r.finishReconcile(err, false)
					} else {
						k8sgptNumberOfResultsByType.With(prometheus.Labels{
							"kind": result.Spec.Kind,
							"name": result.Name,
						}).Inc()
					}
				} else {
					k8sgptReconcileErrorCount.Inc()
					return r.finishReconcile(err, false)
				}
			} else {
				// If the result already exists we will update it
				existingResult.Spec = result.Spec
				existingResult.Labels = result.Labels
				err = r.Update(ctx, &existingResult)
				if err != nil {
					k8sgptReconcileErrorCount.Inc()
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

	r.k8sGPTClients = make(map[string]*kclient.Client)
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
