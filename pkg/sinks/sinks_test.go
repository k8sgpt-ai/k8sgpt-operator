package sinks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func Test_NewSink(t *testing.T) {
	tests := []struct {
		name     string
		sinkType string
		want     ISink
	}{
		{
			name:     "slack sink",
			sinkType: "slack",
			want:     &SlackSink{},
		},
		{
			name:     "mattermost sink",
			sinkType: "mattermost",
			want:     &MattermostSink{},
		},
		{
			name:     "default sink",
			sinkType: "unknown",
			want:     &SlackSink{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSink(tt.sinkType)
			assert.NotNil(t, got, "NewSink() should not return nil")
		})
	}
}

func Test_NewClient(t *testing.T) {
	timeout := 2 * time.Second
	client := NewClient(timeout)

	assert.Equal(t, timeout, client.hclient.Timeout, "Expected timeout to be %v, got %v", timeout, client.hclient.Timeout)
}

func Test_MattermostSinkConfigure(t *testing.T) {
	sink := &MattermostSink{}
	client := NewClient(2 * time.Second)
	config := v1alpha1.K8sGPT{
		Spec: v1alpha1.K8sGPTSpec{
			Sink: &v1alpha1.WebhookRef{
				Endpoint: "http://example.com",
				Channel:  "channel",
				UserName: "username",
				IconURL:  "http://icon.url",
			},
		},
	}

	sink.Configure(config, *client, "")

	assert.Equal(t, "http://example.com", sink.Endpoint)
	assert.Equal(t, "channel", sink.Channel)
	assert.Equal(t, "username", sink.UserName)
	assert.Equal(t, "http://icon.url", sink.IconURL)
	assert.Equal(t, client, &sink.Client)
}

func Test_MattermostSinkEmit(t *testing.T) {
    tests := []struct {
        name         string
        results      v1alpha1.ResultSpec
        responseCode int
        expectError  bool
    }{
        {
            name: "Error details with successful response",
            results: v1alpha1.ResultSpec{
                Error: []v1alpha1.Failure{
                    {Text: "First error"},
                    {Text: "Second error"},
                },
            },
            responseCode: http.StatusOK,
            expectError:  false,
        },
        {
            name: "Non-empty details with failed response",
            results: v1alpha1.ResultSpec{
                Details: "Some details",
            },
            responseCode: http.StatusInternalServerError,
            expectError:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sink := &MattermostSink{
                Endpoint: "",
                Client:   *NewClient(2 * time.Second),
            }
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.responseCode)
            }))
            defer server.Close()

            sink.Endpoint = server.URL

            err := sink.Emit(tt.results)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func Test_SlackSinkConfigure(t *testing.T) {
    sink := &SlackSink{}
    client := NewClient(2 * time.Second)
    config := v1alpha1.K8sGPT{
        Spec: v1alpha1.K8sGPTSpec{
            Sink: &v1alpha1.WebhookRef{
                Endpoint: "http://example.com",
            },
        },
    }

    sink.Configure(config, *client, "")

    assert.Equal(t, "http://example.com", sink.Endpoint)
    assert.Equal(t, client, &sink.Client)
}

func Test_SlackSinkEmit(t *testing.T) {
    tests := []struct {
        name         string
        results      v1alpha1.ResultSpec
        responseCode int
        expectError  bool
    }{
        {
            name: "Successful response",
            results: v1alpha1.ResultSpec{
                Kind: "kind",
                Name: "name",
            },
            responseCode: http.StatusOK,
            expectError:  false,
        },
        {
            name: "Failed response",
            results: v1alpha1.ResultSpec{
                Kind: "kind",
                Name: "name",
            },
            responseCode: http.StatusInternalServerError,
            expectError:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sink := &SlackSink{
                Endpoint: "",
                Client:   *NewClient(2 * time.Second),
            }
            server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.responseCode)
            }))
            defer server.Close()

            sink.Endpoint = server.URL

            err := sink.Emit(tt.results)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}