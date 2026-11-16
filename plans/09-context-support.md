# 09 — No context.Context support

**Severity:** High (breaking change — batch with [10](10-client-injection.md))
**Files:** all of [../oauth.go](../oauth.go), [../basicauth.go](../basicauth.go), [../vivawallet.go](../vivawallet.go), and every endpoint file

## Problem

No method accepts a `context.Context`. Every request is uncancellable and bounded only by the
package-level 60-second timeout on the shared `http.Client`.

## Impact

For a payments SDK that sits in an HTTP request path, this is the most likely thing a serious
consumer asks for:

- An abandoned upstream request cannot cancel its in-flight Viva call; the goroutine and connection
  are held for up to 60s.
- No per-call deadline. A checkout create and a wallet list get the same budget.
- Distributed tracing spans cannot be propagated, since there is no context to carry them.
- `errors.Is(err, context.DeadlineExceeded)` is unavailable for retry logic.

60 seconds is also a very long default for an interactive payment flow.

## Fix

Thread context from the public methods down to `http.NewRequestWithContext`.

**Signature shape.** Two options:

- **A — new signatures (recommended for a v2).** `CreateOrderPayment(ctx, payload)`. Clean, one
  way to do it, breaks every caller.
- **B — additive `...Context` variants.** Keep `GetTransaction(id)` delegating to
  `GetTransactionContext(ctx, id)` with `context.Background()`. Non-breaking, but doubles the API
  surface and the stutter (`GetTransactionContext`) is ugly.

Given the SDK is at v1.0.3 with two known consumers, **A with a major version bump** is cleaner
than carrying B's duplication forever. Batch it with [10](10-client-injection.md) and any other
breaking changes into a single v2.

**Mechanics:**

1. `newRequest` takes a `ctx` and calls `http.NewRequestWithContext` (pairs with
   [05](05-newrequest-swallows-error.md), which is already changing this function — do them
   together).
2. `Get`/`Post`/`Patch`/`Delete` on both clients take `ctx` as the first parameter. Update the
   `Client` interface to match — see [03](03-client-interface-unsatisfied.md), which should be
   folded in here rather than done twice.
3. `Authenticate(ctx)` likewise; the auth-refresh preamble passes the caller's ctx through, so a
   cancelled request also cancels its token refresh.
4. Every endpoint method in [../order_payments.go](../order_payments.go),
   [../transactions.go](../transactions.go), [../wallet.go](../wallet.go) gains `ctx` and forwards
   it.
5. Update [../example/main.go](../example/main.go) and the README examples.

**Timeout policy.** Once callers can pass deadlines, drop the hardcoded 60s `Timeout` on the shared
client, or keep it as a backstop and document it. Do not have both a silent 60s ceiling and a
caller deadline without saying which wins — `http.Client.Timeout` is the shorter-wins minimum, and
that surprises people.

## Migration

Ship a short `MIGRATING.md`: every call gains `ctx` as the first argument; pass
`context.Background()` for identical prior behaviour, or `r.Context()` in an HTTP handler to get
the cancellation benefit.

## Verification

- Test that a cancelled context aborts an in-flight request: stub server sleeps, caller cancels,
  assert the error satisfies `errors.Is(err, context.Canceled)`.
- Test that a deadline shorter than the stub's delay yields `context.DeadlineExceeded`.
- `go test -race ./...`, `make vet`, `staticcheck`.
