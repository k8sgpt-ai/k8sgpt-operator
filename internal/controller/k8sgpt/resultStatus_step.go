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
package k8sgpt

import (
	"fmt"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/sinks"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResultStatusStep struct {
	next K8sGPT
}

func (step *ResultStatusStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting ResultStatusStep")

	// We emit when result Status is not historical
	// and when user configures a sink for the first time
	latestResultList, err := EmitIfNotHistorical(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	if len(latestResultList.Items) == 0 {
		return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name)
	}

	sinkEnabled, sinkType, err := step.initSinkType(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	err = step.processLatestResults(instance, sinkEnabled, sinkType, latestResultList)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	instance.logger.Info("ending ResultStatusStep")

	return step.next.execute(instance)
}

func (step *ResultStatusStep) setNext(next K8sGPT) {
	step.next = next
}

func (step *ResultStatusStep) initSinkType(instance *K8sGPTInstance) (bool, sinks.ISink, error) {
	sinkEnabled := instance.K8sgptConfig.Spec.Sink != nil && instance.K8sgptConfig.Spec.Sink.Type != "" && (instance.K8sgptConfig.Spec.Sink.Endpoint != "" || instance.K8sgptConfig.Spec.Sink.Secret != nil)
	var sinkType sinks.ISink

	if sinkEnabled {
		var sinkSecretValue string

		if instance.K8sgptConfig.Spec.Sink.Secret != nil {
			secret := &kcorev1.Secret{}
			secretNamespacedName := types.NamespacedName{
				Namespace: instance.req.Namespace,
				Name:      instance.K8sgptConfig.Spec.Sink.Secret.Name,
			}
			if err := instance.R.Get(instance.Ctx, secretNamespacedName, secret); err != nil {

				return sinkEnabled, sinkType, fmt.Errorf("could not find sink secret: %w", err)
			}

			sinkSecretValue = string(secret.Data[instance.K8sgptConfig.Spec.Sink.Secret.Key])
		}
		sinkType = sinks.NewSink(instance.K8sgptConfig.Spec.Sink.Type)
		sinkType.Configure(*instance.K8sgptConfig, *instance.R.SinkClient, sinkSecretValue)
	}

	return sinkEnabled, sinkType, nil

}

func (step *ResultStatusStep) processLatestResults(instance *K8sGPTInstance, sinkEnabled bool, sinkType sinks.ISink, latestResultList *corev1alpha1.ResultList) error {
	for _, result := range latestResultList.Items {
		var res corev1alpha1.Result
		if err := instance.R.Get(instance.Ctx, client.ObjectKey{Namespace: result.Namespace, Name: result.Name}, &res); err != nil {
			return err
		}

		if sinkEnabled {
			if res.Status.LifeCycle != string(resources.NoOpResult) || res.Status.Webhook == "" {
				if err := sinkType.Emit(res.Spec); err != nil {
					return err
				}
				res.Status.Webhook = instance.K8sgptConfig.Spec.Sink.Endpoint
			}
		} else {
			// Remove the Webhook status from results
			res.Status.Webhook = ""
		}
		if err := instance.R.Status().Update(instance.Ctx, &res); err != nil {
			return err
		}
	}

	return nil
}
