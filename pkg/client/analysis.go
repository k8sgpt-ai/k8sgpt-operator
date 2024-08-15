package client

import (
	"context"
	"encoding/json"
	"fmt"

	rpc "buf.build/gen/go/ronaldpetty/ronk8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/ronaldpetty/ronk8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/common"
	v1 "k8s.io/api/apps/v1"
)

func (c *Client) ProcessAnalysis(deployment v1.Deployment, config *v1alpha1.K8sGPT, allowAIRequest bool) (*common.K8sGPTReponse, error) {

	client := rpc.NewServerAnalyzerServiceClient(c.conn)
	req := &schemav1.AnalyzeRequest{
		Explain:   config.Spec.AI.Enabled && allowAIRequest,
		Nocache:   config.Spec.NoCache,
		Backend:   config.Spec.AI.Backend,
		Namespace: config.Spec.TargetNamespace,
		Filters:   config.Spec.Filters,
		Anonymize: *config.Spec.AI.Anonymize,
		Language:  config.Spec.AI.Language,
	}

	res, err := client.Analyze(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Analyze RPC: %v", err)
	}

	var target []v1alpha1.ResultSpec

	jsonBytes, err := json.Marshal(res.Results)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, &target)
	if err != nil {
		return nil, err
	}

	response := &common.K8sGPTReponse{
		Status:   res.Status,
		Results:  target,
		Problems: int(res.Problems),
	}
	return response, nil
}
