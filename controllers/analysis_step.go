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
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// allowBackendAIRequest a circuit breaker that switching on/off backend AI calls
	allowBackendAIRequest = true
	// analysisRetryCount is for the number of analysis failures
	analysisRetryCount int
)

type AnalysisStep struct {
	next                K8sGPT
	enableResultLogging bool
	logger              logr.Logger
}

type AnalysisLogStatement struct {
	Name    string
	Error   string
	Details string
}

func (step *AnalysisStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting AnalysisStep")

	response, err := instance.kclient.ProcessAnalysis(*instance.k8sgptDeployment, instance.k8sgptConfig, allowBackendAIRequest)
	if err != nil {
		if instance.k8sgptConfig.Spec.AI.Enabled {
			step.incK8sgptNumberOfFailedBackendAICalls(instance)
			step.handleAIFailureBackoff(instance)
		}
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	// reset analysisRetryCount
	analysisRetryCount = 0

	// Update metrics count
	if instance.k8sgptConfig.Spec.AI.Enabled && len(response.Results) > 0 {
		step.incK8sgptNumberOfFailedBackendAICalls(instance)
	}

	// Parse the k8sgpt-deployment response into a list of results
	step.setk8sgptNumberOfResults(instance, float64(len(response.Results)))

	rawResults, err := resources.MapResults(*instance.r.Integrations, response.Results, *instance.k8sgptConfig)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	// Prior to creating or updating any results we will delete any stale results that
	// no longer are relevent, we can do this by using the resultSpec composed name against
	// the custom resource name
	err = step.cleanUpStaleResults(rawResults, instance)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	// At this point we are able to loop through our rawResults and create them or update
	// them as needed
	err = step.processRawResults(rawResults, instance)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	instance.logger.Info("ending AnalysisStep")

	return step.next.execute(instance)

}

func (step *AnalysisStep) setNext(next K8sGPT) {
	step.next = next
}

func (step *AnalysisStep) handleAIFailureBackoff(instance *K8sGPTInstance) {
	if instance.k8sgptConfig.Spec.AI.BackOff.Enabled {
		if analysisRetryCount > instance.k8sgptConfig.Spec.AI.BackOff.MaxRetries {
			allowBackendAIRequest = false
			instance.logger.Info(fmt.Sprintf("Disabled AI backend %s due to failures exceeding max retries\n", instance.k8sgptConfig.Spec.AI.Backend))
			analysisRetryCount = 0
		}
		analysisRetryCount++
	}
}

func (step *AnalysisStep) incK8sgptNumberOfFailedBackendAICalls(instance *K8sGPTInstance) {
	reconcileErrorCounter := instance.r.MetricsBuilder.GetCounterVec("k8sgpt_number_of_failed_backend_ai_calls")
	if reconcileErrorCounter != nil {
		reconcileErrorCounter.WithLabelValues(instance.k8sgptConfig.Spec.AI.Backend, instance.k8sgptDeployment.Name, instance.k8sgptDeployment.Namespace, instance.k8sgptConfig.Name).Inc()
	}
}

func (step *AnalysisStep) setk8sgptNumberOfResults(instance *K8sGPTInstance, result float64) {
	numberOfResultsGauge := instance.r.MetricsBuilder.GetGaugeVec("k8sgpt_number_of_results")
	if numberOfResultsGauge != nil {
		numberOfResultsGauge.WithLabelValues(instance.k8sgptConfig.Name).Set(result)
	}

}

func (step *AnalysisStep) cleanUpStaleResults(rawResults map[string]corev1alpha1.Result, instance *K8sGPTInstance) error {
	resultList := &corev1alpha1.ResultList{}
	err := instance.r.List(instance.ctx, resultList, client.MatchingLabels(map[string]string{
		"k8sgpts.k8sgpt.ai/name":      instance.k8sgptConfig.Name,
		"k8sgpts.k8sgpt.ai/namespace": instance.k8sgptConfig.Namespace,
	}))
	if err != nil {
		return err
	}

	if len(resultList.Items) > 0 {
		for _, result := range resultList.Items {
			instance.logger.Info(fmt.Sprintf("checking if %s is still relevant", result.Name))
			if _, ok := rawResults[result.Name]; !ok {
				err := instance.r.Delete(instance.ctx, &result)
				if err != nil {
					return err
				}
				numberOfResultsByType := instance.r.MetricsBuilder.GetGaugeVec("k8sgpt_number_of_results_by_type")
				if numberOfResultsByType != nil {
					numberOfResultsByType.WithLabelValues(result.Spec.Kind, result.Spec.Name, instance.k8sgptConfig.Name).Desc()
				}

			}
		}
	}
	return nil
}

func (step *AnalysisStep) processRawResults(rawResults map[string]corev1alpha1.Result, instance *K8sGPTInstance) error {

	numberOfResultsByType := instance.r.MetricsBuilder.GetGaugeVec("k8sgpt_number_of_results_by_type")
	if numberOfResultsByType != nil {
		numberOfResultsByType.Reset()
	}
	for _, result := range rawResults {
		result, err := resources.CreateOrUpdateResult(instance.ctx, instance.r.Client, result)
		if err != nil {
			return err
		}
		// Rather than using the raw corev1alpha.Result from the RPC, we log on the v1alpha.Result from KubeBuilder
		if step.enableResultLogging {

			// check if result.spec.error is nil
			var errorString = ""
			if len(result.Spec.Error) > 0 {
				errorString = fmt.Sprintf("Error %s", result.Spec.Error)
			}
			logStatement := AnalysisLogStatement{
				Name:    result.Spec.Name,
				Details: result.Spec.Details,
				Error:   errorString,
			}
			// to json
			jsonBytes, err := json.Marshal(logStatement)
			if err != nil {
				step.logger.Error(err, "Error marshalling logStatement")
			}
			step.logger.Info(string(jsonBytes))
		}
		numberOfResultsByType.WithLabelValues(result.Spec.Kind, result.Spec.Name, instance.k8sgptConfig.Name).Inc()
	}

	return nil
}
