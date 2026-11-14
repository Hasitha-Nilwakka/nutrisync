# Architecture Decision Records (ADRs) — NutriSync (`numbers-dont-lie`)

This directory contains the **Architecture Decision Records (ADRs)** for the **NutriSync (`numbers-dont-lie`)** repository. ADRs serve as long-term institutional memory so developers and AI coding agents understand **what** architectural boundaries exist and **why** they were chosen.

---

## How Agents and Developers Use This Directory

1. **Progressive Disclosure (Scan First, Read On-Demand):**
   - Always scan the **ADR Index Table** below before running `/plan` or proposing changes that cross service boundaries (`Next.js` <-> `services/go-api` <-> `services/py-ai`).
   - Read individual ADR files (`000X-*.md`) only when your task touches their domain.
2. **Recording or Superseding Decisions (`/adr`):**
   - Run `/adr <decision title>` (defined in [`.agents/skills/adr/SKILL.md`](../../.agents/skills/adr/SKILL.md)) using [`_template.md`](./_template.md) whenever a significant architectural, database, security/privacy, or AI pipeline decision is made.
   - **Never delete or rewrite historical ADRs.** If a decision changes, create a new ADR and mark the old one's status as `Superseded by ADR-NNNN`.

---

## ADR Index

| ID | Title | Status | Affected Services | One-Sentence Summary |
| :--- | :--- | :--- | :--- | :--- |
| [ADR-0001](./0001-record-architecture-decisions.md) | Record Architecture Decisions | **Accepted** | All (`/` · `go-api` · `py-ai`) | Use numbered Markdown ADRs in `docs/adr/` indexed by `README.md` as the repository's institutional memory. |
| [ADR-0002](./0002-single-db-ownership-go-gorm.md) | Single Database Ownership in Go API via GORM Over Legacy Prisma | **Accepted** | `Next.js` · `go-api` · `py-ai` | PostgreSQL is owned and accessed exclusively by `services/go-api` via GORM; Next.js and `services/py-ai` never connect to PostgreSQL directly, and legacy Prisma files are frozen. |
| [ADR-0003](./0003-two-tier-pii-and-3-layer-ai.md) | Stateless Python AI Service with Two-Tier PII Anonymization and 3-Layer Guardrails | **Accepted** | `go-api` · `py-ai` | Isolate health analytics and LLM inference inside stateless `services/py-ai` behind Go outbound PII stripping and a 3-layer SciSpacy/LLaMA guardrail & clinical fallback pipeline. |
