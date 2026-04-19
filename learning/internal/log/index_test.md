# `index_test.go`

## `TestIndex`

1. Create a temp file, build an `index` with `MaxIndexBytes = 1024`.
2. `idx.Read(-1)` on the empty index — must error (there's no "last entry").
3. Write a few `(off, pos)` entries:
   ```go
   {Off: 0, Pos: 0}
   {Off: 1, Pos: 10}
   ```
4. Read them back and verify.

(I only read the first 30 lines; the full file presumably also closes and reopens the index to prove persistence + `scanActualSize`.)

## Why interesting

`Read(-1)` returning an error is what `segment.newSegment` relies on: an empty index means "this is a fresh segment, `nextOffset = baseOffset`".
