package types

import (
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	kclient "github.com/k8sgpt-ai/k8sgpt-operator/pkg/client"
)

type InterControllerSignal struct {
	K8sGPTClient *kclient.Client
	// We need a bit of context around the current backend for the query we will send to
	// the K8sGPT server. This will be used to determine the correct backend for the query.
	// This is a bit of a hack, but it will work for now.
	Backend string
	// We'll also transfer the K8sGPT instance, because this helps the mutation controller know it's configuration
	K8sGPT *corev1alpha1.K8sGPT
}
