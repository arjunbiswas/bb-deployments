# Buildbarn Operator Architecture

## Overview

This document describes the architecture, flow, and hierarchy of the Buildbarn Kubernetes Operator.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "Operator Namespace (buildbarn-system)"
            Manager[Operator Manager<br/>controller-runtime]
            Manager -->|Manages| SchedulerCtrl[BuildbarnScheduler Controller]
            Manager -->|Manages| WorkerCtrl[BuildbarnWorker Controller]
            Manager -->|Manages| StorageCtrl[BuildbarnStorage Controller]
            Manager -->|Manages| FrontendCtrl[BuildbarnFrontend Controller]
            Manager -->|Manages| BrowserCtrl[BuildbarnBrowser Controller]
        end

        subgraph "Application Namespace (buildbarn)"
            subgraph "Custom Resources"
                SchedulerCR[BuildbarnScheduler CR]
                WorkerCR[BuildbarnWorker CR]
                StorageCR[BuildbarnStorage CR]
                FrontendCR[BuildbarnFrontend CR]
                BrowserCR[BuildbarnBrowser CR]
            end

            subgraph "Kubernetes Resources"
                SchedulerDeploy[Deployment: scheduler]
                SchedulerSvc[Service: scheduler]
                SchedulerIng[Ingress: scheduler]
                
                WorkerDeploy[Deployment: worker]
                
                StorageSTS[StatefulSet: storage]
                StorageSvc[Service: storage]
                
                FrontendDeploy[Deployment: frontend]
                FrontendSvc[Service: frontend]
                
                BrowserDeploy[Deployment: browser]
                BrowserSvc[Service: browser]
                BrowserIng[Ingress: browser]
            end

            ConfigMap[ConfigMap: buildbarn-config]
        end
    end

    SchedulerCtrl -->|Watches & Reconciles| SchedulerCR
    SchedulerCR -->|Creates| SchedulerDeploy
    SchedulerCR -->|Creates| SchedulerSvc
    SchedulerCR -->|Creates| SchedulerIng
    SchedulerDeploy -->|Mounts| ConfigMap

    WorkerCtrl -->|Watches & Reconciles| WorkerCR
    WorkerCR -->|Creates| WorkerDeploy
    WorkerDeploy -->|Mounts| ConfigMap

    StorageCtrl -->|Watches & Reconciles| StorageCR
    StorageCR -->|Creates| StorageSTS
    StorageCR -->|Creates| StorageSvc
    StorageSTS -->|Mounts| ConfigMap
    StorageSTS -->|Uses| PVC[PersistentVolumeClaims]

    FrontendCtrl -->|Watches & Reconciles| FrontendCR
    FrontendCR -->|Creates| FrontendDeploy
    FrontendCR -->|Creates| FrontendSvc
    FrontendDeploy -->|Mounts| ConfigMap

    BrowserCtrl -->|Watches & Reconciles| BrowserCR
    BrowserCR -->|Creates| BrowserDeploy
    BrowserCR -->|Creates| BrowserSvc
    BrowserCR -->|Creates| BrowserIng
    BrowserDeploy -->|Mounts| ConfigMap

    style Manager fill:#4a90e2,stroke:#2e5c8a,color:#fff
    style SchedulerCtrl fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style WorkerCtrl fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style StorageCtrl fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style FrontendCtrl fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style BrowserCtrl fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style SchedulerCR fill:#ffd700,stroke:#ccaa00,color:#000
    style WorkerCR fill:#ffd700,stroke:#ccaa00,color:#000
    style StorageCR fill:#ffd700,stroke:#ccaa00,color:#000
    style FrontendCR fill:#ffd700,stroke:#ccaa00,color:#000
    style BrowserCR fill:#ffd700,stroke:#ccaa00,color:#000
```

## Reconciliation Flow

```mermaid
sequenceDiagram
    participant User
    participant API as Kubernetes API
    participant Controller
    participant CR as Custom Resource
    participant K8sRes as Kubernetes Resources

    User->>API: Create/Update BuildbarnScheduler CR
    API->>Controller: Watch Event (Add/Update)
    Controller->>API: Fetch CR
    Controller->>Controller: Check Finalizer
    Controller->>Controller: Reconcile Logic
    
    alt Resource Exists
        Controller->>API: Get Deployment/Service/Ingress
        API-->>Controller: Return Resource
        Controller->>Controller: Compare Spec vs Status
        alt Needs Update
            Controller->>API: Update Resource
        end
    else Resource Missing
        Controller->>API: Create Deployment
        Controller->>API: Create Service
        Controller->>API: Create Ingress
    end
    
    Controller->>API: Get Resource Status
    API-->>Controller: Return Status
    Controller->>API: Update CR Status
    
    Note over Controller,K8sRes: Controller watches owned resources<br/>for changes and reconciles
```

## Component Hierarchy

```mermaid
graph TD
    subgraph "Operator Layer"
        A[Operator Manager<br/>main.go]
    end

    subgraph "Controller Layer"
        B1[BuildbarnScheduler<br/>Reconciler]
        B2[BuildbarnWorker<br/>Reconciler]
        B3[BuildbarnStorage<br/>Reconciler]
        B4[BuildbarnFrontend<br/>Reconciler]
        B5[BuildbarnBrowser<br/>Reconciler]
    end

    subgraph "API Layer (CRDs)"
        C1[BuildbarnScheduler<br/>v1alpha1]
        C2[BuildbarnWorker<br/>v1alpha1]
        C3[BuildbarnStorage<br/>v1alpha1]
        C4[BuildbarnFrontend<br/>v1alpha1]
        C5[BuildbarnBrowser<br/>v1alpha1]
    end

    subgraph "Kubernetes Resources"
        D1[Deployment<br/>Service<br/>Ingress]
        D2[Deployment]
        D3[StatefulSet<br/>Service<br/>PVCs]
        D4[Deployment<br/>Service]
        D5[Deployment<br/>Service<br/>Ingress]
    end

    A --> B1
    A --> B2
    A --> B3
    A --> B4
    A --> B5

    B1 -->|Reconciles| C1
    B2 -->|Reconciles| C2
    B3 -->|Reconciles| C3
    B4 -->|Reconciles| C4
    B5 -->|Reconciles| C5

    C1 -->|Creates| D1
    C2 -->|Creates| D2
    C3 -->|Creates| D3
    C4 -->|Creates| D4
    C5 -->|Creates| D5

    style A fill:#4a90e2,stroke:#2e5c8a,color:#fff
    style B1 fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style B2 fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style B3 fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style B4 fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style B5 fill:#7b68ee,stroke:#5a4fcf,color:#fff
    style C1 fill:#ffd700,stroke:#ccaa00,color:#000
    style C2 fill:#ffd700,stroke:#ccaa00,color:#000
    style C3 fill:#ffd700,stroke:#ccaa00,color:#000
    style C4 fill:#ffd700,stroke:#ccaa00,color:#000
    style C5 fill:#ffd700,stroke:#ccaa00,color:#000
```

## Controller Responsibilities

### BuildbarnScheduler Controller
- **Manages**: Deployment, Service, Ingress
- **Ports**: 8982 (client-grpc), 8983 (worker-grpc), 7982 (http)
- **Default Replicas**: 1
- **Service Type**: ClusterIP

### BuildbarnWorker Controller
- **Manages**: Deployment (with init containers and sidecars)
- **Components**: 
  - Worker container (bb-worker)
  - Runner container (runner sidecar)
  - Runner installer init container
  - Volume init container
- **Default Replicas**: 8
- **Instance**: Supports multiple instances (e.g., ubuntu22-04)

### BuildbarnStorage Controller
- **Manages**: StatefulSet, Service, PersistentVolumeClaims
- **Ports**: 8981 (grpc)
- **Default Replicas**: 2
- **Storage**: 
  - CAS: 33Gi (default)
  - AC: 1Gi (default)
- **Service Type**: Headless (ClusterIP: None)

### BuildbarnFrontend Controller
- **Manages**: Deployment, Service
- **Ports**: 8980 (grpc)
- **Default Replicas**: 3
- **Service Type**: LoadBalancer

### BuildbarnBrowser Controller
- **Manages**: Deployment, Service, Ingress
- **Ports**: 7984 (http)
- **Default Replicas**: 3
- **Service Type**: ClusterIP
- **Ingress**: Optional (created if IngressHost specified)

## Lifecycle Management

```mermaid
stateDiagram-v2
    [*] --> Creating: CR Created
    Creating --> Reconciling: Finalizer Added
    Reconciling --> Ready: Resources Created & Healthy
    Reconciling --> NotReady: Resources Unhealthy
    NotReady --> Reconciling: Retry
    Ready --> Reconciling: Spec Changed
    Reconciling --> Deleting: CR Deleted
    Deleting --> [*]: Finalizer Removed
```

## Key Features

1. **Finalizers**: All controllers use finalizers to ensure proper cleanup
2. **Owner References**: All created resources have owner references to the CR
3. **Status Updates**: Controllers update CR status based on underlying resource states
4. **Watch**: Controllers watch both CRs and owned resources for changes
5. **Reconciliation**: Continuous reconciliation loop ensures desired state

## Resource Dependencies

```
ConfigMap (buildbarn-config)
    ├── Common configuration (common.libsonnet)
    ├── Scheduler configuration (scheduler.jsonnet)
    ├── Worker configuration (worker-{instance}.jsonnet)
    ├── Runner configuration (runner-{instance}.jsonnet)
    ├── Storage configuration (storage.jsonnet)
    ├── Frontend configuration (frontend.jsonnet)
    └── Browser configuration (browser.jsonnet)
```

All components mount the ConfigMap to `/config/` directory in their containers.
