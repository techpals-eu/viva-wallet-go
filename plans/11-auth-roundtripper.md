# 11 — Auth refresh belongs in a RoundTripper

**Severity:** Medium (refactor; subsumes [01](01-auth-refresh-always-errors.md) and [02](02-oauth-value-receivers.md))
**Files:** [../oauth.go](../oauth.go), [../order_payments.go](../order_payments.go), [../transactions.go](../transactions.go)

## Problem

The same auth-refresh preamble is copy-pasted into every OAuth endpoint method:

```go
if c.HasAuthExpired() {
    _, authErr := c.Authenticate()
    return nil, fmt.Errorf("authentication error %s", authErr)
}
```

The existing `// TODO: use RoundTripper to avoid rewriting this` at
[../transactions.go:35](../transactions.go#L35) already identifies the fix — and the TODO is right.

Three copies exist. All three carry bug [01](01-auth-refresh-always-errors.md), because the bug was
copy-pasted along with the block. That is the argument for this refactor in one sentence: a
duplicated concern produced a duplicated bug, and every new OAuth endpoint will inherit both.

Worse, the block is *easy to forget entirely* — a new endpoint that omits it simply sends an
expired token and gets a confusing 401.

## Fix

Move refresh into the transport, so it is impossible to forget and exists once:

```go
type authTransport struct {
    base   http.RoundTripper
    client *OAuthClient
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    if t.client.HasAuthExpired() {
        if _, err := t.client.authenticate(req.Context()); err != nil {
            return nil, fmt.Errorf("viva: refresh access token: %w", err)
        }
    }
    r := req.Clone(req.Context())
    r.Header.Set("Authorization", "Bearer "+t.client.AuthToken())
    return t.base.RoundTrip(r)
}
```

Then delete all three preamble blocks and `setBearerToken`.

**Required care:**

- **Clone the request.** `RoundTrip` must not mutate the request it is given — clone before setting
  the header.
- **No recursion.** The token request must bypass this transport; `Authenticate` uses basic auth
  against a different host. Give it a plain client, or guard on host.
- **Single-flight.** Concurrent requests arriving on an expired token will each trigger a refresh.
  Use `golang.org/x/sync/singleflight` — *but* this repo is deliberately dependency-free
  (empty `go.sum`), so implement it with a `sync.Mutex` held across the refresh plus a
  double-check of expiry after acquiring. Keeping zero dependencies is worth more here than the
  elegance.
- **Refresh early.** Treat the token as expired ~30s before `expires` to avoid losing a race with
  the server's clock. `HasAuthExpired` currently compares against exact expiry.
- **Optional 401 retry.** Retry once on a 401 with a forced refresh, covering server-side
  revocation. Only for idempotent requests — do **not** blind-retry a POST that creates a payment.
  Given the stakes, consider leaving this out initially.

## Relationship to plans 01 and 02

This subsumes both. But **do [01](01-auth-refresh-always-errors.md) first anyway**: it is a
one-line fix for a live bug and should not wait behind a design change. This plan then removes the
code that 01 fixed.

[02](02-oauth-value-receivers.md) becomes mandatory here — the transport holds an `*OAuthClient`
and mutates its token, so pointer receivers are required, not merely advisable.

[09](09-context-support.md) also matters: `RoundTrip` gets `req.Context()`, so refresh inherits the
caller's deadline for free — which is only useful once contexts are threaded through.

## Verification

- Expired-token test: stub token endpoint + stub API; assert exactly one token request and that
  the API request carries the new bearer.
- Concurrency test: 50 parallel calls on an expired token under `-race`; assert the token endpoint
  was hit exactly once.
- Assert the token request itself does not carry a bearer header (no recursion).
