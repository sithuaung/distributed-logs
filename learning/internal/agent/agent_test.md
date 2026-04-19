# `agent_test.go`

End-to-end test (`TestAgent`) that boots a real 3-node cluster and verifies replication.

## What it does

1. Builds a server TLS config and a peer TLS config from the certs in `internal/config`.
2. Starts **3 agents** on dynamically picked ports (`go-dynaport`).
   - Agent `0` is `Bootstrap: true` (becomes the initial Raft leader).
   - Agents `1` and `2` set `StartJoinAddrs = [agent[0].BindAddr]` so Serf pulls them into the cluster.
3. Sleeps 3 s to let Serf gossip settle and Raft replicate membership.
4. Writes a record `"foo"` via the leader's gRPC client (`Produce`).
5. Sleeps 3 s again (wait for Raft to apply on followers).
6. Reads it back from the leader — expects `"foo"`.
7. Reads it back from a **follower** — expects `"foo"` (this is the replication check).

## The `client` helper

Creates a gRPC client that dials via the custom resolver:

```go
grpc.Dial(fmt.Sprintf("%s:///%s", loadbalance.Name, rpcAddr), ...)
```

`loadbalance.Name == "proglog"`, so the URL scheme is `proglog://`. This triggers the custom `Resolver` (from `internal/loadbalance`) which calls `GetServers` on the cluster and hands the addresses to the custom `Picker`.

## Things worth noting

- The test sleeps instead of polling because the code under test doesn't expose a "ready" signal — a common weakness, but fine for a textbook-style test.
- Each agent uses its own tmp `DataDir`, cleaned up with `os.RemoveAll` in the defer.
- `peerTLSConfig` is reused both for peer-to-peer Raft traffic **and** by the test client — the `root-client` identity happens to be authorized for everything.
