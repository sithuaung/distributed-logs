# Learning Notes — `internal/`

This folder mirrors the project's `internal/` tree. Each subfolder has a `README.md` that explains the purpose of the package, and a `.md` file per Go source file that explains what the file does, the key types/functions, and why they exist.

Read order if you are brand new to the project:

1. `internal/log/` — how a single log (store + index + segment) works on disk.
2. `internal/log/distributed.md` — how multiple logs are coordinated with Raft.
3. `internal/server/` — the gRPC API that exposes the log.
4. `internal/auth/` — who can call which RPC.
5. `internal/config/` — TLS certs and ACL file locations.
6. `internal/discovery/` — how nodes find each other with Serf.
7. `internal/loadbalance/` — how a client picks which node to talk to.
8. `internal/agent/` — the top-level "node" that wires everything together.

## Big picture

A **node** ("Agent") in this system:

- Listens on a single TCP port, uses `cmux` to multiplex Raft traffic and gRPC traffic on the same port.
- Stores records in a local **log** made of **segments**. Each segment has a `store` (the raw record bytes) and an `index` (memory-mapped offset → position lookup).
- Replicates the log across nodes using **Raft** (Hashicorp implementation). Writes go to the leader and are committed through Raft's FSM.
- Discovers peers using **Serf** (gossip). When a new node joins, the leader adds it as a Raft voter.
- Exposes a **gRPC** service (`Log`) with `Produce`, `Consume`, streaming variants, and `GetServers`.
- Authenticates clients with **mTLS**, and authorizes them using **Casbin** ACLs.
- Clients use a custom gRPC **resolver + picker** that sends `Produce` to the leader and round-robins `Consume` across followers.
