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
	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/channel_types"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// MutationReconciler reconciles a Mutation object
type MutationReconciler struct {
	client.Client
	logger            logr.Logger
	Scheme            *runtime.Scheme
	ServerQueryClient *rpc.ServerQueryServiceClient
	Signal            chan channel_types.InterControllerSignal
	RemoteBackend     string
}

var (
	mutationControllerLog = ctrl.Log.WithName("mutation-controller")
)

// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.k8sgpt.ai,resources=mutations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mutation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its ResultRef here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *MutationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// get the mutation resource
	var mutation corev1alpha1.Mutation
	if err := r.Get(ctx, req.NamespacedName, &mutation); err != nil {
		mutationControllerLog.Error(err, "unable to fetch mutation")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	mutationControllerLog.Info("Reconciling mutation", "mutation", mutation.Name)

	if r.ServerQueryClient == nil {
		mutationControllerLog.Info("Awaiting signal for K8sGPT connection")
		signal := <-r.Signal
		c := rpc.NewServerQueryServiceClient(signal.K8sGPTClient.Conn)
		r.ServerQueryClient = &c
		r.RemoteBackend = signal.Backend
		mutationControllerLog.Info("Received signal for K8sGPT connection")
	}

	switch mutation.Status.Phase {
	case corev1alpha1.AutoRemediationPhaseNotStarted:
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
		}

		queryResponse, err := (*r.ServerQueryClient).Query(context.Background(), &schemav1.QueryRequest{
			Backend: r.RemoteBackend,
			Query: fmt.Sprintf(mutation_prompt, result.Spec.Details,
				mutation.Spec.OriginConfiguration),
		})
		if err != nil {
			mutationControllerLog.Error(err, "unable to query K8sGPT")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// compute similarity score
		score := util.SimilarityScore(mutation.Spec.OriginConfiguration, queryResponse.GetResponse())
		mutationControllerLog.Info("Similarity score", "score", score)

		mutationControllerLog.Info("Got mutation targetConfiguration for", "mutation", mutation.Name)
		mutation.Spec.TargetConfiguration = queryResponse.GetResponse()
		mutation.Spec.SimilarityScore = fmt.Sprintf("%f", score)
		// Update the spec (if needed)
		if err := r.Client.Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		mutationControllerLog.Info("Updated mutation with targetConfiguration", "mutation", mutation.Name)
		// Update the status
		mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseInProgress
		if err := r.Client.Status().Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation status")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		mutationControllerLog.Info("Updated mutation status to InProgress", "mutation", mutation.Name)
		return ctrl.Result{}, nil
	case corev1alpha1.AutoRemediationPhaseInProgress:
		// This means that the executor has applied the configuration and we are
		// in a period of waiting for result to expire, therefore showing success
		// here we loop through mutations and apply them
		// we will also check if the result has expired

		if mutation.Spec.TargetConfiguration == "" {
			mutationControllerLog.Info("Target configuration is not set, this shouldn't occur at this phase", "mutation", mutation.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// Convert the spec.targetConfiguration to an Object
		// 1. Get the GVK from the Kind string
		gv, err := schema.ParseGroupVersion(mutation.Spec.ResourceGVK)
		if err != nil {
			mutationControllerLog.Error(err, "unable to parse group version from kind", "kind", mutation.Kind)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		gvk := gv.WithKind(mutation.Spec.ResourceRef.Kind)
		// 2. Create an unstructured object
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		mutationControllerLog.Info(mutation.Spec.TargetConfiguration)
		// 3. Decode the targetConfiguration into the unstructured object
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(mutation.Spec.TargetConfiguration), 1000)
		if err := decoder.Decode(obj); err != nil {
			mutationControllerLog.Error(err, "unable to decode target configuration", "configuration", mutation.Spec.TargetConfiguration)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		// 4. Set the object's name and namespace (important for updates!)
		obj.SetName(mutation.Spec.ResourceRef.Name)
		obj.SetNamespace(mutation.Spec.ResourceRef.Namespace)

		// 5. Get the existing object
		var existingObj *unstructured.Unstructured
		if existingObj, err = getObjectFromObjectReference(ctx, r.Client, mutation.Spec.ResourceRef); err != nil {
			mutationControllerLog.Error(err, "unable to get existing object", "object", mutation.Spec.ResourceRef.Name)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		// 6. Apply the update using Patch
		patch := client.MergeFrom(obj) // Create a patch based on the current state of the object
		// print out the patch
		mutationControllerLog.Info("Apply patch to object", "patch", patch)
		if err := r.Client.Patch(ctx, existingObj, patch); err != nil {
			mutationControllerLog.Error(err, "unable to patch object", "object", obj.GetName())
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		mutationControllerLog.Info("Successfully patched object", "object", obj.GetName())
		mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseCompleted
		if err := r.Client.Status().Update(ctx, &mutation); err != nil {
			mutationControllerLog.Error(err, "unable to update mutation status")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		// This should be fine tuned, there needs to be time for the patch to be applied
		// And for a settling period
		return ctrl.Result{RequeueAfter: time.Second * 60}, nil
	case corev1alpha1.AutoRemediationPhaseCompleted:
		// this 	is when the execute/apply is completed
		mutationControllerLog.Info("Mutation has been completed", "mutation", mutation.Name)
		// find the original result
		var result corev1alpha1.Result
		if err := r.Get(ctx, client.ObjectKey{
			Name: mutation.Spec.ResultRef.Name, Namespace: mutation.Spec.ResultRef.Namespace}, &result); err != nil {
			mutationControllerLog.Error(err, "unable to get result mutation successful", "mutation", mutation.Name, "result", mutation.Spec.ResultRef.Name)
			mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseSuccessful
			// update status
			if err := r.Client.Status().Update(ctx, &mutation); err != nil {
				mutationControllerLog.Error(err, "unable to update mutation status")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		} else {
			mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseFailed
			if err := r.Client.Status().Update(ctx, &mutation); err != nil {
				mutationControllerLog.Error(err, "unable to update mutation status")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
		}
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	case corev1alpha1.AutoRemediationPhaseSuccessful:
		// This phase occurs when the result has expired and no longer exists
		mutationControllerLog.Info("Mutation has been successful", "mutation", mutation.Name)
		return ctrl.Result{RequeueAfter: time.Second * 120}, nil
	case corev1alpha1.AutoRemediationPhaseFailed:
		// This phase will occur when a result does not expire after phase completed
		mutationControllerLog.Info("Mutation has failed, result still exists", "mutation", mutation.Name)
		return ctrl.Result{RequeueAfter: time.Second * 120}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MutationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Mutation{}).
		Named("mutation").
		Complete(r)
}
func getObjectFromObjectReference(ctx context.Context, c client.Client, objRef corev1.ObjectReference) (*unstructured.Unstructured, error) {
	// 1. Parse GroupVersion from the ObjectReference
	gv, err := schema.ParseGroupVersion(objRef.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group version: %w", err)
	}

	// 2. Construct GroupVersionKind
	gvk := gv.WithKind(objRef.Kind)

	// 3. Create an Unstructured object
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	// 4. Fetch the object using the client
	err = c.Get(ctx, client.ObjectKey{Namespace: objRef.Namespace, Name: objRef.Name}, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return obj, nil
}
