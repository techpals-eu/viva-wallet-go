# 14 — CancelOrderPayment response type naming

**Severity:** Low
**File:** [../order_payments.go](../order_payments.go)

## Problem

```go
type CancelOrderPayment struct { OrderCode int64; ErrorCode int; ... }

func (c BasicAuthClient) CancelOrderPayment(orderCode int64) (*CancelOrderPayment, error)
```

The method and its return type share a name, and the type is a *response* while every other
response type in the package carries the `...Response` suffix:

- `CheckoutOrderResponse`, `GetOrderPaymentResponse`, `GetTransactionResponse`,
  `TransactionResponse`, `BalanceTransferResponse`, `TokenResponse`

`CancelOrderPayment` is the odd one out, and its name reads like a request payload — which is
actively misleading, since the call takes no body.

## Fix

```go
type CancelOrderPaymentResponse struct { ... }

// Deprecated: use CancelOrderPaymentResponse.
type CancelOrderPayment = CancelOrderPaymentResponse
```

A **type alias** (`=`) keeps existing code compiling and is exactly the right tool: the two names
denote the identical type, so callers holding a `*CancelOrderPayment` are unaffected. Drop the
alias in v2.

## Related observation

This struct duplicates the error envelope — `ErrorCode`, `ErrorText`, `TimeStamp`,
`CorrelationID`, `EventID`, `Success` — that also appears in `TransactionResponse`. Once
[04](04-typed-api-errors.md) introduces a typed error carrying those fields, consider whether these
response structs should embed a shared envelope type instead of repeating it:

```go
type ResponseEnvelope struct {
    ErrorCode     int       `json:"ErrorCode,omitempty"`
    ErrorText     string    `json:"ErrorText,omitempty"`
    TimeStamp     time.Time `json:"TimeStamp,omitempty"`
    CorrelationID string    `json:"CorrelationId,omitempty"`
    EventID       int       `json:"EventId,omitempty"`
    Success       bool      `json:"Success,omitempty"`
}
```

Embedding preserves the JSON shape and field access (`resp.ErrorText` still works), so this is
non-breaking. Worth doing alongside 04 rather than separately.

## Verification

- `make build-example && make vet && staticcheck`
- Confirm the alias compiles by keeping a reference to the old name in a test.
