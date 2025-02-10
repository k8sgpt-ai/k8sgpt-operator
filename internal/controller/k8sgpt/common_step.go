package k8sgpt

import (
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func EmitIfNotHistorical(instance *K8sGPTInstance) (*corev1alpha1.ResultList, error) {
	latestResultList := &corev1alpha1.ResultList{}
	err := instance.R.List(instance.Ctx, latestResultList, client.MatchingLabels(map[string]string{
		"k8sgpts.k8sgpt.ai/name":      instance.K8sgptConfig.Name,
		"k8sgpts.k8sgpt.ai/namespace": instance.K8sgptConfig.Namespace,
	}))
	return latestResultList, err
}
