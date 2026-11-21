#!/usr/bin/env bash
# Run the test suite.
#
# Usage:
#   scripts/test.sh            # readable output, writes build/coverage.out
#   scripts/test.sh --json     # go test -json into build/test-results.json,
#                              # which is what the CI test reporter reads
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

json=false
if [[ "${1:-}" == "--json" ]]; then
  json=true
  shift
fi

mkdir -p build

if [[ "$json" == true ]]; then
  go test -race -covermode=atomic -coverprofile=build/coverage.out -json "$@" ./... \
    | tee build/test-results.json
else
  go test -race -covermode=atomic -coverprofile=build/coverage.out "$@" ./...
  go tool cover -func=build/coverage.out | tail -n 1
fi
