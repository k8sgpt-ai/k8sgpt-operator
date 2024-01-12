# Contributing to K8sGPT operator
Instructions to help with developing the operator

## Description
The operator has been bootstrapped with kubebuilder which also leverages a Makefile for
development operations
## Installing/Updating CRDs
By invoking `make install` you will install to your current K8s cluster the K8sGPT CRDs.

There will be times when your feature work will require a CRD update, after changing the corresponding
GO API struct you can simply run `make manifests` and `make install` to generate the new CRDs and deploy them
to you cluster.

Note: At the moment, you have to place the updated CRDs manually to the HELM chart's [CRD templates](https://github.com/k8sgpt-ai/k8sgpt-operator/blob/main/chart/operator/templates/k8sgpt-crd.yaml)
## Running in Local Mode
In order to run your operator locally(without the need of Helm charts) you have to specify an env variable `LOCAL_MODE`, apply a K8sGPT [CR](https://github.com/k8sgpt-ai/k8sgpt-operator?tab=readme-ov-file#run-the-example) and port forward K8sGPT's service.
E.g 
`LOCAL_MODE=1 make run` and `kubectl port-forward svc/k8sgpt 8080:8080`

---  
### Local mode with out-of-cluster K8sGPT service

In a nutshell, the operator is communicating over GRPC with the K8sGPT instance which runs as a long-running [service](https://github.com/k8sgpt-ai/k8sgpt/blob/main/cmd/serve/serve.go)

In the scenario where you are testing an uncomitted or unreleased K8sGPT with your operator you can run locally both the operator and the K8sGPT server,
by running again `LOCAL_MODE=1 make run` and in your local K8sGPT clone you can run `k8sgpt serve` or simply ` go run main.go serve`
Note: You should always deploy a K8sGPT Custom Resource so the operator's reconcilaition can be triggered and you can set arbitrary values for the k8sgpt's version since you will bypass them.

## Help
Feel free to join our slack [channel](https://k8sgpt.slack.com) and open GH issues, so we can make the development experience better for all K8sGPT contributors

