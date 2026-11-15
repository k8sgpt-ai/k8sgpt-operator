package client

import (
	"context"
	"net"
	"testing"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/api/apps/v1"
)

type fakeAnalyzerServer struct {
	rpc.UnimplementedServerAnalyzerServiceServer
	filters []string
}

func (s *fakeAnalyzerServer) Analyze(_ context.Context, req *schemav1.AnalyzeRequest) (*schemav1.AnalyzeResponse, error) {
	s.filters = append([]string(nil), req.Filters...)
	return &schemav1.AnalyzeResponse{
		Results: []*schemav1.Result{
			{Kind: "Pod", Name: "shop/api-0"},
			{Kind: "Deployment", Name: "shop/api"},
			{Kind: "Service", Name: "shop/api"},
		},
	}, nil
}

func TestProcessAnalysisSendsEveryConfiguredFilter(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	server := grpc.NewServer()
	fake := &fakeAnalyzerServer{}
	rpc.RegisterServerAnalyzerServiceServer(server, fake)
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	anonymize := true
	config := &v1alpha1.K8sGPT{
		Spec: v1alpha1.K8sGPTSpec{
			Filters: []string{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "Ingress"},
			AI: &v1alpha1.AISpec{
				Backend:   "openai",
				Enabled:   true,
				Anonymize: &anonymize,
				Language:  "english",
			},
		},
	}

	client := &Client{Conn: conn}
	response, err := client.ProcessAnalysis(v1.Deployment{}, config, false)
	require.NoError(t, err)
	require.Len(t, response.Results, 3)
	require.Equal(t, config.Spec.Filters, fake.filters)
}
