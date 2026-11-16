# 10 — Package-level http.Client singleton, no injection

**Severity:** Medium (breaking if done via constructors — batch with [09](09-context-support.md))
**File:** [../vivawallet.go](../vivawallet.go)

## Problem

```go
var httpClient *http.Client

func init() {
    httpClient = &http.Client{Timeout: defaultTimeout}
}
```

Both constructors hand this single shared instance to every client they create. Neither
`NewOAuth` nor `NewBasicAuth` accepts an alternative.

## Impact

- **No per-consumer tuning.** Timeout, transport, connection pool sizing, and proxy settings are
  fixed at 60s / `http.DefaultTransport` for everyone.
- **No instrumentation.** A wrapping `RoundTripper` for metrics, tracing, or retries cannot be
  installed.
- **Shared mutable global.** Two `vivawallet` clients in one process share a client; anyone
  mutating `c.Client` affects the other. The `init()` + package var is also unnecessary — a
  composite literal at the constructor would do.
- **Tests must mutate the exported field** after construction, which works but is undiscoverable
  and racy if done concurrently. See [08](08-add-tests.md).

The `Client` field being exported is the current escape hatch, but nothing documents it and it
cannot be set atomically at construction.

## Fix

Adopt functional options — it keeps one constructor per auth type and stays extensible:

```go
type Option func(*options)

func WithHTTPClient(hc *http.Client) Option
func WithTimeout(d time.Duration) Option
func WithBaseURLs(api, app, auth string) Option   // also unblocks plan 08

func NewOAuth(clientID, clientSecret string, demo bool, opts ...Option) *OAuthClient
func NewBasicAuth(merchantID, apiKey string, demo bool, opts ...Option) *BasicAuthClient
```

Variadic options are **not** a breaking change to existing call sites, so this part can ship
before v2. `WithBaseURLs` is what lets [08](08-add-tests.md) point the SDK at an `httptest`
server without exported-field mutation.

Drop the `init()` and give each client its own default `&http.Client{Timeout: defaultTimeout}`
unless one is supplied. Keep the `Client` field exported for now, but document that
`WithHTTPClient` is the supported path.

Consider whether the `demo bool` positional argument should become `WithDemo()` in v2 — it reads
poorly at call sites (`NewOAuth(id, secret, true)`) and is easy to invert by accident, which for
this SDK means sending real payments at a test.

## Interaction

- [09](09-context-support.md): once callers pass deadlines, the client-level timeout becomes a
  backstop. Decide the policy in one place.
- [11](11-auth-roundtripper.md): the auth `RoundTripper` wraps whatever transport the injected
  client carries — ordering matters, so implement it as a wrapper around `hc.Transport`, not a
  replacement.

## Verification

- Test that `WithHTTPClient` is actually used (stub client records the call).
- Test that two clients constructed without options do not share an `*http.Client`.
- Existing call sites in [../example/main.go](../example/main.go) compile unchanged.
