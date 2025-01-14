package controller

import (
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func emitIfNotHistorical(instance *K8sGPTInstance) (*corev1alpha1.ResultList, error) {
	latestResultList := &corev1alpha1.ResultList{}
	err := instance.r.List(instance.ctx, latestResultList, client.MatchingLabels(map[string]string{
		"k8sgpts.k8sgpt.ai/name":      instance.k8sgptConfig.Name,
		"k8sgpts.k8sgpt.ai/namespace": instance.k8sgptConfig.Namespace,
	}))
	return latestResultList, err
}
