#!/usr/bin/env bash
# Everything CI runs, in the same order, so a red build can be reproduced
# locally with one command.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

scripts/build.sh
scripts/lint.sh
scripts/test.sh
scripts/crap.sh
