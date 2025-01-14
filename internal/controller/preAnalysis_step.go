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
	"fmt"
	"strings"

	Kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime"
)

type PreAnalysisStep struct {
	next K8sGPT
}

func (step *PreAnalysisStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting PreAnalysisStep")

	// try to upgrade version if already deployed
	if instance.hasReadyReplicas {
		deployment := instance.k8sgptDeployment

		imageURI := deployment.Spec.Template.Spec.Containers[0].Image
		imageRepository, imageVersion := step.parseImageURI(imageURI)

		// Update deployment image if change
		if imageRepository != instance.k8sgptConfig.Spec.Repository || imageVersion != instance.k8sgptConfig.Spec.Version {
			return step.updateDeploymentImage(instance)
		}

	}

	// If the deployment is active, we will query it directly for sis data
	address, err := Kclient.GenerateAddress(instance.ctx, instance.r.Client, instance.k8sgptConfig)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	instance.logger.Info(fmt.Sprintf("K8sGPT address: %s\n", address))

	instance.kclient, err = Kclient.NewClient(address)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	// This will need a refactor in future...
	err = step.addRemoteCache(instance)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	err = step.addIntegrations(instance)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	instance.logger.Info("ending PreAnalysisStep")

	return step.next.execute(instance)

}

func (step *PreAnalysisStep) setNext(next K8sGPT) {
	step.next = next
}

func (step *PreAnalysisStep) addRemoteCache(instance *K8sGPTInstance) error {
	if instance.k8sgptConfig.Spec.RemoteCache != nil || instance.k8sgptConfig.Spec.CustomAnalyzers != nil {
		return instance.kclient.AddConfig(instance.k8sgptConfig)
	}
	return nil
}

func (step *PreAnalysisStep) addIntegrations(instance *K8sGPTInstance) error {
	if instance.k8sgptConfig.Spec.Integrations != nil {
		return instance.kclient.AddIntegration(instance.k8sgptConfig)
	}
	return nil
}

func (step *PreAnalysisStep) updateDeploymentImage(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.k8sgptDeployment.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf(
		"%s:%s",
		instance.k8sgptConfig.Spec.Repository,
		instance.k8sgptConfig.Spec.Version,
	)
	err := instance.r.Update(instance.ctx, instance.k8sgptDeployment)
	if err != nil {
		return instance.r.FinishReconcile(err, false, instance.k8sgptConfig.Name)
	}

	return instance.r.FinishReconcile(nil, false, instance.k8sgptConfig.Name)
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
