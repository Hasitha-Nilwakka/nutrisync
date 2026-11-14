#!/usr/bin/env bash
# Deterministic verification script for the create-go-endpoint skill.
# Runs gofmt, go vet, and go test across services/go-api.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
GO_API_DIR="$REPO_ROOT/services/go-api"

echo "==> [1/3] Checking Go formatting (gofmt -l)..."
UNFORMATTED="$(cd "$GO_API_DIR" && gofmt -l .)"
if [[ -n "$UNFORMATTED" ]]; then
  echo "Formatting Go files with gofmt -w:"
  echo "$UNFORMATTED"
  (cd "$GO_API_DIR" && gofmt -w .)
fi

echo "==> [2/3] Running go vet ./..."
(cd "$GO_API_DIR" && go vet ./...)

echo "==> [3/3] Running Go unit tests (go test -v ./...)..."
(cd "$GO_API_DIR" && go test -v ./...)

echo "==> SUCCESS: All Go API checks and unit tests passed."
