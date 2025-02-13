# Auto Remediation

Status: Alpha 
Supported AI Backends:
- Amazonbedrock
- OpenAI

Auto Remediation will attempt to fix problems encountered in your cluster.
To accomplish this, it interprets K8sGPT results and applying a patch to fix the issue on the target resource (or parent ).

This feature is **highly** experimental and is not ready for use in a production environment.
To enable this feature, you need to set the following K8sGPT custom resource field:

```bash
cat<<EOF | kubectl apply -f -
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: default
spec:
  ai:
    autoRemediation:
      enabled: true
      riskThreshold: 90
      resources:
        - Pod
        - Service
        - Deployment
...
```
### AISpec Fields

`autoRemediation`: Configures automatic remediation of identified issues.

`enabled`: A boolean value. If true, enables automatic remediation.

`similarityRequirement`: A string representing the required similarity with the original manifest (e.g., "90"). 
New proposed manifests with a requirement above this threshold will be automatically remediated.

`resources`: A list of Kubernetes resource types to consider for automatic remediation (e.g., Pod, Service, Deployment, Ingress).

Complete example available [here](./config/samples/autoremediation/valid_k8sgpt_remediation_sample.yaml)

## How does it work?

Opting-in to auto remediation will enable the following processes:
- K8sGPT operator will parse results that have been created, and calculate
kinds that auto remediation has been [enabled on](#supported_Kinds). Upon doing so, it will also create a [Mutation](#mutations).
- After Mutations are created they will attempt to reconcile the differenc in the origin resource vs the target changes.
- Once a patch has been calculated ( in-part based on similarity score), they will attempt to apply it.
- The resource change will be watched until the result either is removed ( as the resource is now fixed ) or persists.
- The mutation will keep an entire log of the changes and events that occured.


## Supported Kinds

Currently in Alpha state, the supported kinds are:
- Service
- Pod
  - Owned (RS/Deployment)
  - Static

## Mutations

Mutations are custom resources that hold the state and intent for mutating resources in the cluster.
Eventually this will be compatible with a GitOps process ( you can pull the mutations out of cluster and re-apply).

Currently Mutations will reside in the same namespaces as your `K8sGPT` custom resource.
Mutations are controlled by a finaliser and will require `k8sgpt-operator` running for deletion automatically.
## Rollback 

TODO: Deleting a mutation will revert the applied changes to the cluster resource. 