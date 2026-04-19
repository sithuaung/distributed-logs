# `picker_test.go`

Unit tests for the `Picker` using a fake `balancer.SubConn` — no real gRPC.

## `TestPickerNoSubConnAvailable`

Brand-new `Picker{}` (nothing `Build`-ed into it). Picking anything must return `balancer.ErrNoSubConnAvailable`. This is the "cluster state isn't ready yet" path.

## `TestPickerProducesToLeader`

`setupTest` creates 3 fake sub-conns; the 0th is marked leader (`is_leader: true`). After `picker.Build(...)`, calling `Pick` 5 times with method `"/log.vX.Log/Produce"` must always return sub-conn 0.

## `TestPickerConsumesFromFollowers`

Same setup. Method `"/log.vX.Log/Consume"`. Each of 5 picks should alternate between sub-conns 1 and 2 (indices `i%2 + 1`). Confirms round-robin starts at 1 and wraps.

## `setupTest` helper

Builds a `base.PickerBuildInfo` by hand — each sub-conn gets a `resolver.Address` with an `is_leader` attribute, all stuffed into `ReadySCs`. Then calls `picker.Build(buildInfo)`.

## `subConn` fake

Implements the `balancer.SubConn` interface minimally — only `UpdateAddresses` actually does anything (stores the addrs). `Shutdown`/`GetOrBuildProducer` just `panic` since the test never hits them.
