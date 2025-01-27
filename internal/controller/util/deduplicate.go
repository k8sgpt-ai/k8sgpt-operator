package util

import (
	"github.com/go-logr/logr"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/types"
)

func Deduplicate(input []types.EligibleResource, log logr.Logger) []types.EligibleResource {
	var eligibleResource []types.EligibleResource
	var ownedResources = make(map[string]types.EligibleResource)
	for _, resource := range input {
		switch resource.ObjectRef.Kind {
		case "Pod":
			// Get the object and look for owner references
			obj, err := FromConfig(FromObjectConfig{
				Kind:      resource.ObjectRef.Kind,
				GvkStr:    resource.GVK,
				Config:    resource.OriginConfiguration,
				Name:      resource.ObjectRef.Name,
				Namespace: resource.ObjectRef.Namespace,
			})
			if err != nil {
				log.Error(err, "error deduplicating object")
				continue
			}
			ownerReferences := obj.GetOwnerReferences()
			if len(ownerReferences) == 0 {
				eligibleResource = append(eligibleResource, resource)
				continue
			}
			// Check if the owner is already in the map
			ownerKey := ownerReferences[0].Name + ownerReferences[0].Kind
			if _, ok := ownedResources[ownerKey]; !ok {
				ownedResources[ownerKey] = resource
			}
		}
	}
	for _, resource := range ownedResources {
		eligibleResource = append(eligibleResource, resource)
	}

	return eligibleResource
}
