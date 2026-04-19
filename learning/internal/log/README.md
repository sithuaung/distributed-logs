# `internal/log`

The heart of the project. Three layers:

### 1. Single-node storage (on-disk commit log)

```
Log
└── Segments[]         (many, each a contiguous range of offsets)
    ├── store          (append-only file of length-prefixed records)
    └── index          (memory-mapped file: relative offset → store position)
```

- **store.go** — length-prefixed record file. Buffered writes, random reads.
- **index.go** — fixed-width (12 byte) entries via `mmap`. Quick `offset → file position` lookups. Pre-allocated to `MaxIndexBytes` so mmap works; trimmed back on close.
- **segment.go** — one `store` + one `index`, both named by `baseOffset`. Decides when it's "full" (`IsMaxed`).
- **log.go** — orchestrates segments: appends go to the active segment, creates a new one when full; reads find the right segment by offset range.
- **config.go** — shared config struct for Raft + segment tuning.

### 2. Distributed (multi-node) log

- **distributed.go** — wraps `Log` with **Hashicorp Raft**. Exposes `Append/Read/Join/Leave/GetServers`. Also defines:
  - `fsm` — Raft's state-machine hook that actually calls `Log.Append` after consensus.
  - `logStore` — the `raft.LogStore` implementation (where Raft puts its own log entries — on top of our `Log`!).
  - `StreamLayer` — the `raft.StreamLayer` that carries Raft RPCs over our cmux'd TLS connection.
  - `snapshot` — Raft snapshot that streams the whole commit log.

### 3. Old-style replicator (not used by the current agent)

- **replicator.go** — pre-Raft replication via `ConsumeStream` on followers. Historical / reference; the agent uses Raft now.

## Read order

1. `store.md`
2. `index.md`
3. `segment.md`
4. `log.md`
5. `config.md`
6. `distributed.md`
7. `replicator.md` (optional — older design)

And the corresponding `_test.md` files.
