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
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/resources"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	FinalizerName = "k8sgpt.ai/finalizer"
)

type FinalizerStep struct {
	next K8sGPT
}

func (step *FinalizerStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting FinalizerStep")
	FinalizerName := FinalizerName
	if !instance.k8sgptConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if utils.ContainsString(instance.k8sgptConfig.GetFinalizers(), FinalizerName) {

			// Delete any external resources associated with the instance
			err := resources.Sync(instance.ctx, instance.r.Client, *instance.k8sgptConfig, resources.DestroyOp)
			if err != nil {
				return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
			}
			controllerutil.RemoveFinalizer(instance.k8sgptConfig, FinalizerName)
			if err := instance.r.Update(instance.ctx, instance.k8sgptConfig); err != nil {
				return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
			}
		}
		// Stop reconciliation as the item is being deleted
		return instance.r.FinishReconcile(nil, false, instance.k8sgptConfig.Name)
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !utils.ContainsString(instance.k8sgptConfig.GetFinalizers(), FinalizerName) {
		controllerutil.AddFinalizer(instance.k8sgptConfig, FinalizerName)
		if err := instance.r.Update(instance.ctx, instance.k8sgptConfig); err != nil {
			return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
		}
	}
	instance.logger.Info("ending FinalizerStep")

	return step.next.execute(instance)

}

func (step *FinalizerStep) setNext(next K8sGPT) {
	step.next = next
}
