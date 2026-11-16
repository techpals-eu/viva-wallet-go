# 04 — API error bodies are discarded

**Severity:** High — highest value-per-line change for consumers
**Files:** [../oauth.go](../oauth.go), [../basicauth.go](../basicauth.go)

## Problem

Both clients collapse every non-200 into a status code and throw the body away:

```go
if resp.StatusCode != 200 {
    return fmt.Errorf("failed to perform request with status %d", resp.StatusCode)
}
```

Viva returns the actionable detail in that body — `ErrorCode`, `ErrorText`, `EventId`,
`CorrelationId` (the same shape already modelled in `TransactionResponse` and
`CancelOrderPayment`). A declined card, an invalid amount, and an expired token are all
indistinguishable to a caller today.

Also note `!= 200` rejects `201` and `204`, which some of these endpoints legitimately return.

## Impact

Debugging a failed payment means reproducing it by hand with curl. A consumer cannot branch on
"card declined" vs "bad request" vs "auth expired" without string-matching a status code, and
cannot log the `CorrelationId` that Viva support asks for.

## Fix

Add a typed error in a new `errors.go`:

```go
type APIError struct {
    StatusCode    int
    Body          string
    ErrorCode     int    `json:"ErrorCode"`
    ErrorText     string `json:"ErrorText"`
    EventID       int    `json:"EventId"`
    CorrelationID string `json:"CorrelationId"`
}

func (e *APIError) Error() string {
    if e.ErrorText != "" {
        return fmt.Sprintf("viva: status %d: %s (code %d, correlation %s)",
            e.StatusCode, e.ErrorText, e.ErrorCode, e.CorrelationID)
    }
    return fmt.Sprintf("viva: status %d: %s", e.StatusCode, e.Body)
}
```

In both `performReq` implementations, on a non-2xx: read the body, best-effort
`json.Unmarshal` it into the `APIError` (ignore the unmarshal error — some responses are
plain text or HTML), always keep the raw `Body`, and return it.

Widen the success check from `resp.StatusCode != 200` to `< 200 || > 299`.

Cap the retained body (e.g. `io.LimitReader` at 64KB) so an HTML error page does not end up
wholesale in a log line.

## Consumer-facing shape

```go
var apiErr *vivawallet.APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == 401 { ... }
```

This pairs with [01](01-auth-refresh-always-errors.md) using `%w` — keep wrapping consistent so
`errors.As` reaches through.

## Verification

- `httptest` stub returning 400 with a Viva-shaped error body; assert `errors.As` yields an
  `*APIError` with `ErrorText` and `CorrelationID` populated.
- Stub returning 500 with an HTML body; assert `Body` is retained and `Error()` does not panic.
- Stub returning 204 with an empty body; assert it is treated as success, not an error.
