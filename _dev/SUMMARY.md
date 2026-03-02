# Distributed Log Service

A Raft-replicated distributed commit log over gRPC, built in Go. Similar in concept to Apache Kafka but built from scratch.

## What It Does

- **Append-only commit log**: clients produce (write) and consume (read) records identified by sequential offsets
- **Raft consensus replication**: writes go through the leader and are replicated to a quorum of followers before being acknowledged
- **Automatic leader election**: if the leader crashes, Raft elects a new one from the remaining nodes
- **Cluster membership via Serf**: nodes discover and monitor each other using gossip protocol; join/leave events automatically add/remove Raft voters
- **Leader-aware client load balancing**: a custom gRPC resolver and picker route writes to the leader and reads to followers (round-robin)
- **Mutual TLS authentication**: client identity is extracted from the TLS certificate's Common Name
- **Casbin authorization**: policy-based access control for produce/consume actions
- **Observability**: structured logging (zap), distributed tracing, and metrics via OpenCensus

## Architecture

```
Single TCP port per node (cmux multiplexed)
  |
  +-- first byte == 0x01 --> Raft transport (StreamLayer)
  |
  +-- everything else ----> gRPC server
                              |
                         authenticate (TLS CN)
                         authorize (Casbin)
                              |
                        DistributedLog
                         /    |    \
                      Raft  BoltDB  Snapshots
                        |
                    local Log
                    (segments: store file + mmap index)
```

## Components

### Storage (`internal/log/`)

| Component | What it does |
|---|---|
| **store** | Append-only file of length-prefixed records with buffered writes |
| **index** | Memory-mapped file of 12-byte entries (4-byte relative offset + 8-byte store position) for O(1) lookups |
| **segment** | Pairs one store + one index; rotates when either hits its size limit |
| **log** | Manages ordered segments; binary-searches by offset to find the right segment |
| **distributed log** | Wraps the local log with Raft consensus for replication |

### Raft (`internal/log/distributed.go`)

- **FSM (Finite State Machine)**: applies committed Raft entries by appending records to the local log
- **logStore**: adapts the local commit log to Raft's `LogStore` interface
- **StreamLayer**: multiplexed TCP transport with optional peer TLS; identified by a magic first byte (`0x01`)
- **Snapshot/Restore**: snapshots serialize the entire log; restores replay it
- **Stable store**: BoltDB for Raft metadata (current term, last vote)

### Discovery (`internal/discovery/membership.go`)

- Uses HashiCorp Serf (gossip protocol) for cluster membership
- Each node tags itself with its `rpc_addr`
- `MemberJoin` events call `DistributedLog.Join()` which calls `raft.AddVoter()`
- `MemberLeave`/`MemberFailed` events call `DistributedLog.Leave()` which calls `raft.RemoveServer()`

### gRPC Server (`internal/server/server.go`)

| RPC | Description |
|---|---|
| `Produce` | Append a record (goes through Raft on the leader) |
| `Consume` | Read a record by offset (local read, no Raft round-trip) |
| `ProduceStream` | Bidirectional streaming: send records, receive offsets |
| `ConsumeStream` | Server streaming: continuously stream records from a given offset |
| `GetServers` | Return all cluster members with addresses and leader status |

Middleware chain: context tags, zap logging, auth interceptor, OpenCensus stats/tracing.

Registers gRPC health check service for Kubernetes probes.

### Load Balancing (`internal/loadbalance/`)

- **Resolver**: calls `GetServers` to discover cluster members; tags each address with `is_leader`
- **Picker**: routes `Produce` calls to the leader, `Consume` calls round-robin across followers

### Agent (`internal/agent/agent.go`)

Top-level orchestrator. Startup order:
1. Open TCP listener, wrap with cmux
2. Create distributed log (Raft + local storage)
3. Create gRPC server with auth middleware
4. Create Serf membership (wired to Raft join/leave)
5. Start serving

Shutdown order: Serf leave, gRPC graceful stop, log close.

### Auth (`internal/auth/` and `internal/config/`)

- **TLS config**: supports separate server TLS (client-facing mTLS) and peer TLS (node-to-node mTLS)
- **Authorizer**: Casbin enforcer with RBAC model; default policy grants `root` identity produce + consume on `*`

## CLI

| Binary | Purpose |
|---|---|
| `cmd/proglog` | The server. Flags: `--data-dir`, `--bind-addr`, `--rpc-port`, `--bootstrap`, `--start-join-addrs`, `--node-name`, TLS and ACL paths. Blocks on SIGINT/SIGTERM. |
| `cmd/getservers` | Diagnostic tool. Connects to a node and prints all cluster members with leader status. |

## Kubernetes Deployment (`deploy/`)

- **StatefulSet** with 3 replicas and PersistentVolumeClaims (1Gi each)
- **Init container** generates per-pod config: pod-0 bootstraps, pods 1+ join pod-0 via stable DNS (`proglog-0.proglog.<ns>.svc.cluster.local`)
- **Headless Service** with `publishNotReadyAddresses: true` for stable pod DNS
- **Metacontroller** creates one LoadBalancer Service per pod so external clients can reach individual nodes (needed for leader-aware routing)
- **Native gRPC health probes** for readiness and liveness
- **ACL ConfigMap** with Casbin model and policy

## Data Flow

**Write path**: client -> picker (to leader) -> gRPC authenticate/authorize -> `DistributedLog.Append()` -> `raft.Apply()` -> replicate to quorum -> FSM applies to local log on each node -> return offset

**Read path**: client -> picker (to follower, round-robin) -> gRPC authenticate/authorize -> `DistributedLog.Read()` -> local log binary search -> segment index lookup -> store file read -> return record
