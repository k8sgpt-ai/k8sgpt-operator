#!/bin/bash

# Default values, can be overridden with environment variables or command-line args
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-k8sgpt-cluster}"
HELM_RELEASE_NAME="${HELM_RELEASE_NAME:-k8sgpt}"
NAMESPACE="${NAMESPACE:-k8sgpt}"
OPENAI_TOKEN="${OPENAI_TOKEN:-}"

# Function to check if a command is installed
check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo "$1 is not installed. Please install $1 and try again."
    exit 1
  fi
}

# Check for required tools
check_tools() {
  check_command kind
  check_command helm
  check_command kubectl
}

# Check for OpenAI token
check_token() {
  if [[ -z "$OPENAI_TOKEN" ]]; then
    echo "OPENAI_TOKEN is not defined. Please provide the token as an argument or set it as an environment variable."
    exit 1
  fi
}

# Function to check if k8sgpt is already deployed
check_k8sgpt_deployed() {
  if kubectl get deployment -n "$NAMESPACE" -l app.kubernetes.io/name=k8sgpt-operator &> /dev/null; then
    echo "k8sgpt-operator is already deployed in namespace $NAMESPACE. Skipping deployment."
    return 0 # Indicate success
  else
    return 1 # Indicate failure
  fi
}

# Function to apply embedded k8s resources
apply_k8s_resources() {
  local k8sgpt_yaml=$(cat <<EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-auto-remediation-sample
spec:
  ai:
    autoRemediation:
      enabled: true
      similarityRequirement: "90"
      resources:
        - Pod
        - Service
        - Deployment
    enabled: true
    model: gpt-4o-mini
    backend: openai
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  remoteCache:
    interplex:
      endpoint: k8sgpt-interplex-service.k8sgpt.svc.cluster.local:8084
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.4.1
EOF
)

  local deployment_yaml=$(cat <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: missing-image-deployment
  labels:
    app: missing-image-deployment
spec:
  selector:
    matchLabels:
      app: missing-image-deployment
  replicas: 1
  template:
    metadata:
      labels:
        app: missing-image-deployment
    spec:
      containers:
        - name: missing-image-container
          image: nginxxx
EOF
)

  echo "Applying K8sGPT resource..."
  echo "$k8sgpt_yaml" | kubectl apply -f - -n "$NAMESPACE"

  echo "Applying deployment with missing image..."
  echo "$deployment_yaml" | kubectl apply -f - -n "$NAMESPACE"
}

# Main script logic
main() {
  check_tools
  check_token

  if check_k8sgpt_deployed; then
    echo "k8sgpt is already set up. Skipping setup."
    exit 0;
  fi

  echo "Creating kind cluster..."
  kind create cluster --name "$KIND_CLUSTER_NAME"

  echo "Adding k8sgpt helm repo..."
  helm repo add k8sgpt https://charts.k8sgpt.ai/
  helm repo update

  echo "Installing k8sgpt-operator helm chart..."
  if helm status "$HELM_RELEASE_NAME" -n "$NAMESPACE" &> /dev/null; then
    echo "k8sgpt-operator is already installed. Skipping installation."
  else
    helm install "$HELM_RELEASE_NAME" k8sgpt/k8sgpt-operator -n "$NAMESPACE" --create-namespace --set interplex.enabled=true
  fi

  echo "Creating secret..."
  if kubectl get secret k8sgpt-sample-secret -n "$NAMESPACE" -o jsonpath="{.data}" &> /dev/null; then
    echo "Secret k8sgpt-sample-secret already exists in namespace $NAMESPACE. Skipping creation."
  else
    kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key="$OPENAI_TOKEN" -n "$NAMESPACE"
  fi

  apply_k8s_resources

  echo "k8sgpt local setup complete."
}

# Execute the main function, passing any command-line arguments.
main "$@"
