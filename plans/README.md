# Plans

One file per known issue in the SDK, each self-contained: what is wrong, why it matters,
the change to make, and how to verify it.

Numbering is stable and does not imply order. The suggested order is at the bottom.

## Bugs — broken in normal use

| # | Plan | Severity |
|---|------|----------|
| 01 | [Auth-refresh block always returns an error](01-auth-refresh-always-errors.md) | Critical |
| 02 | [OAuthClient uses value receivers for mutating methods](02-oauth-value-receivers.md) | High (latent) |
| 03 | [BasicAuthClient does not satisfy the Client interface](03-client-interface-unsatisfied.md) | Medium |
| 04 | [API error bodies are discarded](04-typed-api-errors.md) | High |
| 05 | [newRequest swallows http.NewRequest errors](05-newrequest-swallows-error.md) | Medium |
| 06 | [tokenEndpoint builds a double slash](06-token-endpoint-double-slash.md) | Low |
| 07 | [basicauth performReq checks status before read error](07-basicauth-error-ordering.md) | Low |

## Structural

| # | Plan | Severity |
|---|------|----------|
| 08 | [No tests anywhere; `make test` hits the live API](08-add-tests.md) | High |
| 09 | [No context.Context support](09-context-support.md) | High |
| 10 | [Package-level http.Client singleton, no injection](10-client-injection.md) | Medium |
| 11 | [Auth refresh belongs in a RoundTripper](11-auth-roundtripper.md) | Medium |

## Smaller

| # | Plan | Severity |
|---|------|----------|
| 12 | [BalanceTranfer is misspelled and public](12-balancetranfer-typo.md) | Low |
| 13 | [CheckoutOrder.Customer is an anonymous struct](13-named-customer-type.md) | Low |
| 14 | [CancelOrderPayment response type naming](14-cancelorderpayment-naming.md) | Low |
| 15 | [Inconsistent status/state field types](15-status-field-types.md) | Low (needs API check) |
| 16 | [README documents a non-existent CreateCardToken](16-readme-createcardtoken.md) | Low |
| 17 | [gofmt not enforced; three files unformatted](17-enforce-gofmt.md) | Low |
| 18 | [go.mod and CI matrix disagree on minimum Go](18-go-version-floor.md) | Low |

## Features — ISV / fiscalization branch

Derived from [email-data-fiscalization.md](email-data-fiscalization.md). These are new API surface,
not fixes. **19 gates all of them.**

| # | Plan | Severity |
|---|------|----------|
| 19 | [ISV auth: three-part basic auth and an ISV client](19-isv-credentials-and-client.md) | Blocker |
| 20 | [ISV order creation and Smart Checkout redirect](20-isv-order-payments.md) | High |
| 21 | [EFT POS Cloud REST API](21-eftpos-cloud-rest.md) | High |
| 22 | [Webhook types, verification, and a Gin handler](22-webhooks.md) | Medium |
| 23 | [Fiscalization scope gap — decision required](23-fiscalization-scope-gap.md) | Medium |
| 24 | [Docs, example and credential hygiene](24-isv-docs-and-example.md) | Low (one urgent item) |

## Suggested order

1. **01** — the only bug that breaks the SDK in normal use. Do it with a test.
2. **08** — tests, so 01 cannot regress and the rest can be changed safely.
3. **04**, **09** — what consumers actually hit.
4. **11** — subsumes 01 and 02 cleanly if you would rather refactor once.
5. The rest, opportunistically.

Note that **11** is an alternative route to **01** and **02**: if you do 11 first, those two
collapse into it. 01 is still worth doing standalone first, because it is a one-line fix for a
live bug and 11 is a design change.

**09** and **10** are breaking API changes and should be batched into one release.

## Feature order

**23 first** — it is a five-minute conversation that determines whether 19–22 are the right work at
all, and it flags the committed credentials in
[email-data-fiscalization.md](email-data-fiscalization.md) that should not outlive this branch.

Then:

1. **19** — nothing else compiles without it. Resolve the two unknowns it names (the exact ISV
   basic-auth composition, the OAuth scope) against the demo account *before* writing code; both are
   undocumented in the email and unfetchable from Viva's client-rendered docs portal.
2. **20** and **21** in parallel — independent of each other once 19 lands.
3. **22** — parallel with either; the root-module half touches no client code. Its `ginviva/`
   submodule is the only place a third-party dependency enters this repo, and it must stay a
   *separate* Go module — Gin's floor is go 1.20+ (1.25 on master) against this SDK's 1.17.
4. **24** — last, once the surface is settled.

Consider doing **11** (auth RoundTripper) before **19**: a third client otherwise means a third copy
of the auth-refresh preamble, and **09** (context) before **21**, since terminal polling is the
strongest case in the repo for cancellable requests. Both are judgement calls about whether to pay
down the debt now or ship the feature first.
