# `picker.go`

Implements a gRPC **Picker** (and `PickerBuilder`) that routes RPCs based on method name:

- `Produce` → the leader.
- `Consume` → a follower, round-robin.
- If no followers exist, `Consume` falls back to the leader.

## Types

```go
type Picker struct {
    mu        sync.RWMutex
    leader    balancer.SubConn
    followers []balancer.SubConn
    current   uint64 // atomic counter for round-robin
}
```

A `balancer.SubConn` is gRPC's handle to a single backend connection.

## `Build(buildInfo base.PickerBuildInfo) balancer.Picker`

Called by gRPC whenever the set of ready sub-connections changes (e.g. after the `Resolver` reports new addresses). Walks the ready sub-conns, looks at each one's `resolver.Address.Attributes["is_leader"]` (set by the `Resolver`), and splits them into `leader` + `followers`.

Returns `p` (the picker itself implements `balancer.Picker`).

## `Pick(info balancer.PickInfo) (balancer.PickResult, error)`

Called for every RPC. Decides using `info.FullMethodName`:

- Contains `"Produce"` **or** there are no followers → pick the leader.
- Contains `"Consume"` → pick `nextFollower()`.

If the resolved connection is `nil` (e.g. we haven't seen the leader yet), return `balancer.ErrNoSubConnAvailable`. gRPC interprets that as "retry / wait for new state".

## `nextFollower()`

```go
cur := atomic.AddUint64(&p.current, 1)
idx := int(cur % uint64(len(p.followers)))
return p.followers[idx]
```

Round-robin, atomic-incremented counter. Modulo on `len(followers)` — safe because `Pick` is guarded by `RLock` and `followers` can only change inside `Build` (which takes the write lock).

## `init()`

Registers the picker under the name `Name` via `base.NewBalancerBuilder`. This makes it available to gRPC as a load-balancing policy. The `Resolver` tells gRPC to use it via:

```json
{"loadBalancingConfig":[{"proglog":{}}]}
```

## Gotcha

`ProduceStream` and `ConsumeStream` method names contain `"Produce"` and `"Consume"` respectively, so streaming RPCs route the same way. That's intentional.
