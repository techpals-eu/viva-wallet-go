# 08 — No tests anywhere; `make test` hits the live API

**Severity:** High — blocks safe work on every other plan
**Files:** new `*_test.go`, [../Makefile](../Makefile), [../.github/workflows/ci.yml](../.github/workflows/ci.yml)

## Problem

There are no `_test.go` files in the repository. The `test` target does not run tests:

```make
test:
	go run example/main.go -race ./...
```

That runs the example program against the **live demo API**, requires four environment variables,
creates real demo orders, and passes `-race ./...` as arguments to the program rather than to the
compiler — where they are ignored. It cannot fail meaningfully and cannot run in CI.

CI (`.github/workflows/ci.yml`) runs staticcheck, vet, and a build. Nothing executes code.

## Why this is the highest-leverage structural fix

Bug [01](01-auth-refresh-always-errors.md) is a one-line error that has been live across several
releases. One test would have caught it the day it was written. Every other plan here —
particularly [04](04-typed-api-errors.md), [09](09-context-support.md), and
[11](11-auth-roundtripper.md) — changes shared request plumbing, and there is currently no way to
verify those changes short of running payments by hand.

This package is unusually easy to test: it is pure URI construction, header setting, and JSON
decoding, with a single injectable `*http.Client`.

## Plan

**1. Make the client injectable.** Depends on [10](10-client-injection.md) — or, as an interim,
tests can overwrite the exported `Client` field after construction:

```go
c := vivawallet.NewOAuth("id", "secret", true)
c.Client = srv.Client()
```

That works today. It does not solve the base-URL problem below, though.

**2. Solve base URLs.** `ApiUri`/`AppUri` hardcode Viva's hosts, so a stub server cannot be
targeted. Two options:

- *Preferred:* add unexported `apiBase`/`appBase`/`authBase` fields to `Config`, defaulting to the
  current constants, and have the three URI helpers prefer them when set. Tests set them via an
  internal test helper in the same package. No public API change.
- *Alternative:* keep tests in-package (`package vivawallet`) and stub via a package-level
  overridable var. Simpler, slightly grubbier.

**3. Table tests for URI builders.** Cheapest, highest-signal tests in the repo. For every
`...Uri` function, assert demo and production output. This pins the `ApiUri` vs `AppUri` choice
per endpoint — the documented easiest-to-get-wrong thing when adding an endpoint.

**4. Round-trip tests per endpoint.** With `httptest.NewServer`, for each public method assert:
method and path, `Authorization` header (Bearer vs Basic), `Content-Type`, request body JSON
matches the struct's tags, and the response decodes into the right fields. Use real captured Viva
payloads as fixtures under `testdata/` — this is where the camelCase/PascalCase split gets locked
down.

**5. Auth-specific tests.** The regression test for [01](01-auth-refresh-always-errors.md):
expired token → stub issues a new one → request carries the new bearer. Plus token-endpoint
failure propagation, and a concurrency test (`t.Parallel` + `-race`) exercising
`AuthToken`/`SetToken` under load to justify the mutex.

**6. Fix the Makefile:**

```make
test:
	go test -race ./...

example:
	go run ./example
```

Rename the live-API run to `example` so it stays available without masquerading as a test suite.
Document the four required env vars next to it.

**7. Add to CI:** a `test` job running `go test -race ./...` on the matrix. Needs no secrets,
because nothing touches the network.

## Scope note

Aim for the URI table tests plus the auth regression test first — that is a couple of hours and
covers the live bug. Full per-endpoint coverage can follow incrementally.

## Verification

- `go test -race ./...` green locally and in CI.
- Deliberately reintroduce bug 01 and confirm the suite goes red.
