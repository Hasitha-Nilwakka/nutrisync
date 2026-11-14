# ADR-NNNN: [Short Title of Solved Problem and Solution]

- **Status:** Proposed | Accepted | Deprecated | Superseded by [ADR-XXXX](./XXXX-slug.md)
- **Date:** YYYY-MM-DD
- **Deciders:** [Engineering Team / Architecture Owner]
- **Affected Services:** [Next.js Frontend (`/`) | Go Core API (`services/go-api`) | Python Analytics & AI (`services/py-ai`)]
- **Supersedes:** None (or [ADR-XXXX](./XXXX-slug.md))

---

## 1. Context & Problem Statement

Describe the architectural, security/privacy, or domain problem being addressed. What technical or product forces are in play, and why must a durable decision be made now?

## 2. Decision Drivers

- [Driver 1, e.g., Strict user data isolation and GDPR compliance]
- [Driver 2, e.g., Avoiding schema drift across polyglot services]
- [Driver 3, e.g., Deterministic verification and testability]

## 3. Considered Options

1. **Option 1:** [Name of Option 1] — Brief summary of pros/cons.
2. **Option 2:** [Name of Option 2] — Brief summary of pros/cons.
3. **Option 3:** [Name of Option 3] — Brief summary of pros/cons.

## 4. Decision Outcome

**Chosen Option:** **[Option X]**, because [concise justification explaining why this option best satisfies the decision drivers compared to the alternatives].

## 5. Consequences

### Positive
- [Benefit 1]
- [Benefit 2]

### Negative / Trade-offs
- [Cost, complexity, or constraint introduced by this decision]

## 6. Compliance & Enforcement

How is this architectural decision enforced across the repository?
- **`AGENTS.md` / Service Rules:** [Which section codifies this rule?]
- **Hooks & Permissions (`.agents/hooks.json`):** [Does a `PreToolUse` or `PostToolUse` hook guard this boundary?]
- **Automated Tests / Skills:** [Which test suites or verification scripts verify compliance?]
