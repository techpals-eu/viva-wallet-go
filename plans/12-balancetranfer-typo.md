# 12 — BalanceTranfer is misspelled and public

**Severity:** Low
**File:** [../wallet.go](../wallet.go)

## Problem

```go
func (c BasicAuthClient) BalanceTranfer(...)   // "Tranfer" — missing the 's'
```

It is an exported method at v1.0.3, so it is part of the published API surface. It cannot simply be
renamed without breaking any importer.

The request struct `BalanceTransfer` *is* spelled correctly, which makes the pair especially
confusing — and means the corrected method name would collide with the struct name.

## Fix

That collision is the real design question here. Options:

**A. Rename the method to `TransferBalance` (recommended).** Verb-first reads better for an
action, and it sidesteps the struct-name collision entirely:

```go
// TransferBalance transfers money from one wallet to another.
func (c *BasicAuthClient) TransferBalance(walletID, targetWalletID string, payload BalanceTransfer) (*BalanceTransferResponse, error)

// Deprecated: use TransferBalance. Retained for compatibility; will be removed in v2.
func (c *BasicAuthClient) BalanceTranfer(walletID, targetWalletID string, payload BalanceTransfer) (*BalanceTransferResponse, error) {
    return c.TransferBalance(walletID, targetWalletID, payload)
}
```

**B. Rename the struct** to `BalanceTransferRequest`, freeing `BalanceTransfer` for the method.
More churn, and breaks callers who construct the struct — which is all of them.

Go with A. Keep the deprecated wrapper through v1, drop it in the v2 that carries
[09](09-context-support.md). Use the `// Deprecated:` prefix exactly — staticcheck and editors key
off that form.

## Verification

- `make build-example && staticcheck`
- [../example/main.go](../example/main.go) does not currently call this method; consider adding a
  call so the path gets exercised.
