# `distributed.go`

The big one. Wraps `Log` with **Hashicorp Raft** so the log is replicated across nodes.

## `DistributedLog` type

```go
type DistributedLog struct {
    config Config
    log    *Log       // local, on-disk commit log
    raft   *raft.Raft // Raft consensus engine
}
```

## `NewDistributedLog(dataDir, config)`

1. `setupLog(dataDir)` — builds the user-data `Log` at `dataDir/log`.
2. `setupRaft(dataDir)` — builds the Raft engine.

## `setupRaft(dataDir)`

Assembles five things Raft needs:

1. **FSM** — `&fsm{log: l.log}`. Raft calls `fsm.Apply(...)` for each committed log entry.
2. **LogStore** — `newLogStore(dataDir/raft/log, config)`. This is a second `Log` (ours!) used to store Raft's own log entries. `logConfig.Segment.InitialOffset = 1` because Raft indices start at 1.
3. **StableStore** — `raftboltdb.NewBoltStore(dataDir/raft/stable)`. Raft persists metadata (current term, voted-for) here. BoltDB is used because it gives strong fsync guarantees for tiny key-value data.
4. **SnapshotStore** — `raft.NewFileSnapshotStore(dataDir/raft, retain=1, ...)`. Keeps the most recent 1 snapshot on disk.
5. **Transport** — `raft.NewNetworkTransport(StreamLayer, maxPool=5, timeout=10s, ...)`. Our `StreamLayer` (below) is what actually shuttles bytes.

Then build `raft.DefaultConfig`, override timeouts if the caller set any, and call `raft.NewRaft(...)`.

If `Bootstrap: true`, call `raft.BootstrapCluster` with a single-server configuration (this node). Swallow the "bootstrap only works on new clusters" error — harmless on restart.

## Writing: `Append(record) → apply(AppendRequestType, ProduceRequest)`

`apply` is the **only** path for mutations:

1. Start with a 1-byte `RequestType` (`AppendRequestType = 0`).
2. Append the protobuf-marshaled request.
3. `raft.Apply(bytes, 10s)` — submits to Raft. Blocks until the entry is committed across a quorum.
4. Raft then calls `fsm.Apply` on every node to advance the local `Log`.
5. Return the FSM's response (`*api.ProduceResponse{Offset}`).

So even the leader's own append goes through Raft → strong consistency, no split brain.

## Reading: `Read(offset)`

Pure **local** read from this node's `Log`. On followers this is "eventually consistent" (the record will show up after the FSM applies it). This is why the picker routes `Consume` to followers freely — it's OK.

## Cluster membership

- `Join(id, addr)` — called by `discovery.Membership` handler. Gets current Raft config, removes stale entries with the same id/addr, then `AddVoter`.
- `Leave(id)` — `RemoveServer`.
- `WaitForLeader(timeout)` — polls `raft.Leader()` every second.
- `GetServers() → []*api.Server` — returns every server in the Raft config with an `IsLeader` boolean. Used by the client-side `Resolver`.

## `Close()`

Shuts down Raft (which fsyncs stable state), then closes the local `Log`.

## FSM

```go
type fsm struct { log *Log }
```

- `Apply(record *raft.Log)` — reads first byte (request type), dispatches. `AppendRequestType` → `applyAppend` (unmarshal the protobuf, call `log.Append`, return `*api.ProduceResponse`).
- `Snapshot()` — returns a `snapshot` wrapping `log.Reader()`.
- `Restore(r io.ReadCloser)` — replays a snapshot: read length-prefixed records one at a time, unmarshal, `log.Append`. On the very first record, resets the log with `InitialOffset = record.Offset` so the restored log starts at the right place.

## `snapshot`

```go
type snapshot struct { reader io.Reader }
```

- `Persist(sink)` — `io.Copy(sink, s.reader)`, then `sink.Close`. On error, `sink.Cancel`.
- `Release()` — noop.

## `logStore` (raft.LogStore adapter)

A `*Log` wrapped to satisfy `raft.LogStore`:

- `FirstIndex / LastIndex` → `LowestOffset / HighestOffset`.
- `GetLog(index, out)` → `log.Read(index)` into a `raft.Log` struct (copying Value, Term, Type).
- `StoreLog / StoreLogs` → `log.Append(...)`.
- `DeleteRange(min, max) → log.Truncate(max)`.

In other words: Raft thinks it has a log store, but it's actually **our own segmented log** under the hood. Neat reuse.

## `StreamLayer` (raft.StreamLayer)

The transport for Raft RPCs. Wraps the `net.Listener` from cmux plus the TLS configs.

- `Dial(addr, timeout)` — dial TCP, write a single byte `RaftRPC` (= `1`) so the remote cmux knows to route this to the Raft listener, then wrap in `tls.Client(..., peerTLSConfig)`.
- `Accept()` — accept a conn, read one byte, verify it's `RaftRPC`, wrap with `tls.Server(..., serverTLSConfig)`.
- `Close() / Addr()` — delegate to underlying listener.

This is the piece that makes "one TCP port per node" work. Both Raft and gRPC live on it.
