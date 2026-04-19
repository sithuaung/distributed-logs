# `agent.go`

Defines the `Agent` — one full node of the cluster — and the `Config` used to build one.

## `Config`

Fields you pass in when creating a node:

| field | meaning |
|---|---|
| `ServerTLSConfig`, `PeerTLSConfig` | mTLS configs. Server cert is for clients talking in; peer cert is for Raft talking node-to-node. |
| `DataDir` | where the log and Raft state live on disk. |
| `BindAddr` | host:port Serf listens on (gossip). |
| `RPCPort` | port for client gRPC **and** Raft. Both are multiplexed here. |
| `NodeName` | unique id for this node (also used as Raft `ServerID`). |
| `StartJoinAddrs` | Serf addresses of existing nodes to contact on startup. Empty = bootstrap node. |
| `ACLModelFile`, `ACLPolicyFile` | Casbin model + policy files for auth. |
| `Bootstrap` | true only for the first node — it becomes the initial Raft leader. |

`Config.RPCAddr()` returns `host:RPCPort` derived from `BindAddr`.

## `Agent`

```go
type Agent struct {
    Config Config
    mux        cmux.CMux           // multiplex Raft + gRPC on one port
    log        *log.DistributedLog // Raft-replicated log
    server     *grpc.Server        // public API
    membership *discovery.Membership
    // + shutdown plumbing
}
```

## `New(config) (*Agent, error)`

Runs four setup steps in order and then calls `go a.serve()`.

### `setupMux`
Opens a TCP listener at `RPCAddr` and wraps it in `cmux.New`. cmux lets us decide which incoming connection is Raft vs gRPC without needing two ports.

### `setupLog`
Tells cmux: "if the first byte of a connection is `log.RaftRPC` (`1`), route it here." Those connections become the Raft transport. Builds a `log.Config` with a `StreamLayer` over that listener, then constructs the `DistributedLog`. If this node is bootstrapping, it blocks up to 3 s waiting to become the leader.

### `setupServer`
Builds a Casbin `Authorizer`, then a gRPC server (with TLS if configured). `a.mux.Match(cmux.Any())` catches whatever wasn't Raft and hands it to the gRPC server in a goroutine. Crucially, the server's `GetServerer` is the `DistributedLog` itself — so `GetServers` returns the live Raft membership.

### `setupMembership`
Creates a Serf `Membership` whose handler is the `DistributedLog`. Every join tells the leader to `AddVoter`; every leave tells it to `RemoveServer`. Tags carry the peer's `rpc_addr` so others know where to send Raft traffic.

### `serve`
Blocks on `mux.Serve()`. If cmux dies, the agent shuts itself down.

## `Shutdown`

Idempotent (guarded by `shutdownLock` + `shutdown` bool). Order:

1. `membership.Leave` — stop gossiping so peers notice us leaving.
2. `server.GracefulStop` — drain in-flight RPCs.
3. `log.Close` — shuts down Raft and closes segment files.

## Key takeaway

`agent.go` is pure glue. The interesting code lives in the packages it imports; this file's job is to pass the right things to the right constructors in the right order.
