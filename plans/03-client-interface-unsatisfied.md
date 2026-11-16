# 03 — BasicAuthClient does not satisfy the Client interface

**Severity:** Medium
**Files:** [../vivawallet.go](../vivawallet.go), [../basicauth.go](../basicauth.go), [../oauth.go](../oauth.go)

## Problem

`vivawallet.go` declares:

```go
type Client interface {
    Get(uri string, v interface{}) error
    Post(uri string, reader *bytes.Reader, v interface{}) error
    Patch(uri string, reader *bytes.Reader, v interface{}) error
}
```

Two mismatches:

1. `BasicAuthClient.Patch(uri string, reader *bytes.Reader) error` omits `v interface{}`, so
   `BasicAuthClient` does **not** implement `Client`.
2. `Delete` exists on both concrete clients but is absent from the interface.

The interface is referenced nowhere in the package and satisfied only by `OAuthClient`, by
accident. It is currently decorative — it documents an abstraction that does not hold.

## Why it matters

It is a trap for the next person: it reads as "both clients are interchangeable", which is the one
thing this SDK's two-client design specifically does *not* guarantee. Someone will write a helper
taking a `Client`, pass a `BasicAuthClient`, and get a confusing compile error.

## Options

**A. Align and enforce (recommended).** Give `BasicAuthClient.Patch` the `v interface{}` parameter
for symmetry, add `Delete` to the interface, and add compile-time assertions:

```go
var (
    _ Client = (*OAuthClient)(nil)
    _ Client = (*BasicAuthClient)(nil)
)
```

The assertions are the real value — they make any future drift a compile error.

`BasicAuthClient.Patch` currently discards the body because `UpdateOrderPayment` has nothing to
decode. Adding `v` means threading it through; pass `nil` from `UpdateOrderPayment` and have the
implementation skip `json.Unmarshal` when `v == nil`. Note `OAuthClient.performReq` does *not*
currently nil-check `v`, so it would need the same guard.

**B. Delete the interface.** If nothing consumes it, removing it is honest and smaller. Reintroduce
one when there is a real consumer.

Pick A if you expect to add a shared helper or test double; pick B if not. A is the better bet
given [08](08-add-tests.md) and [09](09-context-support.md) both benefit from a seam.

## Interaction with other plans

- [02](02-oauth-value-receivers.md) changes receivers to pointers — write the assertions against
  pointer types as shown above.
- [09](09-context-support.md) will change every signature in this interface. If context is coming
  soon, do this plan *with* it rather than twice.

## Verification

- `make build-example && make vet && staticcheck` — the assertions fail the build if unsatisfied.
