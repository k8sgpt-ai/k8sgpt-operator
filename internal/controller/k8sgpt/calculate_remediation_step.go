package k8sgpt

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/conversions"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type calculateRemediationStep struct {
	logger logr.Logger
	next   K8sGPT
}

const (
	mutationFinalizer = "mutation.finalizer.k8sgpt.ai"
)

func (step *calculateRemediationStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting RemediationStep")
	if !instance.K8sgptConfig.Spec.AI.AutoRemediation.Enabled {
		instance.logger.Info("calculateRemediationStep skipped because auto-remediation disabled")
		return ctrl.Result{Requeue: true, RequeueAfter: ReconcileSuccessInterval}, nil
	}
	latestResultList, err := EmitIfNotHistorical(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name, instance.K8sgptConfig)
	}

	if len(latestResultList.Items) == 0 {
		return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name, instance.K8sgptConfig)
	}
	for _, result := range latestResultList.Items {
		var res corev1alpha1.Result
		if err := instance.R.Get(instance.Ctx, client.ObjectKey{Namespace: result.Namespace, Name: result.Name}, &res); err != nil {
			return instance.R.FinishReconcile(err, false, result.Name, instance.K8sgptConfig)
		}
	}
	preEligibleResources := conversions.ResultsToEligibleResources(instance.K8sgptConfig,
		instance.R.Client,
		instance.R.Scheme,
		instance.logger.WithName("conversion"),
		latestResultList)

	// Merge results if they are duplicates or from the same origin e.g., pods in a rs
	eligibleResources := util.Deduplicate(preEligibleResources, step.logger)

	step.logger.Info("eligibleResources", "count", len(eligibleResources))

	// Create mutations for eligible resources
	for _, eligibleResource := range eligibleResources {
		// Get the GVK from the scheme
		gvks, _, err := instance.R.Scheme.ObjectKinds(instance.K8sgptConfig)
		if err != nil {
			instance.logger.Error(err, "Failed to get GVK for K8sGPT resource")
			return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name, instance.K8sgptConfig)
		}
		if len(gvks) == 0 {
			err := fmt.Errorf("no GVK found for K8sGPT resource")
			instance.logger.Error(err, "Unable to set OwnerReference for Mutation")
			return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name, instance.K8sgptConfig)
		}
		mutation := corev1alpha1.Mutation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eligibleResource.ResultRef.Name,
				Namespace: instance.K8sgptConfig.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(instance.K8sgptConfig, gvks[0]),
				},
			},

			Spec: corev1alpha1.MutationSpec{
				ResourceRef:         eligibleResource.ObjectRef,
				ResourceGVK:         eligibleResource.GVK,
				ResultRef:           eligibleResource.ResultRef,
				OriginConfiguration: eligibleResource.OriginConfiguration,
				TargetConfiguration: "",
			},
			Status: corev1alpha1.MutationStatus{
				Phase:   corev1alpha1.AutoRemediationPhaseNotStarted,
				Message: "Not Started",
			},
		}
		mutation.Finalizers = append(mutation.Finalizers, mutationFinalizer)
		// Check if the mutation exists, else create it
		mutationKey := client.ObjectKey{Namespace: instance.K8sgptConfig.Namespace, Name: eligibleResource.ResultRef.Name}
		var existingMutation corev1alpha1.Mutation
		if err := instance.R.Get(instance.Ctx, mutationKey, &existingMutation); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name, instance.K8sgptConfig)
			}
			if err := instance.R.Create(instance.Ctx, &mutation); err != nil {
				return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name, instance.K8sgptConfig)
			}
		} else {
			// keep track of it's status
		}
	}
	mutationCounter := instance.R.MetricsBuilder.GetGaugeVec("k8sgpt_mutations_count")
	if mutationCounter != nil {
		mutationCounter.WithLabelValues("mutations", "pk8sgpt").Set(float64(len(eligibleResources)))
	}
	step.logger.Info("ending calculateRemediationStep")
	return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name, instance.K8sgptConfig)
}

func (step *calculateRemediationStep) setNext(next K8sGPT) {
	step.next = next
}
