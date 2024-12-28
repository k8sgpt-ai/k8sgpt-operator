<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/banner-white.png" width="600px;">
  <img alt="Text changing depending on mode. Light: 'So light!' Dark: 'So dark!'" src="./images/banner-black.png" width="600px;">
</picture>
<br/>

[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/k8sgpt)](https://artifacthub.io/packages/search?repo=k8sgpt)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fk8sgpt-ai%2Fk8sgpt-operator.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fk8sgpt-ai%2Fk8sgpt-operator?ref=badge_shield)

This Operator is designed to enable [K8sGPT](https://github.com/k8sgpt-ai/k8sgpt/) within a Kubernetes cluster.
It will allow you to create a custom resource that defines the behaviour and scope of a managed K8sGPT workload. Analysis and outputs will also be configurable to enable integration into existing workflows.

<img src="images/demo2.gif" width="600px;"/>

## Installation

```
helm repo add k8sgpt https://charts.k8sgpt.ai/
helm repo update
helm install release k8sgpt/k8sgpt-operator -n k8sgpt-operator-system --create-namespace
```

## Run the example

1. Install the operator from the [Installation](#installation) section.

2. Create secret:

```sh
kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key=$OPENAI_TOKEN -n k8sgpt-operator-system
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
  ai:
    enabled: true
    model: gpt-3.5-turbo
    backend: openai
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
    # backOff:
    #  enabled: false
    #  maxRetries: 5
    # anonymized: false
    # language: english
    # proxyEndpoint: https://10.255.30.150 # use proxyEndpoint to setup backend through an HTTP/HTTPS proxy
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
  #integrations:
  # trivy:
  #  enabled: true
  #  namespace: trivy-system
  # filters:
  #   - Ingress
  # sink:
  #   type: slack
  #   webhook: <webhook-url> # use the sink secret if you want to keep your webhook url private
  #   secret:
  #     name: slack-webhook
  #     key: url
  #extraOptions:
  #   backstage:
  #     enabled: true
EOF
```

4. Once the custom resource has been applied the K8sGPT-deployment will be installed and
   you will be able to see the Results objects of the analysis after some minutes (if there are any issues in your cluster):

```bash
❯ kubectl get results -n k8sgpt-operator-system -o json | jq .
{
  "apiVersion": "v1",
  "items": [
    {
      "apiVersion": "core.k8sgpt.ai/v1alpha1",
      "kind": "Result",
      "spec": {
        "details": "The error message means that the service in Kubernetes doesn't have any associated endpoints, which should have been labeled with \"control-plane=controller-manager\". \n\nTo solve this issue, you need to add the \"control-plane=controller-manager\" label to the endpoint that matches the service. Once the endpoint is labeled correctly, Kubernetes can associate it with the service, and the error should be resolved.",
```

## Monitor multiple clusters

The `k8sgpt.ai` Operator allows monitoring multiple clusters by providing a `kubeconfig` value.

This feature could be fascinating if you want to embrace Platform Engineering such as running a fleet of Kubernetes clusters for multiple stakeholders.
Especially designed for the Cluster API-based infrastructures, `k8sgpt.ai` Operator is going to be installed in the same Cluster API management cluster:
this one is responsible for creating the required clusters according to the infrastructure provider for the seed clusters.

Once a Cluster API-based cluster has been provisioned a `kubeconfig` according to the naming convention `${CLUSTERNAME}-kubeconfig` will be available in the same namespace:
the conventional Secret data key is `value`, this can be used to instruct the `k8sgpt.ai` Operator to monitor a remote cluster without installing any resource deployed to the seed cluster.

```
$: kubectl get clusters
NAME              PHASE         AGE   VERSION
capi-quickstart   Provisioned   8s    v1.28.0

$: kubectl get secrets
NAME                         TYPE     DATA   AGE
capi-quickstart-kubeconfig   Opaque   1      8s
```

> **A security concern**
>
> If your setup requires the least privilege approach,
> a different `kubeconfig` must be provided since the Cluster API generated one is bounded to the `admin` user which has `clustr-admin` permissions.

Once you have a valid `kubeconfig`, a `k8sgpt` instance can be created as it follows.

```yaml
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: capi-quickstart
  namespace: default
spec:
  ai:
    anonymized: true
    backend: openai
    language: english
    model: gpt-3.5-turbo
    secret:
      key: api_key
      name: my_openai_secret
  kubeconfig:
    key: value
    name: capi-quickstart-kubeconfig
```

Once applied the `k8sgpt.ai` Operator will create the `k8sgpt.ai` Deployment by using the seed cluster `kubeconfig` defined in the field `/spec/kubeconfig`.

The resulting `Result` objects will be available in the same Namespace where the `k8sgpt.ai` instance has been deployed,
accordingly labelled with the following keys:

- `k8sgpts.k8sgpt.ai/name`: the `k8sgpt.ai` instance Name
- `k8sgpts.k8sgpt.ai/namespace`: the `k8sgpt.ai` instance Namespace
- `k8sgpts.k8sgpt.ai/backend`: the AI backend (if specified)

Thanks to these labels, the results can be filtered according to the specified monitored cluster,
without polluting the underlying cluster with the `k8sgpt.ai` CRDs and consuming seed compute workloads,
as well as keeping confidentiality about the AI backend driver credentials.

> In case of missing `/spec/kubeconfig` field, `k8sgpt.ai` Operator will track the cluster on which has been deployed:
> this is possible by mounting the provided `ServiceAccount`.

## Distributed Cache

<details>

<summary>Interplex cache</summary>

[Interplex](https://github.com/interplex-ai/interplex.git) is a caching system designed to work over RPC and optimised for K8sGPT. This cache can be installed without any credentials in your local cluster as part of your normal helm install.

1. Install K8sGPT Operator with Interplex

```
helm install release k8sgpt/k8sgpt-operator -n k8sgpt-operator-system --create-namespace --set interplex.enabled=true
```

2. Create the secret for your AI backend (_in this example we use OPENAI_):
```
kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key=$OPENAI_TOKEN -n k8sgpt-operator-system
```

3. Point your K8sGPT Custom resource to the interplex cache: (match the helm release name with the cache prefix e.g., myrelease-interplex-service:8084)

```
  kubectl apply -f - << EOF
  apiVersion: core.k8sgpt.ai/v1alpha1
  kind: K8sGPT
  metadata:
    name: k8sgpt-sample
    namespace: k8sgpt-operator-system
  spec:
    ai:
      enabled: true
      model: gpt-3.5-turbo
      backend: openai
      secret:
        name: k8sgpt-sample-secret
        key: openai-api-key
    noCache: false
    remoteCache:
      interplex:
        endpoint: release-interplex-service:8084
    repository: ghcr.io/k8sgpt-ai/k8sgpt
    version: v0.3.48
  EOF
```

</details>

## Remote Cache

<details>

<summary>Azure Blob storage</summary>

1. Install the operator from the [Installation](#installation) section.

2. Create secret:

```sh
kubectl create secret generic k8sgpt-sample-cache-secret --from-literal=azure_client_id=<AZURE_CLIENT_ID>  --from-literal=azure_tenant_id=<AZURE_TENANT_ID> --from-literal=azure_client_secret=<AZURE_CLIENT_SECRET> -n k8sgpt-
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
  ai:
    model: gpt-3.5-turbo
    backend: openai
    enabled: true
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
  remoteCache:
    credentials:
      name: k8sgpt-sample-cache-secret
    azure:
      # Storage account must already exist
      storageAccount: "account_name"
      containerName: "container_name"
EOF
```

</details>

<details>

<summary>S3</summary>

1. Install the operator from the [Installation](#installation) section.

2. Create secret:

```sh
kubectl create secret generic k8sgpt-sample-cache-secret --from-literal=aws_access_key_id=<AWS_ACCESS_KEY_ID>  --from-literal=aws_secret_access_key=<AWS_SECRET_ACCESS_KEY> -n k8sgpt-
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
  ai:
    model: gpt-3.5-turbo
    backend: openai
    enabled: true
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
  remoteCache:
    credentials:
      name: k8sgpt-sample-cache-secret
    s3:
      bucketName: foo
      region: us-west-1
EOF
```

</details>

## Other AI Backend Examples

<details>

<summary>AzureOpenAI</summary>

1. Install the operator from the [Installation](#installation) section.

2. Create secret:

```sh
kubectl create secret generic k8sgpt-sample-secret --from-literal=azure-api-key=$AZURE_TOKEN -n k8sgpt-operator-system
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
  ai:
    enabled: true
    secret:
      name: k8sgpt-sample-secret
      key: azure-api-key
    model: gpt-35-turbo
    backend: azureopenai
    baseUrl: https://k8sgpt.openai.azure.com/
    engine: llm
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
EOF
```

</details>

<details>

<summary>Amazon Bedrock</summary>

1. Install the operator from the [Installation](#installation) section.

2. When running on AWS, you have a number of ways to give permission to the managed K8sGPT workload to access Amazon Bedrock.

- Grant access to Bedrock using the Kubernetes Service Account. This is the [best practices method for assigning permissions to Kubernetes Pods](https://aws.github.io/aws-eks-best-practices/security/docs/iam/#identities-and-credentials-for-eks-pods). There are a few ways to do this:
  - On Amazon EKS, using [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
  - On Amazon EKS, using [IAM Roles for Service Accounts (IRSA)](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
  - On self-managed Kubernetes, using IAM Roles for Service Accounts (IRSA) with the [Pod Identity Webhook](https://github.com/aws/amazon-eks-pod-identity-webhook)
- Grant access to Bedrock using AWS credentials in a Kubernetes Secret. Note this goes [against AWS best practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html#bp-workloads-use-roles) and should be used with caution.

To grant access to Bedrock using a Kubernetes Service account, create an IAM role with Bedrock permissions. An example policy is included below:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream"
      ],
      "Resource": "*"
    }
  ]
}
```

To grant access to Bedrock using AWS credentials in a Kubernetes secret you can create a secret:

```sh
kubectl create secret generic bedrock-sample-secret --from-literal=AWS_ACCESS_KEY_ID="$(echo $AWS_ACCESS_KEY_ID)" --from-literal=AWS_SECRET_ACCESS_KEY="$(echo $AWS_SECRET_ACCESS_KEY)" -n k8sgpt-operator-system
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
  ai:
    enabled: true
    secret:
     name: bedrock-sample-secret
    model: anthropic.claude-v2
    region: eu-central-1
    backend: amazonbedrock
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
EOF
```

</details>

<details>

<summary>LocalAI</summary>

1. Install the operator from the [Installation](#installation) section.

2. Follow the [LocalAI installation guide](https://github.com/go-skynet/helm-charts#readme) to install LocalAI. (_No OpenAI secret is required when using LocalAI_).

3. Apply the K8sGPT configuration object:

```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-local-ai
  namespace: default
spec:
  ai:
    enabled: true
    model: ggml-gpt4all-j
    backend: localai
    baseUrl: http://local-ai.local-ai.svc.cluster.local:8080/v1
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
EOF
```

Note: ensure that the value of `baseUrl` is a properly constructed [DNS name](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#services) for the LocalAI Service. It should take the form: `http://local-ai.<namespace_local_ai_was_installed_in>.svc.cluster.local:8080/v1`.

1. Same as step 4. in the example above.

</details>

## K8sGPT Configuration Options

<details>

<summary>ImagePullSecrets</summary>
You can use custom k8sgpt image by modifying `repository`, `version`, `imagePullSecrets`.
`version` actually works as image tag.

```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: k8sgpt-operator-system
spec:
  ai:
    enabled: true
    model: gpt-3.5-turbo
    backend: openai
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  noCache: false
  repository: sample.repository/k8sgpt
  version: sample-tag
  imagePullSecrets:
    - name: sample-secret
EOF
```

</details>

<details>

<summary>Resources</summary>
You can use custom k8sgpt container resource usage by `resources`.

```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: k8sgpt-operator-system
spec:
  ai:
    enabled: true
    model: gpt-3.5-turbo
    backend: openai
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  resources:
    limits:
      cpu: 10
      memory: 512Mi
    requests:
      cpu: 200m
      memory: 156Mi
EOF
```

</details>

<details>
<summary>sink (integrations) </summary>

Optional parameters available for sink.  
('type', 'webhook' are required parameters.)

| tool       | channel | icon_url | username |
| ---------- | ------- | -------- | -------- |
| Slack      |         |          |          |
| Mattermost | ✔️      | ✔️       | ✔️       |

</details>

## Helm values

For details please see [here](chart/operator/values.yaml)

## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fk8sgpt-ai%2Fk8sgpt-operator.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fk8sgpt-ai%2Fk8sgpt-operator?ref=badge_large)
