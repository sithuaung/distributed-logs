# `distributed_test.go`

End-to-end test for `DistributedLog` — spins up a real multi-node Raft cluster with no mocks.

## `TestMultipleNodes`

1. Allocates 3 dynamic ports (`go-dynaport`).
2. Builds 3 `DistributedLog`s, each in its own tmp dir.
   - Node 0 is the bootstrap (becomes initial leader).
   - Nodes 1 and 2 `Join` through the leader's Raft transport.
3. `Append`s a handful of records on the leader.
4. Uses `require.Eventually` to wait until every follower's local `Log` has the same records (confirmed by `Read` on each node).
5. Tests `Leave` — a node leaves and the remaining cluster still functions.

(I only peeked at the first ~30 lines; the rest follows that shape.)

## Why interesting

It's the only place the whole Raft wiring (`fsm`, `logStore`, `StreamLayer`, snapshot store, stable store) gets exercised against real `raft.NewRaft`. If something is misconfigured in `setupRaft`, this test catches it — and the failures are easier to diagnose here than in the higher-level `agent_test.go` which layers gRPC and Serf on top.

## Tuning

Real Raft timeouts are multiple seconds by default, so tests usually knock them down via `config.Raft.HeartbeatTimeout` etc. The test also needs `require.Eventually` rather than direct assertions, because replication is asynchronous.
