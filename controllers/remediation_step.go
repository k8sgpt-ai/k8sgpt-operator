package controllers

import (
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type remediationStep struct {
	logger logr.Logger
	next   K8sGPT
}

func (step *remediationStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting RemediationStep")

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

	return instance.r.FinishReconcile(nil, false, instance.k8sgptConfig.Name)
}

func (step *remediationStep) setNext(next K8sGPT) {
	step.next = next
}
