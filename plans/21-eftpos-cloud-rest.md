# 21 — EFT POS Cloud REST API (card terminals)

**Severity:** High — the second half of the email
**Depends on:** [19](19-isv-credentials-and-client.md)
**Files:** new `eftpos.go`

## Problem

The PMS needs to drive Viva physical card terminals over HTTPS: send a sale to a terminal, refund,
and poll the session for the outcome. Nothing in the SDK touches `/ecr/*`.

This is a genuinely different API surface from the rest of the SDK — it is asynchronous. A sale
request returns immediately; the actual result arrives only when the customer finishes at the
terminal, which is why the email prescribes polling every 3s for up to 3 minutes.

## Fix

### Unresolved before implementation

Two constants could not be confirmed — the Viva docs portal renders client-side and is not
fetchable, and neither the email nor the public GitHub issues state them:

1. **The base host.** Very likely `(demo-)api.vivapayments.com` (i.e. `ApiUri`), since `/ecr/*` sits
   with the other modern APIs, but **verify against a demo call before shipping**. Wrong host here
   is the silent-bug case CLAUDE.md warns about.
2. **The OAuth scope** for ECR tokens. Pair with the scope work in [19](19-isv-credentials-and-client.md).

Resolve both first; everything below is independent of the answers.

### Endpoints

Paths from the email's doc links. Note the `:verb` colon suffix — this is Google-style custom-method
syntax, it is part of the path and must **not** be percent-encoded. `fmt.Sprintf` is fine here;
`url.PathEscape` on the segment would break it.

| Method | Path | Notes |
|---|---|---|
| `Sale` | `POST /ecr/isv/v1/transactions:sale` | the payload in the email |
| `Refund` | `POST /ecr/isv/v1/transactions:refund` | refund to any card |
| `GetSession` | `GET /ecr/isv/v1/sessions/{sessionId}` | poll for the outcome |

There are merchant-scoped equivalents without the `isv` segment (`/ecr/v1/sessions/{sessionId}`
appears in Viva's own issue tracker). Scope this plan to the ISV variants the email specifies; if
the merchant variants are wanted later, the `isv` segment is the only difference and the URI
builders should take it as a parameter rather than being duplicated.

### Request struct

From the email, all camelCase:

| Field | JSON | Notes |
|---|---|---|
| SessionID | `sessionId` | caller-generated UUID, unique per request |
| TerminalID | `terminalId` | the TID; one terminal per service point |
| CashRegisterID | `cashRegisterId` | which station triggered it |
| Amount | `amount` | minor units (`734` = €7.34) |
| CurrencyCode | `currencyCode` | **string** `"978"` here — unlike the order API's int. Do not share a type between the two |
| MerchantReference | `merchantReference` | ties payment to booking; searchable in e-banking |
| CustomerTrns | `customerTrns` | customer-facing description |
| TipAmount | `tipAmount,omitempty` | **excluded from `amount`** — Viva sums them. Document this on the field; it is the easiest way to double-charge a tip |
| ShowTransactionResult | `showTransactionResult,omitempty` | |
| ShowReceipt | `showReceipt,omitempty` | |
| ISVDetails | `isvDetails,omitempty` | `amount` (ISV fee, *not* added to the sale total) and `terminalMerchantId` |

`sessionId` is caller-supplied and must be unique. The SDK targets go 1.17 and is stdlib-only, so
there is no `uuid` package available and adding a dependency would break the SDK's defining
constraint. Do **not** hand-roll a UUID generator. Leave generation to the caller and say so in the
doc comment; optionally offer `NewSessionID()` built on `crypto/rand` hex if a helper is wanted —
but never `math/rand`, where collisions across restarts are a real failure mode with real money
attached.

### Session response

Field names, confirmed from a real payload in Viva's issue tracker
([VivaPayments/API#2461](https://github.com/VivaPayments/API/issues/2461)):

`sessionId`, `terminalId`, `tid`, `cashRegisterId`, `amount`, `success`, `message`, `orderCode`,
`shortOrderCode`, `referenceNumber`, `retrievalReferenceNumber`, `authorizationId`, `transactionId`,
`transactionTypeId`, `transactionDateTime`, `abortOperation`, `abortSuccess`, `abortAckTime`.

Note `orderCode` and `shortOrderCode` arrive as **strings** (`"0"`) in that payload, while
`OrderCode` elsewhere in this SDK is an `int64` — see [15](15-status-field-types.md). Use the wire
type; do not "fix" it to an int and get an unmarshal error in production.

**Semantics caveat worth a doc comment:** that same issue reports `success: false` on sessions that
settled successfully in the portal. A `false` is therefore not conclusive proof of non-payment —
callers reconciling money should treat a failed/ambiguous session as "verify", not "refund". This
is exactly the kind of thing that belongs in the SDK's documentation rather than being rediscovered.

### Polling

Same position as [20](20-isv-order-payments.md): expose `GetSession` and document the cadence (3s,
3-minute cap per the email). Do not bury a blocking sleep loop in the SDK. Terminal interactions
are the strongest argument in the repo for [09](09-context-support.md) — a customer who walks away
leaves a caller blocked for three minutes with no way to cancel.

## Verification

- httptest per endpoint asserting method, exact path including the `:sale` / `:refund` suffix
  (a test that catches accidental percent-encoding), and auth header.
- A fixture in `testdata/` built from the real session payload above, including the string
  `orderCode`, asserted to unmarshal cleanly.
- Test that `tipAmount` is serialised separately from `amount` and never folded in.
- `go test -race ./...`, `make vet`, `staticcheck`.
