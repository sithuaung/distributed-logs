# `tls.go`

Builds an `*tls.Config` from the project's `TLSConfig` struct. Used anywhere we set up mTLS — agent startup, tests, etc.

## `TLSConfig`

```go
type TLSConfig struct {
    CertFile      string // PEM cert for this side (optional)
    KeyFile       string // PEM key for this side (optional)
    CAFile        string // CA that signs the other side (optional)
    ServerAddress string // SNI / expected server name when Server==false
    Server        bool   // true = we are the server; false = we are the client
}
```

## `SetupTLSConfig(cfg) (*tls.Config, error)`

Logic:

1. If `CertFile` + `KeyFile` are both set, load them with `tls.LoadX509KeyPair` and add to `tlsConfig.Certificates`. This presents our identity.
2. If `CAFile` is set:
   - Read and parse the PEM into an `x509.CertPool`.
   - If `Server: true` — set `ClientCAs = ca` and `ClientAuth = RequireAndVerifyClientCert`. This is the **mTLS** piece: the server won't accept anyone whose cert isn't signed by our CA.
   - If `Server: false` — set `RootCAs = ca`. We only trust server certs signed by our CA.
   - Either way, set `ServerName = cfg.ServerAddress` (used for SNI on clients and for verification).

## Why it matters

Every connection in the cluster — client↔server gRPC, node↔node Raft — is mTLS-authenticated, and the server extracts the cert's `CommonName` to feed to the Casbin authorizer (see `internal/server/authenticate`). So this tiny file is the foundation of the whole auth story.
