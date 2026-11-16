# CLAUDE.md

Guidance for Claude Code working in this repository.

## What this is

`github.com/techpals-eu/viva-wallet-go` — a dependency-free Go SDK (package `vivawallet`) wrapping
the [Viva Wallet payments API](https://developer.vivawallet.com/apis-for-payments/payment-api/).
Flat package at the repo root; stdlib only (`go.sum` is empty). Targets go 1.17, CI builds 1.16–1.20.

## Commands

```sh
make build-example   # go build -o examp ./example/main.go  — the de-facto compile check
make vet             # go vet ./...
make lint            # staticcheck (must be installed: go install honnef.co/go/tools/cmd/staticcheck@latest)
```

CI (`.github/workflows/ci.yml`) runs staticcheck + vet, and builds the example across five Go versions.

`make test` is **not** a test target — it runs `go run example/main.go -race ./...`, which hits the
live demo API and needs `VIVA_CLIENT_ID`, `VIVA_CLIENT_SECRET`, `VIVA_MERCHANT_ID`, `VIVA_API_KEY`.
There are no `_test.go` files anywhere in the repo.

## Architecture

Viva's API uses two different auth schemes across endpoints, so the SDK exposes **two clients** and a
method lives on whichever client its endpoint requires. This split is forced by the API, not a choice.

- `OAuthClient` ([oauth.go](oauth.go)) — `NewOAuth(clientID, clientSecret, demo)`. Client-credentials
  token from `accounts.vivapayments.com/connect/token`, cached in a mutex-guarded `token` struct.
  Used for the newer `/checkout/v2/*` and `/acquiring/v1/*` endpoints.
- `BasicAuthClient` ([basicauth.go](basicauth.go)) — `NewBasicAuth(merchantID, apiKey, demo)`.
  HTTP basic auth with merchant ID + API key. Used for the older (often doc-flagged "Deprecated")
  `/api/*` endpoints.

[vivawallet.go](vivawallet.go) holds the shared pieces: `Config`, the package-level `httpClient`
(60s timeout), `newRequest`, and the two host helpers that are easy to mix up:

- `ApiUri(c)` → `(demo-)api.vivapayments.com` — for `/checkout/v2`, `/acquiring/v1`
- `AppUri(c)` → `(demo.|www.)vivapayments.com` — for `/api/*`

Picking the wrong one is the most likely silent bug when adding an endpoint. Check the Viva doc URL
for the endpoint and match the host.

### Endpoint file layout

One file per API area, each following the same shape: request/response structs, the method on the
appropriate client, then an unexported `...Uri(c Config, ...)` builder below it.

- [order_payments.go](order_payments.go) — create (OAuth) / get / update / cancel (basic auth)
- [transactions.go](transactions.go) — get (OAuth), create / cancel (basic auth), cancel partial auth (OAuth)
- [wallet.go](wallet.go) — balance transfer, list wallets (basic auth)

## Conventions

When adding an endpoint, mirror the existing files exactly:

- Doc comment on every exported method with a `// Ref: <viva doc URL>` line — this is how the API
  surface is traced back to the docs. Also add it to the README's "Supported Endpoints" list.
- Private URI builder per endpoint taking `Config`, using `ApiUri`/`AppUri`.
- Request structs use `omitempty` on optional fields. Note that JSON casing differs by API era:
  `/checkout/v2` responses are camelCase, `/api/*` responses are PascalCase — copy the casing from
  the actual API, not from a neighbouring struct.
- Errors are wrapped with `fmt.Errorf`; non-200 responses become
  `failed to perform request with status %d` with the body discarded. There is no typed error yet.
- `VERSION` is a plain file, bumped manually.

## Known rough edges

Don't treat these as intentional patterns to copy; fixing one in passing is fine, but they explain
odd-looking code:

- **Auth refresh lives in the transport** ([transport.go](transport.go)): `authTransport` refreshes
  an expired token (single-flight, 30s early) and attaches the bearer. OAuth endpoint methods must
  **not** check `HasAuthExpired` themselves. Token requests go on the plain `Client`, never the
  bearer client, or they would recurse.
- `Authenticate` / `SetToken` are on a **value** receiver but mutate through the `*token` and
  `*sync.RWMutex` pointers, so they do work — the `OAuthClient` value itself is never updated.
- `tokenEndpoint` produces a double slash (`.../%s/` + `/connect/token`).
- `Delete` is on both concrete clients but is missing from the `Client` interface;
  `BasicAuthClient.Patch` has a different signature than `OAuthClient.Patch` (no `v interface{}`),
  so `BasicAuthClient` does not satisfy `Client`.
- `BalanceTranfer` is misspelled and part of the public API — don't rename without a version bump.
- `gofmt` is not enforced by CI and `example/main.go`, `transactions.go`, `wallet.go` are currently
  unformatted. Format code you write; avoid reformatting whole files, which buries the real diff.

## Current branch

`feat/fiscalization` — work in progress toward Viva fiscalization / ISV support. An empty `cards/`
directory is staged (card token endpoints; the README already documents a `CreateCardToken` that
does not exist yet). [email-data-fiscalization.md](email-data-fiscalization.md) is a Greek-language
vendor email holding the spec: ISV order creation at `/checkout/v2/isv/orders` (adds `isvAmount`,
`currencyCode`), Smart Checkout redirect URLs, and the EFT POS Cloud REST protocol. Credentials in
that file are demo/`<tbd>` placeholders — never commit real ones.
