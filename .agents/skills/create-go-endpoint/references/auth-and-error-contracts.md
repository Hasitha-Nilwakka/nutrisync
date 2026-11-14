# Go API Authentication, User Isolation & Error Contracts (`services/go-api`)

## 1. Trusted Authentication & User Isolation (`AGENTS.md` §6)

- **Cookie-Derived Identity Only:**
  - Read the `access_token` cookie (`r.Cookie("access_token")`).
  - Validate the JWT signature and expiration using `auth.ValidateAccessToken` / `userSessionRepository` session verification.
  - Extract `authenticatedUserID` strictly from the verified token claims.
  - **Prohibited:** Never accept `user_id` or `email` from request JSON bodies, URL path parameters, or query strings to authorize reading or mutating health profiles, weight logs, activity entries, or analytics snapshots.
- **User-Scoped GORM Queries (`services/go-api/repositories/`):**
  - Every `First`, `Find`, `Create`, `Save`, `Updates`, and `Delete` query on user-owned tables (`health_profiles`, `weight_entries`, `activity_entries`, `analytics_snapshots`, `sessions`, `accounts`) must include `.Where("user_id = ?", authenticatedUserID)`.

---

## 2. Standard HTTP Status Codes & Error Payload Shapes (`AGENTS.md` §9)

All error responses must set `Content-Type: application/json` and return a structured JSON object without internal stack traces or raw GORM/PostgreSQL errors:

| HTTP Status | Constant | When to Use | Canonical JSON Shape |
| :--- | :--- | :--- | :--- |
| `200 OK` | `http.StatusOK` | Successful `GET`, `PUT`, or action | Domain response struct (`camelCase` JSON tags) |
| `201 Created` | `http.StatusCreated` | Successful `POST` resource creation | Domain response struct (`camelCase` JSON tags) |
| `400 Bad Request` | `http.StatusBadRequest` | Malformed JSON, missing required fields, or physiological out-of-bounds values | `{"error": " Descriptive validation message"}` |
| `401 Unauthorized` | `http.StatusUnauthorized` | Missing, expired, or invalid `access_token` cookie / revoked session | `{"error": "Authentication required"}` |
| `403 Forbidden` | `http.StatusForbidden` | Authenticated user lacks GDPR consent or permission for the action | `{"error": "Forbidden"}` |
| `404 Not Found` | `http.StatusNotFound` | Resource does not exist for the authenticated `userID` | `{"error": "Resource not found"}` |
| `405 Method Not Allowed` | `http.StatusMethodNotAllowed` | Unsupported HTTP verb | `{"error": "Method not allowed"}` |
| `500 Internal Server Error` | `http.StatusInternalServerError` | Unexpected database or upstream error (never leak raw SQL error strings) | `{"error": "Internal server error"}` |

---

## 3. Outbound AI Orchestration Boundary (`ai_orchestrator_client.go`)

When a Go handler invokes `services/py-ai` (`POST /api/v1/analyze`):
- Strip `UserID`, `Email`, `Name`, and OAuth tokens.
- Send only `AnonymizedAIAnalysisPayload` with a hashed/opaque `subject_id` and normalized metric units (`kg`, `cm`).
- Always handle `services/py-ai` timeouts or `5xx` responses gracefully by falling back to cached `AnalyticsSnapshot` or deterministic Go wellness calculations.
