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
	"net"
	"os"
	"time"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/common"
	"google.golang.org/grpc"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This is the client for communicating with the K8sGPT in cluster deployment
type Client struct {
	conn *grpc.ClientConn
}

func (c *Client) Close() error {
	return c.conn.Close()
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

func GenerateAddress(ctx context.Context, cli client.Client, k8sgptConfig *v1alpha1.K8sGPT) (string, error) {
	var address string
	if os.Getenv("LOCAL_MODE") != "" {
		address = "localhost:8080"
	} else {
		// Get service IP and port for k8sgpt-deployment
		svc := &corev1.Service{}
		err := cli.Get(ctx, client.ObjectKey{Namespace: k8sgptConfig.Namespace,
			Name: "k8sgpt"}, svc)
		if err != nil {
			return "", nil
		}
		address = fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
	}

	fmt.Printf("Creating new client for %s\n", address)
	// Test if the port is open
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return "", err
	}

	fmt.Printf("Connection established between %s and localhost with time out of %d seconds.\n", address, int64(1))
	fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())

	return address, nil
}

func (c *Client) ProcessAnalysis(deployment v1.Deployment, config *v1alpha1.K8sGPT) (*common.K8sGPTReponse, error) {

	client := rpc.NewServerServiceClient(c.conn)
	req := &schemav1.AnalyzeRequest{
		Explain:   config.Spec.AI.Enabled,
		Nocache:   config.Spec.NoCache,
		Backend:   config.Spec.AI.Backend,
		Filters:   config.Spec.Filters,
		Anonymize: config.Spec.AI.Anonymize,
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
