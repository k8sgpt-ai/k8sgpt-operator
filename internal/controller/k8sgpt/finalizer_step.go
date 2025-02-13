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
	if !instance.K8sgptConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if utils.ContainsString(instance.K8sgptConfig.GetFinalizers(), FinalizerName) {

			// Delete any external resources associated with the instance
			err := resources.Sync(instance.Ctx, instance.R.Client, *instance.K8sgptConfig, resources.DestroyOp)
			if err != nil {
				return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
			}
			controllerutil.RemoveFinalizer(instance.K8sgptConfig, FinalizerName)
			if err := instance.R.Update(instance.Ctx, instance.K8sgptConfig); err != nil {
				return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
			}
		}
		// Stop reconciliation as the item is being deleted
		return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name)
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object. This is equivalent
	// registering our finalizer.
	if !utils.ContainsString(instance.K8sgptConfig.GetFinalizers(), FinalizerName) {
		controllerutil.AddFinalizer(instance.K8sgptConfig, FinalizerName)
		if err := instance.R.Update(instance.Ctx, instance.K8sgptConfig); err != nil {
			return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
		}
	}
	instance.logger.Info("ending FinalizerStep")

	return step.next.execute(instance)

}

func (step *FinalizerStep) setNext(next K8sGPT) {
	step.next = next
}
