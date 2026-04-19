# How Membership Discovery Works

This explains `internal/discovery/membership.go` — the service discovery layer that uses [HashiCorp Serf](https://www.serf.io/) to manage cluster membership via a gossip protocol.

## Overview

The `Membership` struct is responsible for:
1. Creating/joining a Serf cluster
2. Listening for membership events (nodes joining or leaving)
3. Delegating those events to a `Handler` (e.g., Raft consensus layer)

```
Node A starts ──► Serf cluster created
Node B starts ──► Joins Node A via StartJoinAddrs
                  ──► Serf gossips the join event to all nodes
                  ──► Each node's eventHandler() fires handler.Join()
Node B leaves ──► Serf gossips the leave event
                  ──► Each node's eventHandler() fires handler.Leave()
```

## Key Components

### Membership struct (line 12-18)

| Field     | Purpose |
|-----------|---------|
| `Config`  | Embedded config: node name, bind address, tags, seed addresses |
| `handler` | Callback interface to notify the application layer of join/leave events |
| `serf`    | The Serf agent — control handle for the cluster |
| `events`  | Channel where Serf pushes membership events |
| `logger`  | Structured logger |

### Handler interface (line 66-69)

```go
type Handler interface {
    Join(name, addr string) error
    Leave(name string) error
}
```

This is how Membership talks to the rest of the application. Whatever implements this interface (typically the Raft/replication layer) gets notified when nodes join or leave the cluster.

## Lifecycle: Step by Step

### 1. New() — Creating a Membership instance (line 20-30)

```go
func New(handler Handler, config Config) (*Membership, error)
```

- Stores the config and handler
- Calls `setupSerf()` to initialize the cluster

### 2. setupSerf() — Initializing Serf (line 39-64)

This is where the real setup happens:

1. **Resolve bind address** (line 40) — Parses the TCP address (e.g., `127.0.0.1:8401`) into IP and port.

2. **Configure Serf** (line 44-51):
   - Sets the bind IP and port for gossip communication
   - Creates an event channel and assigns it to `config.EventCh` — this tells Serf "push all events into this channel"
   - Sets node tags (e.g., `rpc_addr`) so other nodes know how to reach this node's RPC server
   - Sets the node name for identification

3. **Create the Serf agent** (line 52) — `serf.Create(config)` starts the Serf agent, which begins listening for gossip traffic.

4. **Start the event loop** (line 56) — `go m.eventHandler()` launches a background goroutine that continuously reads from the events channel.

5. **Join existing cluster** (line 57-62) — If `StartJoinAddrs` is provided, the node contacts those addresses to join an existing cluster. If `nil`, this is the first node (bootstrap) and it skips this step.

### 3. eventHandler() — The Event Loop (line 71-90)

```go
func (m *Membership) eventHandler() {
    for e := range m.events {
        switch e.EventType() {
        case serf.EventMemberJoin:
            // ...
        case serf.EventMemberLeave, serf.EventMemberFailed:
            // ...
        }
    }
}
```

Runs forever in a goroutine, reading events from the channel:

- **EventMemberJoin** — A new node joined. For each member in the event, call `handler.Join()` (skipping the local node to avoid self-joining).
- **EventMemberLeave / EventMemberFailed** — A node left gracefully or was detected as failed. Call `handler.Leave()` for each (if the local node left, stop processing).

### 4. isLocal() — Self-check (line 109-111)

Compares the event member's name with the local Serf member's name. This prevents a node from trying to join/remove itself, which would cause issues in the Raft layer.

### 5. Members() and Leave() — Cluster operations (line 113-119)

- `Members()` returns all known members in the cluster (used for health checks, debugging)
- `Leave()` gracefully removes this node from the cluster, triggering `EventMemberLeave` on other nodes

### 6. logError() — Smart error logging (line 121-132)

Logs errors from `handler.Join()` / `handler.Leave()`. If the error is `raft.ErrNotLeader`, it logs at Debug level instead of Error — because only the Raft leader can add/remove servers, so non-leader nodes getting this error is expected and not a real problem.

## How It Fits Together

```
┌─────────────────────────────────────────────┐
│                  Node                       │
│                                             │
│  ┌─────────────┐       ┌─────────────────┐  │
│  │  Membership  │──────►│  Handler (Raft) │  │
│  │  (discovery) │ Join/ │  Add/remove     │  │
│  │              │ Leave │  servers         │  │
│  └──────┬───────┘       └─────────────────┘  │
│         │                                    │
│  ┌──────▼───────┐                            │
│  │  Serf Agent   │◄── gossip protocol ──► other nodes
│  │  (gossip)     │                            │
│  └───────────────┘                            │
└─────────────────────────────────────────────┘
```

1. **Serf** handles the low-level gossip protocol — detecting when nodes appear or disappear in the network
2. **Membership** translates those raw Serf events into application-level `Join`/`Leave` calls
3. **Handler** (implemented by the Raft/replication layer) reacts by adding or removing servers from the consensus group

This separation means the discovery mechanism (Serf/gossip) is decoupled from the consensus mechanism (Raft). You could swap Serf for a different discovery method without changing the Raft layer.
