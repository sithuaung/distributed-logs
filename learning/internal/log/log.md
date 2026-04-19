# `log.go`

The single-node `Log`. A list of `segment`s plus the logic to append, read, and rotate.

## `Log` type

```go
type Log struct {
    mu sync.RWMutex

    Dir    string // where segment files live
    Config Config

    activeSegment *segment   // the one currently being appended to
    segments      []*segment // all of them, sorted by baseOffset
}
```

## `NewLog(dir, c) (*Log, error)`

1. Default `MaxStoreBytes` and `MaxIndexBytes` to 1024 if unset (tiny — makes tests roll segments quickly).
2. `setup()` — scan existing files to rebuild segments.

## `setup()`

1. `os.ReadDir(dir)` — list files.
2. For each file, strip the extension to get the base offset number, collect into `baseOffsets`.
3. Sort ascending.
4. Walk the sorted list and call `newSegment(baseOffsets[i])` — **skipping every other entry** because each offset appears twice (one `.store`, one `.index`). That's the `i++` inside the loop.
5. If no segments were found, create a fresh one at `Config.Segment.InitialOffset`.

The last segment seen becomes `activeSegment`.

## `Append(record) (uint64, error)`

Takes the write lock, appends to `activeSegment`, and if the segment is now maxed, calls `newSegment(off+1)` to roll to a new one. Returns the assigned offset.

## `Read(off) (*api.Record, error)`

Takes the read lock, walks `l.segments` to find the one whose `[baseOffset, nextOffset)` window contains `off`, and reads from it. Returns `api.ErrOffsetOutOfRange` if no segment matches.

Note: this is an O(n-segments) linear scan. Fine for small numbers; would want a binary search for very long logs.

## `newSegment(off)`

Creates a new segment at `off`, appends it to `segments`, sets it as `activeSegment`. Caller holds the write lock.

## `Close() / Remove() / Reset()`

- `Close` closes every segment.
- `Remove` closes then `os.RemoveAll(Dir)`.
- `Reset` removes everything and re-runs `setup()` — used by the Raft FSM during snapshot restore.

## `LowestOffset() / HighestOffset()`

Bookends:
- Lowest = `segments[0].baseOffset`.
- Highest = last segment's `nextOffset - 1`, or 0 if empty.

These are used as Raft's `FirstIndex`/`LastIndex` via the `logStore` adapter in `distributed.go`.

## `Truncate(lowest uint64) error`

Drops every segment whose entire offset range is `<= lowest`. Used by Raft (`DeleteRange`) during log compaction.

## `Reader() io.Reader`

Returns a reader that concatenates every segment's store file — used by Raft `Snapshot`. Wraps each store in an `originReader` that tracks its own `off` and calls `ReadAt`:

```go
type originReader struct {
    *store
    off int64
}
```

The `*store` embed lets `io.MultiReader` see it as a reader, but `originReader.Read` actually just advances `off` through `ReadAt`.
