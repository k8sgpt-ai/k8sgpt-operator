package shared

import (
	"sync"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

var (
	serverQueryClient     *rpc.ServerQueryServiceClient
	serverQueryClientLock sync.RWMutex
	remoteBackend         string
	remoteBackendLock     sync.RWMutex
	k8sgptConfig          *corev1alpha1.K8sGPT
	k8sgptConfigLock      sync.RWMutex
)

// SetServerQueryClient sets the shared ServerQueryClient.
func SetServerQueryClient(client *rpc.ServerQueryServiceClient) {
	serverQueryClientLock.Lock()
	defer serverQueryClientLock.Unlock()
	serverQueryClient = client
}

// GetServerQueryClient gets the shared ServerQueryClient.
func GetServerQueryClient() *rpc.ServerQueryServiceClient {
	serverQueryClientLock.RLock()
	defer serverQueryClientLock.RUnlock()
	return serverQueryClient
}

// SetRemoteBackend sets the shared backend.
func SetRemoteBackend(backend string) {
	remoteBackendLock.Lock()
	defer remoteBackendLock.Unlock()
	remoteBackend = backend
}

// GetRemoteBackend gets the shared backend.
func GetRemoteBackend() string {
	remoteBackendLock.RLock()
	defer remoteBackendLock.RUnlock()
	return remoteBackend
}

// SetK8sGPTConfig sets the shared K8sGPT config.
func SetK8sGPTConfig(config *corev1alpha1.K8sGPT) {
	k8sgptConfigLock.Lock()
	defer k8sgptConfigLock.Unlock()
	k8sgptConfig = config
}

// GetK8sGPTConfig gets the shared K8sGPT config.
func GetK8sGPTConfig() *corev1alpha1.K8sGPT {
	k8sgptConfigLock.RLock()
	defer k8sgptConfigLock.RUnlock()
	return k8sgptConfig
}
