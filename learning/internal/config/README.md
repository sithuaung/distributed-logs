# `internal/config`

Two small helpers used everywhere:

1. **Certificate & ACL file locations** (`files.go`) — resolves where cert/key/CA PEM files and the Casbin model+policy live on disk.
2. **TLS config builder** (`tls.go`) — turns a `TLSConfig` struct into a `*tls.Config` for client or server mTLS.

Both are used by `internal/agent` (to start a node), `internal/server` tests, `internal/loadbalance` tests, and `test/` end-to-end tests.

## Files
- [files.md](files.md) — where certs live (`$CONFIG_DIR` or `~/.proglog`).
- [tls.md](tls.md) — how mTLS `*tls.Config` is constructed.
