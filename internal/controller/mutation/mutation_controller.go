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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MutationReconciler reconciles a Mutation object
type MutationReconciler struct {
	client.Client
	logger            logr.Logger
	Scheme            *runtime.Scheme
	ServerQueryClient rpc.ServerQueryServiceClient
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
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *MutationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	mutationControllerLog.Info("Awaiting signal for K8sGPT connection")
	select {
	case signal := <-r.Signal:
		c := rpc.NewServerQueryServiceClient(signal.K8sGPTClient.Conn)
		r.ServerQueryClient = c
		r.RemoteBackend = signal.Backend
		mutationControllerLog.Info("Received signal for K8sGPT connection")
	}

	// List all mutations in all namespaces
	var mutations corev1alpha1.MutationList
	if err := r.List(ctx, &mutations); err != nil {
		mutationControllerLog.Error(err, "unable to list mutations")
		return ctrl.Result{}, err
	}
	mutationControllerLog.Info("Number of mutations", "count", len(mutations.Items))
	// Loop through mutation states

	for _, mutation := range mutations.Items {
		switch mutation.Status.Phase {
		case corev1alpha1.AutoRemediationPhaseNotStarted:
			// This phase means that there is an origin configuration, resource and result
			// It needs an additional API call to determine targetConfiguration (mutation)
			// The goal now is to set the target Configuration and move phases to InProgress
			queryResponse, err := r.ServerQueryClient.Query(context.Background(), &schemav1.QueryRequest{
				Backend: r.RemoteBackend,
				Query: fmt.Sprintf(mutation_prompt, mutation.Spec.Result.Spec.Details,
					mutation.Spec.OriginConfiguration),
			})
			if err != nil {
				mutationControllerLog.Error(err, "unable to query K8sGPT")
				return ctrl.Result{}, err
			}
			mutation.Spec.TargetConfiguration = queryResponse.GetResponse()
			mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseInProgress
			if err := r.Update(ctx, &mutation); err != nil {
				mutationControllerLog.Error(err, "unable to update mutation")
				return ctrl.Result{}, err
			}
			break
		case corev1alpha1.AutoRemediationPhaseInProgress:
			// This means that the executor has applied the configuration and we are
			// in a period of waiting for result to expire, therefore showing success
			break
		case corev1alpha1.AutoRemediationPhaseCompleted:
			// this is when the execute/apply is completed
			break
		case corev1alpha1.AutoRemediationPhaseFailed:
			// This phase will occur when a result does not expire after phase completed
			break
		}
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
