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
package controller

import (
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InitStep struct {
	next K8sGPT
}

func (step *InitStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting InitStep")
	k8sgptConfig := &corev1alpha1.K8sGPT{}
	err := instance.r.Get(instance.ctx, instance.req.NamespacedName, k8sgptConfig)
	if err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	instance.k8sgptConfig = k8sgptConfig

	instance.logger.Info("ending InitStep")

	return step.next.execute(instance)

}

func (step *InitStep) setNext(next K8sGPT) {
	step.next = next
}
