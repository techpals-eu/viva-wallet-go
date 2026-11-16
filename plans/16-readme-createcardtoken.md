# 16 — README documents a non-existent CreateCardToken

**Severity:** Low (documentation correctness)
**File:** [../README.md](../README.md)

## Problem

The README has a "Create card token" section under Transactions:

```golang
createCardToken := CreateCardToken{
        TransactionID: "some-trx-id",
}
cardToken, err2 := CreateCardToken(cardToken)
```

Neither `CreateCardToken` the type nor the function exists anywhere in the package. The snippet is
also internally broken: it declares `createCardToken` then passes `cardToken` — an undefined
variable — to a free function that would need to be a method on a client.

## Impact

A user copying this gets a compile error and has no way to tell whether they are holding it wrong
or the docs are wrong. It is the first thing someone evaluating the SDK would try if card tokens
are what they need.

The empty `cards/` directory on the current branch suggests this was written in anticipation of an
implementation that never landed.

## Options

**A. Remove the section** (recommended for now). The README should describe what exists. Removing
is a one-minute change and immediately correct.

**B. Implement it.** Viva exposes card tokenisation at
`GET /acquiring/v1/cards/{transactionId}/token` (OAuth). If card tokens are wanted, implement it in
`cards.go` following the conventions in [../CLAUDE.md](../CLAUDE.md) — note `CheckoutOrder` already
has a `CardTokens []string` field, so the create side is half-present and a retrieve method would
complete the loop.

Pick A now; open an issue for B. Do not leave the broken snippet in place while B is pending.

## Also check while in the README

- The install line says `go get -u github.com/techpals-eu/viva-wallet` — missing the `-go` suffix.
  The module is `github.com/techpals-eu/viva-wallet-go` ([../go.mod](../go.mod)). This one is worse
  than the card-token snippet: it fails at install time, before the user sees anything else.
- The "Supported Endpoints" list should stay in sync as endpoints are added — per CLAUDE.md that is
  part of the convention for a new endpoint.

## Verification

- Every code block in the README compiles when pasted into a `main`. Worth considering a
  `README_test.go` with the examples as real compiled test functions so this cannot rot again.
