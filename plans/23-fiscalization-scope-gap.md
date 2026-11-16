# 23 — Fiscalization proper: the branch name promises more than the email delivers

**Severity:** Medium — a scoping decision to make before calling the branch done
**Files:** none yet — this plan is a decision, not an edit

## Problem

The branch is `feat/fiscalization` and [email-data-fiscalization.md](email-data-fiscalization.md) is
the stated spec. But read closely, the email contains **no fiscalization content**. It covers:

- ISV order creation + Smart Checkout redirect ([20](20-isv-order-payments.md))
- EFT POS Cloud REST ([21](21-eftpos-cloud-rest.md))
- Webhooks ([22](22-webhooks.md))

Fiscalization in the Greek market (myDATA / ΑΑΔΕ receipt issuance, the tax-signature flow that
couples an EFT POS terminal to an ERP under Greek law 4308/2014 and the 2024 POS–cash-register
interconnection mandate) is a different problem: issuing and transmitting fiscal receipts, not
taking payments.

The EFT POS work in [21](21-eftpos-cloud-rest.md) is the *prerequisite* for that interconnection —
which is plausibly why the email was filed under this branch name — but it is not the same thing.

## Decision required

Pick one, and make the branch name and README match:

- **A — the email is the whole scope.** Rename the branch (`feat/isv-payments`), and say in the
  README that fiscal receipt issuance is out of scope. Cleanest; the deliverable then matches what
  the vendor actually specified.
- **B — fiscalization is genuinely wanted.** Then the email is insufficient input and the vendor
  needs a follow-up: which fiscalization provider, whether Viva issues the receipt or the PMS does,
  and what the ΑΑΔΕ transmission path is. Do not infer this from the payment API docs.

Raise this with the vendor before building. The risk is shipping 19–22, marking the branch done,
and discovering the actual compliance requirement was never addressed.

## Related loose end: `cards/`

An empty `cards/` directory is staged on this branch, and the README already documents a
`CreateCardToken` that does not exist — see [16](16-readme-createcardtoken.md). The email does not
mention card tokens. Either implement it or drop both the directory and the README claim; a staged
empty directory and a documented phantom method are exactly the kind of thing that survives a merge
unnoticed.

Note also that a flat package is this SDK's established layout (CLAUDE.md: "Flat package at the repo
root"). A `cards/` subpackage would be the first departure from that and would need `vivawallet.Config`
exported across a package boundary. Prefer `cards.go` at the root unless there is a reason not to.

## Status (2026-11-16)

**Decision: B — fiscalization is wanted.** The email was re-checked line by line against the code.

Already implemented from the email:

| Email item | Code |
|---|---|
| ISV order `POST /checkout/v2/isv/orders` (amount, isvAmount, customer, currencyCode, paymentTimeout, disableCash) | `ISVClient.CreateISVOrder` |
| Smart Checkout URL `web2?ref=…&color=…` | `SmartCheckoutURL`, `WithColor` |
| Order result `GET /api/orders/{code}` | `ISVClient.GetISVOrder` |
| EFT POS sale / refund incl. `isvDetails`, `showReceipt`, `showTransactionResult`, tip excluded from amount | `ISVClient.Sale`, `Refund` |
| Session result `GET /ecr/isv/v1/sessions/{id}` | `ISVClient.GetSession` |
| Webhooks 1796, 1797, 1798, 8193, 8194 | `ParseWebhook` + accessors |

Not implemented, specified by the email:

- **Polling helpers.** Every 3 s, giving up after 10 min for orders and 3 min for terminal sessions.
  The doc comments say "poll it", but callers write the loop themselves. Small and buildable now.

Still missing for fiscalization, and **not in the email**. The only "receipt" (`showReceipt`) is
the terminal's card slip, not a fiscal document. Ask the vendor:

1. Who issues the fiscal receipt (ΑΑΔΕ myDATA): Viva, a certified provider (πάροχος ηλεκτρονικής
   τιμολόγησης), or our PMS?
2. Which provider and which API, with sandbox credentials?
3. How does the receipt get onto the POS transaction for the POS–ERP interconnection
   (διασύνδεση POS–ταμειακής)? Does a field on the Sale request carry the receipt/MARK id, and does
   Viva need to be sent the receipt signature?
4. Is the fiscal step synchronous with payment (before `Sale`) or after (on webhook 1796)?

**Superseded:** Viva's fiscalisation docs answer these questions (Viva signs via fiskaltrust and
transmits to myDATA; `fiscalisationData` rides on the terminal sale or goes to
`/fiscalization/v1/isv/receipts:sign`). The implementation plan is
[25](25-greek-fiscalisation.md).
