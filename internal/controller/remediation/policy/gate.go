// Package policy provides deterministic, local safety checks for automatic
// remediation proposals. It deliberately does not decide whether a diagnosis
// is correct: it only constrains what an accepted proposal can change.
package policy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

const defaultMaxChangedPaths = 1

// AllowedPathsAnnotation lets an owner opt an otherwise-unregistered standard
// API resource into remediation for exact JSON Pointer paths. It is deliberately
// an object-local, explicit allowlist; a K8sGPT resource selector never grants
// arbitrary writes. Example: /data/log-level,/spec/ports/0/targetPort.
const AllowedPathsAnnotation = "core.k8sgpt.ai/auto-remediation-allowed-paths"

// Input contains the snapshot used to create the proposal, the object fetched
// immediately before execution, and the LLM's proposed replacement. A caller
// must fetch Live directly from the API server; this gate does not perform I/O.
type Input struct {
	Original *unstructured.Unstructured
	Live     *unstructured.Unstructured
	Proposed *unstructured.Unstructured

	// MaxChangedPaths defaults to one. A value greater than zero permits a
	// narrowly-bounded multi-container image correction.
	MaxChangedPaths int
}

// Change is one semantic JSON-pointer change calculated by the operator, not
// trusted from the proposal.
type Change struct {
	Path string
	Old  any
	New  any
}

// PatchOperation is an RFC 6902 operation. Current rules only allow replace
// operations, but the type makes the output usable by a later patch executor.
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// Decision records all reasons for rejection so callers can persist useful
// audit information on a Mutation without exposing a permissive boolean only.
type Decision struct {
	Allowed bool
	Reasons []string
	Changes []Change
	Patch   []PatchOperation
}

// ResourcePolicy is the deny-by-default policy for one exact Kubernetes GVK.
// A policy must be registered before the gate can admit a change for that
// resource. Keeping the key as a GVK (rather than a Kind) avoids accidentally
// applying rules written for one API version or API group to another resource
// that happens to have the same kind name.
type ResourcePolicy struct {
	GVK schema.GroupVersionKind

	// AllowedPath receives an operator-calculated JSON Pointer. It must return
	// true only for paths the policy explicitly supports.
	AllowedPath func(path string) bool
}

// registeredPolicies is intentionally a closed registry. Adding support for a
// resource is a code and review decision, rather than an effect of a user
// enabling a broad resource selector.
var registeredPolicies = map[schema.GroupVersionKind]ResourcePolicy{
	{Group: "", Version: "v1", Kind: "Pod"}: {
		GVK:         schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
		AllowedPath: podImagePath,
	},
	{Group: "apps", Version: "v1", Kind: "Deployment"}: {
		GVK:         schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
		AllowedPath: podTemplateImagePath,
	},
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}: {
		GVK:         schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		AllowedPath: podTemplateImagePath,
	},
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: {
		GVK:         schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		AllowedPath: podTemplateImagePath,
	},
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}: {
		GVK:         schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
		AllowedPath: podTemplateImagePath,
	},
	{Group: "batch", Version: "v1", Kind: "Job"}: {
		GVK:         schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"},
		AllowedPath: podTemplateImagePath,
	},
	{Group: "batch", Version: "v1", Kind: "CronJob"}: {
		GVK:         schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"},
		AllowedPath: cronJobImagePath,
	},
}

// RegisteredGVKs returns a stable copy of the currently registered policies.
// It is primarily useful to expose the actual auto-remediation surface in
// status and documentation without treating configured resource kinds as a
// promise that arbitrary mutations are allowed.
func RegisteredGVKs() []schema.GroupVersionKind {
	gvks := make([]schema.GroupVersionKind, 0, len(registeredPolicies))
	for gvk := range registeredPolicies {
		gvks = append(gvks, gvk)
	}
	sort.Slice(gvks, func(i, j int) bool { return gvks[i].String() < gvks[j].String() })
	return gvks
}

// Evaluate validates immutable object identity, rejects stale snapshots,
// calculates a semantic diff, and applies the closed GVK policy registry.
// No model-provided confidence score can override this decision.
func Evaluate(in Input) Decision {
	result := Decision{}
	if in.Original == nil || in.Live == nil || in.Proposed == nil {
		return rejected("original, live, and proposed objects are required")
	}

	if reason := liveMatchesOriginal(in.Original, in.Live); reason != "" {
		return rejected(reason)
	}
	if reason := proposedMatchesLive(in.Proposed, in.Live); reason != "" {
		return rejected(reason)
	}

	gvk := in.Live.GroupVersionKind()
	resourcePolicy, ok := registeredPolicies[gvk]
	if !ok {
		paths := annotationAllowedPaths(in.Live)
		if len(paths) == 0 {
			return rejected(fmt.Sprintf("no remediation policy registered for GVK %s; set %s to explicit JSON Pointer paths", gvk.String(), AllowedPathsAnnotation))
		}
		resourcePolicy = ResourcePolicy{GVK: gvk, AllowedPath: func(path string) bool { return paths[path] }}
	}
	if gvk.Group == "" && gvk.Version == "v1" && gvk.Kind == "Pod" && len(in.Live.GetOwnerReferences()) > 0 {
		return rejected("owned Pods are not remediated directly; target the owning workload instead")
	}

	live := normalized(in.Live.Object)
	proposed := normalized(in.Proposed.Object)
	changes := diff("", live, proposed)
	result.Changes = changes
	if len(changes) == 0 {
		return rejectedWithChanges(changes, "proposal makes no semantic change")
	}
	limit := in.MaxChangedPaths
	if limit == 0 {
		limit = defaultMaxChangedPaths
	}
	if limit < 0 {
		return rejectedWithChanges(changes, "max changed paths must not be negative")
	}
	if len(changes) > limit {
		return rejectedWithChanges(changes, fmt.Sprintf("proposal changes %d paths; maximum is %d", len(changes), limit))
	}

	for _, change := range changes {
		if isHighRisk(change.Path) {
			return rejectedWithChanges(changes, fmt.Sprintf("high-risk path is forbidden: %s", change.Path))
		}
		if !resourcePolicy.AllowedPath(change.Path) {
			return rejectedWithChanges(changes, fmt.Sprintf("path is not allowlisted: %s", change.Path))
		}
		// The initial policy intentionally allows replacement only. This rules
		// out creating/removing containers or changing list structure.
		if change.Old == nil || change.New == nil {
			return rejectedWithChanges(changes, fmt.Sprintf("add/remove operation is forbidden: %s", change.Path))
		}
		result.Patch = append(result.Patch, PatchOperation{Op: "replace", Path: change.Path, Value: change.New})
	}

	result.Allowed = true
	return result
}

func annotationAllowedPaths(object *unstructured.Unstructured) map[string]bool {
	value := object.GetAnnotations()[AllowedPathsAnnotation]
	paths := map[string]bool{}
	for _, path := range strings.Split(value, ",") {
		path = strings.TrimSpace(path)
		if strings.HasPrefix(path, "/") {
			paths[path] = true
		}
	}
	return paths
}

// ParseYAMLProposal converts one Kubernetes YAML document to an unstructured
// object. Parsing is intentionally separate from Evaluate so integrations can
// persist parse failures distinctly from policy rejections.
func ParseYAMLProposal(data []byte) (*unstructured.Unstructured, error) {
	jsonData, err := kyaml.ToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("convert proposed YAML to JSON: %w", err)
	}
	object := map[string]any{}
	if err := json.Unmarshal(jsonData, &object); err != nil {
		return nil, fmt.Errorf("decode proposed object: %w", err)
	}
	if len(object) == 0 {
		return nil, fmt.Errorf("proposed object is empty")
	}
	return &unstructured.Unstructured{Object: object}, nil
}

func rejected(reason string) Decision { return Decision{Reasons: []string{reason}} }
func rejectedWithChanges(changes []Change, reason string) Decision {
	return Decision{Reasons: []string{reason}, Changes: changes}
}

func liveMatchesOriginal(original, live *unstructured.Unstructured) string {
	if original.GetUID() == "" || live.GetUID() == "" || original.GetUID() != live.GetUID() {
		return "live object UID does not match the proposal snapshot"
	}
	if original.GetResourceVersion() == "" || live.GetResourceVersion() == "" || original.GetResourceVersion() != live.GetResourceVersion() {
		return "live object resourceVersion does not match the proposal snapshot"
	}
	if !sameIdentity(original, live) {
		return "live object identity does not match the proposal snapshot"
	}
	return ""
}

func proposedMatchesLive(proposed, live *unstructured.Unstructured) string {
	if !sameIdentity(proposed, live) {
		return "proposed object must exactly match target apiVersion, kind, namespace, and name"
	}
	if uid := proposed.GetUID(); uid != "" && uid != live.GetUID() {
		return "proposed object UID does not match target"
	}
	if resourceVersion := proposed.GetResourceVersion(); resourceVersion != "" && resourceVersion != live.GetResourceVersion() {
		return "proposed object resourceVersion does not match target"
	}
	if proposed.GetGenerateName() != live.GetGenerateName() {
		return "proposed object generateName does not match target"
	}
	return ""
}

func sameIdentity(a, b *unstructured.Unstructured) bool {
	return a.GetAPIVersion() == b.GetAPIVersion() && a.GetKind() == b.GetKind() &&
		a.GetNamespace() == b.GetNamespace() && a.GetName() == b.GetName()
}

func podImagePath(path string) bool {
	return imagePath("/spec/containers/", path) || imagePath("/spec/initContainers/", path)
}

func podTemplateImagePath(path string) bool {
	return imagePath("/spec/template/spec/containers/", path) || imagePath("/spec/template/spec/initContainers/", path)
}

func cronJobImagePath(path string) bool {
	return imagePath("/spec/jobTemplate/spec/template/spec/containers/", path) || imagePath("/spec/jobTemplate/spec/template/spec/initContainers/", path)
}

func imagePath(prefix, path string) bool {
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, "/image") {
		return false
	}
	// exactly one (numeric) list index between the prefix and image field
	middle := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/image")
	if middle == "" || strings.Contains(middle, "/") {
		return false
	}
	for _, c := range middle {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isHighRisk(path string) bool {
	for _, prefix := range []string{
		"/metadata/ownerReferences", "/metadata/finalizers", "/metadata/managedFields",
		"/metadata/annotations", "/metadata/labels", "/status", "/spec/serviceAccountName",
		"/spec/hostNetwork", "/spec/hostPID", "/spec/hostIPC", "/spec/securityContext",
		"/spec/containers", "/spec/initContainers", "/spec/template/spec/serviceAccountName",
		"/spec/template/spec/hostNetwork", "/spec/template/spec/hostPID", "/spec/template/spec/hostIPC",
		"/spec/template/spec/securityContext", "/spec/template/spec/containers", "/spec/template/spec/initContainers",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			// Image paths are checked by the allowlist, but are not high risk.
			if strings.HasSuffix(path, "/image") && (strings.Contains(prefix, "/containers") || strings.Contains(prefix, "/initContainers")) {
				continue
			}
			return true
		}
	}
	return false
}

func normalized(object map[string]any) any {
	copy := deepCopy(object).(map[string]any)
	delete(copy, "status")
	if metadata, ok := copy["metadata"].(map[string]any); ok {
		for _, key := range []string{"uid", "resourceVersion", "generation", "creationTimestamp", "deletionTimestamp", "managedFields", "selfLink"} {
			delete(metadata, key)
		}
	}
	return copy
}

func deepCopy(value any) any {
	data, _ := json.Marshal(value)
	var copied any
	_ = json.Unmarshal(data, &copied)
	return copied
}

func diff(path string, old, new any) []Change {
	if oldMap, ok := old.(map[string]any); ok {
		if newMap, same := new.(map[string]any); same {
			keys := make(map[string]struct{}, len(oldMap)+len(newMap))
			for key := range oldMap {
				keys[key] = struct{}{}
			}
			for key := range newMap {
				keys[key] = struct{}{}
			}
			ordered := make([]string, 0, len(keys))
			for key := range keys {
				ordered = append(ordered, key)
			}
			sort.Strings(ordered)
			var result []Change
			for _, key := range ordered {
				result = append(result, diff(path+"/"+escape(key), oldMap[key], newMap[key])...)
			}
			return result
		}
	}
	if oldList, ok := old.([]any); ok {
		if newList, same := new.([]any); same {
			max := len(oldList)
			if len(newList) > max {
				max = len(newList)
			}
			var result []Change
			for i := 0; i < max; i++ {
				var before, after any
				if i < len(oldList) {
					before = oldList[i]
				}
				if i < len(newList) {
					after = newList[i]
				}
				result = append(result, diff(fmt.Sprintf("%s/%d", path, i), before, after)...)
			}
			return result
		}
	}
	if equal(old, new) {
		return nil
	}
	return []Change{{Path: path, Old: old, New: new}}
}

func equal(a, b any) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func escape(segment string) string {
	return strings.ReplaceAll(strings.ReplaceAll(segment, "~", "~0"), "/", "~1")
}
