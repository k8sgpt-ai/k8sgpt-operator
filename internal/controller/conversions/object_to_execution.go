package conversions

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/remediation/policy"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectExecutionConfig struct {
	Ctx      context.Context
	Rc       client.Client
	Obj      client.Object
	Mutation corev1alpha1.Mutation
	Log      logr.Logger
}

func ResourceToExecution(config ObjectExecutionConfig) (ctrl.Result, error) {
	// Never apply an LLM-produced replacement directly. Fetch the object again,
	// calculate the semantic patch locally, and let the policy gate restrict it.
	live := &unstructured.Unstructured{}
	live.SetGroupVersionKind(config.Obj.GetObjectKind().GroupVersionKind())
	if err := config.Rc.Get(config.Ctx, client.ObjectKeyFromObject(config.Obj), live); err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	original, err := policy.ParseYAMLProposal([]byte(config.Mutation.Spec.OriginConfiguration))
	if err != nil {
		return rejectMutation(config, "Invalid origin snapshot: "+err.Error(), policy.Decision{})
	}
	proposed, ok := config.Obj.(*unstructured.Unstructured)
	if !ok {
		return rejectMutation(config, "Proposed remediation is not an unstructured Kubernetes object", policy.Decision{})
	}
	decision := policy.Evaluate(policy.Input{Original: original, Live: live, Proposed: proposed})
	if !decision.Allowed {
		return rejectMutation(config, "Policy rejected remediation: "+decision.Reasons[0], decision)
	}
	// The test operation closes the dry-run/apply race: a concurrent update makes
	// the persistent patch fail rather than applying a decision made from stale state.
	operations := append([]policy.PatchOperation{{
		Op:    "test",
		Path:  "/metadata/resourceVersion",
		Value: live.GetResourceVersion(),
	}}, decision.Patch...)
	patch, err := json.Marshal(operations)
	if err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	// Ask the API server to validate admission, schema, and immutability rules
	// before making the same narrow patch persistent.
	if err := config.Rc.Patch(config.Ctx, live.DeepCopy(), client.RawPatch(types.JSONPatchType, patch), client.DryRunAll); err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	if err := config.Rc.Patch(config.Ctx, live, client.RawPatch(types.JSONPatchType, patch)); err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	config.Mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseCompleted
	config.Mutation.Status.Message = "Completed: policy-approved patch applied"
	config.Mutation.Status.PolicyDecision = "approved"
	config.Mutation.Status.ChangedPaths = decisionPaths(decision)
	now := metav1.Now()
	config.Mutation.Status.AppliedAt = &now
	if err := config.Rc.Update(config.Ctx, &config.Mutation); err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	return ctrl.Result{RequeueAfter: util.SuccessfulRequeueTime}, nil
}

func rejectMutation(config ObjectExecutionConfig, reason string, decision policy.Decision) (ctrl.Result, error) {
	config.Log.Info(reason, "mutation", config.Mutation.Name)
	config.Mutation.Status.Phase = corev1alpha1.AutoRemediationAborted
	config.Mutation.Status.Message = reason
	config.Mutation.Status.PolicyDecision = "rejected"
	config.Mutation.Status.ChangedPaths = decisionPaths(decision)
	if err := config.Rc.Update(config.Ctx, &config.Mutation); err != nil {
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	return ctrl.Result{}, nil
}

func decisionPaths(decision policy.Decision) []string {
	paths := make([]string, 0, len(decision.Changes))
	for _, change := range decision.Changes {
		paths = append(paths, change.Path)
	}
	return paths
}
