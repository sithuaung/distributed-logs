# `membership_test.go`

Tests `Membership` without Raft — uses a fake `Handler` that just records calls.

## `TestMembership`

1. Spin up 3 `Membership`s via `setupMember`:
   - Member 0 has no `StartJoinAddrs` → it is the seed.
   - Members 1 and 2 point `StartJoinAddrs` at member 0's address.
2. Wait (with `require.Eventually`) until:
   - `handler.joins` has 2 entries (members 1 and 2 joined — member 0 saw them, did not see itself).
   - `m[0].Members()` has 3 entries (all three, as Serf sees them).
   - `handler.leaves` is empty.
3. Call `m[2].Leave()`.
4. Eventually check:
   - Member 0's view of member 2 becomes `serf.StatusLeft`.
   - `handler.leaves` has one entry.
5. Drain `handler.leaves` and require the leaver was named `"2"`.

## `setupMember` helper

- `len(members) == 0` case builds the channels on the `handler` (so the seed's handler can record events).
- Subsequent members reuse the channels by **not** setting them (they'd be nil and writes would block). Actually, re-reading: every new member gets a **fresh** `handler{}` — only the first one has channels. The test only asserts against the first handler. The other handlers are throwaway. That's why the function returns `handler` only conditionally useful for the first call.

## `handler` fake

```go
type handler struct {
    joins  chan map[string]string
    leaves chan string
}
```

`Join(id, addr)` sends a map into `joins` (non-blocking because the channel is buffered 3). `Leave(id)` sends the id into `leaves`.

## Why no Raft?

Discovery is a plain gossip abstraction here — testing it with a real `DistributedLog` would be slow and would tangle two concerns. A mock `Handler` proves the bridging logic works.
