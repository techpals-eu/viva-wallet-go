# 07 — basicauth performReq checks status before read error

**Severity:** Low (consistency / correctness-adjacent)
**Files:** [../basicauth.go](../basicauth.go), [../oauth.go](../oauth.go)

## Problem

The two clients order their error handling differently.

`basicauth.go`:

```go
body, bodyErr := io.ReadAll(resp.Body)

if resp.StatusCode != 200 {
    return nil, fmt.Errorf("failed to perform request with status %d", resp.StatusCode)
}

return body, bodyErr
```

`oauth.go`:

```go
body, bodyErr := io.ReadAll(resp.Body)
if bodyErr == nil {
    resp.Body.Close()
}
if bodyErr != nil {
    return bodyErr
}
if resp.StatusCode != 200 {
    return fmt.Errorf(...)
}
```

Two separate issues:

1. **Ordering differs.** In `basicauth`, a read failure on a non-200 is masked by the status error;
   in `oauth`, the read error wins. Neither is wrong per se, but they should agree.
2. **`oauth.go` leaks the body on read error.** `resp.Body.Close()` is called only when
   `bodyErr == nil`. A read error therefore leaks the connection — exactly backwards, and the one
   case where closing matters most. `basicauth.go` correctly uses `defer resp.Body.Close()`.

## Fix

Standardise both on the `basicauth` close pattern and the `oauth` ordering:

```go
defer resp.Body.Close()

body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
if err != nil {
    return nil, fmt.Errorf("viva: read response body: %w", err)
}

if resp.StatusCode < 200 || resp.StatusCode > 299 {
    return nil, newAPIError(resp.StatusCode, body)   // see plan 04
}

return body, nil
```

This is best done *as part of* [04](04-typed-api-errors.md), which rewrites these same few lines —
doing them separately means touching the same block twice.

## Verification

- Stub server that closes the connection mid-body; assert an error mentioning the read failure and
  no goroutine/connection leak (`httptest.Server.Close()` blocks on outstanding connections, so a
  leaked body surfaces as a hung test — a useful natural assertion).
- Existing endpoint tests continue to pass unchanged.
