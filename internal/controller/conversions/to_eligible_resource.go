package conversions

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// AutoRemediationResourceEnabled reports whether selector permits gvk. A bare
// Kind remains supported for backwards compatibility. Qualified selectors are
// exact group/kind matches: "apps/Deployment" or "v1/ConfigMap". The latter
// uses the Kubernetes apiVersion spelling for the core group.
func AutoRemediationResourceEnabled(config *corev1alpha1.K8sGPT, gvk schema.GroupVersionKind) bool {
	if config == nil || config.Spec.AI == nil {
		return false
	}
	for _, selector := range config.Spec.AI.AutoRemediation.Resources {
		selector = strings.TrimSpace(selector)
		if selector == gvk.Kind { // legacy exact Kind selector
			return true
		}
		if selector == gvk.Group+"/"+gvk.Kind || selector == gvk.GroupVersion().String()+"/"+gvk.Kind {
			return true
		}
	}
	return false
}

// autoRemediationResourceEnabled is retained for packages which used the old
// unexported helper. New callers should use AutoRemediationResourceEnabled.
func autoRemediationResourceEnabled(config *corev1alpha1.K8sGPT, kind string) bool {
	return AutoRemediationResourceEnabled(config, schema.GroupVersionKind{Kind: kind})
}

// resultTarget returns the analyzer-supplied exact target. For legacy Results,
// a fully qualified user selector can safely supply the missing GVK, but only
// when exactly one such selector matches the reported Kind.
func resultTarget(config *corev1alpha1.K8sGPT, item *corev1alpha1.Result) (schema.GroupVersionKind, string, string, error) {
	if target := item.Spec.TargetRef; target != nil {
		if target.APIVersion == "" || target.Kind == "" || target.Name == "" {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("targetRef must include apiVersion, kind, and name")
		}
		gv, err := schema.ParseGroupVersion(target.APIVersion)
		if err != nil {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("invalid targetRef apiVersion: %w", err)
		}
		return gv.WithKind(target.Kind), target.Namespace, target.Name, nil
	}

	parts := strings.Split(item.Spec.Name, "/")
	if len(parts) > 2 || len(parts) == 0 || parts[len(parts)-1] == "" {
		return schema.GroupVersionKind{}, "", "", fmt.Errorf("legacy resource name must be namespace/name or a cluster-scoped name")
	}
	namespace, name := "", parts[0]
	if len(parts) == 2 {
		if parts[0] == "" {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("legacy resource namespace must not be empty")
		}
		namespace, name = parts[0], parts[1]
	}
	if gvk, ok := configuredLegacyGVK(config, item.Spec.Kind); ok {
		return gvk, namespace, name, nil
	}
	switch item.Spec.Kind {
	case "Pod":
		if namespace == "" {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("legacy Pod result must include namespace/name")
		}
		return corev1.SchemeGroupVersion.WithKind("Pod"), namespace, name, nil
	case "Deployment":
		if namespace == "" {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("legacy Deployment result must include namespace/name")
		}
		return schema.GroupVersion{Group: "apps", Version: "v1"}.WithKind("Deployment"), namespace, name, nil
	default:
		return schema.GroupVersionKind{}, "", "", fmt.Errorf("legacy Result lacks targetRef or a unique fully qualified resource selector for kind %q", item.Spec.Kind)
	}
}

func configuredLegacyGVK(config *corev1alpha1.K8sGPT, kind string) (schema.GroupVersionKind, bool) {
	if config == nil || config.Spec.AI == nil {
		return schema.GroupVersionKind{}, false
	}
	var matches []schema.GroupVersionKind
	for _, selector := range config.Spec.AI.AutoRemediation.Resources {
		gvk, ok := fullyQualifiedSelector(strings.TrimSpace(selector))
		if ok && gvk.Kind == kind {
			matches = append(matches, gvk)
		}
	}
	if len(matches) != 1 {
		return schema.GroupVersionKind{}, false
	}
	return matches[0], true
}

func fullyQualifiedSelector(selector string) (schema.GroupVersionKind, bool) {
	parts := strings.Split(selector, "/")
	var groupVersion, kind string
	switch len(parts) {
	case 3:
		groupVersion, kind = parts[0]+"/"+parts[1], parts[2]
	case 2:
		// v1/ConfigMap is the fully-qualified spelling for the core group.
		if !strings.HasPrefix(parts[0], "v") || parts[0] == "v" {
			return schema.GroupVersionKind{}, false
		}
		groupVersion, kind = parts[0], parts[1]
	default:
		return schema.GroupVersionKind{}, false
	}
	if kind == "" {
		return schema.GroupVersionKind{}, false
	}
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return schema.GroupVersionKind{}, false
	}
	return gv.WithKind(kind), true
}

func eligibleResource(ctx context.Context, rc client.Client, resultRef *corev1.ObjectReference, gvk schema.GroupVersionKind, namespace, name string) (types.EligibleResource, error) {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	if err := rc.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, object); err != nil {
		return types.EligibleResource{}, err
	}
	// Managed fields are controller bookkeeping, not part of an LLM proposal.
	unstructured.RemoveNestedField(object.Object, "metadata", "managedFields")
	origin, err := yaml.Marshal(object.Object)
	if err != nil {
		return types.EligibleResource{}, err
	}
	return types.EligibleResource{
		ResultRef: *resultRef,
		ObjectRef: corev1.ObjectReference{
			APIVersion:      gvk.GroupVersion().String(),
			Kind:            gvk.Kind,
			Namespace:       object.GetNamespace(),
			Name:            object.GetName(),
			UID:             object.GetUID(),
			ResourceVersion: object.GetResourceVersion(),
		},
		GVK:                 gvk.GroupVersion().String(),
		OriginConfiguration: string(origin),
	}, nil
}

// resolveOwningWorkload follows controller ownership from a Pod finding to the
// nearest explicitly-enabled workload. Pods created by a Deployment are owned
// by a ReplicaSet, so the walk deliberately continues until it finds an opted
// in ancestor. This preserves the analyzer finding while changing the desired
// state rather than trying to mutate an immutable, controller-owned Pod.
func resolveOwningWorkload(ctx context.Context, rc client.Client, config *corev1alpha1.K8sGPT, gvk schema.GroupVersionKind, namespace, name string) (schema.GroupVersionKind, string, string, error) {
	if gvk != corev1.SchemeGroupVersion.WithKind("Pod") {
		return gvk, namespace, name, nil
	}
	currentGVK, currentNamespace, currentName := gvk, namespace, name
	for range 8 { // protects against malformed ownership cycles
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(currentGVK)
		if err := rc.Get(ctx, client.ObjectKey{Namespace: currentNamespace, Name: currentName}, current); err != nil {
			return schema.GroupVersionKind{}, "", "", err
		}
		owner := controllerOwner(current.GetOwnerReferences())
		if owner == nil {
			return gvk, namespace, name, nil
		}
		ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
		if err != nil {
			return schema.GroupVersionKind{}, "", "", fmt.Errorf("invalid controller owner apiVersion %q: %w", owner.APIVersion, err)
		}
		currentGVK, currentName = ownerGV.WithKind(owner.Kind), owner.Name
		if AutoRemediationResourceEnabled(config, currentGVK) {
			return currentGVK, currentNamespace, currentName, nil
		}
	}
	return schema.GroupVersionKind{}, "", "", fmt.Errorf("controller ownership depth exceeded while resolving Pod %s/%s", namespace, name)
}

func controllerOwner(owners []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range owners {
		if owners[i].Controller != nil && *owners[i].Controller {
			return &owners[i]
		}
	}
	return nil
}

// ResultsToEligibleResources converts only explicitly-enabled, exactly
// identified analyzer targets. It works for namespaced and cluster-scoped API
// resources because namespace is taken from targetRef rather than inferred.
func ResultsToEligibleResources(config *corev1alpha1.K8sGPT, rc client.Client, scheme *runtime.Scheme, logger logr.Logger, items *corev1alpha1.ResultList) []types.EligibleResource {
	eligible := []types.EligibleResource{}
	for i := range items.Items {
		item := &items.Items[i]
		gvk, namespace, name, err := resultTarget(config, item)
		if err != nil {
			logger.Error(err, "unable to determine exact remediation target", "ResultRef", item.Name)
			continue
		}
		originalGVK := gvk
		gvk, namespace, name, err = resolveOwningWorkload(context.Background(), rc, config, gvk, namespace, name)
		if err != nil {
			logger.Error(err, "unable to resolve owning remediation workload", "ResultRef", item.Name)
			continue
		}
		if !AutoRemediationResourceEnabled(config, gvk) {
			logger.Info("Resource not enabled for auto-remediation", "ResultRef", item.Name, "GVK", gvk.String())
			continue
		}
		resultRef, err := reference.GetReference(scheme, item)
		if err != nil {
			logger.Error(err, "unable to create Result reference", "Name", item.Name)
			continue
		}
		resource, err := eligibleResource(context.Background(), rc, resultRef, gvk, namespace, name)
		if err != nil {
			logger.Error(err, "unable to load remediation target", "ResultRef", item.Name, "GVK", gvk.String())
			continue
		}
		// An exact reference must still agree with the fetched object. UID and
		// resourceVersion are optional while producers migrate, but when supplied
		// they protect against recreated/stale objects.
		if target := item.Spec.TargetRef; target != nil && originalGVK == gvk && ((target.UID != "" && target.UID != resource.ObjectRef.UID) || (target.ResourceVersion != "" && target.ResourceVersion != resource.ObjectRef.ResourceVersion)) {
			logger.Info("targetRef does not match live object", "ResultRef", item.Name, "GVK", gvk.String())
			continue
		}
		eligible = append(eligible, resource)
	}
	return eligible
}
