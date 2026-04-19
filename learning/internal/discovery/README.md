# `internal/discovery`

Service discovery using **Hashicorp Serf** — a gossip protocol library. Each node runs a Serf agent; when nodes join/leave the cluster, Serf fires events that this package translates into method calls on a `Handler`.

In this project the `Handler` is the `DistributedLog` (from `internal/log`). So:

- Serf says "a new node joined" → `DistributedLog.Join(name, rpcAddr)` → leader adds a Raft voter.
- Serf says "a node left" → `DistributedLog.Leave(name)` → leader removes the voter.

Serf and Raft are separate concerns:

- **Serf** = "who's alive?" (eventual, gossip, cheap).
- **Raft** = "who agrees on the log?" (strong, quorum, expensive).

This package bridges them.

## Files
- [membership.md](membership.md) — the `Membership` type wrapping Serf.
- [membership_test.md](membership_test.md) — 3-node gossip test with a fake handler.
