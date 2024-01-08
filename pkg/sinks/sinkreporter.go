package sinks

import (
	"net/http"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
)

type ISink interface {
	Configure(config v1alpha1.K8sGPT, c Client, secret string)
	Emit(results v1alpha1.ResultSpec) error
}

func NewSink(sinkType string) ISink {
	switch sinkType {
	case "slack":
		return &SlackSink{}
		//Introduce more Sink Providers
	default:
		return &SlackSink{}
	}
}

type Client struct {
	hclient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	client := &http.Client{
		Timeout: timeout,
	}
	return &Client{
		hclient: client,
	}
}
