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
	"fmt"

	"google.golang.org/grpc"
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
