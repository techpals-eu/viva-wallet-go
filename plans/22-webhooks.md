# 22 — Webhook payload types, verification, and a Gin handler

**Severity:** Medium — recommended by the email, and the correct alternative to polling
**Depends on:** nothing (can be done in parallel with [20](20-isv-order-payments.md)/[21](21-eftpos-cloud-rest.md))
**Files:** new `webhooks.go`; new **`ginviva/` submodule** (own `go.mod`)

## Problem

The email recommends five webhooks. The SDK has no webhook support at all, so every consumer writes
their own structs and their own verification.

| EventTypeID | Event |
|---|---|
| 1796 | Transaction Payment Created |
| 1797 | Transaction Reversal Created |
| 1798 | Transaction Failed |
| 8193 | Account Connected |
| 8194 | Account Verification Status Changed |

The consuming PMS is a **Gin** application, so the SDK should carry the Gin integration rather than
leaving every consumer to rewrite the same handler and rediscover the `GET` verification handshake.

## The dependency problem — read this before writing any code

Gin cannot go in the root module. This is not a style preference:

- The SDK declares `go 1.17` and CI builds **1.17, 1.18, 1.19, 1.20**
  ([../.github/workflows/ci.yml](../.github/workflows/ci.yml)).
- Gin's own floor has moved well past that: v1.9.1/v1.10.0 require **go 1.20**, v1.11 requires
  **1.23**, and current master declares **go 1.25**.
- Gin pulls ~15 direct dependencies — `validator/v10`, `sonic`, `quic-go`,
  `go.mongodb.org/mongo-driver/v2`, `protobuf`, `golang.org/x/net`.

Adding it to the root module would end the SDK's defining property — `go.sum` is empty, stdlib only,
per CLAUDE.md — and would force every existing consumer, including ones that never touch webhooks,
onto go 1.25 and a MongoDB driver to create a payment order. That is not an acceptable trade for a
convenience handler.

### Fix: a separate Go module in a subdirectory

```
viva-wallet-go/
├── go.mod              module github.com/techpals-eu/viva-wallet-go        (go 1.17, stdlib only)
├── webhooks.go         parsing + types — no Gin import
└── ginviva/
    ├── go.mod          module github.com/techpals-eu/viva-wallet-go/ginviva  (go 1.23, requires gin)
    └── handler.go
```

A nested module is **excluded from the parent's build** — `go build ./...` at the root does not
descend into a directory containing its own `go.mod`. So the root module's dependency graph stays
empty, 1.17 consumers are unaffected, and Gin users add one import path:

```
go get github.com/techpals-eu/viva-wallet-go/ginviva
```

The submodule depends on the parent via a normal `require` plus a `replace ../` for local
development. **Versioning caveat:** nested modules are tagged `ginviva/v1.2.0`, not `v1.2.0` — a
detail worth writing into [24](24-isv-docs-and-example.md), since `VERSION` is bumped by hand and
the two now move independently.

Pick the Gin version deliberately. **v1.10.0 (go 1.20)** is the lowest that is still maintained and
keeps the submodule close to the parent's era; current gin's go 1.25 floor is a large ask for a
webhook handler. Decide and pin it, rather than taking whatever `go get` resolves.

## Fix — part 1: root module, no Gin

Everything here stays framework-agnostic and stdlib-only. A consumer on Echo, chi or `net/http`
gets full webhook support without the submodule.

### 1. Event constants

```go
const (
    EventTransactionPaymentCreated        = 1796
    EventTransactionReversalCreated       = 1797
    EventTransactionFailed                = 1798
    EventAccountConnected                 = 8193
    EventAccountVerificationStatusChanged = 8194
)
```

### 2. Envelope + per-event payload structs

Viva webhooks share an envelope (`EventData`, `Created`, `CorrelationId`, `EventTypeId`,
`Delay`, `MessageId`, `RecipientId`, `MessageTypeId`) with the event-specific body in `EventData`.

Casing here is **PascalCase** on the envelope — webhooks follow the older `/api/*` convention, not
`/checkout/v2` camelCase. Verify each payload against the doc pages linked in the email rather than
copying from a neighbouring struct; CLAUDE.md is explicit that casing differs by API era and this
is the easiest place in the codebase to get it wrong.

Parse in two steps, since `EventData` differs per event:

```go
// ParseWebhook decodes the envelope, leaving EventData as json.RawMessage.
func ParseWebhook(body []byte) (*WebhookEvent, error)

// Typed accessors: func (e *WebhookEvent) TransactionPaymentCreated() (*TransactionPaymentCreatedData, error)
```

Using `json.RawMessage` for `EventData` means an unknown or newly-added event type does not fail the
whole parse — important, because a webhook endpoint that 500s on an unrecognised event will have
Viva retrying it forever.

**Do not use Gin's `c.ShouldBindJSON` for this**, even in the submodule. It would put
`validator/v10` struct tags on the payload types and move parsing into the framework layer, so the
same types could no longer be used by a non-Gin consumer. Read the body, call `ParseWebhook`.

### 3. Webhook verification key

Viva verifies webhook endpoint ownership with a `GET` to the endpoint returning a `Key` fetched from
`/api/messages/config/token` (basic auth, `AppUri`). Add:

```go
// Ref: https://developer.viva.com/webhooks-for-payments/
func (c BasicAuthClient) GetWebhookVerificationKey() (*WebhookVerificationKey, error)
```

Consumers must return that key as JSON from a `GET` on their webhook URL or Viva will not activate
it. It is the single most common reason a Viva webhook setup fails, and it is not obvious from the
payload docs — which is the main argument for shipping the Gin handler at all, since the handler can
simply do it for them.

Confirm the exact path against the docs before wiring; it is an `/api/*` path, so **`AppUri`**.

## Fix — part 2: the `ginviva` submodule

### Handler

```go
package ginviva

// Handler serves both halves of a Viva webhook endpoint: the GET ownership
// handshake and the POST event delivery. Register it on one route:
//
//	h := ginviva.NewHandler(client, ginviva.Config{
//	    OnPaymentCreated: func(c *gin.Context, d *vivawallet.TransactionPaymentCreatedData) error { ... },
//	})
//	r.Any("/webhooks/viva", h.Handle)
func NewHandler(c *vivawallet.BasicAuthClient, cfg Config) *Handler
func (h *Handler) Handle(c *gin.Context)
```

`r.Any` on a single route is the point: both verbs must live at the same URL, which is precisely
what consumers get wrong when wiring this by hand.

**Cache the verification key.** Fetch it once on first `GET` (or eagerly at construction) and store
it behind a mutex — do not call Viva's API on every `GET`. Viva re-verifies periodically, and an
endpoint that makes an authenticated upstream call per probe is a self-inflicted rate limit.

### Handler semantics that matter

These are the decisions that make the difference between a handler that works and one that causes
an overnight retry storm:

- **Unknown `EventTypeId` → 200, not 404.** Viva retries non-2xx. An event type you have no callback
  for is not an error; log it and acknowledge.
- **A callback returning an error → 500**, so Viva retries. This is the one case where retry is
  wanted. Make the callback signature return `error` so consumers can express "I failed, send it
  again" — a `func(...)` with no return cannot.
- **Malformed JSON → 400 and no retry.** A body that will never parse should not be retried forever.
- **Do not run callbacks in a goroutine.** The handler must not return 200 before the consumer has
  durably handled the event, or a crash loses it silently.
- **Idempotency stays with the consumer.** Viva can deliver the same event more than once; expose
  `MessageId` on the envelope and say in the doc comment that consumers must dedupe on it. The SDK
  has no storage and must not pretend to.

### What the submodule must not do

No storage, no dedupe, no business logic, no `gin.Engine` construction. It adapts Gin's request and
response to `ParseWebhook` and the verification key — nothing more. Everything it does should remain
expressible in ~30 lines of consumer code against the root module alone; if it starts needing more,
the boundary has drifted.

## Verification

- **Root module:** fixtures in `testdata/` for each of the five event types, asserting the envelope
  parses and the typed accessor returns the right fields; a test with an unknown `EventTypeId`
  asserting `ParseWebhook` succeeds rather than erroring; a test that a typed accessor called for
  the wrong event type returns an error rather than a zero-valued struct.
- **Submodule** (`gin.SetMode(gin.TestMode)` + `httptest`): `GET` returns the key JSON; `POST` with
  a known event invokes the right callback; unknown event type returns 200; callback error returns
  500; malformed body returns 400; the key is fetched once across several `GET`s.
- **CI:** add a job for `ginviva` — the root matrix must not try to build it, since 1.17–1.19 cannot
  compile Gin. Run it on the submodule's own Go version, in its own directory. Also assert the root
  module stays clean — `[ "$(go list -m all | wc -l)" -eq 1 ]`, which holds today and fails the day
  someone adds a dependency to the root. (Use the *module* list, not `go list -deps`: that reports
  the standard library's own vendored `golang.org/x/...` packages and is never empty.)
- `go test -race ./...` in both modules, `make vet`, `staticcheck`.
