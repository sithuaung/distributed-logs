# `store_test.go`

Round-trip tests for the `store` type.

## Fixtures

```go
write = []byte("hello world")
width = uint64(len(write)) + lenWidth  // total on-disk size per record
```

## `TestStoreAppendRead`

1. Create a temp file, wrap in `newStore`.
2. `testAppend(t, s)` — append the `write` fixture many times, checking that `pos` increases by `width` each time and that cumulative `size` matches.
3. `testRead(t, s)` — `Read` at each expected `pos` and verify the payload comes back.
4. `testReadAt(t, s)` — `ReadAt` the length prefix then the payload at the right offset; verify both.
5. Reopen the file with `newStore` again — this tests that `newStore` picks up the existing file size correctly — and re-run `testRead`.

The helper functions `testAppend`, `testRead`, `testReadAt` aren't shown in the snippet I read but follow the names — they exercise the three public methods.

## Why interesting

The close/reopen step confirms that `Close → Flush → reopen` is a durable cycle, not something that loses the last buffered write.
