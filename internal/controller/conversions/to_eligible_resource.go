package conversions

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
)

// TODO: how do we stop results colliding by modifying the same resources
// e.g. pod and deployment in the same replica set
var (
	SupportedResources = map[string]func(*[]types.EligibleResource,
		client.Client, *runtime.Scheme,
		logr.Logger, *corev1.ObjectReference, string, string) error{

		"Deployment": func(eligibleResources *[]types.EligibleResource, c client.Client, scheme *runtime.Scheme,
			logger logr.Logger, resultRef *corev1.ObjectReference, namespace string, name string) error {
			var deployment appsv1.Deployment
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &deployment); err != nil {
				return err
			}
			deploymentRef, err := reference.GetReference(scheme, &deployment)
			if err != nil {
				return err
			}
			deployment.ManagedFields = nil
			yamlData, err := yaml.Marshal(deployment)
			if err != nil {
				return err
			}
			*eligibleResources = append(*eligibleResources, types.EligibleResource{ResultRef: *resultRef,
				ObjectRef: *deploymentRef, OriginConfiguration: string(yamlData),
				GVK: deploymentRef.GroupVersionKind().String()})
			return nil
		},
		"Pod": func(eligibleResources *[]types.EligibleResource, c client.Client, scheme *runtime.Scheme,
			logger logr.Logger, resultRef *corev1.ObjectReference, namespace string, name string) error {
			var pod corev1.Pod
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &pod); err != nil {
				return err
			}
			podRef, err := reference.GetReference(scheme, &pod)
			if err != nil {
				logger.Error(err, "unable to create reference for Pod", "Pod", name)
			}
			// Strip out the stuff we don't need
			pod.ManagedFields = nil
			yamlData, err := yaml.Marshal(pod)
			if err != nil {
				logger.Error(err, "unable to marshal Pod to yaml", "Pod", name)
			}
			*eligibleResources = append(*eligibleResources, types.EligibleResource{ResultRef: *resultRef,
				ObjectRef: *podRef, OriginConfiguration: string(yamlData),
				GVK: podRef.GroupVersionKind().String()})
			return nil
		},
		"Ingress": func(eligibleResources *[]types.EligibleResource, c client.Client, scheme *runtime.Scheme,
			logger logr.Logger, resultRef *corev1.ObjectReference, namespace string, name string) error {

			var ingress networkingv1.Ingress
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &ingress); err != nil {
				return err
			}
			ingressRef, err := reference.GetReference(scheme, &ingress)
			ingress.ManagedFields = nil
			if err != nil {
				return err
			}
			yamlData, err := yaml.Marshal(ingress)
			if err != nil {
				return err
			}
			*eligibleResources = append(*eligibleResources, types.EligibleResource{ResultRef: *resultRef,
				ObjectRef: *ingressRef, OriginConfiguration: string(yamlData),
				GVK: ingressRef.GroupVersionKind().String()})

			return nil
		},
		"Service": func(eligibleResources *[]types.EligibleResource, c client.Client, scheme *runtime.Scheme,
			logger logr.Logger, resultRef *corev1.ObjectReference, namespace string, name string) error {
			var service corev1.Service
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &service); err != nil {
				return err
			}
			// Strip out the stuff we don't need
			service.ManagedFields = nil
			serviceRef, err := reference.GetReference(scheme, &service)
			if err != nil {
				return err
			}
			yamlData, err := yaml.Marshal(service)
			if err != nil {
				return err
			}
			*eligibleResources = append(*eligibleResources, types.EligibleResource{ResultRef: *resultRef, ObjectRef: *serviceRef, OriginConfiguration: string(yamlData),
				GVK: serviceRef.GroupVersionKind().String()})

			return nil
		},
	}
)

func ResultsToEligibleResources(config *corev1alpha1.K8sGPT,
	rc client.Client, scheme *runtime.Scheme,
	logger logr.Logger, items *corev1alpha1.ResultList) []types.EligibleResource {
	// Currently this step is a watershed to ensure we are able to control directly what resources
	// are going to be mutated
	// In the future, we will have a more sophisticated way to determine which resources are eligible
	// for remediation
	var eligibleResources = []types.EligibleResource{}

	for _, item := range items.Items {
		//demangle the name of the resource
		names := strings.Split(item.Spec.Name, "/")
		namespace := names[0]
		name := names[1]
		if len(names) != 2 {
			logger.Error(fmt.Errorf("invalid resource name"), "unable to parse resource name", "ResourceRef", item.Name)
			continue
		}
		// create reference from the result
		resultRef, err := reference.GetReference(scheme, &item)
		if err != nil {
			logger.Error(err, "Unable to create reference for ResultRef", "Name", item.Name)
		}
		// check if it's in the SupportedResources map
		if supportedResource, ok := SupportedResources[item.Spec.Kind]; ok {
			err := supportedResource(&eligibleResources, rc, scheme, logger, resultRef, namespace, name)
			if err != nil {
				logger.Error(err, "unable to create eligible resource", "ResourceRef", item.Name)
			}
		} else {
			logger.Info("Resource not supported", "ResourceRef", item.Name, "Kind", item.Spec.Kind)
		}
	}
	return eligibleResources
}
