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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/pkg/common"
	v1 "k8s.io/api/apps/v1"
)

// This is the client for communicating with the K8sGPT in cluster deployment
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) ProcessAnalysis(deployment v1.Deployment, config *v1alpha1.K8sGPT) (*common.K8sGPTReponse, error) {

	// Construct the request
	// <service-name>.<namespace>:8080/analyze
	var url string
	if os.Getenv("LOCAL_MODE") != "" {
		url = "http://localhost:8080/analyze"
	} else {
		url = fmt.Sprintf("http://%s.%s:8080/analyze", "k8sgpt", deployment.Namespace)
	}

	if config.Spec.EnableAI {
		url = url + "?explain=true"
	}

	if config.Spec.NoCache {
		url = url + "?nocache=true"
	}

	r, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var target common.K8sGPTReponse

	err = json.NewDecoder(r.Body).Decode(&target)
	if err != nil {
		return nil, err
	}
	return &target, nil
}
