# `internal/auth`

Thin wrapper around **Casbin** that answers "is subject S allowed to do action A on object O?".

It is used by `internal/server` inside every `Produce` / `Consume` handler:

```go
s.Authorizer.Authorize(subject(ctx), objectWildcard, produceAction)
```

The `subject` comes from the mTLS client certificate's `CommonName` (see `internal/server/server.go` — `authenticate` function).

## Files
- [authorizer.md](authorizer.md) — the single file in this package.
