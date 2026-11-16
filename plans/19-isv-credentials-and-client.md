# 19 — ISV auth: three-part basic auth and an ISV-scoped OAuth client

**Severity:** Blocker for the whole fiscalization feature set
**Files:** [../vivawallet.go](../vivawallet.go), [../basicauth.go](../basicauth.go), [../oauth.go](../oauth.go), [../options.go](../options.go), new `isv.go`

## Problem

Every other plan in the 19–24 range depends on this one, so it goes first.

The ISV endpoints do not fit either existing client:

1. **ISV basic auth needs three credentials, not two.** Viva's ISV credentials page describes the
   ISV auth structure as *Reseller ID + Merchant ID + Reseller API Key*. `BasicAuthClient` holds
   exactly two fields (`MerchantID`, `APIKey`) and `performReq` calls
   `req.SetBasicAuth(c.Config.MerchantID, c.Config.APIKey)`. There is nowhere to put the Reseller
   ID, and the merchant id is *per-call variable* for an ISV — the email notes
   `Merchant ID: 393969b6-... (Είναι μεταβλητό)`, i.e. one ISV client serves many merchants.
2. **The OAuth client is single-tenant too.** ISV order creation uses the ISV partner's own
   client credentials, but the order belongs to a merchant.

Bolting `ResellerID` onto the existing `Config` and mutating it per call is not viable: clients are
shared across goroutines, and `Config` is copied by value into every `...Uri` builder.

## Fix

**Do not widen `BasicAuthClient`.** Add a third client in a new `isv.go`:

```go
// ISVClient talks to Viva's ISV Partner APIs on behalf of a merchant. One client serves
// many merchants: the merchant id is a per-call argument, never client state.
type ISVClient struct {
    Config Config
    Client *http.Client

    lock       sync.RWMutex
    tokenValue token
}

func NewISV(resellerID, resellerAPIKey, clientID, clientSecret string, demo bool, opts ...Option) *ISVClient
```

Add `ResellerID` and `ResellerAPIKey` to `Config` alongside the existing credential fields.

**Open question to resolve before writing the auth header** — the exact basic-auth composition.
The docs say "Reseller ID + Merchant ID + Reseller API Key" without spelling out the encoding. The
two candidates are:

- username `resellerID:merchantID`, password `resellerAPIKey`
- username `resellerID`, password `merchantID:resellerAPIKey`

Confirm against a live demo call before committing — the email's `Username = <tdb:tdb>` hint (note
the colon *inside* the username placeholder) points at the first form, but that is inference, not
documentation. Isolate it in one function so a wrong guess is a one-line fix:

```go
func isvBasicAuth(c Config, merchantID string) (username, password string)
```

Mirror the existing `BasicAuth(c Config)` helper's placement in [../oauth.go](../oauth.go).

**Reuse, do not duplicate.** `ISVClient`'s `Get`/`Post`/`Delete` and its token handling are the
same code as `OAuthClient`'s. Two options:

- **A (recommended if [11](11-auth-roundtripper.md) is done first):** the auth difference becomes a
  `RoundTripper`, and `ISVClient` is `OAuthClient` with a different transport plus a merchant-id
  argument on each method. Least duplication.
- **B (standalone):** extract the token cache into an embedded unexported `tokenCache` struct that
  both `OAuthClient` and `ISVClient` embed, so `Authenticate`, `AuthToken`, `SetToken` and
  `HasAuthExpired` are written once.

Prefer **A**. This is the point where [11](11-auth-roundtripper.md) stops being a nice-to-have: a
third client would triple the copy-pasted auth-refresh preamble that [01](01-auth-refresh-always-errors.md)
already had to fix in three places.

**OAuth scope.** The ISV and EFT POS OAuth flows are documented as using scoped tokens. The scope
string could not be confirmed from the docs portal (it renders client-side and is not fetchable).
Resolve it against the demo account, then add a `scope` field to the token request in
`Authenticate` — currently the body is a bare `grant_type=client_credentials`. Make the scope a
field on the client, not a constant, since ISV and ECR ([21](21-eftpos-cloud-rest.md)) need
different ones.

## Verification

- `isvBasicAuth` unit test asserting the exact header for known inputs, so the composition is
  pinned once confirmed.
- httptest server asserting `Authorization` on both an ISV basic-auth call and a bearer call.
- A test with two different merchant ids through one client, asserting each request carries its
  own merchant id — this is the property that makes ISV clients concurrency-safe.
- `go test -race ./...`, `make vet`, `staticcheck`.
