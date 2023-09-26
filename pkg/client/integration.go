package client

import (
	"context"
	"errors"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

func (c *Client) AddIntegration(config *v1alpha1.K8sGPT) error {

	// Check if the integration is active already

	client := rpc.NewServerServiceClient(c.conn)
	req := &schemav1.ListIntegrationsRequest{}

	resp, err := client.ListIntegrations(context.Background(),
		req)
	if err != nil {
		return err
	}

	if config.Spec.Integrations.Trivy == nil {
		return errors.New("integrations: only Trivy is currently supported")
	}

	// Check for active integrations
	for _, active := range resp.Integrations {
		switch active {
		case "trivy":
			// Integration is active, nothing to do
			return nil
		}
	}
	// If the integration is inactive, make it active
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
	if config.Spec.Integrations.Trivy.Enabled {
		fmt.Println("Activated integration")
	} else {
		fmt.Println("Deactivated integration")
	}
	return nil
}
