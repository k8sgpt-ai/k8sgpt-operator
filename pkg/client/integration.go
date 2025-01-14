package client

import (
	"context"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

func (c *Client) AddIntegration(config *v1alpha1.K8sGPT) error {

	// Check if the integration is active already
	client := rpc.NewServerConfigServiceClient(c.Conn)
	req := &schemav1.ListIntegrationsRequest{}

	resp, err := client.ListIntegrations(context.Background(),
		req)
	if err != nil {
		return err
	}

	if resp.Trivy.Enabled == config.Spec.Integrations.Trivy.Enabled {
		fmt.Println("Skipping trivy installation, already enabled")
		return nil
	}
	// If the integration is inactive, make it active
	// Equally, if the flag has been deactivated we should also account for this
	// TODO: Currently this only support trivy
	configUpdatereq := &schemav1.AddConfigRequest{
		Integrations: &schemav1.Integrations{
			Trivy: &schemav1.Trivy{
				Enabled:     config.Spec.Integrations.Trivy.Enabled,
				SkipInstall: config.Spec.Integrations.Trivy.SkipInstall,
				Namespace:   config.Spec.Integrations.Trivy.Namespace,
			},
		},
	}
	_, err = client.AddConfig(context.Background(), configUpdatereq)
	if err != nil {
		return fmt.Errorf("failed to call AddConfig RPC: %v", err)
	}

	return nil
}
