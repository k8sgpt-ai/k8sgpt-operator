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
	"fmt"
	"net"
	"os"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"google.golang.org/grpc"
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
		return nil, fmt.Errorf("failed to create context: %v", err)
	}
	client := &Client{conn: conn}

	return client, nil
}

func GenerateAddress(ctx context.Context, cli client.Client, k8sgptConfig *v1alpha1.K8sGPT) (string, error) {
	var address string
	var ip net.IP

	if os.Getenv("LOCAL_MODE") != "" {
		address = "localhost:8080"
	} else {
		// Get service IP and port for k8sgpt-deployment
		svc := &corev1.Service{}
		err := cli.Get(ctx, client.ObjectKey{Namespace: k8sgptConfig.Namespace,
			Name: k8sgptConfig.Name}, svc)
		if err != nil {
			return "", nil
		}
		ip = net.ParseIP(svc.Spec.ClusterIP)
		if ip.To4() != nil {
			address = fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
		} else {
			address = fmt.Sprintf("[%s]:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
		}
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
