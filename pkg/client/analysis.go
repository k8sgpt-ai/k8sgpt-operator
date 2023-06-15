package client

import (
	"context"
	"encoding/json"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/common"
)

func (c *Client) ProcessAnalysis(config *v1alpha1.K8sGPT) (*common.K8sGPTReponse, error) {

	client := rpc.NewServerServiceClient(c.conn)

	req := &schemav1.AnalyzeRequest{
		Explain: config.Spec.AI.Enable,
		Nocache: config.Spec.NoCache,
		Backend: string(config.Spec.AI.Backend),
		Filters: config.Spec.Filters,
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
