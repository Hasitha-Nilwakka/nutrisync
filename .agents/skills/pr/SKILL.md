---
name: pr
description: >-
  Run full polyglot verification checks across Next.js, Go API, and Python AI,
  generate a structured changelog and PR description, and prepare or open a clean Pull Request.
  Trigger whenever the user types /pr or asks to prepare a commit, changelog, or pull request.
---

# Pre-PR Verification, Changelog & Pull Request Workflow (`/pr`)

You are operating in **Release & Pull Request Preparation Mode** for **NutriSync (`numbers-dont-lie`)**.
Your job is to verify that all affected services pass their quality gates, ensure no secrets or stray files are staged, synthesize a clean changelog/PR body, and request explicit user confirmation before committing, pushing, or opening the PR.

---

## Step 1: Workspace Hygiene & Secret Audit

1. Run `git status` and `git diff` to inspect all staged, unstaged, and untracked files.
2. **Verify Zero Prohibited Artifacts (`AGENTS.md` §5 & §8):**
   - Confirm no `.env` / `.env.*` secret files, temporary debug logs, or unapproved edits to legacy Prisma files (`docs/legacy-prisma-schema.prisma`, `prisma.config.ts`, `src/generated/prisma/`) are present in the working tree.
   - Confirm all unrelated user changes are preserved untouched.

---

## Step 2: Full Polyglot Verification Gate (`AGENTS.md` §4)

Invoke the `test-runner` subagent ([`.agents/agents/test-runner.md`](../../agents/test-runner.md)) (or run directly) to execute the verification suite across all 3 services and record the exact results:

1. **Frontend (`/`):**
   - `./.agents/skills/create-nextjs-feature/scripts/verify-frontend.sh` (`pnpm exec tsc --noEmit`, `pnpm lint`, `pnpm test`)
2. **Go Core API (`services/go-api/`):**
   - `./.agents/skills/create-go-endpoint/scripts/verify-go-endpoint.sh` (`gofmt`, `go vet`, `go test -v ./...`)
3. **Python Analytics & AI (`services/py-ai/`):**
   - `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` (stateless DB-import check + `.venv/bin/pytest -v`)

> **Hard Gate:** If any check fails, **stop immediately**, report the exact failing test or lint error, and do **not** proceed to commit or PR creation.

---

## Step 3: Generate Structured Changelog & PR Description

1. Inspect `git log` and the associated feature spec in `docs/specs/<feature-name>.md` (if present).
2. Draft a clean **Pull Request Title** (conventional commit style, e.g., `feat(health): add weekly wellness trend summary`) and **Pull Request Body** using this template:

```markdown
## Summary
- **Spec:** `docs/specs/<feature-name>.md` (if applicable)
- **Issue:** Closes #<number>
- **Overview:** <1-3 bullet summary of what changed and why>

## Changes by Service
- **Frontend (`/`):** <Routes, feature components, and Zod schemas added/updated>
- **Go Core API (`services/go-api`):** <Handlers, GORM repositories/models, normalization, or orchestrator updates>
- **Python AI (`services/py-ai`):** <Pydantic schemas, Pandas/NumPy analytics, or 3-Layer pipeline updates>

## Security, Privacy & Guardrail Compliance
- [x] User isolation enforced via server-side JWT cookies (`userID`-scoped GORM queries)
- [x] Two-Tier PII Anonymization preserved (`AnonymizedAIAnalysisPayload` + Layer 1 scrubber)
- [x] Wellness Score weights & Layer 3 contraindication guardrails intact
- [x] Single-DB ownership (`services/go-api` via GORM) preserved

## Verification Results
| Suite | Command | Status |
| :--- | :--- | :--- |
| Frontend Typecheck | `pnpm exec tsc --noEmit` | PASS |
| Frontend Lint | `pnpm lint` | PASS |
| Frontend Unit Tests | `pnpm test` | PASS |
| Go API Unit Tests | `cd services/go-api && go test -v ./...` | PASS |
| Python AI Unit Tests | `cd services/py-ai && .venv/bin/pytest -v` | PASS |
```

---

## Step 4: Approval Gate for Commit, Push & PR Creation

1. Present the verification summary, proposed commit message, and drafted PR description in chat.
2. Per `AGENTS.md` §5 and `.agents/hooks.json`, **ask for explicit user confirmation** before executing `git commit`, `git push`, or creating the Pull Request via the `github` MCP server (`mcp__github__create_pull_request`).
