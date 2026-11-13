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
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/conversions"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/shared"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/util"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/prompts"
	metricspkg "github.com/k8sgpt-ai/k8sgpt-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// MutationReconciler reconciles a Mutation object
type MutationReconciler struct {
	client.Client
	logger            logr.Logger
	Scheme            *runtime.Scheme
	ServerQueryClient *rpc.ServerQueryServiceClient
	MetricsBuilder    *metricspkg.MetricBuilder
	RemoteBackend     string
	K8sGPT            *corev1alpha1.K8sGPT
}

var (
	mutationControllerLog = ctrl.Log.WithName("mutation-controller")
)

// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources="*",verbs="*"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mutation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its ResultRef here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"
// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources="*",verbs="*"
func (r *MutationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// get the mutation resource
	var mutation corev1alpha1.Mutation
	if err := r.Get(ctx, req.NamespacedName, &mutation); err != nil {
		mutationControllerLog.Error(err, "unable to fetch mutation")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	mutationControllerLog.Info("Reconciling mutation", "mutation", mutation.Name)

	// Get the shared backend.
	backend := shared.GetRemoteBackend()

	// check if the object is being deleted

	// check if object has a finalizer
	if mutation.ObjectMeta.Finalizers != nil && !mutation.ObjectMeta.DeletionTimestamp.IsZero() {
		finalizer := mutation.ObjectMeta.GetFinalizers()
		if util.IsStringInSlice("mutation.finalizer.k8sgpt.ai", finalizer) {
			mutationControllerLog.Info("Mutation has finalizer, proceeding")
		}
		// After our inspection we must delete the object finaliser to release the hold
		mutation.Finalizers = removeFinalizer(mutation.Finalizers, "mutation.finalizer.k8sgpt.ai")
		// update object
		if err := r.Client.Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
	}

	switch mutation.Status.Phase {
	case corev1alpha1.AutoRemediationPhaseNotStarted:
		if r.ServerQueryClient == nil {
			r.ServerQueryClient = shared.GetServerQueryClient()
		}
		if r.ServerQueryClient == nil {
			mutationControllerLog.Info("K8sGPT client not ready, requeuing proposal")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, nil
		}
		// This phase means that there is an origin configuration, resource and result
		// It needs an additional API call to determine targetConfiguration (mutation)
		// The goal now is to set the target Configuration and move phases to InProgress
		// Get the actual result from the reference
		var result corev1alpha1.Result
		err := r.Client.Get(ctx, client.ObjectKey{Name: mutation.Spec.ResultRef.Name,
			Namespace: mutation.Spec.ResultRef.Namespace}, &result)
		if err != nil {
			mutationControllerLog.Error(err, "Unable to retrieve result from reference",
				"Name", mutation.Spec.ResultRef.Name)
			return ctrl.Result{Requeue: false}, err
		}

		queryResponse, err := (*r.ServerQueryClient).Query(context.Background(), &schemav1.QueryRequest{
			Backend: backend,
			Query: fmt.Sprintf(prompts.Mutation_prompt, result.Spec.Details,
				mutation.Spec.OriginConfiguration),
		})
		if err != nil {
			mutationControllerLog.Error(err, "unable to query K8sGPT")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, nil
		}
		if queryResponse.GetResponse() == "{null}" {
			mutationControllerLog.Info("Unable to progress with this mutation, unknown solution", "name", mutation.Name)
			mutation.Status.Message = "No known fix"

			err := r.Client.Status().Update(ctx, &mutation)
			if err != nil {
				mutationControllerLog.Error(err, "unable to update mutation status")
				return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
			}
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime * 10}, nil
		}
		proposal, err := imageSelectionProposal(mutation.Spec.OriginConfiguration, queryResponse.GetResponse())
		if err != nil {
			mutationControllerLog.Info("Rejecting invalid image selection proposal", "mutation", mutation.Name, "error", err.Error())
			mutation.Status.Phase = corev1alpha1.AutoRemediationAborted
			mutation.Status.Message = "Invalid image selection proposal: " + err.Error()
			mutation.Status.PolicyDecision = "rejected"
			if updateErr := r.Client.Update(ctx, &mutation); updateErr != nil {
				return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, updateErr
			}
			return ctrl.Result{}, nil
		}
		mutationControllerLog.Info("Got mutation targetConfiguration for", "mutation", mutation.Name)
		mutation.Spec.TargetConfiguration = proposal
		mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseInProgress
		mutation.Status.Message = "In Progress"
		if err := r.Client.Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		mutationControllerLog.Info("Updated mutation status to InProgress", "mutation", mutation.Name)
		return ctrl.Result{RequeueAfter: util.NotStartedRequeueTime}, err
	case corev1alpha1.AutoRemediationPhaseInProgress:
		// This means that the executor has applied the configuration, and we are
		// in a period of waiting for result to expire, therefore showing success
		// here we loop through mutations and apply them
		// we will also check if the result has expired

		if mutation.Spec.TargetConfiguration == "" {
			mutationControllerLog.Info("Target configuration is not set, this shouldn't occur at this phase", "mutation", mutation.Name)
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, nil
		}
		// Authorization is determined by the semantic policy gate in ResourceToExecution.
		// Convert the spec.targetConfiguration to an Object
		// 1. Get the GVK from the Kind string
		obj, err := util.FromConfig(util.FromObjectConfig{
			Kind:      mutation.Spec.ResourceRef.Kind,
			GvkStr:    mutation.Spec.ResourceGVK,
			Config:    mutation.Spec.TargetConfiguration,
			Name:      mutation.Spec.ResourceRef.Name,
			Namespace: mutation.Spec.ResourceRef.Namespace,
		})
		if err != nil {
			mutationControllerLog.Error(err, "unable to convert targetConfiguration to object", "mutation", mutation.Name)
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		return conversions.ResourceToExecution(conversions.ObjectExecutionConfig{
			Ctx:      ctx,
			Rc:       r.Client,
			Log:      mutationControllerLog,
			Obj:      obj,
			Mutation: mutation,
		})
	case corev1alpha1.AutoRemediationPhaseCompleted:
		// this    is when the execute/apply is completed
		mutationControllerLog.Info("Mutation has been completed", "mutation", mutation.Name)
		// find the original result
		return r.doesResultExist(ctx, mutation)
	case corev1alpha1.AutoRemediationPhaseSuccessful:
		// This phase occurs when the result has expired and no longer exists
		mutationControllerLog.Info("Mutation has been successful", "mutation", mutation.Name)
		return ctrl.Result{RequeueAfter: util.SuccessfulRequeueTime}, nil
	case corev1alpha1.AutoRemediationPending:
		// This phase will occur when a result does not expire after phase completed
		mutationControllerLog.Info("Mutation is pending, result still exists", "mutation", mutation.Name)
		return r.doesResultExist(ctx, mutation)
	}
	return ctrl.Result{}, nil
}

func removeFinalizer(finalizers []string, target string) []string {
	filtered := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		if finalizer != target {
			filtered = append(filtered, finalizer)
		}
	}
	return filtered
}

// SetupWithManager sets up the controller with the Manager.
func (r *MutationReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Mutation count will be set in the k8sgpt_controller, but this controller owns the metric
	mutationCount := r.MetricsBuilder.GetGaugeVec("k8sgpt_mutations_count")

	// Register the metrics
	metrics.Registry.MustRegister(
		mutationCount,
	)
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Mutation{}).
		Named("mutation").
		Complete(r)
}
func (r *MutationReconciler) doesResultExist(ctx context.Context, mutation corev1alpha1.Mutation) (ctrl.Result, error) {
	ready, err := r.targetReady(ctx, mutation)
	if err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	if !ready {
		mutation.Status.Phase = corev1alpha1.AutoRemediationPending
		mutation.Status.Message = "Waiting for target rollout"
		if err := r.Client.Update(ctx, &mutation); err != nil {
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		return ctrl.Result{RequeueAfter: util.PendingRequeueTime}, nil
	}
	var result corev1alpha1.Result
	if err := r.Get(ctx, client.ObjectKey{
		Name: mutation.Spec.ResultRef.Name, Namespace: mutation.Spec.ResultRef.Namespace}, &result); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseSuccessful
		mutation.Status.Message = "Successful"
		// update status
		if err := r.Client.Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation status")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
	} else {
		mutation.Status.Phase = corev1alpha1.AutoRemediationPending
		mutation.Status.Message = "Pending"
		if err := r.Client.Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation status")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		return ctrl.Result{RequeueAfter: util.PendingRequeueTime}, nil
	}
	return ctrl.Result{RequeueAfter: util.CompletedRequeueTime}, nil
}

// targetReady checks post-mutation health where Kubernetes exposes a
// meaningful rollout condition. Other registered policies may add their own
// verification rules; unknown resource types remain dependent on re-analysis.
func (r *MutationReconciler) targetReady(ctx context.Context, mutation corev1alpha1.Mutation) (bool, error) {
	gv, err := schema.ParseGroupVersion(mutation.Spec.ResourceGVK)
	if err != nil {
		return false, fmt.Errorf("parse target GVK: %w", err)
	}
	gvk := gv.WithKind(mutation.Spec.ResourceRef.Kind)
	if gvk.Group != "apps" || gvk.Version != "v1" || gvk.Kind != "Deployment" {
		return true, nil
	}
	target := &unstructured.Unstructured{}
	target.SetGroupVersionKind(gvk)
	if err := r.Get(ctx, client.ObjectKey{Namespace: mutation.Spec.ResourceRef.Namespace, Name: mutation.Spec.ResourceRef.Name}, target); err != nil {
		return false, err
	}
	if expectedUID := mutation.Spec.ResourceRef.UID; expectedUID != "" && target.GetUID() != expectedUID {
		return false, fmt.Errorf("target Deployment UID changed since proposal")
	}
	expectedReplicas, found, err := unstructured.NestedInt64(target.Object, "spec", "replicas")
	if err != nil {
		return false, err
	}
	if !found {
		expectedReplicas = 1
	}
	observedGeneration, _, err := unstructured.NestedInt64(target.Object, "status", "observedGeneration")
	if err != nil {
		return false, err
	}
	availableReplicas, _, err := unstructured.NestedInt64(target.Object, "status", "availableReplicas")
	if err != nil {
		return false, err
	}
	return observedGeneration >= target.GetGeneration() && availableReplicas >= expectedReplicas, nil
}
