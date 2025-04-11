# Development Guide for k8sgpt-operator

This document provides instructions for setting up and working with the k8sgpt-operator codebase in Visual Studio Code.

## Prerequisites

- Go 1.22.0 or later
- Visual Studio Code with Go extension
- Docker
- Kubernetes cluster (local or remote)
- kubectl configured for your cluster

## Setup

1. Clone the repository:
   ```
   git clone https://github.com/k8sgpt-ai/k8sgpt-operator.git
   cd k8sgpt-operator
   ```

2. Open the project in VS Code:
   ```
   code .
   ```
   
   Or use the workspace file:
   ```
   code k8sgpt-operator.code-workspace
   ```

3. Install recommended extensions when prompted.

4. Verify Go modules:
   ```
   go mod tidy
   ```

## Building and Testing

### Using VS Code Tasks

VS Code tasks have been configured for common operations. Access them by:
- Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
- Type "Tasks: Run Task"
- Select from available tasks:
  - Build
  - Test
  - Make All
  - Make Build
  - Make Install

### Debugging

Two launch configurations are provided:

1. **Launch Package** - Basic debugging with local mode enabled
2. **Launch with Metrics** - Debugging with metrics and result logging enabled

To start debugging:
- Open cmd/main.go
- Set breakpoints as needed
- Press F5 or select Run > Start Debugging

## Project Structure

- `api/` - Contains API definitions for Custom Resource Definitions (CRDs)
- `cmd/` - Application entry points
- `internal/` - Private application and library code
- `pkg/` - Public library code
- `hack/` - Scripts for development
- `config/` - Configuration files
- `chart/` - Helm chart for deployment

## Development Workflow

1. Make changes to the code
2. Use VS Code tasks to build and test
3. Debug using the provided launch configurations
4. Use `make install` to install CRDs to your cluster
5. Deploy with `helm install` for testing

## Common Issues

### Port Conflicts
If you encounter port conflicts when debugging, you may need to change the ports in the launch configuration.

### Kubernetes Connectivity
Make sure your KUBECONFIG is properly set up before debugging.

## Further Reading

Refer to the main [README.md](README.md) for general information and usage instructions. 
