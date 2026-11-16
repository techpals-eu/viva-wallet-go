# 24 — Documentation, example and credential hygiene for the ISV surface

**Severity:** Low individually, but this is what makes 19–22 usable
**Depends on:** [19](19-isv-credentials-and-client.md)–[22](22-webhooks.md)
**Files:** [../README.md](../README.md), [../example/main.go](../example/main.go), [../VERSION](../VERSION)

## Fix

### README

- Add every new method to "Supported Endpoints" with its `// Ref:` doc URL, per CLAUDE.md.
- New "ISV Partners" section: the three-part credential story from
  [19](19-isv-credentials-and-client.md), and that one client serves many merchants.
- Document the full Smart Checkout flow end to end — create order → redirect to
  `web2?ref=<orderCode>` → receive result by webhook or polling. The redirect step is the part a
  new consumer will not guess from a method list.
- State the polling cadences the SDK deliberately does *not* implement (3s/10min for orders,
  3s/3min for EFT POS sessions) so callers know the intended policy.
- Note the webhook verification-key requirement from [22](22-webhooks.md).

### Example

`make build-example` is the de-facto compile check, so anything not referenced there is not
compile-checked in CI. Add ISV usage to [../example/main.go](../example/main.go), gated on new env
vars (`VIVA_RESELLER_ID`, `VIVA_RESELLER_API_KEY`) so the existing flow still runs without them.

Note the existing file is not gofmt-clean ([17](17-enforce-gofmt.md)) — format what you add, do not
reformat the file wholesale, per CLAUDE.md.

### Credential hygiene — do this first, it is the only urgent item here

[email-data-fiscalization.md](email-data-fiscalization.md) is committed in `plans/` and contains a
**real, non-placeholder ISV client ID**:

```
2xjaujh2ars66jpx20x3zygqllbzg8e1rpjz0k53ryep7.apps.vivapayments.com
```

and a real demo merchant id (`393969b6-c18e-4770-ba9a-2838c2beafee`). The secrets are `<tbd>`
placeholders, so nothing is immediately exploitable and this is a demo account — but CLAUDE.md says
plainly: *never commit real ones*. A client id is an identifier, not a secret, yet it does not belong
in a public repo either.

Decide before this branch merges: redact them, or move the email out of the repo entirely into
wherever the team keeps vendor correspondence. If the repo is or will be public, prefer removing the
file — and note that redacting in a later commit does not remove it from history.

Never let real credentials reach the README or the example; keep everything env-var driven, as the
existing example already does.

### Versioning with the new submodule

[22](22-webhooks.md) adds `ginviva/` as a **nested Go module**, which changes how tagging works:
nested modules are released under a path-prefixed tag (`ginviva/v1.2.0`), independently of the root
module's `v1.2.0`. Document that in the README, and note that `go get .../ginviva` needs go 1.20+
while the root SDK stays on 1.17.

Keep the README explicit that the root module remains stdlib-only and that Gin is opt-in — that
property is the reason the submodule exists.

### Version

`VERSION` is bumped by hand. These are additive (new clients, new methods, no signature changes to
existing ones), so a minor bump from `1.1.0` → `1.2.0` fits — unless [19](19-isv-credentials-and-client.md)
takes route A and folds in [11](11-auth-roundtripper.md), or [09](09-context-support.md) lands in the
same release, either of which makes it a v2.
