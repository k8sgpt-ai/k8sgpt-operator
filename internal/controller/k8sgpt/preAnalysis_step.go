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
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"
	Kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime"
)

type PreAnalysisStep struct {
	next   K8sGPT
	Signal chan types.InterControllerSignal
}

func (step *PreAnalysisStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting PreAnalysisStep")

	// try to upgrade version if already deployed
	if instance.hasReadyReplicas {
		deployment := instance.k8sgptDeployment

		imageURI := deployment.Spec.Template.Spec.Containers[0].Image
		imageRepository, imageVersion := step.parseImageURI(imageURI)

		// Update deployment image if change
		if imageRepository != instance.K8sgptConfig.Spec.Repository || imageVersion != instance.K8sgptConfig.Spec.Version {
			return step.updateDeploymentImage(instance)
		}

	}

	// If the deployment is active, we will query it directly for sis data
	address, err := Kclient.GenerateAddress(instance.Ctx, instance.R.Client, instance.K8sgptConfig)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	instance.logger.Info("K8sGPT address: " + address)

	instance.kclient, err = Kclient.NewClient(address)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	instance.logger.Info("K8sGPT client: " + fmt.Sprintf("%v", instance.kclient))
	instance.logger.Info("Sending signal to configure step")

	if instance.K8sgptConfig.Spec.AI.AutoRemediation.Enabled {
		step.Signal <- types.InterControllerSignal{
			K8sGPTClient: instance.kclient,
			Backend:      instance.K8sgptConfig.Spec.AI.Backend,
			K8sGPT:       instance.K8sgptConfig,
		}
	}
	instance.logger.Info("Signal sent to configure step")

	instance.logger.Info("Adding remote cache")
	// This will need a refactor in future...
	err = step.addRemoteCache(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}
	instance.logger.Info("Remote cache added")

	instance.logger.Info("Adding integrations")

	err = step.addIntegrations(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}
	instance.logger.Info("Integrations added")

	instance.logger.Info("ending PreAnalysisStep")

	return step.next.execute(instance)

}

func (step *PreAnalysisStep) setNext(next K8sGPT) {
	step.next = next
}

func (step *PreAnalysisStep) addRemoteCache(instance *K8sGPTInstance) error {
	if instance.K8sgptConfig.Spec.RemoteCache != nil || instance.K8sgptConfig.Spec.CustomAnalyzers != nil {
		return instance.kclient.AddConfig(instance.K8sgptConfig)
	}
	return nil
}

func (step *PreAnalysisStep) addIntegrations(instance *K8sGPTInstance) error {
	if instance.K8sgptConfig.Spec.Integrations != nil {
		return instance.kclient.AddIntegration(instance.K8sgptConfig)
	}
	return nil
}

func (step *PreAnalysisStep) updateDeploymentImage(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.k8sgptDeployment.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf(
		"%s:%s",
		instance.K8sgptConfig.Spec.Repository,
		instance.K8sgptConfig.Spec.Version,
	)
	err := instance.R.Update(instance.Ctx, instance.k8sgptDeployment)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name)
}

// https://kubernetes.io/docs/concepts/containers/images/#image-names
func (step *PreAnalysisStep) parseImageURI(uri string) (string, string) {
	// We have possible image variants:
	// - pause
	// - pause:v1.0.0
	// With registry
	// - fictional.Registry.example/imagename
	// - fictional.Registry.example:10443/imagename
	// - fictional.Registry.example/imagename:v1.0.0
	// - fictional.Registry.example:10443/imagename:v1.0.0

	var (
		repository string
		version    string
	)

	if strings.Contains(uri, "/") {
		parts := strings.SplitN(uri, "/", 2)
		registry := parts[0]
		name := parts[1]
		if strings.Contains(name, ":") {
			nameParts := strings.SplitN(name, ":", 2)
			repository = registry + "/" + nameParts[0]
			version = nameParts[1]
		} else {
			repository = registry + "/" + name
		}
	} else if strings.Contains(uri, ":") {
		imageParts := strings.SplitN(uri, ":", 2)
		repository = imageParts[0]
		version = imageParts[1]
	} else {
		repository = uri
	}

	return repository, version
}
