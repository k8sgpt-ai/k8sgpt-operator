package shared

import (
	"sync"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
)

var (
	serverQueryClient     *rpc.ServerQueryServiceClient
	serverQueryClientLock sync.RWMutex
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
