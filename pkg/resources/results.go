package resources

import (
	"context"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CreatedResult string = "created"
	UpdatedResult        = "updated"
	NoOpResult           = "historical"
)

func MapResults(i integrations.Integrations, resultsSpec []v1alpha1.ResultSpec, config v1alpha1.K8sGPT) (map[string]v1alpha1.Result, error) {
	namespace := config.Namespace
	backend := config.Spec.Backend
	backstageEnabled := config.Spec.ExtraOptions != nil && config.Spec.ExtraOptions.Backstage.Enabled
	rawResults := make(map[string]v1alpha1.Result)
	for _, resultSpec := range resultsSpec {
		name := strings.ReplaceAll(resultSpec.Name, "-", "")
		name = strings.ReplaceAll(name, "/", "")
		result := GetResult(resultSpec, name, namespace, backend)
		if backstageEnabled {
			backstageLabel, err := i.BackstageLabel(resultSpec)
			if err != nil {
				return nil, err
			}
			// add Backstage label
			result.ObjectMeta.Labels = backstageLabel
		}

		rawResults[name] = result
	}
	return rawResults, nil
}

func GetResult(resultSpec v1alpha1.ResultSpec, name, namespace, backend string) v1alpha1.Result {
	resultSpec.Backend = backend
	return v1alpha1.Result{
		Spec: resultSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func CreateOrUpdateResult(ctx context.Context, c client.Client, result v1alpha1.Result, config v1alpha1.K8sGPT) (string, error) {
	// Check if the result already exists
	var existingResult v1alpha1.Result
	err := c.Get(ctx, client.ObjectKey{Namespace: config.Namespace,
		Name: result.Name}, &existingResult)
	if err != nil {
		// if the result doesn't exist, we will create it
		if errors.IsNotFound(err) {
			err = c.Create(ctx, &result)
			if err != nil {
				return "", err
			}
			result.Status.Type = CreatedResult
			_ = c.Status().Update(ctx, &result)
			return CreatedResult, nil

		} else {
			return "", err
		}
	}

	// If the result error and solution has changed, we will update CR
	updateRequired := existingResult.Spec.Details != result.Spec.Details || existingResult.Spec.Name != result.Spec.Name || existingResult.Spec.Backend != result.Spec.Backend
	if updateRequired {
		existingResult.Spec = result.Spec
		existingResult.Labels = result.Labels
		err = c.Update(ctx, &existingResult)
		if err != nil {
			return "", err
		}
		existingResult.Status.Type = UpdatedResult
		_ = c.Status().Update(ctx, &existingResult)
		return UpdatedResult, err
	}
	existingResult.Status.Type = NoOpResult
	_ = c.Status().Update(ctx, &existingResult)
	return "", nil
}
