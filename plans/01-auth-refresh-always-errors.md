# 01 — Auth-refresh block always returns an error

**Severity:** Critical — breaks the SDK in normal use
**Files:** [../order_payments.go](../order_payments.go), [../transactions.go](../transactions.go)

## Problem

Three OAuth methods re-authenticate when the token has expired, then unconditionally return an
error — including when the re-auth succeeded:

```go
if c.HasAuthExpired() {
    _, authErr := c.Authenticate()
    return nil, fmt.Errorf("authentication error %s", authErr)
}
```

`authErr` is `nil` on success, so the caller gets `authentication error %!s(<nil>)` and no data.
The request is never sent.

Call sites:

- `CreateOrderPayment` — order_payments.go:43-46
- `GetTransaction` — transactions.go:35-38
- `CancelPartialAuthorization` — transactions.go:138-141 (returns bare `error`, same shape)

## Impact

Any `OAuthClient` that outlives its token is permanently broken: the first call after expiry fails,
and so does every call after it, because the refreshed token is never used. A fresh client works
only until `expires` passes.

There is a second-order effect: a brand-new client created by `NewOAuth` has a zero-value
`expires`, so `HasAuthExpired()` is `true` immediately. A caller who does not call `Authenticate()`
manually first hits this on their very first request — which is why [../example/main.go](../example/main.go)
calls `Authenticate()` explicitly before anything else.

## Fix

Return only on real failure, then fall through to the request:

```go
if c.HasAuthExpired() {
    if _, authErr := c.Authenticate(); authErr != nil {
        return nil, fmt.Errorf("authentication error: %w", authErr)
    }
}
```

Use `%w` rather than `%s` so callers can `errors.Is`/`errors.As` the cause. For
`CancelPartialAuthorization`, the same with a bare `return fmt.Errorf(...)`.

Apply to all three call sites identically. Do not try to factor it out here — that is plan
[11](11-auth-roundtripper.md); keep this change minimal so it is easy to review and backport.

## Verification

- Add a regression test (see [08](08-add-tests.md)) using `httptest`: build a client whose token is
  already expired, point it at a stub server that returns a valid token then a valid payload, and
  assert the call succeeds and sends `Authorization: Bearer <new token>`.
- Add a second test where the token endpoint returns 401 and assert the error propagates.
- `make build-example && make vet && staticcheck`

## Notes

Fixing this makes the token actually refresh mid-flight, which depends on `SetToken` persisting
through the value receiver — see [02](02-oauth-value-receivers.md). It works today, but confirm
the regression test above genuinely observes the *new* token, not a stale one.
