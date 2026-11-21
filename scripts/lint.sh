#!/usr/bin/env bash
# Static checks: formatting, vet, and every linter that is installed.
#
# Usage:
#   scripts/lint.sh             # run the checks that are available
#   scripts/lint.sh --install   # install staticcheck and golangci-lint first
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ "${1:-}" == "--install" ]]; then
  echo "installing linters…"
  command -v staticcheck >/dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
  command -v golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
  command -v go-crap >/dev/null || go install github.com/padiazg/go-crap@latest
fi

section() { printf '\n==> %s\n' "$*"; }

section "gofmt"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  printf '%s\n' "$unformatted"
  echo "gofmt: the files above need formatting (gofmt -w .)" >&2
  exit 1
fi

section "go vet"
go vet ./...

if command -v staticcheck >/dev/null; then
  section "staticcheck"
  staticcheck ./...
else
  echo "staticcheck is not installed — run scripts/lint.sh --install"
fi

if command -v golangci-lint >/dev/null; then
  section "golangci-lint"
  golangci-lint run ./...
else
  echo "golangci-lint is not installed — run scripts/lint.sh --install"
fi
