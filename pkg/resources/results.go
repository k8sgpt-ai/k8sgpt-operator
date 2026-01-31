package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/integrations"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ResultOperation string

const (
	CreatedResult ResultOperation = "created"
	UpdatedResult ResultOperation = "updated"
	NoOpResult    ResultOperation = "historical"
)

// hashResultContent generates a hash of the meaningful parts of a ResultSpec
// to determine if the actual problem has changed, ignoring AI-generated details
// that may vary between analysis runs
func hashResultContent(spec v1alpha1.ResultSpec) string {
	// Create a stable representation of the key fields that identify the issue
	type hashableContent struct {
		Kind         string
		Name         string
		ParentObject string
		ErrorCount   int
		// Include error text but not sensitive data which may vary
		ErrorTexts []string
	}

	content := hashableContent{
		Kind:         spec.Kind,
		Name:         spec.Name,
		ParentObject: spec.ParentObject,
		ErrorCount:   len(spec.Error),
		ErrorTexts:   make([]string, len(spec.Error)),
	}

	// Extract error texts (excluding sensitive data that may vary)
	for i, err := range spec.Error {
		content.ErrorTexts[i] = err.Text
	}

	// Marshal to JSON for consistent hashing
	data, err := json.Marshal(content)
	if err != nil {
		// Fallback to a basic hash if marshaling fails
		return fmt.Sprintf("%s-%s-%s-%d", spec.Kind, spec.Name, spec.ParentObject, len(spec.Error))
	}

	// Generate SHA256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func MapResults(i integrations.Integrations, resultsSpec []v1alpha1.ResultSpec, config v1alpha1.K8sGPT) (map[string]v1alpha1.Result, error) {
	namespace := config.Namespace
	backend := config.Spec.AI.Backend
	backstageEnabled := config.Spec.ExtraOptions != nil && config.Spec.ExtraOptions.Backstage.Enabled
	rawResults := make(map[string]v1alpha1.Result)
	for _, resultSpec := range resultsSpec {
		name := strings.ReplaceAll(resultSpec.Name, "-", "")
		name = strings.ReplaceAll(name, "/", "")
		result := GetResult(resultSpec, name, namespace, backend, resultSpec.Details, &config)
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

func GetResult(resultSpec v1alpha1.ResultSpec, name, namespace, backend string, detail string, owner *v1alpha1.K8sGPT) v1alpha1.Result {
	resultSpec.Backend = backend
	resultSpec.Details = detail
	return v1alpha1.Result{
		Spec: resultSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, owner.GetObjectKind().GroupVersionKind()),
			},
		},
	}
}

func CreateOrUpdateResult(ctx context.Context, c client.Client, res v1alpha1.Result) (*v1alpha1.Result, error) {
	logger := log.FromContext(ctx)

	// Calculate content hash for the new result
	newHash := hashResultContent(res.Spec)

	var finalResult *v1alpha1.Result
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var existing v1alpha1.Result
		if err := c.Get(ctx, client.ObjectKey{Namespace: res.Namespace, Name: res.Name}, &existing); err != nil {
			if errors.IsNotFound(err) {
				// New result - create it with the hash
				res.Status.ContentHash = newHash
				res.Status.LifeCycle = string(CreatedResult)
				if err := c.Create(ctx, &res); err != nil {
					return err
				}
				logger.Info("Created result", "name", res.Name)
				finalResult = &res
				return nil
			}
			return err
		}

		// Check if the meaningful content has changed by comparing hashes
		// and labels (which may include backend changes)
		if existing.Status.ContentHash == newHash && reflect.DeepEqual(res.Labels, existing.Labels) {
			// Content is identical - mark as historical (no notification needed)
			existing.Status.LifeCycle = string(NoOpResult)
			if err := c.Status().Update(ctx, &existing); err != nil {
				return err
			}
			logger.V(1).Info("Result unchanged (historical)", "name", res.Name, "hash", newHash)
			finalResult = &existing
			return nil
		}

		// Content has changed - update the result
		// Capture the old hash before updating
		oldHash := existing.Status.ContentHash
		existing.Spec = res.Spec
		existing.Labels = res.Labels
		if err := c.Update(ctx, &existing); err != nil {
			return err
		}
		existing.Status.LifeCycle = string(UpdatedResult)
		existing.Status.ContentHash = newHash
		if err := c.Status().Update(ctx, &existing); err != nil {
			return err
		}
		logger.Info("Updated result", "name", res.Name, "oldHash", oldHash, "newHash", newHash)
		finalResult = &existing
		return nil
	})

	return finalResult, err
}
