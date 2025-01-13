package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type eligibleResource struct {
	Result corev1alpha1.Result
	Object client.Object
}
type calculateRemediationStep struct {
	logger logr.Logger
	next   K8sGPT
}

func (step *calculateRemediationStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting RemediationStep")
	if !instance.k8sgptConfig.Spec.AI.AutoRemediation.Enabled {
		instance.logger.Info("calculateRemediationStep skipped because auto-remediation disabled")
		return ctrl.Result{}, nil
	}
	latestResultList, err := emitIfNotHistorical(instance)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	if len(latestResultList.Items) == 0 {
		return instance.r.FinishReconcile(nil, false, instance.k8sgptConfig.Name)
	}
	for _, result := range latestResultList.Items {
		var res corev1alpha1.Result
		if err := instance.r.Get(instance.ctx, client.ObjectKey{Namespace: result.Namespace, Name: result.Name}, &res); err != nil {
			return instance.r.FinishReconcile(err, false, result.Name)
		}
	}
	//
	eligibleResources := step.parseEligibleResources(instance, latestResultList)
	step.logger.Info("eligibleResources", "count", len(eligibleResources))

	step.logger.Info("ending calculateRemediationStep")
	return instance.r.FinishReconcile(nil, false, instance.k8sgptConfig.Name)
}

func (step *calculateRemediationStep) parseEligibleResources(instance *K8sGPTInstance, items *corev1alpha1.ResultList) []eligibleResource {
	// Currently this step is a watershed to ensure we are able to control directly what resources
	// are going to be mutated
	// In the future, we will have a more sophisticated way to determine which resources are eligible
	// for remediation
	var eligibleResources = []eligibleResource{}
	context := context.Background()
	for _, item := range items.Items {
		//demangle the name of the resource
		names := strings.Split(item.Spec.Name, "/")
		namespace := names[0]
		name := names[1]
		if len(names) != 2 {
			instance.logger.Error(fmt.Errorf("invalid resource name"), "unable to parse resource name", "Resource", item.Name)
			continue
		}
		// Support Service/Ingress currently
		switch item.Spec.Kind {
		case "Service":
			var service corev1.Service
			if err := instance.r.Get(context, client.ObjectKey{Namespace: namespace, Name: name}, &service); err != nil {
				instance.logger.Error(err, "unable to fetch Service", "Service", item.Name)
				continue
			}
			eligibleResources = append(eligibleResources, eligibleResource{Result: item, Object: &service})

		case "Ingress":
			var ingress networkingv1.Ingress
			if err := instance.r.Get(context, client.ObjectKey{Namespace: namespace, Name: name}, &ingress); err != nil {
				instance.logger.Error(err, "unable to fetch Ingress", "Ingress", item.Name)
				continue
			}
			eligibleResources = append(eligibleResources, eligibleResource{Result: item, Object: &ingress})
		}
	}
	return eligibleResources
}

func (step *calculateRemediationStep) setNext(next K8sGPT) {
	step.next = next
}
