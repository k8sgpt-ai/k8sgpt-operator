package integrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	backstageLabelKey = "backstage.io/kubernetes-id"
)

type Integrations struct {
	restMapper meta.RESTMapper
	client     client.Client
	ctx        context.Context
}

func NewIntegrations(client client.Client, ctx context.Context) (*Integrations, error) {
	restMapper, err := cmdutil.NewFactory(genericclioptions.NewConfigFlags(true)).ToRESTMapper()
	if err != nil {
		return &Integrations{}, err
	}
	return &Integrations{
		restMapper: restMapper,
		client:     client,
		ctx:        ctx,
	}, nil
}

func (i *Integrations) BackstageLabel(result v1alpha1.ResultSpec) (map[string]string, error) {
	namespace, resourceName, _ := strings.Cut(result.Name, "/")
	gvr, err := i.restMapper.ResourceFor(schema.GroupVersionResource{
		Resource: result.Kind,
	})
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{
		Group:   gvr.Group,
		Kind:    result.Kind,
		Version: gvr.Version,
	}
	obj.SetGroupVersionKind(gvk)
	// Retrieve the resource using the client
	err = i.client.Get(i.ctx, client.ObjectKey{Name: resourceName, Namespace: namespace}, obj)
	if err != nil {
		return nil, err
	}
	labels := obj.GetLabels()
	value, exists := labels[backstageLabelKey]
	if !exists {
		fmt.Printf("Label key '%s' does not exist in %s resource: %s\n", backstageLabelKey, result.Kind, resourceName)
	}
	// Assign the same label key/value to result CR
	return map[string]string{backstageLabelKey: value}, nil
}
