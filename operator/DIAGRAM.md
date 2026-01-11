# Buildbarn Operator - Visual Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  buildbarn-system Namespace                              │   │
│  │  ┌────────────────────────────────────────────────────┐ │   │
│  │  │  Operator Manager (controller-runtime)              │ │   │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐          │ │   │
│  │  │  │Scheduler │ │ Worker   │ │ Storage  │ ...  │ │   │
│  │  │  │Controller│ │Controller│ │Controller│      │ │   │
│  │  │  └──────────┘ └──────────┘ └──────────┘      │ │   │
│  │  └────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  buildbarn Namespace (Application)                        │   │
│  │                                                            │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │Buildbarn     │  │Buildbarn     │  │Buildbarn     │   │   │
│  │  │Scheduler CR  │  │Worker CR     │  │Storage CR    │   │   │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │   │
│  │         │                  │                  │           │   │
│  │         ▼                  ▼                  ▼           │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │Deployment     │  │Deployment     │  │StatefulSet   │   │   │
│  │  │Service        │  │(with sidecars)│  │Service       │   │   │
│  │  │Ingress        │  │               │  │PVCs          │   │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │   │
│  │                                                            │   │
│  │  ┌──────────────┐  ┌──────────────┐                      │   │
│  │  │Buildbarn     │  │Buildbarn     │                      │   │
│  │  │Frontend CR    │  │Browser CR    │                      │   │
│  │  └──────┬───────┘  └──────┬───────┘                      │   │
│  │         │                  │                              │   │
│  │         ▼                  ▼                              │   │
│  │  ┌──────────────┐  ┌──────────────┐                      │   │
│  │  │Deployment     │  │Deployment     │                      │   │
│  │  │Service        │  │Service        │                      │   │
│  │  │(LoadBalancer) │  │Ingress        │                      │   │
│  │  └──────────────┘  └──────────────┘                      │   │
│  │                                                            │   │
│  │  ┌──────────────────────────────────────┐                 │   │
│  │  │  ConfigMap: buildbarn-config         │                 │   │
│  │  │  (Mounted by all components)         │                 │   │
│  │  └──────────────────────────────────────┘                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Controller Reconciliation Flow

```
User Action
    │
    ▼
┌─────────────────┐
│ Create/Update   │
│ Custom Resource │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Kubernetes API  │
│ (CR Stored)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Controller      │
│ Watch Event     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Reconcile Loop  │
│ 1. Fetch CR     │
│ 2. Check Finalizer│
│ 3. Compare State │
└────────┬────────┘
         │
         ├─────────────────┐
         │                 │
         ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ Resource     │  │ Resource     │
│ Exists?      │  │ Missing?     │
└──────┬───────┘  └──────┬───────┘
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ Update      │  │ Create       │
│ Resource    │  │ Resource     │
└──────┬───────┘  └──────┬───────┘
       │                 │
       └────────┬────────┘
                │
                ▼
       ┌──────────────┐
       │ Update CR    │
       │ Status       │
       └──────────────┘
```

## Component Details

### BuildbarnScheduler
```
BuildbarnScheduler CR
    │
    ├──> Deployment (scheduler)
    │       └──> Pod
    │             ├──> Container: bb-scheduler
    │             │     ├──> Port 8982 (client-grpc)
    │             │     ├──> Port 8983 (worker-grpc)
    │             │     └──> Port 7982 (http)
    │             └──> Volume: configs (ConfigMap)
    │
    ├──> Service (ClusterIP)
    │       ├──> Port 8982
    │       ├──> Port 8983
    │       └──> Port 7982
    │
    └──> Ingress
            └──> Host: bb-scheduler.{namespace}
```

### BuildbarnWorker
```
BuildbarnWorker CR
    │
    └──> Deployment (worker)
            └──> Pod
                  ├──> InitContainer: bb-runner-installer
                  ├──> InitContainer: volume-init
                  ├──> Container: worker (bb-worker)
                  │     ├──> Env: NODE_NAME, POD_NAME
                  │     └──> Volume: configs, worker
                  └──> Container: runner (sidecar)
                        ├──> SecurityContext: runAsUser 65534
                        └──> Volume: configs, worker, empty
```

### BuildbarnStorage
```
BuildbarnStorage CR
    │
    ├──> StatefulSet (storage)
    │       └──> Pod (replicated)
    │             ├──> InitContainer: volume-init
    │             ├──> Container: storage (bb-storage)
    │             │     └──> Port 8981 (grpc)
    │             ├──> Volume: configs (ConfigMap)
    │             ├──> Volume: cas (PVC - 33Gi)
    │             └──> Volume: ac (PVC - 1Gi)
    │
    └──> Service (Headless)
            └──> Port 8981
```

### BuildbarnFrontend
```
BuildbarnFrontend CR
    │
    ├──> Deployment (frontend)
    │       └──> Pod (replicated)
    │             └──> Container: storage (bb-storage)
    │                   ├──> Port 8980 (grpc)
    │                   └──> Volume: configs (ConfigMap)
    │
    └──> Service (LoadBalancer)
            └──> Port 8980
```

### BuildbarnBrowser
```
BuildbarnBrowser CR
    │
    ├──> Deployment (browser)
    │       └──> Pod (replicated)
    │             └──> Container: browser (bb-browser)
    │                   ├──> Port 7984 (http)
    │                   └──> Volume: configs (ConfigMap)
    │
    ├──> Service (ClusterIP)
    │       └──> Port 7984
    │
    └──> Ingress (optional)
            └──> Host: {IngressHost} or bb-browser.{namespace}
```

## Data Flow

```
┌─────────────┐
│   User      │
│  (kubectl)  │
└──────┬──────┘
       │
       │ kubectl apply -f scheduler.yaml
       ▼
┌─────────────────┐
│ BuildbarnScheduler│
│      CR          │
└──────┬───────────┘
       │
       │ Controller watches
       ▼
┌─────────────────┐
│ Scheduler       │
│ Controller      │
└──────┬───────────┘
       │
       │ Creates/Updates
       ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Deployment    │      │    Service      │      │    Ingress     │
│  (scheduler)    │      │  (scheduler)   │      │  (scheduler)   │
└─────────────────┘      └─────────────────┘      └─────────────────┘
       │
       │ Pods run
       ▼
┌─────────────────┐
│  Buildbarn      │
│  Scheduler      │
│  (Running)      │
└─────────────────┘
```

## Status Flow

```
CR Status Fields:
├── State (Ready/NotReady/NotFound)
├── Replicas (total)
├── ReadyReplicas (ready)
└── Conditions (array)

Controller Updates Status:
1. Fetches underlying resource (Deployment/StatefulSet)
2. Reads resource status
3. Compares replicas vs readyReplicas
4. Sets State accordingly
5. Updates CR status subresource
```
