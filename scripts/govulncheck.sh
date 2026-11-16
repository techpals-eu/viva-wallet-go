#!/usr/bin/env bash
# Runs govulncheck on the module in each given directory (default: .) and fails only on
# reachable vulnerabilities that are not listed in .govulncheck-ignore at the repo root.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
govulncheck="${GOVULNCHECK_BIN:-govulncheck}"
ignored="$(sed 's/#.*//; s/[[:space:]]//g' "$root/.govulncheck-ignore" | grep -v '^$' || true)"

status=0
for dir in "${@:-.}"; do
  echo "== govulncheck: $dir"
  # JSON mode always exits 0, so failing is up to us. A finding whose first trace frame has
  # a function is reachable from this module's code: the same rule text mode fails on.
  json="$(cd "$dir" && "$govulncheck" -format json ./...)"
  found="$(jq -r 'select(.finding.trace[0].function? != null) | .finding.osv' <<<"$json" | sort -u)"
  for id in $found; do
    if grep -qxF "$id" <<<"$ignored"; then
      echo "ignored: $id (listed in .govulncheck-ignore)"
    else
      echo "::error::$dir: reachable vulnerability $id: https://pkg.go.dev/vuln/$id"
      status=1
    fi
  done
done
exit "$status"
