package resources

import (
	"context"
	"reflect"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ResultOperation string

const (
	CreatedResult ResultOperation = "created"
	UpdatedResult ResultOperation = "updated"
	NoOpResult    ResultOperation = "historical"
)

func MapResults(i integrations.Integrations, resultsSpec []v1alpha1.ResultSpec, config v1alpha1.K8sGPT) (map[string]v1alpha1.Result, error) {
	namespace := config.Namespace
	backend := config.Spec.AI.Backend
	backstageEnabled := config.Spec.ExtraOptions != nil && config.Spec.ExtraOptions.Backstage.Enabled
	rawResults := make(map[string]v1alpha1.Result)
	for _, resultSpec := range resultsSpec {
		name := strings.ReplaceAll(resultSpec.Name, "-", "")
		name = strings.ReplaceAll(name, "/", "")
		result := GetResult(resultSpec, name, namespace, backend, resultSpec.Details)
		labels := map[string]string{
			"k8sgpts.k8sgpt.ai/name":      config.Name,
			"k8sgpts.k8sgpt.ai/namespace": config.Namespace,
		}
		if config.Spec.AI != nil {
			labels["k8sgpts.k8sgpt.ai/backend"] = config.Spec.AI.Backend
		}
		if backstageEnabled {
			// add Backstage label
			backstageLabel := i.BackstageLabel(resultSpec)
			for k, v := range backstageLabel {
				labels[k] = v
			}
		}
		result.SetLabels(labels)

		rawResults[name] = result
	}
	return rawResults, nil
}

func GetResult(resultSpec v1alpha1.ResultSpec, name, namespace, backend string, detail string) v1alpha1.Result {
	resultSpec.Backend = backend
	resultSpec.Details = detail
	return v1alpha1.Result{
		Spec: resultSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
func CreateOrUpdateResult(ctx context.Context, c client.Client, res v1alpha1.Result) (*v1alpha1.Result, error) {
	logger := log.FromContext(ctx)
	var existing v1alpha1.Result
	if err := c.Get(ctx, client.ObjectKey{Namespace: res.Namespace, Name: res.Name}, &existing); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := c.Create(ctx, &res); err != nil {
			return nil, err
		}
		logger.Info("Created result", "name", res.Name)
		return &existing, nil
	}
	if len(existing.Spec.Error) == len(res.Spec.Error) && reflect.DeepEqual(res.Labels, existing.Labels) {
		existing.Status.LifeCycle = string(NoOpResult)
		err := c.Status().Update(ctx, &existing)
		return &existing, err
	}

	existing.Spec = res.Spec
	existing.Labels = res.Labels
	if err := c.Update(ctx, &existing); err != nil {
		return nil, err
	}
	existing.Status.LifeCycle = string(UpdatedResult)
	if err := c.Status().Update(ctx, &existing); err != nil {
		return nil, err
	}
	logger.Info("Updated result", "name", res.Name)
	return &existing, nil
}
