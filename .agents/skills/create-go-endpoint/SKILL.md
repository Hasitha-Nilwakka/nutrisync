---
name: create-go-endpoint
description: >-
  Scaffold, implement, and verify an authenticated Go API endpoint in services/go-api
  with GORM user-scoped repository queries, Next.js /api/* rewrite wiring, and Go unit tests.
  Use whenever adding or updating HTTP handlers, routes, or repositories in services/go-api.
---

# Create Go API Endpoint (`create-go-endpoint`)

Follow this runbook when creating or modifying HTTP endpoints in `services/go-api` (`:8080`) for **NutriSync (`numbers-dont-lie`)**.

---

## 1. Progressive Context & Boilerplate

Before writing code, consult these bundled resources as needed:
- **Auth & Error Contracts:** Read [`references/auth-and-error-contracts.md`](./references/auth-and-error-contracts.md) for cookie-based JWT `userID` extraction, GORM transaction patterns, and structured HTTP JSON error shapes.
- **Handler Starter Template:** Copy and adapt [`assets/handler.template.go`](./assets/handler.template.go).
- **Unit Test Starter Template:** Copy and adapt [`assets/handler_test.template.go`](./assets/handler_test.template.go) (includes `200 OK`, `400 Bad Request`, and `401`/`403` negative user-isolation test cases).

---

## 2. Step-by-Step Implementation Flow

1. **Check Live Database Schema (`postgres-readonly` MCP):**
   - If the endpoint reads or mutates PostgreSQL tables, verify table/column names via `postgres-readonly` MCP and inspect `services/go-api/models/`.
   - **Approval Gate:** Any modification to GORM models (`services/go-api/models/`) or `database/postgres.go` requires explicit user approval before editing (`AGENTS.md` §8).
2. **Implement User-Scoped Repository Method (`services/go-api/repositories/`):**
   - Every GORM query (`Where("user_id = ?", authenticatedUserID)`) must be strictly scoped to the authenticated user's ID.
   - Wrap multi-table writes in `db.Transaction(...)`.
   - Use full, descriptive, single-purpose function names (never abbreviate).
3. **Implement HTTP Handler (`services/go-api/handlers/`):**
   - Enforce the HTTP method (`http.StatusMethodNotAllowed` on mismatch).
   - Extract `authenticatedUserID` solely from the validated `access_token` JWT cookie (`auth.ValidateAccessTokenAndExtractUserID`) and verify the active session—**never** trust a `userId` passed in the JSON body or query string.
   - Normalize biometric units via `services/go-api/health/normalization.go` when applicable.
   - Return structured JSON (`{"error": "..."}` or `{"message": "..."}`) with zero stack traces or raw SQL errors.
4. **Register Route & Next.js Proxy Rewrite:**
   - Register the route in `CreateApplicationMux` inside [`services/go-api/main.go`](../../../services/go-api/main.go).
   - If the frontend calls this route directly, verify or add the matching `/api/*` rewrite in [`next.config.ts`](../../../next.config.ts).
5. **Add Unit & Negative Authorization Tests (`services/go-api/handlers/*_test.go`):**
   - Using [`assets/handler_test.template.go`](./assets/handler_test.template.go), test:
     1. Authenticated valid request (`200 OK` / `201 Created`).
     2. Invalid payload or out-of-bounds value (`400 Bad Request`).
     3. Unauthenticated, expired-cookie, or cross-user request (`401 Unauthorized` / `403 Forbidden`).

---

## 3. Deterministic Verification

Run the bundled verification script to check `gofmt`, `go vet`, all Go unit tests, and `next.config.ts` rewrite alignment:

```bash
./.agents/skills/create-go-endpoint/scripts/verify-go-endpoint.sh
```
