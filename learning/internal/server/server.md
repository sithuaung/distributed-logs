# `server.go`

The gRPC server. This is where HTTP/2 + TLS + middleware + the Log API come together.

## Interfaces it depends on

```go
type CommitLog interface {
    Append(*api.Record) (uint64, error)
    Read(uint64) (*api.Record, error)
}
type Authorizer interface {
    Authorize(subject, object, action string) error
}
type GetServerer interface {
    GetServers() ([]*api.Server, error)
}
```

Clean separation — the server doesn't import `log` or `discovery`; callers plug in whatever satisfies these.

## `Config`

```go
type Config struct {
    CommitLog   CommitLog
    Authorizer  Authorizer
    GetServerer GetServerer
}
```

## `NewGRPCServer(config, grpcOpts...) (*grpc.Server, error)`

Assembly function:

1. Named `zap` logger (`"server"`).
2. Enable OpenCensus tracing (`AlwaysSample`) and metrics views (`ocgrpc.DefaultServerViews`).
3. Build interceptor chains (both Unary and Stream):
   - `grpc_ctxtags` — attach fields to the request context for logging.
   - `grpc_zap` — log every RPC with duration.
   - `grpc_auth` — extract the client identity via `authenticate` and stash it in the context.
4. Plug `ocgrpc.ServerHandler` as the stats handler (metrics/tracing).
5. Build the actual `*grpc.Server` with all options.
6. Register a gRPC **health** service (always serving) — used by load balancers / orchestrators.
7. Register the **Log** service via `api.RegisterLogServer`.

## `grpcServer` methods

### `Produce(ctx, req) (ProduceResponse, error)`
1. `Authorizer.Authorize(subject(ctx), "*", "produce")`.
2. `CommitLog.Append(req.Record)` → offset.
3. Return `{Offset}`.

### `Consume(ctx, req) (ConsumeResponse, error)`
1. Authorize with `"consume"`.
2. `CommitLog.Read(req.Offset)`.
3. Wraps errors in `api.ErrOffsetOutOfRange` — a custom type that maps to a specific gRPC status code.

### `ProduceStream(stream) error`
Bidirectional: read a request, produce, send back the response, loop. Each message is authorized individually (because it calls `s.Produce`).

### `ConsumeStream(req, stream) error`
Server streaming: keep reading incrementing offsets. On `ErrOffsetOutOfRange` it `continue`s — so clients can tail the log; the loop idles until new data arrives. Exits when `stream.Context().Done()` fires.

### `GetServers(ctx, req)`
Delegates to `GetServerer.GetServers()` and wraps the result. No auth — this is used by the resolver during Dial, before any credentials are really established in a useful way.

## mTLS → subject extraction

### `authenticate(ctx) (context.Context, error)`

Called by `grpc_auth.UnaryServerInterceptor`. Pulls the `peer.Peer` from context:

- If there's no peer info at all → error.
- If `p.AuthInfo == nil` (no TLS) → set subject = `""` and continue. This allows the server tests that don't use client certs.
- Otherwise cast to `credentials.TLSInfo`, read `VerifiedChains[0][0].Subject.CommonName`, and stash it in the context under `subjectContextKey{}`.

### `subject(ctx) string`

Reads the value back out. **Will panic with a type assertion error if there's no subject in context** — callers must be behind the `grpc_auth` interceptor.

## Why middleware order matters

`grpc_ctxtags` must come first so later middleware (logging, auth) can attach fields to the tags. `grpc_auth` must run before the handler so `subject(ctx)` works in `Produce`/`Consume`.
