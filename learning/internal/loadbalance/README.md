# `internal/loadbalance`

Client-side smarts for gRPC. Clients don't know which node is the leader — this package teaches gRPC to:

1. **Discover** the cluster by calling `GetServers` on any node (the **Resolver**).
2. **Pick** a connection per request: leader for writes, round-robin followers for reads (the **Picker**).

Both are registered with gRPC in `init()` so just importing the package plugs them in. The scheme is `"proglog"` (constant `Name`), so clients dial:

```go
grpc.Dial("proglog:///" + anyNodeAddr, ...)
```

## Files
- [resolver.md](resolver.md) — discovers cluster members via `GetServers` RPC.
- [picker.md](picker.md) — routes `Produce` to leader, `Consume` to followers.
- [resolver_test.md](resolver_test.md) — resolver unit test with a mock `GetServerer`.
- [picker_test.md](picker_test.md) — picker unit tests with fake sub-conns.
