#!/usr/bin/env bash
# Deterministic verification script for the create-nextjs-feature skill.
# Runs TypeScript typechecking, ESLint, and Vitest unit tests from the repository root.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "==> [1/3] Running TypeScript typecheck (pnpm exec tsc --noEmit)..."
(cd "$REPO_ROOT" && pnpm exec tsc --noEmit)

echo "==> [2/3] Running ESLint (pnpm lint)..."
(cd "$REPO_ROOT" && pnpm lint)

echo "==> [3/3] Running Vitest unit tests (pnpm test)..."
(cd "$REPO_ROOT" && pnpm test)

echo "==> SUCCESS: All Next.js frontend typechecks, lint rules, and Vitest suites passed."
