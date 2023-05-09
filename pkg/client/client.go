/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"encoding/json"
	"fmt"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/common"
	"google.golang.org/grpc"
	v1 "k8s.io/api/apps/v1"
)

// This is the client for communicating with the K8sGPT in cluster deployment
type Client struct {
	conn *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	// Connect to the K8sGPT server and create a new client
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to dial K8sGPT server: %v", err)
	}

	client := &Client{conn: conn}

	return client, nil
}

func (c *Client) ProcessAnalysis(deployment v1.Deployment, config *v1alpha1.K8sGPT) (*common.K8sGPTReponse, error) {

	client := rpc.NewServerClient(c.conn)

	req := &schemav1.AnalyzeRequest{
		Explain: config.Spec.EnableAI,
		Nocache: config.Spec.NoCache,
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
