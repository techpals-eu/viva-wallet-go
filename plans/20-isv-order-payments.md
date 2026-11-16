# 20 — ISV order creation and Smart Checkout redirect

**Severity:** High — the primary deliverable of the email
**Depends on:** [19](19-isv-credentials-and-client.md)
**Files:** new `isv_orders.go`, [../README.md](../README.md)

## Problem

The email's main ask: create a payment order as an ISV partner, redirect the customer to Smart
Checkout, then poll for the result. None of it exists.

`CreateOrderPayment` in [../order_payments.go](../order_payments.go) is close but not reusable
as-is: it posts to `/checkout/v2/orders`, and `CheckoutOrder` has no `isvAmount` or `currencyCode`
field.

## Fix

### 1. `CreateISVOrder`

```go
// Ref: https://developer.viva.com/isv-partner-program/payment-isv-api/
func (c *ISVClient) CreateISVOrder(merchantID string, payload ISVCheckoutOrder) (*CheckoutOrderResponse, error)
```

`POST {ApiUri}/checkout/v2/isv/orders` — note **`ApiUri`**, not `AppUri`; this is a `/checkout/v2`
path, matching the email's `https://demo-api.vivapayments.com/checkout/v2/isv/orders`.

**Struct.** Do not extend `CheckoutOrder` — it is public API used by the non-ISV path, and adding
ISV-only fields to it would put `isvAmount` on ordinary checkout orders. Define a separate
`ISVCheckoutOrder`. It duplicates most of `CheckoutOrder`, which is acceptable: the two are
independently versioned by Viva.

Fields the email documents, in camelCase (`/checkout/v2` era):

| Field | JSON | Notes |
|---|---|---|
| Amount | `amount` | required, minor units |
| ISVAmount | `isvAmount,omitempty` | the ISV commission. **Must not exceed `amount`** — the API rejects it |
| CustomerTransactions | `customerTrns,omitempty` | shown on Smart Checkout |
| Customer | `customer,omitempty` | email, fullName, phone, countryCode, requestLang |
| CurrencyCode | `currencyCode,omitempty` | `978` for EUR — **int**, see below |
| PaymentTimeout | `paymentTimeout,omitempty` | seconds; email uses 600 |
| DisableCash | `disableCash,omitempty` | |

**Currency code type trap.** The email uses `"currencyCode": 978` (int) for the order API but
`"currencyCode": "978"` (string) for EFT POS ([21](21-eftpos-cloud-rest.md)), and
`GetTransactionResponse.CurrencyCode` in [../transactions.go](../transactions.go) is already a
`string`. Do not assume they match. Copy the casing and type from the actual API per endpoint, as
CLAUDE.md requires. Add a `CurrencyEUR` constant so callers are not typing `978` by hand.

Use a named `Customer` type here rather than the anonymous struct `CheckoutOrder` uses — see
[13](13-named-customer-type.md); this is a new struct, so there is no reason to repeat that mistake.

### 2. Smart Checkout URL builder

The email is explicit that the redirect URL is static base + dynamic order code:
`https://demo.vivapayments.com/web2?ref=<orderCode>&color=<hex>`.

```go
// SmartCheckoutURL returns the URL to redirect a customer to in order to complete payment.
func SmartCheckoutURL(c Config, orderCode int64, opts ...CheckoutURLOption) string
```

- Host is **`AppUri`** (`demo.vivapayments.com` / `www.vivapayments.com`), not `ApiUri`. This is
  the exact `ApiUri`/`AppUri` mix-up CLAUDE.md flags as the most likely silent bug.
- `color` is a per-redirect hex value, so it belongs as an option, not a client-level setting.
  Accept it with or without a leading `#` and strip it; `#` in a query value is a fragment
  delimiter and would silently truncate the URL.
- Build with `net/url` and `url.Values.Encode()`, not `fmt.Sprintf`.

This is a pure function with no I/O — the cheapest thing in the whole feature set to test
exhaustively, and worth doing so.

### 3. Order status polling

`GET /api/orders/{orderCode}` under ISV auth. Note this is an **`/api/*`** path → **`AppUri`**, and
a *different host to the creation call*. Confirm against the doc link in the email
(`~1api~1orders~1{order_code}/get`) before wiring it.

Expose the single fetch as `GetISVOrder(merchantID string, orderCode int64)`. **Do not build the
polling loop into the SDK** as a blocking `for` with `time.Sleep`. The email suggests 3s intervals
up to a 10-minute timeout; that is a caller policy decision, and an uncancellable 10-minute sleep
inside a dependency-free SDK is hostile in a request path. Document the recommended cadence in the
README and let callers own the loop — or, if a helper is wanted, gate it behind
[09](09-context-support.md) so it takes a `context.Context` and is cancellable.

Prefer webhooks over polling where possible — see [22](22-webhooks.md).

## Verification

- httptest: asserts path `/checkout/v2/isv/orders`, method POST, and that the marshalled body
  carries `isvAmount` and `currencyCode` with the right types.
- Table test for `SmartCheckoutURL`: demo vs production host, color with and without `#`, no color,
  and that the order code lands in `ref`.
- A test asserting `isvAmount > amount` is *not* silently sent — decide whether the SDK validates
  client-side or lets the API reject; document whichever.
- Add both methods to the README "Supported Endpoints" list with their `// Ref:` doc URLs.
