package mutation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/remediation/policy"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type imageSelection struct {
	Container string `json:"container"`
	Image     string `json:"image"`
}

// imageSelectionProposal turns a deliberately tiny LLM response into a full
// candidate object locally. The model never supplies metadata or any other
// mutable field; policy.Evaluate remains responsible for authorizing the one
// resulting semantic change.
func imageSelectionProposal(origin, response string) (string, error) {
	// Keep the established full-manifest contract for existing providers and
	// deterministic test doubles. It is still treated only as a candidate by
	// the policy gate; the compact form below is preferred for real providers.
	if object, err := policy.ParseYAMLProposal([]byte(response)); err == nil && object.GetAPIVersion() != "" && object.GetKind() != "" {
		return response, nil
	}
	selection := imageSelection{}
	response = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(response), "```"), "```json"))
	if err := json.Unmarshal([]byte(response), &selection); err != nil {
		return "", fmt.Errorf("response must be a JSON image selection: %w", err)
	}
	if selection.Container == "" || selection.Image == "" || strings.ContainsAny(selection.Image, "\n\r") {
		return "", fmt.Errorf("response must include non-empty container and image")
	}
	object, err := policy.ParseYAMLProposal([]byte(origin))
	if err != nil {
		return "", err
	}
	if !setContainerImage(object, selection.Container, selection.Image) {
		return "", fmt.Errorf("container %q is not present in the target manifest", selection.Container)
	}
	data, err := yaml.Marshal(object.Object)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func setContainerImage(object *unstructured.Unstructured, name, image string) bool {
	paths := [][]string{{"spec", "containers"}, {"spec", "initContainers"}, {"spec", "template", "spec", "containers"}, {"spec", "template", "spec", "initContainers"}, {"spec", "jobTemplate", "spec", "template", "spec", "containers"}, {"spec", "jobTemplate", "spec", "template", "spec", "initContainers"}}
	for _, path := range paths {
		containers, found, err := unstructured.NestedSlice(object.Object, path...)
		if err != nil || !found {
			continue
		}
		for i, value := range containers {
			container, ok := value.(map[string]any)
			if !ok || container["name"] != name {
				continue
			}
			container["image"] = image
			containers[i] = container
			return unstructured.SetNestedSlice(object.Object, containers, path...) == nil
		}
	}
	return false
}
