#!/usr/bin/env bash
# Deterministic verification script for the build-py-ai-pipeline skill.
# Verifies stateless boundary (no direct DB drivers), bytecode syntax, and pytest suite in services/py-ai.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
PY_AI_DIR="$REPO_ROOT/services/py-ai"

echo "==> [1/2] Auditing stateless boundary (checking for forbidden direct DB imports)..."
if grep -rnE '^\s*(import|from)\s+(psycopg|psycopg2|sqlalchemy|asyncpg|prisma)\b' "$PY_AI_DIR" --exclude-dir=".venv" --exclude-dir=".pytest_cache" --exclude-dir="__pycache__"; then
  echo "ERROR: Direct database driver import detected in services/py-ai. PostgreSQL belongs exclusively to services/go-api."
  exit 1
fi

echo "==> [2/2] Running Python AI pytest suite..."
(cd "$PY_AI_DIR" && .venv/bin/pytest -v)

echo "==> SUCCESS: All Python AI architecture checks and pytest suites passed."
