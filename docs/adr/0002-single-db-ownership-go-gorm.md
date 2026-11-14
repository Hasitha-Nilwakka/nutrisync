# ADR-0002: Single Database Ownership in Go API via GORM Over Legacy Prisma

- **Status:** Accepted
- **Date:** 2026-11-14
- **Deciders:** NutriSync Engineering Team
- **Affected Services:** `Next.js Frontend` (`/`), `Go Core API` (`services/go-api`), `Python Analytics & AI` (`services/py-ai`)
- **Supersedes:** Legacy Prisma schema (`docs/legacy-prisma-schema.prisma`, `prisma.config.ts`, `src/generated/prisma/`)

---

## 1. Context & Problem Statement

Early prototypes of `numbers-dont-lie` used Prisma in the Next.js layer (`docs/legacy-prisma-schema.prisma` and `src/generated/prisma/`). As the platform evolved into a 3-service architecture with dedicated Go authentication/core APIs (`services/go-api`) and Python health analytics (`services/py-ai`), allowing multiple services or ORMs to manage schema migrations and query PostgreSQL created severe risks: split schema authority, inconsistent user-isolation enforcement, and duplicated session validation logic.

## 2. Decision Drivers

- Guarantee a **single source of truth** for PostgreSQL schema migrations and data access.
- Enforce user data isolation (`Where("user_id = ?", authenticatedUserID)`) and JWT/refresh session validation in one hardened backend layer.
- Keep `services/py-ai` purely stateless so AI/analytics workers never hold database credentials or access raw account tables.
- Prevent AI coding agents from mistakenly writing Prisma queries when they see legacy Prisma artifacts in the repository.

## 3. Considered Options

1. **Option 1: Exclusive PostgreSQL ownership in `services/go-api` via GORM (`gorm.io/gorm`), with Next.js communicating over `/api/*` HTTP rewrites and `services/py-ai` operating statelessly.**
2. **Option 2: Shared database access across Next.js (Prisma) and Go (GORM).** (High risk of migration conflicts and bypassed auth rules.)
3. **Option 3: Direct PostgreSQL reads from `services/py-ai` for historical weight/activity series.** (Violates PII isolation by exposing raw user tables to the AI service.)

## 4. Decision Outcome

**Chosen Option:** **Option 1**. `services/go-api` is the sole owner of PostgreSQL (`services/go-api/models/` and `services/go-api/database/postgres.go` via GORM `AutoMigrate`). Next.js proxies all backend calls through `next.config.ts` rewrites, `services/py-ai` receives all required metrics in the request payload from Go, and legacy Prisma files are frozen as read-only historical references.

## 5. Consequences

### Positive
- All authentication, GDPR consent/export (`GET /api/health/export`), and user-scoped data access are centrally audited in Go.
- `services/py-ai` can scale and be tested independently using synthetic JSON payloads without a running database.

### Negative / Trade-offs
- Adding a new persisted field requires updating the Go GORM model and handler in `services/go-api` before consuming it in Next.js.

## 6. Compliance & Enforcement

- **`AGENTS.md` / Service Rules:** Enforced in `AGENTS.md` §3.4 & §8, `services/go-api/AGENTS.md`, and `services/py-ai/AGENTS.md`.
- **Hooks & Permissions (`.agents/hooks.json`):** `PreToolUse` hard-denies edits to `docs/legacy-prisma-schema.prisma`, `prisma.config.ts`, and `src/generated/prisma/*`, and requires explicit user approval (`ask`) before editing `services/go-api/models/*` or `services/go-api/database/*`.
- **Automated Skills:** `.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` fails if any database driver (`psycopg`, `sqlalchemy`, `asyncpg`, `prisma`) is imported in `services/py-ai`.
