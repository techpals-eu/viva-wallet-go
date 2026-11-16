# 05 — newRequest swallows http.NewRequest errors

**Severity:** Medium
**File:** [../vivawallet.go](../vivawallet.go)

## Problem

```go
func newRequest(method string, uri string, reader *bytes.Reader) *http.Request {
    var req *http.Request
    if reader != nil {
        req, _ = http.NewRequest(method, uri, reader)
    } else {
        req, _ = http.NewRequest(method, uri, nil)
    }
    return req
}
```

Both errors are discarded. `http.NewRequest` fails on an unparseable URL or an invalid method — so
a malformed URI yields a `nil` request, which then panics at `req.Header.Add` inside `performReq`,
several frames from the cause.

## Impact

A nil-pointer dereference in place of an error. The panic message points at `performReq`, not at
the URI builder that produced the bad value — the actual bug. This is the kind of thing that turns
a typo in a new endpoint's `fmt.Sprintf` into a half-hour of confusion.

Realistically triggered by: a `sourceCode` containing characters that break the query string in
`getCancelTransactionUri`/`getCancelPartialAuthUri`, which interpolate it raw.

## Fix

Return the error and propagate it:

```go
func newRequest(method string, uri string, reader *bytes.Reader) (*http.Request, error) {
    var body io.Reader
    if reader != nil {
        body = reader
    }
    req, err := http.NewRequest(method, uri, body)
    if err != nil {
        return nil, fmt.Errorf("viva: build %s request for %s: %w", method, uri, err)
    }
    return req, nil
}
```

Note the `if reader != nil` dance is still required even after collapsing the branches: a typed-nil
`*bytes.Reader` assigned to an `io.Reader` is a non-nil interface, and `http.NewRequest` would then
call `Read` on it. Assigning through the `body io.Reader` local as above is the correct form.

Update all eight call sites (`Get`/`Post`/`Patch`/`Delete` on both clients) to return the error.

## Related

While in here, consider `url.QueryEscape` on the interpolated `sourceCode` in
[../transactions.go](../transactions.go) — `getCancelTransactionUri` and `getCancelPartialAuthUri`
build query strings by raw string concatenation. That is the most likely real-world source of a
malformed URI, and escaping it removes the failure mode rather than just reporting it better.

## Verification

- Unit test: call an endpoint via a client whose config produces a URI with a control character;
  assert a non-nil error and no panic.
- Test `getCancelTransactionUri` with a `sourceCode` containing `&` and a space, asserting the
  result parses with `url.Parse` and the `amount` parameter survives intact.
