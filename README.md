<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./images/banner-white.png" width="600px;">
  <img alt="Text changing depending on mode. Light: 'So light!' Dark: 'So dark!'" src="./images/banner-black.png" width="600px;">
</picture>
<br/>
<br />

This Operator is designed to enable K8sGPT within a Kubernetes cluster.
It will allow you to create a custom resource that defines the behaviour and scope of a managed K8sGPT workload. Analysis and outputs will also be configurable to enable integration into existing workflows.


## Installation

```
helm repo add k8sgpt https://charts.k8sgpt.ai/
helm install release k8sgpt/k8sgpt-operator
```

## Run the example

1. Install the operator from the [Installation](#installation) section.

2. Create secret:
```sh 
kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key=$OPENAI_TOKEN -n default
```

1. Apply the K8sGPT configuration object:
```sh
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
spec:
  namespace: default
  model: gpt-3.5-turbo
  backend: openai
  noCache: false
  version: v0.2.4
  enableAI: true
  secret:
    name: k8sgpt-sample-secret
    key: openai-api-key
EOF
```

## Architecture

<img src="images/1.png" width="600px;" />

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Attach from local
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)


