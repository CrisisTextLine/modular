#!/usr/bin/env bash
set -euo pipefail

# Configuration
# GORACE halts immediately on first detected race to shorten feedback loop
export GORACE="halt_on_error=1"
export GOTOOLCHAIN=auto

# CGO must be enabled for -race; default to 1. Callers can explicitly disable with RACE_CGO=0
# Usage: RACE_CGO=0 ./scripts/test_race.sh
: "${RACE_CGO:=1}"
export CGO_ENABLED="${RACE_CGO}"

echo "CGO_ENABLED=${CGO_ENABLED} (override with RACE_CGO=0 to force disable – not recommended)"

echo "==> Running core tests with race detector"
go test -race ./...

echo "==> Running module tests with race detector"
for module in modules/*/; do
  if [ -f "$module/go.mod" ]; then
    echo "==> Module $module"
    (cd "$module" && go test -race ./...)
  fi
done

echo "==> Running CLI tests with race detector"
(cd cmd/modcli && go test -race ./...)

echo "All tests passed under race detector."
