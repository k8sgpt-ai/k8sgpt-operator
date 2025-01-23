package conversions

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type EligibleResource struct {
	ResultRef           corev1.ObjectReference
	ObjectRef           corev1.ObjectReference
	GVK                 string
	OriginConfiguration string
}

func ResultsToEligibleResources(rc client.Client, scheme *runtime.Scheme,
	logger logr.Logger, items *corev1alpha1.ResultList) []EligibleResource {
	// Currently this step is a watershed to ensure we are able to control directly what resources
	// are going to be mutated
	// In the future, we will have a more sophisticated way to determine which resources are eligible
	// for remediation
	var eligibleResources = []EligibleResource{}
	c := context.Background()
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
		// Support Service/Ingress currently
		switch item.Spec.Kind {
		case "Service":
			var service corev1.Service
			if err := rc.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, &service); err != nil {
				logger.Error(err, "unable to fetch Service", "Service", item.Name)
				continue
			}
			serviceRef, err := reference.GetReference(scheme, &service)
			if err != nil {
				logger.Error(err, "unable to create reference for Service", "Service", item.Name)
			}
			yamlData, err := yaml.Marshal(service)
			if err != nil {
				logger.Error(err, "unable to marshal Service to yaml", "Service", item.Name)
			}
			eligibleResources = append(eligibleResources, EligibleResource{ResultRef: *resultRef, ObjectRef: *serviceRef, OriginConfiguration: string(yamlData),
				GVK: serviceRef.GroupVersionKind().String()})

		case "Ingress":
			var ingress networkingv1.Ingress
			if err := rc.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, &ingress); err != nil {
				logger.Error(err, "unable to fetch Ingress", "Ingress", item.Name)
				continue
			}
			ingressRef, err := reference.GetReference(scheme, &ingress)
			if err != nil {
				logger.Error(err, "unable to create reference for Ingress", "Ingress", item.Name)
			}
			yamlData, err := yaml.Marshal(ingress)
			if err != nil {
				logger.Error(err, "unable to marshal Ingress to yaml", "Service", item.Name)
			}
			eligibleResources = append(eligibleResources, EligibleResource{ResultRef: *resultRef,
				ObjectRef: *ingressRef, OriginConfiguration: string(yamlData),
				GVK: ingressRef.GroupVersionKind().String()})

		case "Pod":
			var pod corev1.Pod
			if err := rc.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, &pod); err != nil {
				logger.Error(err, "unable to fetch Pod", "Pod", item.Name)
				continue
			}
			podRef, err := reference.GetReference(scheme, &pod)
			if err != nil {
				logger.Error(err, "unable to create reference for Pod", "Pod", item.Name)
			}
			yamlData, err := yaml.Marshal(pod)
			if err != nil {
				logger.Error(err, "unable to marshal Pod to yaml", "Pod", item.Name)
			}
			eligibleResources = append(eligibleResources, EligibleResource{ResultRef: *resultRef,
				ObjectRef: *podRef, OriginConfiguration: string(yamlData),
				GVK: podRef.GroupVersionKind().String()})
		}

	}
	return eligibleResources
}
