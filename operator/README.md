# Buildbarn Kubernetes Operator

This is a Kubernetes operator for managing Buildbarn components using Custom Resource Definitions (CRDs). The operator is built using [Kubebuilder](https://book.kubebuilder.io/) and follows the controller-runtime pattern.

## Overview

The operator manages the following Buildbarn components:

- **BuildbarnScheduler**: Manages the Buildbarn scheduler deployment
- **BuildbarnWorker**: Manages Buildbarn worker deployments with runner sidecars
- **BuildbarnStorage**: Manages Buildbarn storage StatefulSets with persistent volumes
- **BuildbarnFrontend**: Manages Buildbarn frontend deployments
- **BuildbarnBrowser**: Manages the Buildbarn browser deployment

## Prerequisites

- Go 1.21 or later
- Kubernetes cluster (v1.25+)
- kubectl configured to access your cluster
- [kustomize](https://kustomize.io/) (v5.0.0+)
- [controller-gen](https://github.com/kubernetes-sigs/controller-tools) (v0.13.0+)

## Installation

### 1. Install CRDs

```bash
make install
```

### 2. Deploy the Operator

```bash
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/buildbarn-operator:latest

# Deploy the operator
make deploy IMG=<your-registry>/buildbarn-operator:latest
```

### 3. Verify Installation

```bash
kubectl get pods -n buildbarn-system
```

## Usage

### Example: Deploy a Buildbarn Scheduler

```yaml
apiVersion: buildbarn.io/v1alpha1
kind: BuildbarnScheduler
metadata:
  name: scheduler
  namespace: buildbarn
spec:
  image: ghcr.io/buildbarn/bb-scheduler:latest
  replicas: 1
  configMapName: buildbarn-config
  serviceType: ClusterIP
```

### Example: Deploy a Buildbarn Worker

```yaml
apiVersion: buildbarn.io/v1alpha1
kind: BuildbarnWorker
metadata:
  name: worker-ubuntu22-04
  namespace: buildbarn
spec:
  image: ghcr.io/buildbarn/bb-worker:latest
  runnerImage: ghcr.io/catthehacker/ubuntu:act-22.04
  runnerInstallerImage: ghcr.io/buildbarn/bb-runner-installer:latest
  replicas: 8
  instance: ubuntu22-04
  configMapName: buildbarn-config
```

### Example: Deploy Buildbarn Storage

```yaml
apiVersion: buildbarn.io/v1alpha1
kind: BuildbarnStorage
metadata:
  name: storage
  namespace: buildbarn
spec:
  image: ghcr.io/buildbarn/bb-storage:latest
  replicas: 2
  configMapName: buildbarn-config
```

## Development

### Running Locally

```bash
# Install CRDs
make install

# Run the operator locally
make run
```

### Building

```bash
# Build the manager binary
make build

# Build the Docker image
make docker-build IMG=buildbarn-operator:latest
```

### Testing

```bash
# Run tests
make test
```

### Generating Manifests

```bash
# Generate CRD manifests
make manifests
```

## Project Structure

```
operator/
├── api/                    # API definitions
│   └── v1alpha1/           # v1alpha1 API version
├── config/                 # Kustomize configs
│   ├── crd/               # CRD definitions
│   ├── default/           # Default deployment config
│   ├── manager/           # Manager deployment
│   └── rbac/              # RBAC configs
├── controllers/           # Controller implementations
├── hack/                  # Helper scripts
└── main.go                # Entry point
```

## Uninstallation

```bash
# Remove the operator
make undeploy

# Remove CRDs
make uninstall
```

## License

Licensed under the Apache License, Version 2.0.
