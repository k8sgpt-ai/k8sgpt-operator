# Auto Remediation

Status: Alpha 

This feature is highly experiment and opt-in only.
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
      riskThreshold: 10
...
```

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
- Ingress 
- Pod
  - Owned (RS/Deployment)
  - Static

## Mutations

Mutations are custom resources that hold the state and intent for mutating resources in the cluster.
Eventually this will be compatible with a GitOps process ( you can pull the mutations out of cluster and re-apply).

