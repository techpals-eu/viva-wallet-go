# 02 — OAuthClient uses value receivers for mutating methods

**Severity:** High (latent — works today, fails silently if touched)
**File:** [../oauth.go](../oauth.go)

## Problem

Every `OAuthClient` method is declared on a value receiver, including the two that mutate state:

```go
func (c OAuthClient) SetToken(value string, expires time.Time)
func (c OAuthClient) Authenticate() (*TokenResponse, error)
```

These work only because `tokenValue` is a `*token` and `lock` is a `*sync.RWMutex`
([../vivawallet.go](../vivawallet.go)) — the copy shares the pointee, so the write lands.

## Why it is fragile

The correctness rests on an invisible detail. If anyone changes `tokenValue *token` to
`tokenValue token` — a natural-looking simplification, since the struct is tiny — auth silently
stops persisting. No compile error, no test failure (there are no tests), just every request going
out unauthenticated.

`NewOAuth` returns `*OAuthClient`, so callers already hold a pointer and nothing in the public API
has to change.

A related trap: because the receiver is a value, `c.Config` cannot be updated by any method, and a
zero-value `OAuthClient{}` constructed directly (bypassing `NewOAuth`) has a nil `lock` and nil
`tokenValue` — it panics on first use rather than erroring.

## Fix

Convert all `OAuthClient` methods to pointer receivers:

```go
func (c *OAuthClient) Post(...)  / Get / Patch / Delete
func (c *OAuthClient) performReq(...)
func (c *OAuthClient) setBearerToken(...)
func (c *OAuthClient) Authenticate() ...
func (c *OAuthClient) AuthToken() string
func (c *OAuthClient) SetToken(...)
func (c *OAuthClient) HasAuthExpired() bool
func (c *OAuthClient) CreateOrderPayment(...)   // order_payments.go
func (c *OAuthClient) GetTransaction(...)       // transactions.go
func (c *OAuthClient) CancelPartialAuthorization(...)
```

Then `tokenValue` and `lock` can become non-pointer (`token` and `sync.RWMutex`), which also makes
a zero-value client usable and removes the nil-panic. Do that as a second commit so the receiver
change is reviewable on its own.

Consider guarding against direct construction by making `NewOAuth` the only sane path — or keep
the pointers and just fix the receivers, which is the smaller change.

## Compatibility

Technically breaking: code doing `var c vivawallet.OAuthClient = *vivawallet.NewOAuth(...)` and
calling methods on the value stops compiling. In practice everyone uses the constructor's pointer.
Worth a note in the changelog, not a major version.

Do the same review for `BasicAuthClient` — it has no mutable state, so value receivers there are
fine, but consistency argues for matching.

## Verification

- `make build-example && make vet && staticcheck`
- A test that calls `Authenticate()` then `AuthToken()` on the same client and asserts the token
  round-trips — and, after converting to non-pointer fields, that it still does.
