# `membership.go`

Wraps `hashicorp/serf` into a small `Membership` type that calls a `Handler` on join/leave.

## Types

```go
type Config struct {
    NodeName       string            // unique per-node id
    BindAddr       string            // host:port Serf listens on
    Tags           map[string]string // gossiped with every member — we stash "rpc_addr" here
    StartJoinAddrs []string          // other Serf addresses to contact on startup
}

type Handler interface {
    Join(name, addr string) error
    Leave(name string) error
}

type Membership struct {
    Config
    handler Handler
    serf    *serf.Serf
    events  chan serf.Event
    logger  *zap.Logger
}
```

## `New(handler, config) (*Membership, error)`

Builds the `Membership`, then calls `setupSerf`.

## `setupSerf`

1. Resolve the `BindAddr` to get IP + port for Serf's memberlist.
2. Build a default Serf config, plug in our `events` channel, `Tags`, and `NodeName`.
3. `serf.Create(config)` starts the gossip agent.
4. Launch `eventHandler` goroutine (see below).
5. If `StartJoinAddrs` is non-empty, call `serf.Join(...)` to contact an existing cluster.

## `eventHandler`

Long-running loop that reads from the `events` channel:

- `EventMemberJoin` → call `handleJoin` for each member (skipping ourselves via `isLocal`).
- `EventMemberLeave` / `EventMemberFailed` → call `handleLeave`.

### `handleJoin(member)`

Pulls the `rpc_addr` tag out of the member's gossip tags and calls `handler.Join(name, rpcAddr)`. That tag is what the leader needs — Serf knows the gossip address, but Raft talks on the RPC port, so we smuggle it through tags.

### `handleLeave(member)`

Calls `handler.Leave(name)`.

### `logError`

Logs join/leave errors. Downgrades `raft.ErrNotLeader` to Debug — it's expected for non-leader nodes to get that error when the handler tries to mutate Raft membership, and we don't want log spam.

## `Members()` / `Leave()`

Pass-throughs to Serf. `Leave()` is called during agent shutdown so peers see us go.

## Subtle bit: the `return` on leave

Inside the `EventMemberLeave` branch, the loop does `return` (not `continue`) when the leaving member is **ourselves** — once we've left, the goroutine exits. This is the clean shutdown path.
