
K8sgpt-operator
===========

Automatic SRE Superpowers within your Kubernetes cluster


## Configuration

The following table lists the configurable parameters of the K8sgpt-operator chart and their default values.

## Metrics Configuration

### kube-rbac-proxy

By default, the operator deploys a `kube-rbac-proxy` sidecar container to protect the metrics endpoint with Kubernetes RBAC authorization and HTTPS encryption.

#### Disabling kube-rbac-proxy

You can disable the proxy to expose metrics directly via HTTP:

```yaml
controllerManager:
  kubeRbacProxy:
    enabled: false
```

When disabled, the manager exposes /metrics directly over HTTP on port
8080 without Kubernetes RBAC authorization. Ensure access is protected by
appropriate network policies, service mesh policies, or other infrastructure
controls.

<!---x-release-please-start-version-->
| Parameter                | Description             | Default                                                                       |
| ------------------------ | ----------------------- |-------------------------------------------------------------------------------|
| `serviceMonitor.enabled` |  | `false`                                                                       |
| `serviceMonitor.additionalLabels` |  | `{}`                                                                          |
| `grafanaDashboard.enabled` |  | `false`                                                                       |
| `grafanaDashboard.folder.annotation` |  | `"grafana_folder"`                                                            |
| `grafanaDashboard.folder.name` |  | `"ai"`                                                                        |
| `grafanaDashboard.label.key` |  | `"grafana_dashboard"`                                                         |
| `grafanaDashboard.label.value` |  | `"1"`                                                                         |
| `controllerManager.kubeRbacProxy.enabled` | Enable kube-rbac-proxy for RBAC-protected HTTPS metrics | `true` |
| `controllerManager.kubeRbacProxy.containerSecurityContext.allowPrivilegeEscalation` |  | `false`                                                                       |
| `controllerManager.kubeRbacProxy.containerSecurityContext.capabilities.drop` |  | `["ALL"]`                                                                     |
| `controllerManager.kubeRbacProxy.image.repository` |  | `"gcr.io/kubebuilder/kube-rbac-proxy"`                                        |
| `controllerManager.kubeRbacProxy.image.tag` |  | `"v0.2.29"`                                                                    |
| `controllerManager.kubeRbacProxy.resources.limits.cpu` |  | `"500m"`                                                                      |
| `controllerManager.kubeRbacProxy.resources.limits.memory` |  | `"128Mi"`                                                                     |
| `controllerManager.kubeRbacProxy.resources.requests.cpu` |  | `"5m"`                                                                        |
| `controllerManager.kubeRbacProxy.resources.requests.memory` |  | `"64Mi"`                                                                      |
| `controllerManager.manager.sinkWebhookTimeout` |  | `"30s"`                                                                       |
| `controllerManager.manager.enableResultLogging` |  | `false`                                                                       |
| `controllerManager.manager.containerSecurityContext.allowPrivilegeEscalation` |  | `false`                                                                       |
| `controllerManager.manager.containerSecurityContext.capabilities.drop` |  | `["ALL"]`                                                                     |
| `controllerManager.manager.image.repository` |  | `"ghcr.io/k8sgpt-ai/k8sgpt-operator"`                                         |
| `controllerManager.manager.image.tag` | x-release-please-version | `"v0.2.29"`                                                                    |
| `controllerManager.manager.resources.limits.cpu` |  | `"500m"`                                                                      |
| `controllerManager.manager.resources.limits.memory` |  | `"128Mi"`                                                                     |
| `controllerManager.manager.resources.requests.cpu` |  | `"10m"`                                                                       |
| `controllerManager.manager.resources.requests.memory` |  | `"64Mi"`                                                                      |
| `controllerManager.replicas` |  | `1`                                                                           |
| `controllerManager.labels` | | []                                                                            |                                                                            |
| `kubernetesClusterDomain` |  | `"cluster.local"`                                                             |
| `metricsService.ports` |  | `[{"name": "https", "port": 8443, "protocol": "TCP", "targetPort": "https"}]` |
| `metricsService.type` |  | `"ClusterIP"`                                                                 |

<!---x-release-please-end-->

---
_Documentation generated by [Frigate](https://frigate.readthedocs.io)._

