package shared

import (
	"testing"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServerQueryClient(t *testing.T) {
	// Test that initially the client is nil
	if client := GetServerQueryClient(); client != nil {
		t.Error("Expected GetServerQueryClient to return nil initially")
	}

	// Create a mock client (using nil as we can't easily create a real one in unit tests)
	var mockClient rpc.ServerQueryServiceClient
	SetServerQueryClient(&mockClient)

	// Test that we can retrieve the client
	if client := GetServerQueryClient(); client == nil {
		t.Error("Expected GetServerQueryClient to return non-nil after setting")
	}
}

func TestRemoteBackend(t *testing.T) {
	// Test that initially the backend is empty
	if backend := GetRemoteBackend(); backend != "" {
		t.Errorf("Expected GetRemoteBackend to return empty string initially, got %s", backend)
	}

	// Set a backend
	testBackend := "openai"
	SetRemoteBackend(testBackend)

	// Test that we can retrieve the backend
	if backend := GetRemoteBackend(); backend != testBackend {
		t.Errorf("Expected GetRemoteBackend to return %s, got %s", testBackend, backend)
	}
}

func TestK8sGPTConfig(t *testing.T) {
	// Test that initially the config is nil
	if config := GetK8sGPTConfig(); config != nil {
		t.Error("Expected GetK8sGPTConfig to return nil initially")
	}

	// Create a mock config
	testConfig := &corev1alpha1.K8sGPT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-k8sgpt",
			Namespace: "default",
		},
	}
	SetK8sGPTConfig(testConfig)

	// Test that we can retrieve the config
	if config := GetK8sGPTConfig(); config == nil {
		t.Error("Expected GetK8sGPTConfig to return non-nil after setting")
	} else if config.Name != "test-k8sgpt" {
		t.Errorf("Expected config name to be test-k8sgpt, got %s", config.Name)
	}
}
