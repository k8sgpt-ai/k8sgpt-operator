package k8sgpt

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type eligibleResource struct {
	ResultRef           corev1.ObjectReference
	ObjectRef           corev1.ObjectReference
	GVK                 string
	OriginConfiguration string
}
type calculateRemediationStep struct {
	logger logr.Logger
	next   K8sGPT
}

func (step *calculateRemediationStep) execute(instance *K8sGPTInstance) (ctrl.Result, error) {
	instance.logger.Info("starting RemediationStep")
	if !instance.K8sgptConfig.Spec.AI.AutoRemediation.Enabled {
		instance.logger.Info("calculateRemediationStep skipped because auto-remediation disabled")
		return ctrl.Result{}, nil
	}
	latestResultList, err := EmitIfNotHistorical(instance)
	if err != nil {
		return instance.R.FinishReconcile(err, false, instance.K8sgptConfig.Name)
	}

	if len(latestResultList.Items) == 0 {
		return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name)
	}
	for _, result := range latestResultList.Items {
		var res corev1alpha1.Result
		if err := instance.R.Get(instance.Ctx, client.ObjectKey{Namespace: result.Namespace, Name: result.Name}, &res); err != nil {
			return instance.R.FinishReconcile(err, false, result.Name)
		}
	}
	//
	eligibleResources := step.parseEligibleResources(instance, latestResultList)
	step.logger.Info("eligibleResources", "count", len(eligibleResources))

	// Create mutations for eligible resources
	for _, eligibleResource := range eligibleResources {
		mutation := corev1alpha1.Mutation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      eligibleResource.ResultRef.Name,
				Namespace: instance.K8sgptConfig.Namespace,
			},
			Spec: corev1alpha1.MutationSpec{
				ResourceRef:         eligibleResource.ObjectRef,
				ResourceGVK:         eligibleResource.GVK,
				ResultRef:           eligibleResource.ResultRef,
				OriginConfiguration: eligibleResource.OriginConfiguration,
				TargetConfiguration: "",
			},
			Status: corev1alpha1.MutationStatus{
				Phase: corev1alpha1.AutoRemediationPhaseNotStarted,
			},
		}
		// Check if the mutation exists, else create it
		mutationKey := client.ObjectKey{Namespace: instance.K8sgptConfig.Namespace, Name: eligibleResource.ResultRef.Name}
		var existingMutation corev1alpha1.Mutation
		if err := instance.R.Get(instance.Ctx, mutationKey, &existingMutation); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name)
			}
			if err := instance.R.Create(instance.Ctx, &mutation); err != nil {
				return instance.R.FinishReconcile(err, false, eligibleResource.ResultRef.Name)
			}
		}
	}
	step.logger.Info("ending calculateRemediationStep")
	return instance.R.FinishReconcile(nil, false, instance.K8sgptConfig.Name)
}

func (step *calculateRemediationStep) parseEligibleResources(instance *K8sGPTInstance, items *corev1alpha1.ResultList) []eligibleResource {
	// Currently this step is a watershed to ensure we are able to control directly what resources
	// are going to be mutated
	// In the future, we will have a more sophisticated way to determine which resources are eligible
	// for remediation
	var eligibleResources = []eligibleResource{}
	c := context.Background()
	for _, item := range items.Items {
		//demangle the name of the resource
		names := strings.Split(item.Spec.Name, "/")
		namespace := names[0]
		name := names[1]
		if len(names) != 2 {
			instance.logger.Error(fmt.Errorf("invalid resource name"), "unable to parse resource name", "ResourceRef", item.Name)
			continue
		}
		// create reference from the result
		resultRef, err := reference.GetReference(instance.R.Scheme, &item)
		if err != nil {
			k8sgptControllerLog.Error(err, "Unable to create reference for ResultRef", "Name", item.Name)
		}
		// Support Service/Ingress currently
		switch item.Spec.Kind {
		case "Service":
			var service corev1.Service
			if err := instance.R.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, &service); err != nil {
				instance.logger.Error(err, "unable to fetch Service", "Service", item.Name)
				continue
			}
			serviceRef, err := reference.GetReference(instance.R.Scheme, &service)
			if err != nil {
				step.logger.Error(err, "unable to create reference for Service", "Service", item.Name)
			}
			yamlData, err := yaml.Marshal(service)
			if err != nil {
				step.logger.Error(err, "unable to marshal Service to yaml", "Service", item.Name)
			}
			eligibleResources = append(eligibleResources, eligibleResource{ResultRef: *resultRef, ObjectRef: *serviceRef, OriginConfiguration: string(yamlData),
				GVK: serviceRef.GroupVersionKind().String()})

		case "Ingress":
			var ingress networkingv1.Ingress
			if err := instance.R.Get(c, client.ObjectKey{Namespace: namespace, Name: name}, &ingress); err != nil {
				instance.logger.Error(err, "unable to fetch Ingress", "Ingress", item.Name)
				continue
			}
			ingressRef, err := reference.GetReference(instance.R.Scheme, &ingress)
			if err != nil {
				step.logger.Error(err, "unable to create reference for Ingress", "Ingress", item.Name)
			}
			yamlData, err := yaml.Marshal(ingress)
			if err != nil {
				step.logger.Error(err, "unable to marshal Ingress to yaml", "Service", item.Name)
			}
			eligibleResources = append(eligibleResources, eligibleResource{ResultRef: *resultRef, ObjectRef: *ingressRef, OriginConfiguration: string(yamlData),
				GVK: ingressRef.GroupVersionKind().String()})
		}
	}
	return eligibleResources
}

func (step *calculateRemediationStep) setNext(next K8sGPT) {
	step.next = next
}
