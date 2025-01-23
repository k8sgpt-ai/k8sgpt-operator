package conversions

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type FromObjectConfig struct {
	Config    string
	GvkStr    string
	Kind      string
	Name      string
	Namespace string
}

func FromConfig(objConfig FromObjectConfig) (client.Object, error) {
	gv, err := schema.ParseGroupVersion(objConfig.GvkStr)
	if err != nil {
		return nil, err
	}
	gvk := gv.WithKind(objConfig.Kind)
	// 2. Create an unstructured object
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	// 3. Decode the targetConfiguration into the unstructured object
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(objConfig.Config), 1000)
	if err := decoder.Decode(obj); err != nil {
		return nil, err
	}
	// 4. Set the object's name and namespace (important for updates!)
	obj.SetName(objConfig.Name)
	obj.SetNamespace(objConfig.Namespace)

	return obj, nil
}
