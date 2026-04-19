# `resolver_test.go`

Unit test for the `Resolver`. Runs a **real** gRPC server (so the TLS + `GetServers` path is exercised) but uses a mock `GetServerer` that returns a hard-coded list.

## `TestResolver` steps

1. Listen on 127.0.0.1 with `net.Listen`.
2. Load the **server** TLS config from `internal/config`.
3. Start a real `server.NewGRPCServer` whose `GetServerer` is the `getServers` mock returning two servers:
   - `localhost:9001`, `is_leader: true`
   - `localhost:9002`, `is_leader: false`
4. Load the **client** TLS config and build `resolver.BuildOptions{DialCreds: ...}`.
5. Call `r.Build(target, conn, opts)` — `target.URL.Path` is the server's actual address; `conn` is a fake `clientConn` that records every `UpdateState`.
6. Assert that the recorded `conn.state.Addresses` matches the expected two-entry list with the right `is_leader` attributes.
7. Clear `conn.state.Addresses`, call `ResolveNow` again, assert that it repopulates — proving refresh works.

## `getServers` mock

```go
func (s *getServers) GetServers() ([]*api.Server, error) {
    return []*api.Server{{...localhost:9001, IsLeader:true}, {...localhost:9002}}, nil
}
```

Implements the `server.GetServerer` interface.

## `clientConn` mock

Fake `resolver.ClientConn`. Only `UpdateState` is interesting — it records the state. Other methods (`ReportError`, `NewAddress`, `NewServiceConfig`, `ParseServiceConfig`) are stubs; `ParseServiceConfig` returns `nil` which is why the test's `wantState` doesn't include a `ServiceConfig`.

## Why this test matters

It verifies that:
- TLS is wired correctly for the resolver's internal gRPC connection.
- `GetServers` results are translated into `resolver.Address` objects with the right attribute keys the `Picker` later reads.
