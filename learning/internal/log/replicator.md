# `replicator.go`

An **older** replication strategy, from before the project adopted Raft. Not wired into the current agent.

## Idea

Instead of using Raft consensus, a follower simply opens a `ConsumeStream` to the leader and `Produce`s every record it receives back into its own log. Gossip (Serf) tells it about peers; when a peer joins, it calls `Replicator.Join` which spawns a replication goroutine.

## `Replicator` type

```go
type Replicator struct {
    DialOptions []grpc.DialOption    // how to dial peers
    LocalServer api.LogClient        // our own log to Produce into

    logger *zap.Logger

    mu      sync.Mutex
    servers map[string]chan struct{} // peer name → cancel channel
    closed  bool
    close   chan struct{}
}
```

## `Join(name, addr)`

Idempotent — if we're already replicating from `name`, do nothing. Otherwise create a cancel channel, store it, and `go r.replicate(addr, leave)`.

## `replicate(addr, leave)`

1. `grpc.Dial(addr, ...)`.
2. `client.ConsumeStream(ctx, {Offset: 0})` — start reading from the very beginning.
3. Spawn a goroutine that reads `stream.Recv()` in a loop and pushes onto a `records` channel.
4. Main loop:
   - `<-r.close` → global shutdown → return.
   - `<-leave` → this peer left → return.
   - `<-records` → got a record → call `LocalServer.Produce` to insert it locally.

Bugs in this simple design (not fixed here):
- Always restarts from offset 0, so lots of duplicate work on restart.
- No leader election — every follower replicates from every other peer → N² traffic.
- A split-brain produces duplicate records with no way to resolve.

Raft replaces all of that.

## `Leave(name)`

Closes the cancel channel and deletes the entry from `servers`.

## `init()` / `Close()`

Lazy init (`servers`, `close`, `logger` are set on first use). `Close` sets `closed = true` and closes `r.close`, which knocks every `replicate` goroutine out of its select.

## Why keep it?

Probably so older chapters of the learning material still compile, or as a fallback reference. Safe to ignore when reasoning about the current system.
