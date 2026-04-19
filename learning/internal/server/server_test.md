# `server_test.go`

Table-driven gRPC server test with mTLS, two identities, and telemetry toggled by a `-debug` flag.

## `TestMain` / `-debug` flag

Adds a `-debug` flag (off by default). When on, replaces zap's global logger with a development one — helpful when poking at failures locally.

## `TestServer`

Iterates a `map[string]func(t, rootClient, nobodyClient, cfg)`:

| scenario | function |
|---|---|
| `produce/consume a message to/from the log succeeds` | `testProduceConsume` |
| `produce/consume stream succeeds` | `testProduceConsumeStream` |
| `consume past log boundary fails` | `testConsumePastBoundary` |
| `unauthorized fails` | `testUnauthorized` |

Each sub-test gets a fresh setup from `setupTest`.

## `setupTest(t, fn)` (the one that actually runs)

1. Open a TCP listener on 127.0.0.1 and a random port.
2. Build **two** client `grpc.ClientConn`s — one with the `root` cert, one with the `nobody` cert. Same CA on both.
3. Build the **server** TLS config (with client-cert verification enabled — `Server: true`).
4. Create a tmp dir for the local log (`log.NewLog`).
5. Build the Casbin `Authorizer`.
6. If `-debug`, also spin up an OpenCensus `LogExporter` that writes metrics/traces to temp files.
7. Build `*Config{CommitLog, Authorizer}` (no `GetServerer` needed in this file — but see the `setupTest1`/`setupTest2` leftover variants below).
8. Call `NewGRPCServer`, `go server.Serve(l)`.
9. Return the two clients, the cfg, and a `teardown` that stops the server, closes conns, closes the listener, and shuts down telemetry.

## The scenarios

### `testProduceConsume`
Simple `Produce` → `Consume` at the same offset, compare values.

### `testConsumePastBoundary`
Produce one record, then `Consume(produce.Offset + 1)`. Expect an error whose gRPC code matches `api.ErrOffsetOutOfRange{}.GRPCStatus().Err()`.

### `testProduceConsumeStream`
Open `ProduceStream`, send two records, check returned offsets are 0 and 1. Then open `ConsumeStream` at offset 0 and read both back.

### `testUnauthorized`
Uses the **nobody** client (not in `policy.csv`). Both `Produce` and `Consume` must return `codes.PermissionDenied`.

## `setupTest1` / `setupTest2`

Dead-code leftovers from earlier chapter iterations (the file is following a textbook). They build similar scaffolding with minor config differences but aren't called. Fine to ignore — treat as noise.
