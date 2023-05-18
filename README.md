<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/banner-white.png" width="600px;">
  <img alt="Text changing depending on mode. Light: 'So light!' Dark: 'So dark!'" src="./images/banner-black.png" width="600px;">
</picture>
<br/>

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/k8sgpt)](https://artifacthub.io/packages/search?repo=k8sgpt)

This Operator is designed to enable [K8sGPT](https://github.com/k8sgpt-ai/k8sgpt/) within a Kubernetes cluster.
It will allow you to create a custom resource that defines the behaviour and scope of a managed K8sGPT workload. Analysis and outputs will also be configurable to enable integration into existing workflows.

<img src="images/demo2.gif" width="600px;"/>

## Installation

```
helm repo add k8sgpt https://charts.k8sgpt.ai/
helm install release k8sgpt/k8sgpt-operator -n k8sgpt-operator-system --create-namespace
```

## Run the example

1. Install the operator from the [Installation](#installation) section.

2. Create secret:
```sh 
kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key=$OPENAI_TOKEN -n k8sgpt-
operator-system
```

3. Apply the K8sGPT configuration object:
```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: k8sgpt-operator-system
spec:
  model: gpt-3.5-turbo
  backend: openai
  noCache: false
  version: v0.3.0
  enableAI: true
  # filters:
  #   - Ingress
  secret:
    name: k8sgpt-sample-secret
    key: openai-api-key
EOF
```

4. Once the custom resource has been applied the K8sGPT-deployment will be installed and
you will be able to see the Results objects of the analysis after some minutes (if there are any issues in your cluster):

```bash
❯ kubectl get results -o json | jq .
{
  "apiVersion": "v1",
  "items": [
    {
      "apiVersion": "core.k8sgpt.ai/v1alpha1",
      "kind": "Result",
      "spec": {
        "details": "The error message means that the service in Kubernetes doesn't have any associated endpoints, which should have been labeled with \"control-plane=controller-manager\". \n\nTo solve this issue, you need to add the \"control-plane=controller-manager\" label to the endpoint that matches the service. Once the endpoint is labeled correctly, Kubernetes can associate it with the service, and the error should be resolved.",
```

## Other AI Backend Examples

<details>

<summary>AzureOpenAI</summary>

1. Install the operator from the [Installation](#installation) section.

2. Create secret:
```sh 
kubectl create secret generic k8sgpt-sample-secret --from-literal=azure-api-key=$AZURE_TOKEN -n k8sgpt-
operator-system
```

3. Apply the K8sGPT configuration object:
```
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: k8sgpt-operator-system
spec:
  model: gpt-35-turbo
  backend: azureopenai
  baseUrl: https://k8sgpt.openai.azure.com/
  engine: llm
  noCache: false
  version: v0.3.2
  enableAI: true
  secret:
    name: k8sgpt-sample-secret
    key: azure-api-key
EOF
```

</details>

<details>

<summary>LocalAI</summary>


1. Install the operator from the [Installation](#installation) section.

2. Follow the [LocalAI installation guide](https://github.com/go-skynet/helm-charts#readme) to install LocalAI. (*No OpenAI secret is required when using LocalAI*).

3. Apply the K8sGPT configuration object:
```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-local-ai
  namespace: default
spec:
  model: gpt-3.5-turbo
  backend: localai
  noCache: false
  version: v0.3.0
  enableAI: true
EOF
```

4. Same as step 4. in the example above.

</details>

## Helm values

For details please see [here](chart/values.yaml)

