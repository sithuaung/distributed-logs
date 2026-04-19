# `internal/server`

The **gRPC service** that exposes the log to the outside world. Implements the `Log` service defined in `api/v1`:

- `Produce(record) → offset`
- `Consume(offset) → record`
- `ProduceStream(stream)` / `ConsumeStream(stream)` — bidirectional/server streams.
- `GetServers() → servers[]` — used by `internal/loadbalance/resolver` to discover the cluster.

The server is **stateless** — it's a thin HTTP-ish layer on top of three injected interfaces:

```go
type Config struct {
    CommitLog   CommitLog   // Append/Read — satisfied by log.Log or log.DistributedLog
    Authorizer  Authorizer  // Authorize(subject, object, action) error
    GetServerer GetServerer // GetServers() for the resolver
}
```

The agent passes a `DistributedLog` for both `CommitLog` and `GetServerer`.

## Files
- [server.md](server.md) — the gRPC server, auth, and observability wiring.
- [server_test.md](server_test.md) — produce/consume happy paths and the unauthorized case.
