# 06 — tokenEndpoint builds a double slash

**Severity:** Low (cosmetic; currently harmless)
**File:** [../oauth.go](../oauth.go)

## Problem

```go
func (c OAuthClient) tokenEndpoint() string {
    return fmt.Sprintf("%s/%s", c.authUri(), "/connect/token")
}
```

The format string supplies a `/` and the argument begins with another, producing
`https://accounts.vivapayments.com//connect/token`.

## Impact

None observed — Viva's server evidently normalises it, and the example authenticates successfully.
But it is a latent dependency on server-side leniency: a gateway change, a redirect, or a proxy
that normalises differently could break auth with a confusing 404. It also looks like a bug to
every reader, costing attention each time.

## Fix

```go
func (c *OAuthClient) tokenEndpoint() string {
    return c.authUri() + "/connect/token"
}
```

## Verification

- `make build-example`
- If [08](08-add-tests.md) has landed: assert the stub server's recorded request path is exactly
  `/connect/token`. That test is worth writing regardless — it pins the endpoint shape.
- Manually: run the example against demo with real credentials and confirm auth still succeeds.

## Note

Check the other URI builders for the same pattern while here. As of writing the rest are correct —
they use `fmt.Sprintf("%s/api/...", AppUri(c))` with no leading slash on the literal — but it is a
cheap sweep.
