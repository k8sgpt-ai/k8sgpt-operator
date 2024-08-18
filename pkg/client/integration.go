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
	client := rpc.NewServerConfigServiceClient(c.conn)
	req := &schemav1.ListIntegrationsRequest{}

	resp, err := client.ListIntegrations(context.Background(),
		req)
	if err != nil {
		return err
	}

	skipTrivy := false
	skipKyverno := false

	if resp.Trivy.Enabled {
		if config.Spec.Integrations.Trivy != nil {
			if config.Spec.Integrations.Trivy.Enabled {
				fmt.Println("Skipping trivy installation, already enabled")
				skipTrivy = true
			}
		}
	} else {
		skipTrivy = true
	}

	if resp.Kyverno.Enabled {
		if config.Spec.Integrations.Kyverno != nil {
			if config.Spec.Integrations.Kyverno.Enabled {
				fmt.Println("Skipping kyverno installation, already enabled")
				skipKyverno = true
			}
		}
	} else {
		skipKyverno = true
	}

	if skipTrivy && skipKyverno {
		return nil
	}

	intergrate := &schemav1.Integrations{}

	var trivy *schemav1.Trivy

	if config.Spec.Integrations.Trivy != nil {
		trivy = &schemav1.Trivy{
			Enabled:     config.Spec.Integrations.Trivy.Enabled,
			SkipInstall: config.Spec.Integrations.Trivy.SkipInstall,
			Namespace:   config.Spec.Integrations.Trivy.Namespace,
		}
		intergrate.Trivy = trivy
	}

	var kyverno *schemav1.Kyverno

	if config.Spec.Integrations.Kyverno != nil {
		kyverno = &schemav1.Kyverno{
			Enabled:     config.Spec.Integrations.Kyverno.Enabled,
			SkipInstall: config.Spec.Integrations.Kyverno.SkipInstall,
			Namespace:   config.Spec.Integrations.Kyverno.Namespace,
		}
		intergrate.Kyverno = kyverno
	}

	// If the integration is inactive, make it active
	// Equally, if the flag has been deactivated we should also account for this
	// TODO: Currently this only support trivy
	configUpdatereq := &schemav1.AddConfigRequest{
		Integrations: intergrate,
	}
	_, err = client.AddConfig(context.Background(), configUpdatereq)
	if err != nil {
		return fmt.Errorf("failed to call AddConfig RPC: %v", err)
	}

	return nil
}
