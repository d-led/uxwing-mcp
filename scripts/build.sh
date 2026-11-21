#!/usr/bin/env bash
# Build the MCP server binary into bin/, stamped with the git version.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"

mkdir -p bin
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${version}" -o bin/uxwing-mcp .
echo "built bin/uxwing-mcp (${version})"
