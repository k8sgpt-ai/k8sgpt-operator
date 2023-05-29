package client

import (
	"context"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

func (c *Client) AddConfig(config *v1alpha1.K8sGPT) error {
	client := rpc.NewServerServiceClient(c.conn)

	req := &schemav1.AddConfigRequest{
		Cache: &schemav1.Cache{
			BucketName: config.Spec.RemoteCache.BucketName,
			Region:     config.Spec.RemoteCache.Region,
		},
	}

	_, err := client.AddConfig(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to call AddConfig RPC: %v", err)
	}

	return nil
}

func (c *Client) RemoveConfig(config *v1alpha1.K8sGPT) error {
	client := rpc.NewServerServiceClient(c.conn)

	req := &schemav1.RemoveConfigRequest{
		Cache: &schemav1.Cache{
			BucketName: config.Spec.RemoteCache.BucketName,
			Region:     config.Spec.RemoteCache.Region,
		},
	}

	_, err := client.RemoveConfig(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to call RemoveConfig RPC: %v", err)
	}

	return nil
}
