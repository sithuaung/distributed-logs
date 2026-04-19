# `files.go`

Resolves on-disk paths for all the TLS certs, keys, and Casbin files the system expects.

## Package-level vars

```go
CAFile               = configFile("ca.pem")
ServerCertFile       = configFile("server.pem")
ServerKeyFile        = configFile("server-key.pem")
RootClientCertFile   = configFile("root-client.pem")
RootClientKeyFile    = configFile("root-client-key.pem")
NobodyClientCertFile = configFile("nobody-client.pem")
NobodyClientKeyFile  = configFile("nobody-client-key.pem")
ACLModelFile         = configFile("model.conf")
ACLPolicyFile        = configFile("policy.csv")
```

Three identity sets of cert+key are expected:

- `server.*` — used by the node.
- `root-client.*` — an admin client (authorized for everything in the Casbin policy).
- `nobody-client.*` — an unauthorized client used in `TestServer/unauthorized fails`.

## `configFile(filename string) string`

Resolution rule:

1. If the env var `CONFIG_DIR` is set, use `$CONFIG_DIR/<filename>`.
2. Otherwise use `~/.proglog/<filename>`.
3. If the home dir can't be found, **panic**.

The panic is intentional: if this package can't find cert files at import time, the program is unusable anyway. `CONFIG_DIR` is the hook that tests and the Makefile use to point at the repo's `test/` certs or `_dev/` working copy.

## Generating the files

Look at `Makefile` (`init`, `gencert` targets) — it uses `cfssl` to mint these certs into `~/.proglog`. The Casbin `model.conf` and `policy.csv` are committed in `test/`.
