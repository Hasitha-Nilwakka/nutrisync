---
name: adr
description: >-
  Draft, supersede, and index Architecture Decision Records (ADRs) in docs/adr/
  to preserve long-term institutional memory across Next.js, Go API, and Python AI.
  Trigger whenever the user types /adr or asks to record or supersede an architectural decision.
---

# Architecture Decision Record Workflow (`/adr`)

You are operating in **Institutional Memory & Architecture Governance Mode** for **NutriSync (`numbers-dont-lie`)**.
Your job is to create or supersede an Architecture Decision Record in `docs/adr/` and keep the scannable index in `docs/adr/README.md` strictly synchronized.

---

## Step 1: Inspect Existing Decisions & Assign Next Number

1. Read [`docs/adr/README.md`](../../../docs/adr/README.md) and [`docs/adr/_template.md`](../../../docs/adr/_template.md).
2. Check whether the new decision **supersedes** or **amends** an existing ADR (`0001`, `0002`, `0003`, etc.).
3. Determine the next sequential 4-digit number (`NNNN`, e.g., `0004`) and construct the filename:
   `docs/adr/NNNN-<kebab-case-decision-title>.md`

---

## Step 2: Draft the ADR

1. Create `docs/adr/NNNN-<kebab-case-decision-title>.md` following every section of [`docs/adr/_template.md`](../../../docs/adr/_template.md):
   - **Metadata Header:** Set `Status` (`Proposed` or `Accepted`), `Date` (`YYYY-MM-DD`), `Affected Services` (`Next.js Frontend`, `Go Core API`, `Python Analytics & AI`), and `Supersedes` / `Superseded by` links.
   - **1. Context & Problem Statement:** Explain the architectural, clinical/privacy, or operational forces driving the decision.
   - **2. Decision Drivers:** Bullet the key constraints (e.g., GDPR/PII isolation, deterministic math, latency, single-DB ownership).
   - **3. Considered Options:** List at least two realistic alternatives evaluated.
   - **4. Decision Outcome:** State the chosen option clearly and explain *why* it won over the alternatives.
   - **5. Consequences:** Document positive outcomes, trade-offs/costs, and risks.
   - **6. Compliance & Enforcement:** Specify exactly how this decision is enforced in `AGENTS.md`, `.agents/hooks.json`, `.agents/skills/`, or automated unit tests.
2. **If Superseding an Older ADR:**
   - Never delete an existing ADR file.
   - Update the older ADR's `Status` line to `Superseded by [ADR-NNNN](./NNNN-<slug>.md)` while preserving its original body intact.

---

## Step 3: Update the ADR Index & Report

1. Add the new ADR row (and update any superseded ADR's status column) in the index table inside [`docs/adr/README.md`](../../../docs/adr/README.md).
2. If the decision establishes a new mandatory invariant for agents, note whether [`AGENTS.md`](../../../AGENTS.md) or `.agents/hooks.json` should be updated and ask for user confirmation before modifying protected configuration files.
3. Present a concise summary with clickable links to the new ADR and `docs/adr/README.md`.
