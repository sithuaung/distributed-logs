# `config.go`

The single shared `Config` struct used by everything in the `log` package.

```go
type Config struct {
    Raft struct {
        raft.Config                // embeds Hashicorp Raft's Config
        BindAddr    string         // host:port the Raft transport listens on
        StreamLayer *StreamLayer   // custom transport (see distributed.go)
        Bootstrap   bool           // true only for the first node in the cluster
    }
    Segment struct {
        MaxStoreBytes uint64 // segment rolls when the store file reaches this size
        MaxIndexBytes uint64 // or when the index reaches this size
        InitialOffset uint64 // baseOffset of the very first segment
    }
}
```

Two unrelated chunks in one struct — historical convenience, so callers pass a single value around.

Defaults (applied in `NewLog` if zero): `MaxStoreBytes = 1024`, `MaxIndexBytes = 1024`. Tiny, because they're geared toward tests; agents override these in practice.
