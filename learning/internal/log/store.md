# `store.go`

The lowest-level file. One `store` = one append-only file of records, each prefixed by its 8-byte big-endian length.

## Layout

```
[ 8-byte length | bytes... ][ 8-byte length | bytes... ]...
```

## Constants

```go
var enc = binary.BigEndian
const lenWidth = 8
```

## `store` type

```go
type store struct {
    *os.File
    mu   sync.Mutex
    buf  *bufio.Writer // buffered writes for throughput
    size uint64        // current EOF
}
```

## `newStore(f *os.File) (*store, error)`

Stats the file to pick up existing size (so appends go after current data), wraps it in a `bufio.Writer`.

## `Append(p []byte) (n, pos uint64, err error)`

- `pos` = current size (where this record starts — the caller uses this as the index entry).
- Write 8-byte length prefix, then the payload, into the buffered writer.
- Bump `size` by `lenWidth + len(p)`.
- Return `(bytesWritten, startPos, nil)`.

Writes are buffered → not durable until a `Flush` or `Close`. Reads flush implicitly before reading.

## `Read(pos uint64) ([]byte, error)`

- Flush the buffer (so any pending writes are visible to the underlying file descriptor).
- Read the 8-byte length at `pos`.
- Read `length` bytes at `pos + lenWidth`.
- Return the payload.

## `ReadAt(p []byte, off int64) (int, error)`

Low-level flush-then-`ReadAt`. Used by the `originReader` in `log.go` to stream whole segments for Raft snapshots.

## `Close() error`

Flushes buffered data, then closes the file. Failure to flush means the tail of your log is lost — so check the error in production.

## Concurrency

All three methods take `s.mu`. So one producer + many readers is safe but serialized.
