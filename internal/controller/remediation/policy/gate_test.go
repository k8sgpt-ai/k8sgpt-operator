package policy

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const deploymentYAML = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
  namespace: shop
  uid: deployment-uid
  resourceVersion: "10"
  labels:
    app: checkout
spec:
  replicas: 2
  template:
    spec:
      serviceAccountName: checkout
      containers:
      - name: app
        image: registry.example/checkout:broken
`

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		proposed string
		mutate   func(*unstructured.Unstructured, *unstructured.Unstructured)
		input    func(Input) Input
		allowed  bool
		reason   string
	}{
		{
			name:     "allows a single deployment image replacement",
			proposed: strings.Replace(deploymentYAML, "checkout:broken", "checkout:fixed", 1),
			allowed:  true,
		},
		{
			name:     "rejects non allowlisted replicas change",
			proposed: strings.Replace(deploymentYAML, "replicas: 2", "replicas: 3", 1),
			reason:   "path is not allowlisted: /spec/replicas",
		},
		{
			name:     "rejects service account changes as high risk",
			proposed: strings.Replace(deploymentYAML, "serviceAccountName: checkout", "serviceAccountName: admin", 1),
			reason:   "high-risk path is forbidden: /spec/template/spec/serviceAccountName",
		},
		{
			name:     "rejects stale resource version",
			proposed: strings.Replace(deploymentYAML, "checkout:broken", "checkout:fixed", 1),
			mutate:   func(_, live *unstructured.Unstructured) { live.SetResourceVersion("11") },
			reason:   "live object resourceVersion does not match the proposal snapshot",
		},
		{
			name:     "rejects cross namespace proposal",
			proposed: strings.Replace(strings.Replace(deploymentYAML, "namespace: shop", "namespace: other", 1), "checkout:broken", "checkout:fixed", 1),
			reason:   "proposed object must exactly match target apiVersion, kind, namespace, and name",
		},
		{
			name:     "rejects multiple changes under default bound",
			proposed: strings.Replace(strings.Replace(deploymentYAML, "checkout:broken", "checkout:fixed", 1), "replicas: 2", "replicas: 3", 1),
			reason:   "proposal changes 2 paths; maximum is 1",
		},
		{
			name:     "rejects no op",
			proposed: deploymentYAML,
			reason:   "proposal makes no semantic change",
		},
		{
			name:     "rejects unsupported resource",
			proposed: strings.Replace(strings.Replace(deploymentYAML, "apps/v1", "v1", 1), "kind: Deployment", "kind: Service", 1),
			mutate: func(original, live *unstructured.Unstructured) {
				original.SetAPIVersion("v1")
				original.SetKind("Service")
				live.SetAPIVersion("v1")
				live.SetKind("Service")
			},
			reason: "no remediation policy registered for GVK /v1, Kind=Service; set core.k8sgpt.ai/auto-remediation-allowed-paths to explicit JSON Pointer paths",
		},
		{
			name:     "rejects an unregistered version of a registered kind",
			proposed: strings.Replace(strings.Replace(deploymentYAML, "apps/v1", "apps/v2", 1), "checkout:broken", "checkout:fixed", 1),
			mutate: func(original, live *unstructured.Unstructured) {
				original.SetAPIVersion("apps/v2")
				live.SetAPIVersion("apps/v2")
			},
			reason: "no remediation policy registered for GVK apps/v2, Kind=Deployment; set core.k8sgpt.ai/auto-remediation-allowed-paths to explicit JSON Pointer paths",
		},
		{
			name:     "rejects proposal resource version that differs from live target",
			proposed: strings.Replace(strings.Replace(deploymentYAML, "resourceVersion: \"10\"", "resourceVersion: \"9\"", 1), "checkout:broken", "checkout:fixed", 1),
			reason:   "proposed object resourceVersion does not match target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := mustObject(t, deploymentYAML)
			live := mustObject(t, deploymentYAML)
			proposed := mustObject(t, tt.proposed)
			if tt.mutate != nil {
				tt.mutate(original, live)
			}
			in := Input{Original: original, Live: live, Proposed: proposed}
			if tt.input != nil {
				in = tt.input(in)
			}
			got := Evaluate(in)
			if got.Allowed != tt.allowed {
				t.Fatalf("Allowed = %v, reasons = %v", got.Allowed, got.Reasons)
			}
			if tt.reason != "" && (len(got.Reasons) != 1 || got.Reasons[0] != tt.reason) {
				t.Fatalf("Reasons = %v, want %q", got.Reasons, tt.reason)
			}
			if got.Allowed {
				if len(got.Changes) != 1 || got.Changes[0].Path != "/spec/template/spec/containers/0/image" {
					t.Fatalf("unexpected changes: %#v", got.Changes)
				}
				if len(got.Patch) != 1 || got.Patch[0].Op != "replace" {
					t.Fatalf("unexpected patch: %#v", got.Patch)
				}
			}
		})
	}
}

func TestRegisteredGVKs(t *testing.T) {
	want := []schema.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "Pod"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
		{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		{Group: "batch", Version: "v1", Kind: "CronJob"},
		{Group: "batch", Version: "v1", Kind: "Job"},
	}
	got := RegisteredGVKs()
	if len(got) != len(want) {
		t.Fatalf("registered GVKs = %#v, want %#v", got, want)
	}
	for _, expected := range want {
		found := false
		for _, actual := range got {
			if actual == expected {
				found = true
			}
		}
		if !found {
			t.Errorf("registered GVKs = %#v, missing %#v", got, expected)
		}
	}
}

func TestEvaluateAllowsRegisteredWorkloadImagePaths(t *testing.T) {
	for _, tc := range []struct {
		name       string
		apiVersion string
		kind       string
		imagePath  string
	}{
		{name: "daemonset", apiVersion: "apps/v1", kind: "DaemonSet", imagePath: "spec:\n  template:\n    spec:\n      containers:\n      - name: app\n        image: example/app:bad"},
		{name: "statefulset", apiVersion: "apps/v1", kind: "StatefulSet", imagePath: "spec:\n  template:\n    spec:\n      containers:\n      - name: app\n        image: example/app:bad"},
		{name: "replicaset", apiVersion: "apps/v1", kind: "ReplicaSet", imagePath: "spec:\n  template:\n    spec:\n      containers:\n      - name: app\n        image: example/app:bad"},
		{name: "job", apiVersion: "batch/v1", kind: "Job", imagePath: "spec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n      - name: app\n        image: example/app:bad"},
		{name: "cronjob", apiVersion: "batch/v1", kind: "CronJob", imagePath: "spec:\n  jobTemplate:\n    spec:\n      template:\n        spec:\n          restartPolicy: Never\n          containers:\n          - name: app\n            image: example/app:bad"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := "apiVersion: " + tc.apiVersion + "\nkind: " + tc.kind + "\nmetadata:\n  name: repairable\n  namespace: shop\n  uid: uid\n  resourceVersion: \"1\"\n" + tc.imagePath + "\n"
			original, live := mustObject(t, yaml), mustObject(t, yaml)
			proposed := mustObject(t, strings.Replace(yaml, "app:bad", "app:good", 1))
			if decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed}); !decision.Allowed {
				t.Fatalf("registered workload image replacement rejected: %#v", decision)
			}
		})
	}
}

func TestEvaluateAllowsExplicitObjectOptInForAnyGVK(t *testing.T) {
	configMap := `apiVersion: v1
kind: ConfigMap
metadata:
  name: settings
  namespace: shop
  uid: config-uid
  resourceVersion: "7"
  annotations:
    core.k8sgpt.ai/auto-remediation-allowed-paths: /data/log-level
data:
  log-level: info
`
	original, live := mustObject(t, configMap), mustObject(t, configMap)
	proposed := mustObject(t, strings.Replace(configMap, "log-level: info", "log-level: debug", 1))
	decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed})
	if !decision.Allowed || len(decision.Patch) != 1 || decision.Patch[0].Path != "/data/log-level" {
		t.Fatalf("explicit ConfigMap policy was not honored: %#v", decision)
	}
}

func TestEvaluateRejectsUnregisteredGVKWithoutObjectOptIn(t *testing.T) {
	service := `apiVersion: v1
kind: Service
metadata:
  name: api
  namespace: shop
  uid: service-uid
  resourceVersion: "7"
spec:
  ports:
  - port: 80
    targetPort: 8080
`
	original, live := mustObject(t, service), mustObject(t, service)
	proposed := mustObject(t, strings.Replace(service, "targetPort: 8080", "targetPort: 8081", 1))
	decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed})
	if decision.Allowed || !strings.Contains(decision.Reasons[0], AllowedPathsAnnotation) {
		t.Fatalf("unregistered Service without opt-in must be rejected: %#v", decision)
	}
}

func TestEvaluatePodImageAndMaxPathBound(t *testing.T) {
	pod := `apiVersion: v1
kind: Pod
metadata:
  name: repairable
  namespace: shop
  uid: pod-uid
  resourceVersion: "7"
spec:
  containers:
  - name: app
    image: example/app:bad
  initContainers:
  - name: init
    image: example/init:bad
`
	original, live := mustObject(t, pod), mustObject(t, pod)
	proposed := mustObject(t, strings.Replace(strings.Replace(pod, "app:bad", "app:good", 1), "init:bad", "init:good", 1))
	if decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed}); decision.Allowed || !strings.Contains(decision.Reasons[0], "maximum is 1") {
		t.Fatalf("default limit should reject: %#v", decision)
	}
	decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed, MaxChangedPaths: 2})
	if !decision.Allowed || len(decision.Patch) != 2 {
		t.Fatalf("two image changes should be allowed: %#v", decision)
	}
}

func TestEvaluateRejectsOwnedPod(t *testing.T) {
	pod := `apiVersion: v1
kind: Pod
metadata:
  name: managed
  namespace: shop
  uid: pod-uid
  resourceVersion: "7"
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: managed-abc
    uid: owner-uid
spec:
  containers:
  - name: app
    image: example/app:bad
`
	original, live := mustObject(t, pod), mustObject(t, pod)
	proposed := mustObject(t, strings.Replace(pod, "app:bad", "app:good", 1))
	decision := Evaluate(Input{Original: original, Live: live, Proposed: proposed})
	if decision.Allowed || decision.Reasons[0] != "owned Pods are not remediated directly; target the owning workload instead" {
		t.Fatalf("owned pod must be rejected: %#v", decision)
	}
}

func TestParseYAMLProposal(t *testing.T) {
	object, err := ParseYAMLProposal([]byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n"))
	if err != nil || object.GetKind() != "Pod" || object.GetName() != "x" {
		t.Fatalf("unexpected parse: %#v, %v", object, err)
	}
	if _, err := ParseYAMLProposal([]byte("[")); err == nil {
		t.Fatal("invalid YAML was accepted")
	}
}

func mustObject(t *testing.T, yaml string) *unstructured.Unstructured {
	t.Helper()
	object, err := ParseYAMLProposal([]byte(yaml))
	if err != nil {
		t.Fatalf("parse object: %v", err)
	}
	return object
}
