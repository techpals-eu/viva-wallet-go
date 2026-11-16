# 17 — gofmt not enforced; three files unformatted

**Severity:** Low
**Files:** [../transactions.go](../transactions.go), [../wallet.go](../wallet.go), [../example/main.go](../example/main.go), [../.github/workflows/ci.yml](../.github/workflows/ci.yml), [../Makefile](../Makefile)

## Problem

`gofmt -l .` currently reports three files. CI runs staticcheck and `go vet` but never checks
formatting, so drift accumulates silently.

Known instances:

- [../wallet.go:65](../wallet.go#L65) — `return nil, err` indented with three extra tabs inside
  `GetWallets`.
- [../transactions.go](../transactions.go) — misaligned struct tag columns in `TransactionResponse`
  (the `Emv` field's `string` column does not line up with the rest).
- [../example/main.go](../example/main.go) — alignment in the `CheckoutOrder` literal.

## Why it matters

Unformatted files make every future diff noisier: an editor with format-on-save reformats the whole
file on an unrelated one-line change, burying the real edit. That is already a live hazard here —
per [../CLAUDE.md](../CLAUDE.md), contributors are told to format their own code but avoid
reformatting whole files, which is an awkward rule that only exists because of this drift.

## Fix

Two steps, in this order, as **two separate commits**:

**1. Format everything.** `gofmt -w .` — a pure-whitespace commit with no logic change, easy to
review and easy to skip with `git blame --ignore-rev`. Add the commit SHA to a
`.git-blame-ignore-revs` file so history stays readable.

**2. Enforce it.** Add to the Makefile:

```make
fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "unformatted files:"; gofmt -l .; exit 1; }
```

And to the CI lint job, before staticcheck:

```yaml
- name: fmt
  run: make fmt-check
```

Add `fmt-check` to the `all` target alongside `vet` and `lint`.

Consider `gofumpt` instead for a stricter, still-gofmt-compatible style — but plain `gofmt` is the
right default for a library others may contribute to.

## Follow-up

Once formatting is enforced, the CLAUDE.md note warning against whole-file reformatting can be
deleted — the problem it works around is gone.

## Verification

- `gofmt -l .` outputs nothing.
- Deliberately misformat a file and confirm CI fails.
