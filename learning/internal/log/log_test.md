# `log_test.go`

Table-driven test over multiple scenarios, each with a fresh tmp dir and fresh `Log`.

## Scenarios

| scenario | what it checks |
|---|---|
| `append and read a record succeeds` | basic round-trip |
| `offset out of range error` | `Read(outside range)` returns `ErrOffsetOutOfRange` |
| `init with existing segments` | open a fresh `Log` on a dir that already has files, verify records from the previous life come back |
| `reader` | `log.Reader()` returns a single stream of every record (used by Raft snapshots) |
| `truncate` | `Truncate(lowest)` drops early segments |

## Setup

Each sub-test:

1. Creates a tmp dir.
2. Builds a `Config` with a tiny `MaxStoreBytes = 32` so appends roll segments quickly.
3. Runs the test func.

## Why interesting

The `init with existing segments` scenario is the proof that `setup()`'s "sort offsets, skip duplicates because `.store` and `.index` share a basename" logic actually reconstructs the log correctly.
