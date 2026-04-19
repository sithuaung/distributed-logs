# `internal/agent`

The **agent** is one full node in the distributed-log cluster. It is the highest-level object in `internal/`: everything else (log, server, discovery, auth) gets wired together here.

One `Agent` owns:

- a TCP listener wrapped in **cmux** — multiplexes Raft and gRPC on the same port.
- a **DistributedLog** — the Raft-replicated commit log (from `internal/log`).
- a **gRPC server** — the public API (from `internal/server`).
- a **Membership** — Serf gossip that tells the leader when new nodes appear (from `internal/discovery`).

Startup order (see `agent.go`):

1. `setupMux` — open the TCP listener and wrap it in cmux.
2. `setupLog` — match Raft traffic on the mux (first byte == `RaftRPC`) and hand it to `DistributedLog`. If `Bootstrap` is true, wait to become leader.
3. `setupServer` — build the authorizer + gRPC server, serve everything cmux didn't match as Raft.
4. `setupMembership` — start Serf. Joins call `DistributedLog.Join`, which adds a Raft voter.

Shutdown reverses this: leave Serf → stop gRPC → close the log (which shuts down Raft).

## Files
- [agent.md](agent.md) — the `Agent` struct and its wiring.
- [agent_test.md](agent_test.md) — end-to-end test that spins up 3 agents.
