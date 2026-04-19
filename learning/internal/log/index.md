# `index.go`

The companion file to `store`. Maps **record offset → store position** via a memory-mapped fixed-width table.

## Why mmap?

So lookups are a pointer arithmetic read (no `syscall` per read). The file is pre-allocated to `MaxIndexBytes`, mapped once, and accessed as a `[]byte`. Writes go straight into that mapping; the OS flushes pages to disk eventually.

## Entry layout

```
[ 4-byte offset (uint32) | 8-byte position (uint64) ]  = 12 bytes per entry
```

- `offset` is **relative to the segment's `baseOffset`**, so 4 bytes is enough even for huge global offsets.
- `position` is the byte offset inside that segment's store file.

Constants:

```go
offWidth uint64 = 4
posWidth uint64 = 8
entWidth        = 12
```

## `index` type

```go
type index struct {
    file *os.File
    mmap gommap.MMap
    size uint64 // bytes of actual data (<= len(mmap))
}
```

## `newIndex(f, c) (*index, error)`

1. Stat the file for its current size.
2. **Truncate to `MaxIndexBytes`** — so mmap has a full region to map. This is why the file on disk might look huge while only a handful of entries exist.
3. `gommap.Map(PROT_READ|PROT_WRITE, MAP_SHARED)`.
4. If the file had existing data (`size > 0`), call `scanActualSize()` to find where the real data ends after a crash recovery.

## `scanActualSize()`

The clever recovery piece. After a crash the file is still at `MaxIndexBytes` but only the prefix is real data; the rest is zero bytes from the `Truncate` (or leftover garbage). A sentinel `off==0, pos==0` entry past position 0 means "end of data" (position 0 is the valid first entry's natural value).

Walks forward in entry-sized chunks; returns the position where it first sees a zero entry past `pos > 0`. If no zeros are found, the index is full, so `numEntries * entWidth`.

## `Read(in int64) (out uint32, pos uint64, err error)`

- `in == -1` → read the **last** entry (common when opening a segment to find `nextOffset`).
- Else treat `in` as the index of the entry.
- Decode 4-byte offset and 8-byte position.
- Return `io.EOF` if out of range.

## `Write(off uint32, pos uint64) error`

Appends a new entry at `size`. Returns `io.EOF` if the mmap is full.

## `Close() error`

1. `mmap.Sync(MS_SYNC)` — flush dirty pages.
2. `file.Sync()` — flush OS buffers.
3. **Truncate the file back to the real `size`** — so the next process opens a correctly-sized file and doesn't need `scanActualSize` (though it still works either way).
4. Close.

## `Name() string`

Delegates to `file.Name()`. Used by `segment.Remove()` to `os.Remove` the right path.
