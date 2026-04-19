# `authorizer.go`

A tiny authorizer backed by Casbin.

## Types

```go
type Authorizer struct {
    enforcer *casbin.Enforcer
}
```

`casbin.Enforcer` is Casbin's rule engine. It loads two files:

- **model** (`model.conf`) — a little DSL declaring "what does a rule look like?" e.g. `sub, obj, act` tuples plus a matcher.
- **policy** (`policy.csv`) — the actual rules, e.g. `p, root, *, produce`.

Both file paths are provided by `internal/config` (`ACLModelFile`, `ACLPolicyFile`).

## API

### `New(model, policy string) *Authorizer`
Builds the Casbin enforcer. Note: it does **not** return an error — Casbin panics internally if the files are malformed.

### `Authorize(subject, object, action string) error`
Calls `enforcer.Enforce(subject, object, action)`. If it returns `false`, builds a gRPC `PermissionDenied` status error:

```
"<subject> not permitted to <action> to <object>"
```

Returning a `status.Error` (not a plain `error`) is important — the gRPC machinery turns it into the right status code for the client.

## Mental model

Casbin in this project is effectively an ACL table: "does the cert with CN=`root` have `produce` permission on `*`?" Yes → nil. No → `PermissionDenied`.
