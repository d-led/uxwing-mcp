#!/usr/bin/env bash
# Fail when any function scores above 10 on CRAP: cyclomatic complexity
# weighted by how much of the function the tests cover.
#
# Requires go-crap: go install github.com/padiazg/go-crap@latest
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v go-crap >/dev/null; then
  echo "go-crap is not installed: go install github.com/padiazg/go-crap@latest" >&2
  exit 1
fi

if [[ ! -f build/coverage.out ]]; then
  scripts/test.sh >/dev/null
fi

go-crap scan --coverage-profile build/coverage.out --threshold 10 --fail-above
