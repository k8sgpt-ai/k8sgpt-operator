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
	"testing"

	"google.golang.org/grpc"
)

func TestGetServerQueryServiceClient(t *testing.T) {
	// Create a client with a mock connection
	// Note: We can't fully test this without a real gRPC server, but we can verify it doesn't panic
	conn, err := grpc.Dial("localhost:9999", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create mock connection: %v", err)
	}
	defer conn.Close()

	client := &Client{Conn: conn}

	// This should not panic
	queryClient := client.GetServerQueryServiceClient()
	if queryClient == nil {
		t.Error("Expected GetServerQueryServiceClient to return non-nil client")
	}
}
