# Auto Remediation

Status: Alpha. Auto remediation is experimental and must not be relied on for production changes without independent review.

The operator can create a `Mutation` proposal for any standard API resource when the analyzer supplies an exact target reference. Automatic execution is intentionally narrower: it is deny-by-default and currently has image-only policies for unowned `Pod`, `Deployment`, `DaemonSet`, `StatefulSet`, `ReplicaSet`, `Job`, and `CronJob` findings. Owned Pods are rejected: target the owning workload directly.

Pod findings owned by a selected workload are resolved to that workload before a proposal is requested (for example, `Pod` → `ReplicaSet` → `Deployment`). This changes desired state and allows Kubernetes to perform a normal rollout.

## Enable it

```yaml
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: default
spec:
  ai:
    autoRemediation:
      enabled: true
      similarityRequirement: "90"
      resources:
        - Pod
        - Deployment
```

`enabled` turns the feature on. `resources` is an exact allowlist of resource selectors. Use a legacy `Kind` (`Deployment`), `group/Kind` (`apps/Deployment`), or an exact `group/version/Kind` (`apps/v1/Deployment`, `v1/ConfigMap`). A selector lets the operator create a proposal; it does not grant mutation rights. The matching GVK must also have a registered policy.

`similarityRequirement` is retained for CRD compatibility but no longer authorizes execution. Whole-manifest textual similarity is not a safety proof: YAML formatting, field order, generated fields, and unrelated changes can affect it.

## Safety model and lifecycle

The intended lifecycle is:

1. An allowlisted finding with an exact target creates a `Mutation` proposal with the original resource configuration and the proposed target configuration. This works for namespaced and cluster-scoped standard API resources.
2. Immediately before execution, the operator fetches the target again and admits only a semantic JSON patch that passes the policy gate: exact GVK/name/namespace, unchanged UID and resource version, one changed path, and an allowlisted container or init-container image field. All other proposals are aborted and are not applied.
3. The API server dry-runs the computed patch before it is persisted. The real patch includes a resource-version test, so a concurrent update fails rather than applying a decision made from stale state.
4. For Deployments, the executor also requires the new generation to be observed and the desired replicas available before the finding can be considered resolved. The operator then observes the related result to determine whether remediation was successful.

The generic resolver and executor support all standard API resources, but the field-level policy registry is intentionally narrow. Each registered workload policy permits exactly one container or init-container image replacement. A selected resource without a registered GVK policy must opt in with an object-local exact-path annotation; otherwise it is explicitly aborted rather than being patched generically. Future work should add reviewed, resource-specific policies and post-change health verification. Changes outside the policy remain proposals for human review rather than being applied automatically.

### Opting another standard resource into remediation

For a standard API resource without a built-in GVK policy, its owner can opt
in to an exact, single-field remediation with the
`core.k8sgpt.ai/auto-remediation-allowed-paths` annotation. Its value is a
comma-separated list of RFC 6901 JSON Pointer paths. The K8sGPT resource must
still select the resource GVK and the policy gate still rejects identity,
metadata, security, ownership, status, and multi-path changes.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: application-settings
  annotations:
    core.k8sgpt.ai/auto-remediation-allowed-paths: /data/log-level
data:
  log-level: info
```

This permits a proposal that changes only `/data/log-level`; it does not grant
access to any other ConfigMap field or to another object.

## Exact target identity

`Result.spec.targetRef` is an optional, backward-compatible reference to the
object that produced a finding. New analyzers should populate all of its
fields: `apiVersion`, `kind`, `namespace` (empty for cluster-scoped objects),
`name`, `uid`, and `resourceVersion`.

The legacy `spec.kind` and `spec.name` fields remain supported for reporting.
For generic remediation, prefer `targetRef`. During analyzer migration, the
operator can also resolve a legacy result when `resources` contains exactly one
fully qualified matching selector (for example `v1/ConfigMap`) and the legacy
name is unambiguous. It rejects bare/group-only and ambiguous selectors rather
than guessing a GVK, then re-fetches the object and snapshots its UID and
resource version before proposing a change.

```yaml
spec:
  kind: Deployment # legacy display field
  name: default/web # legacy display field
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    namespace: default
    name: web
    uid: 4b07b05f-75c2-4f5b-9c1d-2ccf0a7b26e6
    resourceVersion: "123"
```

## Current limits and planned work

This implementation deliberately does **not** yet provide:

- A human-approval or proposal-only mode. The feature must remain disabled outside carefully controlled alpha use.
- A complete dry-run/admission audit. `Mutation.status` records the policy decision, changed paths, and apply time; rejection reasons remain in `status.message`.
- Targeted re-analysis that proves the original finding is resolved. Deployment rollout/availability is checked before success, but a missing `Result` remains the finding-resolution signal.
- Remediation classes other than one image replacement. Each future class needs its own allowlist, risk limits, dry-run coverage, and end-to-end tests before it can be enabled.
- Rollback. A `Mutation` records intent but does not restore a previous object version.

The recommended next implementation sequence is: structured Mutation audit fields, proposal-only/approval mode, verification through rollout health plus re-analysis, then one narrowly scoped remediation class at a time.

## System test

Run `make e2e-remediation` on a host with Docker and Kind to exercise the
current Deployment image policy against a real Kubernetes cluster. The test
creates a Deployment with a nonexistent image, waits for `ErrImagePull` or
`ImagePullBackOff`, supplies a deterministic K8sGPT gRPC proposal, then
asserts that the policy-gated patch makes the Deployment available.

### DeepSeek

Set `ai.backend: deepseek` with a Secret containing the API key. The operator
uses K8sGPT's OpenAI-compatible client and configures the DeepSeek endpoint
automatically. The provider is asked only for an existing container name and a
replacement image; the operator builds the candidate manifest locally and the
same one-path policy gate authorizes (or rejects) the resulting patch. See
[the DeepSeek sample](./config/samples/autoremediation/valid_k8sgpt_remediation_deepseek.yaml).

## Mutations and rollback

Mutations are namespaced custom resources that record the intent and lifecycle of a remediation. They are created in the same namespace as the `K8sGPT` resource.

Rollback is not implemented. Deleting a Mutation does not revert a change already applied to the target resource.

See the complete configuration example at [config/samples/autoremediation/valid_k8sgpt_remediation_sample.yaml](./config/samples/autoremediation/valid_k8sgpt_remediation_sample.yaml).
