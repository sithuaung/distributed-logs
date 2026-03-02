# Running Locally

Guide for running the Proglog distributed log service locally using Colima and Kubernetes.

## Prerequisites

### 1. Install Go

Go 1.25+ is required.

```bash
brew install go
```

### 2. Install Protocol Buffers Compiler

```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 3. Install CFSSL (TLS Certificate Generation)

```bash
brew install cfssl
```

### 4. Install Colima and Docker CLI

```bash
brew install colima docker
```

### 5. Install Kubernetes Tools

```bash
brew install kubectl helm
```

## Setup Steps

### Step 1: Start Colima with Kubernetes

```bash
colima start --kubernetes --cpu 4 --memory 8
```

Verify the cluster is running:

```bash
kubectl cluster-info
kubectl get nodes
```

### Step 2: Initialize Config Directory

```bash
make init
```

This creates `~/.proglog/` where certificates and config files will be stored.

### Step 3: Generate TLS Certificates

```bash
make gencert
```

This generates CA, server, and client certificates and copies them along with ACL policy files to `~/.proglog/`.

### Step 4: Compile Protocol Buffers (if needed)

```bash
make compile
```

### Step 5: Run Tests

```bash
make test
```

### Step 6: Build the Docker Image

```bash
make build-docker
```

This builds the image `github.com/igor-baiborodine/proglog:0.0.1`.

Since Colima uses its own Docker daemon, the image is already available to the Kubernetes cluster.

### Step 7: Deploy to Kubernetes

Install the Metacontroller (handles per-pod service creation):

```bash
helm install metacontroller deploy/metacontroller
```

Install Proglog:

```bash
helm install proglog deploy/proglog
```

### Step 8: Verify Deployment

```bash
kubectl get pods -w
```

Wait until all 3 pods (`proglog-0`, `proglog-1`, `proglog-2`) are running and ready.

```bash
kubectl get services
```

## Using the Service

### List Servers

Port-forward the gRPC service:

```bash
kubectl port-forward pod/proglog-0 8400:8400
```

In another terminal:

```bash
go run cmd/getservers/main.go
```

## Useful Commands

| Command | Description |
|---------|-------------|
| `colima start --kubernetes` | Start Colima with k8s |
| `colima stop` | Stop Colima |
| `colima status` | Check Colima status |
| `kubectl logs proglog-0` | View pod logs |
| `kubectl describe pod proglog-0` | Debug pod issues |
| `helm uninstall proglog` | Remove Proglog deployment |
| `helm uninstall metacontroller` | Remove Metacontroller |
| `make clean` | Remove `~/.proglog/` config directory |

## Troubleshooting

- **Pods stuck in `ImagePullBackOff`**: The Docker image must be built inside Colima's Docker daemon. Make sure Colima is running before running `make build-docker`.
- **Certificate errors**: Run `make clean && make init && make gencert` to regenerate certificates.
- **Colima not starting**: Try `colima delete` then start fresh with `colima start --kubernetes`.


kubectl delete pod proglog-0 --force --grace-period=0 
kubectl delete statefulset proglog                    
helm uninstall proglog
kubectl delete pvc datadir-proglog-0
helm install proglog deploy/proglog
kubectl delete configmap proglog-acl
kubectl get all,configmap,secret,serviceaccount -l app.kubernetes.io/name=proglog

kubectl delete statefulset proglog --force --grace-period=0

helm install proglog deploy/proglog

kubectl patch statefulset proglog -p '{"metadata":{"finalizers":[]}}' --type=merge         
kubectl delete statefulset proglog                    
kubectl delete service proglog-0 proglog-1 proglog-2  
kubectl delete configmap proglog-acl --ignore-not-found                                    
                                                        
make build-docker
kubectl rollout restart statefulset/proglog
helm install proglog deploy/proglog

helm upgrade proglog deploy/proglog                   

helm uninstall proglog && helm install proglog deploy/proglog

kubectl scale statefulset proglog --replicas=3
