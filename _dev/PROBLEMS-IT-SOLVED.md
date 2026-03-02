# Problems It Solved & What's Left

## Problems Solved

### 1. Distributed Consensus
Raft-based replication ensures multiple nodes safely agree on log state without split-brain scenarios. Leader election is automatic — if the leader crashes, a new one is elected.

### 2. Cluster Discovery
Nodes find each other automatically via Serf gossip protocol. Join/leave events update Raft voter membership without manual intervention.

### 3. Smart Client Load Balancing
Custom gRPC resolver + picker routes **writes to the leader** and **reads to followers** (round-robin). This reduces leader load and improves read throughput.

### 4. Crash Recovery
Memory-mapped index files recover gracefully after unclean shutdowns using `scanActualSize()` — scans backward to find the actual data boundary.

### 5. Security (mTLS + RBAC)
- Mutual TLS for both client-server and node-to-node communication
- Casbin-based RBAC authorization (CN extracted from TLS cert as identity)

### 6. Efficient Storage
Append-only store with length-prefixed records, memory-mapped index for O(1) offset lookups, and automatic segment rotation when size limits are exceeded.

### 7. Kubernetes-Native Deployment
- StatefulSet with stable DNS (headless service)
- Init container generates per-pod Raft config
- Metacontroller creates per-pod LoadBalancer Services for external leader routing
- Native gRPC health probes (readiness/liveness)
- PersistentVolumeClaims for durability

### 8. Observability Foundation
Structured logging (Zap) and OpenCensus metrics/tracing hooks are wired in.

### 9. Connection Multiplexing
Single TCP port per node using cmux — Raft transport (first byte `0x01`) and gRPC share the same listener.

### 10. Streaming API
Bidirectional streaming for writes (`ProduceStream`) and server streaming for reads (`ConsumeStream`), alongside unary RPCs.

---

## What's Still Left to Do

### High Priority

- **Log Compaction / Retention:** No TTL or size-based retention enforcement; old segments are never deleted
- **Metrics Export:** OpenCensus is registered but not connected to any backend (Prometheus, Jaeger, etc.)
- **Batching:** Each `Produce` = one Raft entry; no batching of commits — major throughput bottleneck
- **Test TODOs:** `picker_test.go` has two stub methods (`subConn.Shutdown()`, `subConn.GetOrBuildProducer()`) marked `// TODO implement me`

### Medium Priority

- **Partitioning / Sharding:** Single log per cluster limits horizontal scalability
- **Consumer Groups:** No consumer group concept; clients must track offsets themselves
- **Offset Management:** Offsets not persisted server-side; clients must remember their position
- **Exactly-Once Semantics:** No deduplication; retries can produce duplicates
- **Compression:** No record compression support
- **Encryption at Rest:** Data on disk is unencrypted (only in-transit via TLS)

### Low Priority / Nice-to-Have

- **Dynamic ACL Reload:** Policy is hard-coded; no hot-reload without restart
- **Audit Logging:** No record of who accessed what
- **Record Metadata:** No timestamps, keys, or headers on records
- **Binary Search in Index:** Currently linear scan through entries
- **Backup/Restore Utilities:** No tooling for operational backup
- **Monitoring Dashboard:** No example Grafana/Prometheus setup
- **Documentation:** Light on operational procedures and troubleshooting
- **Configurable Replication Factor:** Always uses majority quorum (N/2+1)

---

## Production Readiness Gap Analysis

What separates a working distributed system from one you can run in production under real load.

### Reliability & Correctness

- **Idempotent Producers:** No producer ID / sequence number tracking. A client retry after a network timeout can silently duplicate a record. Fix: assign producer IDs and deduplicate on the leader before committing to Raft.
- **Fencing Tokens:** Leader changes can cause a slow ex-leader to still write. Raft term should be threaded through every write path as a fencing token so stale leaders are rejected.
- **Linearizable Reads:** Current follower reads can serve stale data if the follower hasn't caught up. Fix: read-index or lease-based reads (Raft `VerifyLeader` before serving from follower).
- **Graceful Shutdown:** No drain period — in-flight RPCs are dropped when a node exits. Need a shutdown sequence: stop accepting new work, wait for Raft to commit pending entries, transfer leadership, then exit.
- **Backpressure:** No flow control between producer and the Raft pipeline. A burst of writes can OOM the leader. Fix: bounded in-flight queue + `RESOURCE_EXHAUSTED` gRPC status code.

### Operational Maturity

- **Structured Error Codes:** gRPC status codes are inconsistently used. Clients cannot reliably distinguish "not leader" from "disk full" from "authorization denied" without parsing error strings.
- **Leader Transfer API:** No way to gracefully hand off leadership before maintenance (rolling restart). Raft supports `TransferLeadership`; expose it via an admin RPC.
- **Cluster Membership API:** Nodes can only join at startup. No admin API to add/remove voters or non-voters at runtime.
- **Rolling Upgrade Safety:** No documented compatibility contract between versions. A mixed-version cluster can fail silently if Raft log entries or gossip messages change shape.
- **Chaos / Fault Injection Tests:** No tests that kill nodes mid-write, partition the network, or corrupt a segment file. These are the tests that actually validate correctness.

### Scalability

- **Multi-Partition (Topic) Support:** Everything lives in one log. In practice you need N independent Raft groups (multi-Raft) so different topics can scale and fail independently.
- **Horizontal Read Scaling:** Follower reads are round-robin but all followers replicate the full log. Add read replicas that only subscribe to a subset of partitions.
- **Write Batching at the Raft Layer:** Each `Produce` call proposes one entry. Group-commit (collect entries over a configurable window, propose as a batch) can improve throughput by 10-50x.
- **Index Rebuild Performance:** On a large segment the backward scan in `scanActualSize()` is O(n). A separate checkpoint file would make recovery O(1).

### Observability

- **Metrics Backend Integration:** Wire OpenCensus (or migrate to OpenTelemetry) to a real Prometheus scrape endpoint and ship traces to Jaeger/Tempo. Without this, you are flying blind.
- **Per-Partition Lag Metric:** Track consumer offset vs. latest offset per partition so you can alert on consumers falling behind.
- **Raft Health Metrics:** Expose term, commit index, last applied, leader ID, and follower replication lag as Prometheus gauges.
- **Distributed Tracing Propagation:** Trace context (W3C `traceparent`) must flow through gRPC metadata from client produce → Raft commit → consumer deliver.
- **Alerting Runbooks:** Define SLOs (e.g., p99 write latency < 50 ms, replication lag < 1 s) and write runbooks for each alert.

### Security

- **Certificate Rotation:** mTLS certs are static at startup. Production needs automated rotation (cert-manager + short-lived certs) without node restart.
- **Encryption at Rest:** Segment files are plaintext on disk. Encrypt with AES-GCM at the store layer or rely on encrypted PVs — but the choice must be explicit.
- **Secrets Management:** TLS key material is mounted as a plain Kubernetes Secret. Use a secret store (Vault, AWS Secrets Manager) with short-lived dynamic credentials.
- **Network Policy:** No Kubernetes `NetworkPolicy` restricting pod-to-pod traffic. Any pod in the cluster can reach the Raft port.
- **Audit Log:** No record of which identity produced or consumed which offsets. Required for compliance (SOC 2, HIPAA, PCI).

### Data Management

- **Log Compaction:** Append-only segments grow forever. Need time-based and size-based retention, plus key-based compaction (keep only the latest record per key, Kafka-style).
- **Snapshot Shipping:** Raft snapshots are stored locally. A new node joining a large cluster must stream the snapshot over the network; there is no mechanism for this today.
- **Cross-Region Replication:** Single Raft group is latency-bound by the slowest quorum member. For geo-distribution, need async replication between independent clusters (mirror maker pattern).
- **Backup & Point-in-Time Recovery:** No tooling to snapshot segment files + Raft log to object storage (S3/GCS) and replay from a specific offset.
