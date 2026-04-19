# `segment.go`

A `segment` pairs one `store` with one `index`, both named by the segment's `baseOffset`. It's the unit of log rotation.

## Files on disk

```
<dir>/<baseOffset>.store
<dir>/<baseOffset>.index
```

## `segment` type

```go
type segment struct {
    store                  *store
    index                  *index
    baseOffset, nextOffset uint64
    config                 Config
}
```

- `baseOffset` — the offset of the first record in this segment.
- `nextOffset` — offset that will be assigned to the next `Append`.

## `newSegment(dir, baseOffset, c) (*segment, error)`

1. Open (creating if needed) the `.store` file with `O_RDWR|O_CREATE|O_APPEND` and wrap it in a `store`.
2. Open the `.index` file with `O_RDWR|O_CREATE` and wrap it in an `index` (which truncates to `MaxIndexBytes`, mmaps, and recovers `size`).
3. Read the last index entry (`index.Read(-1)`):
   - If it errors (empty index) → `nextOffset = baseOffset`.
   - Else `nextOffset = baseOffset + lastRelativeOffset + 1`.

## `Append(record) (offset, err)`

1. Assign `record.Offset = nextOffset`.
2. `proto.Marshal(record)` → bytes.
3. Write bytes to the store → get back the store position.
4. Write `(nextOffset - baseOffset, storePos)` into the index.
5. `nextOffset++`.
6. Return the offset **before** the increment (so the caller gets this record's offset).

## `Read(off) (*api.Record, error)`

1. `index.Read(off - baseOffset)` → returns relative offset + store position.
2. `store.Read(storePos)` → returns the marshaled bytes.
3. `proto.Unmarshal` → `*api.Record`.

## `IsMaxed() bool`

`true` if **either** the store or the index has reached its max. Used by `Log.Append` to decide when to roll.

## `Close() / Remove()`

`Close` flushes index + store. `Remove` closes then deletes both files.

## `nearestMultiple(j, k uint64)`

An unused helper (returns the biggest multiple of `k` that is `<= j`). Leftover — can be ignored.
