# ADR-0001: Record Architecture Decisions

- **Status:** Accepted
- **Date:** 2026-11-14
- **Deciders:** NutriSync Engineering Team
- **Affected Services:** All (`Next.js Frontend`, `Go Core API`, `Python Analytics & AI`)
- **Supersedes:** None

---

## 1. Context & Problem Statement

NutriSync (`numbers-dont-lie`) is a 3-service polyglot platform combining a Next.js 16 frontend, a Go 1.26 authentication/core backend (`services/go-api`), and a Python FastAPI analytics/AI engine (`services/py-ai`). As multiple engineers and AI coding agents collaborate across the repository, critical architectural choices—such as why legacy Prisma files exist alongside Go GORM, or how health PII is scrubbed before LLM inference—can easily be lost between chat sessions. Without a durable record of *why* decisions were made, agents risk proposing conflicting patterns or re-introducing deprecated approaches.

## 2. Decision Drivers

- Preserve institutional memory across human developers and stateless AI agent sessions.
- Keep `AGENTS.md` concise by separating *current mandatory rules* (`AGENTS.md`) from *historical architectural rationale* (`docs/adr/`).
- Support token-efficient progressive disclosure via a single scannable index (`docs/adr/README.md`).

## 3. Considered Options

1. **Option 1: Numbered Markdown ADRs in `docs/adr/` with a scannable `README.md` index and `/adr` skill.**
2. **Option 2: Put all historical rationale directly inside `AGENTS.md`.** (Bloats the system prompt on every turn.)
3. **Option 3: Track decisions in external wikis or PR comments only.** (Invisible to local repository context and easily becomes stale.)

## 4. Decision Outcome

**Chosen Option:** **Option 1**, storing sequentially numbered Architecture Decision Records (`0001-*.md`) inside `docs/adr/`, indexed by `docs/adr/README.md`, and managed via the `/adr` skill (`.agents/skills/adr/SKILL.md`).

## 5. Consequences

### Positive
- Agents can scan `docs/adr/README.md` in seconds during `/plan` and read individual ADRs only when relevant.
- Superseded decisions remain traceable via `Superseded by ADR-NNNN` links rather than disappearing from history.

### Negative / Trade-offs
- Requires discipline to run `/adr` and update `docs/adr/README.md` when major architectural boundaries change.

## 6. Compliance & Enforcement

- **`AGENTS.md` / Service Rules:** `AGENTS.md` §3.4 and §11 point agents to `docs/adr/README.md`.
- **Automated Skills:** `.agents/skills/plan/SKILL.md` checks `docs/adr/README.md` during Step 1 (Gather Context), and `.agents/skills/adr/SKILL.md` automates creating and indexing new ADRs.
