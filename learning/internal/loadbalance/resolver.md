# `resolver.go`

Implements a gRPC **Resolver** that discovers cluster members by calling the log server's `GetServers` RPC, then feeds those addresses (with an `is_leader` attribute) to the `Picker`.

## Types

```go
type Resolver struct {
    mu            sync.Mutex
    clientConn    resolver.ClientConn       // talks back to gRPC core
    resolverConn  *grpc.ClientConn          // a side gRPC conn used just to call GetServers
    serviceConfig *serviceconfig.ParseResult
    logger        *zap.Logger
}
```

## `Build(target, cc, opts) (resolver.Resolver, error)`

Called by gRPC once when the client dials `proglog:///<addr>`:

1. Save the `cc` (gRPC's channel back to us).
2. Pull creds from `opts.DialCreds` (so the resolver uses the same TLS as the main client).
3. Parse service config `{"loadBalancingConfig":[{"proglog":{}}]}` — tells gRPC "use our picker".
4. Dial `target.Endpoint()` → that's our second, internal gRPC connection, used only to call `GetServers`.
5. Kick off an initial `ResolveNow`.

## `const Name = "proglog"` + `Scheme()`

Both the load-balancer name and the URL scheme. The `init()` block registers this resolver globally:

```go
func init() { resolver.Register(&Resolver{}) }
```

So any import of this package makes `grpc.Dial("proglog:///...")` work.

## `ResolveNow(...)`

Called by gRPC when it wants a refresh. Steps:

1. Build an `api.LogClient` over `resolverConn`.
2. Call `GetServers` (no auth subject needed — the call is authenticated via TLS only).
3. For each returned `*api.Server`, build a `resolver.Address` with `Attributes["is_leader"] = server.IsLeader`.
4. Push the address list + service config to `clientConn.UpdateState(...)`.

That `UpdateState` call is what ultimately triggers the `Picker.Build`.

## `Close()`

Closes `resolverConn`. Logs errors but ignores them — we're shutting down anyway.

## End-to-end flow (clients' perspective)

```
grpc.Dial("proglog:///host:port")
  → Resolver.Build
    → GetServers → []Server{leader, followers...}
    → clientConn.UpdateState(addresses, service config)
  → gRPC picks the "proglog" balancer (our Picker)
    → Picker.Build(subconns with is_leader attribute)
  → for each RPC: Picker.Pick(method) → leader or follower
```
