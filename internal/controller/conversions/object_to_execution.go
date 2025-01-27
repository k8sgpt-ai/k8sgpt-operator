package conversions

import (
	"buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1alpha1 "github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/controller/util"
	"github.com/k8sgpt-ai/k8sgpt-operator/internal/prompts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type ObjectExecutionConfig struct {
	Ctx         context.Context
	Rc          client.Client
	Obj         client.Object
	Mutation    corev1alpha1.Mutation
	QueryClient schemav1grpc.ServerQueryServiceClient
	Backend     string
	Log         logr.Logger
}

// The purpose of this file is to give explicit execution steps depending on the resource type
// E.g., A static pod can only be deleted/created, where as a deployment can be patched
// the supported types within this file must style in alignment with the to_eligible_resources.go file.

func staticPodExecution(config ObjectExecutionConfig) (ctrl.Result, error) {
	// check if the object exists first
	if err := config.Rc.Get(config.Ctx, client.ObjectKey{Name: config.Obj.GetName(),
		Namespace: config.Obj.GetNamespace()}, config.Obj); err != nil {
		// If the object doesn't exist at this point, we should create it based on the targetConfiguration
		if err := config.Rc.Create(config.Ctx, config.Obj); err != nil {
			config.Log.Error(err, "unable to create object", "object", config.Obj.GetName())
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		config.Log.Info("Successfully updated object", "object", config.Obj.GetName())
		config.Mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseCompleted
		config.Mutation.Status.Message = "Completed"
		if err := config.Rc.Update(config.Ctx, &config.Mutation); err != nil {
			config.Log.Error(err, "unable to update mutation status")
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	} else {
		// Let's check if there is a deletion timestamp
		if config.Obj.GetDeletionTimestamp() != nil {
			config.Log.Info("Object has a deletion timestamp, it is being deleted", "object", config.Obj.GetName())
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, nil
		} else {
			// Delete the object
			if err := config.Rc.Delete(config.Ctx, config.Obj); err != nil {
				config.Log.Error(err, "unable to delete object", "object", config.Obj.GetName())
				return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func deploymentExecution(config ObjectExecutionConfig, deployment appsv1.Deployment) (ctrl.Result, error) {
	// deployment to yaml
	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		config.Log.Error(err, "unable to marshal deployment to yaml", "deployment", deployment.GetName())
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	rawQuery := fmt.Sprintf(prompts.Deployment_prompt, config.Mutation.Spec.TargetConfiguration, yamlData)
	response, err := config.QueryClient.Query(context.Background(), &schemav1.QueryRequest{
		Backend: config.Backend,
		Query:   rawQuery,
	})
	if err != nil {
		config.Log.Error(err, "unable to query server", "deployment", deployment.GetName())
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	// Parse the response into a deployment
	var newDeployment appsv1.Deployment
	if err := yaml.Unmarshal([]byte(response.Response), &newDeployment); err != nil {
		config.Log.Error(err, "unable to unmarshal response to deployment", "deployment", deployment.GetName())
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	// Update the new deployment
	if err := config.Rc.Update(config.Ctx, &newDeployment); err != nil {
		config.Log.Error(err, "unable to updated the deployment", "deployment", deployment.GetName())
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	config.Log.Info("Successfully updated deployment", "deployment", deployment.GetName())
	config.Mutation.Status.Phase = corev1alpha1.AutoRemediationPhaseCompleted
	config.Mutation.Status.Message = "Completed"
	if err := config.Rc.Update(config.Ctx, &config.Mutation); err != nil {
		config.Log.Error(err, "unable to update mutation status")
		return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
	}
	return ctrl.Result{RequeueAfter: util.SuccessfulRequeueTime}, nil
}

func ResourceToExecution(config ObjectExecutionConfig) (ctrl.Result, error) {
	config.Log.Info("ResourceToExecution", "kind", config.Obj.GetObjectKind().GroupVersionKind().Kind)
	switch config.Obj.GetObjectKind().GroupVersionKind().Kind {
	case "Pod":
		var pod corev1.Pod
		err := config.Rc.Get(context.Background(),
			client.ObjectKey{Name: config.Obj.GetName(),
				Namespace: config.Obj.GetNamespace()}, &pod)
		if err != nil {
			config.Log.Error(err, "unable to get pod", "pod", pod.GetName())
			return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
		}
		if len(pod.OwnerReferences) > 0 {
			// Fetch the owner replica set
			var rs appsv1.ReplicaSet
			err := config.Rc.Get(context.Background(),
				client.ObjectKey{Name: pod.OwnerReferences[0].Name,
					Namespace: pod.GetNamespace()}, &rs)
			if err != nil {
				config.Log.Error(err, "unable to get replica set",
					"replica set", pod.OwnerReferences[0].Name)
				return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
			}
			// Get deployment
			var deployment appsv1.Deployment
			err = config.Rc.Get(context.Background(),
				client.ObjectKey{Name: rs.OwnerReferences[0].Name,
					Namespace: rs.GetNamespace()}, &deployment)
			if err != nil {
				config.Log.Error(err, "unable to get deployment", "deployment", rs.OwnerReferences[0].Name)
				return ctrl.Result{RequeueAfter: util.ErrorRequeueTime}, err
			}
			return deploymentExecution(config, deployment)

		}
		return staticPodExecution(config)
	}

	return ctrl.Result{}, nil
}
