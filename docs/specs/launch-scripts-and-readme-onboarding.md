# Feature Spec: Launch Scripts for Local Dev & Docker Reviewer Onboarding

- **Status:** Completed
- **Author:** Agent
- **Target Issue / PR:** N/A
- **Risk Level (`AGENTS.md` §10):** Medium (touches `README.md`, adds launch scripts in `scripts/`, and updates `Dockerfile` / `services/go-api/Dockerfile` / `compose.yaml` with explicit user approval)
- **Approval Gate:** Approved by User
- **Affected Services:**
  - [x] **Next.js Frontend (`/`)** — `README.md`, `Dockerfile` (requires approval), `compose.yaml` (requires approval)
  - [x] **Go Core API (`services/go-api/`)** — `services/go-api/Dockerfile` (requires approval)
  - [x] **Python Analytics & AI (`services/py-ai/`)** — `scripts/start-local-dev.sh`, `scripts/start-docker.sh`

---

## 1. Objective & Scope

- **Problem Statement:**
  Starting the polyglot 3-service NutriSync stack (`Next.js` on `:3000`, `Go API` on `:8080`, `Python AI` on `:8000`, `PostgreSQL 17` on `:5432`, plus optional local LLM server) currently requires running multiple manual terminal commands or wrestling with environment variable mismatches between host (`localhost`) and Docker containers (`db`, `go-api`, `py-ai`, `host.docker.internal`). Reviewers need a single, reliable on-launch script to spin up the full Docker stack immediately after cloning, and developers need a single on-launch script to boot the local hot-reload development environment.
- **In Scope:**
  1. `scripts/start-docker.sh`: Reviewer-focused one-command Docker launch script that ensures `.env` is initialized (generating a fallback `JWT_SECRET` if unset), configures optional host Ollama model wiring if present, overrides host-only `localhost` URLs with Docker network service DNS names (`db`, `go-api`, `py-ai`, `host.docker.internal`), builds/starts all 4 containers, and performs health-check readiness polling on `:5432`, `:8000`, `:8080`, and `:3000`.
  2. `scripts/start-local-dev.sh`: Developer-focused one-command local development launch script that starts PostgreSQL (`:5432`), checks/initializes `.venv` and `node_modules` if missing, optionally starts the local LLM server (`mlx_lm.server` on `:11435` or `ollama` on `:11434`), launches `services/py-ai`, `services/go-api`, and `Next.js` (`pnpm dev`) concurrently with unified log output, and traps `SIGINT`/`SIGTERM` for clean shutdown.
  3. Docker build/network fixes (requires explicit user approval per `AGENTS.md` §5):
     - Remove obsolete `prisma generate` and pass build-time `GO_API_URL` in root `Dockerfile`.
     - Update `services/go-api/Dockerfile` from `golang:1.24-alpine` to `golang:1.26-alpine` to match `services/go-api/go.mod` (`go 1.26.0`) and copy `go.sum`.
     - Expose `5432:5432` on the `db` service in `compose.yaml` so `docker compose up -d db` is reachable from the host in local dev mode.
  4. `README.md`: Document both the automated one-command launch scripts (`./scripts/start-docker.sh` and `./scripts/start-local-dev.sh`) and the manual step-by-step procedures.
- **Out of Scope:**
  - Application business logic, database schema migrations, Wellness Score math, or AI pipeline code changes.
  - Modifying `.env` secrets or committing `.env`.
- **Acceptance Criteria:**
  - [x] `./scripts/start-docker.sh` is executable (`chmod +x`), passes `bash -n` syntax validation, initializes `.env` safely from `.env.example` if absent without overwriting existing `.env` files, and launches the full Docker Compose stack with working inter-container URLs.
  - [x] `./scripts/start-local-dev.sh` is executable (`chmod +x`), passes `bash -n` syntax validation, starts PostgreSQL, `py-ai` (`:8000`), `go-api` (`:8080`), and Next.js (`:3000`), and cleanly terminates child processes on `Ctrl+C`.
  - [x] `README.md` clearly documents both the Reviewer Docker Launch Script and the Local Dev Launch Script alongside manual fallback instructions.
- **Pre-Flight MCP & Context Verification:**
  - [x] Inspected `compose.yaml`, `.env.example`, root `Dockerfile`, `services/go-api/Dockerfile`, `services/py-ai/Dockerfile`, and `next.config.ts`.
  - [x] Verified `services/go-api/main.go` and `services/py-ai/main.py` environment variable expectations and `/health` endpoints.

---

## 2. Polyglot Technical Design

### 2.1 Database & GORM Models (`services/go-api/models/` & `database/`)
- **Tables / Models Affected:** None.
- **Columns, Indexes & Constraints:** None.
- **Migration & Transaction Strategy:** `services/go-api` automatically executes `database.RunDatabaseMigrations` (`AutoMigrate`) on startup once PostgreSQL is reachable.

### 2.2 Go API & Cross-Service Contracts (`services/go-api/`)
- **HTTP Endpoints (`:8080`):** Existing `GET /health` endpoint (`http://localhost:8080/health`) used by both launch scripts to verify service readiness.
- **Dockerfile Alignment (`services/go-api/Dockerfile`):** Update builder base image to `golang:1.26-alpine` (matching `go 1.26.0` in `go.mod`) and copy `go.mod go.sum ./` before `go mod download`.

### 2.3 Python Analytics & 3-Layer AI Pipeline (`services/py-ai/`)
- **HTTP Endpoints (`:8000`):** Existing `GET /health` endpoint (`http://localhost:8000/health`) used by both launch scripts to verify service readiness.
- **LLM Fallback:** If neither Ollama (`:11434`) nor `mlx_lm.server` (`:11435`) is running on the host, Layer 3 deterministically falls back to clinical insights so reviewers never hit runtime failures without a GPU/LLM setup.

### 2.4 Frontend UI, Docker Build & Launch Scripts (`scripts/`, `Dockerfile`, `compose.yaml`, `README.md`)
- **`scripts/start-docker.sh` (Reviewer Docker Script):**
  - Verifies `docker` and `docker compose` availability.
  - Creates `.env` from `.env.example` if missing; generates an ephemeral/local `JWT_SECRET` if the `.env` still has the placeholder value so `go-api` boots with a valid 32+ byte secret.
  - Checks if `ollama` is running on the host; if `nutrisync-llama` is not yet registered in Ollama and `ollama` is running, offers/runs `ollama create nutrisync-llama -f services/py-ai/training/Modelfile`.
  - Runs `docker compose up --build` (or `-d` if `--detach` flag is passed) with container-internal networking environment variables (`DATABASE_URL`, `GO_API_URL=http://go-api:8080`, `PY_AI_URL=http://py-ai:8000`, `LLM_API_URL=http://host.docker.internal:11434/api/chat`).
- **`scripts/start-local-dev.sh` (Local Dev Script):**
  - Verifies `docker`, `go`, `python3`, and `pnpm` availability.
  - Creates `.env` from `.env.example` if missing.
  - Starts PostgreSQL container (`docker compose up -d db`) and waits for readiness on `localhost:5432`.
  - Ensures `services/py-ai/.venv` exists (bootstrapping venv + `pip install -r requirements.txt` if missing) and `node_modules` exists (`pnpm install --frozen-lockfile` if missing).
  - Auto-detects or accepts `--llm=mlx|ollama|fallback` to wire `LLM_API_URL`, `LLM_MODEL`, and `LLM_API_FORMAT`.
  - Starts `py-ai` (`:8000`), `go-api` (`:8080`), and `next dev` (`:3000`), waits for health checks, prints service URLs, and cleans up all child PIDs on exit.

---

## 3. Security, Privacy (GDPR/PII) & Clinical Guardrails

- **Zero Secret Exposure (`AGENTS.md` §5 & §9):**
  - Neither launch script ever prints, logs, or commits `.env` secrets (`JWT_SECRET`, `EMAIL_APP_PASSWORD`, OAuth secrets).
  - `.env` remains git-ignored; `.env.example` is used strictly as a template when `.env` does not exist.
- **User Isolation, GDPR & AI Guardrails:**
  - Unchanged; all three services run with their full authentication, GDPR consent, and 3-layer AI guardrail pipelines active.

---

## 4. Implementation & Verification Checklist

### Implementation Steps
- [x] **Step 1 — Docker & Compose Fixes (if approved by user):**
  - Update root `Dockerfile` to remove legacy `prisma generate` and set `ARG GO_API_URL` / `ENV GO_API_URL` during `pnpm build`.
  - Update `services/go-api/Dockerfile` to `golang:1.26-alpine` and copy `go.sum`.
  - Update `compose.yaml` to expose `5432:5432` on `db` and ensure container-internal URLs (`db`, `go-api`, `py-ai`, `host.docker.internal`) are not accidentally overwritten by host `localhost` entries in `.env`.
- [x] **Step 2 — Reviewer Docker Launch Script (`scripts/start-docker.sh`):**
  - Create `scripts/start-docker.sh` with pre-flight checks, `.env` bootstrap, optional Ollama setup, and `docker compose` execution.
- [x] **Step 3 — Local Dev Launch Script (`scripts/start-local-dev.sh`):**
  - Create `scripts/start-local-dev.sh` with DB container startup, dependency checks, optional MLX/Ollama LLM wiring, concurrent service supervision, health-check polling, and `trap` cleanup.
- [x] **Step 4 — Documentation (`README.md`):**
  - Update `README.md` to feature both launch scripts prominently alongside manual startup instructions.

### Verification Commands (`AGENTS.md` §4)
- [x] **Shell Script Syntax Check:** `bash -n scripts/start-docker.sh && bash -n scripts/start-local-dev.sh`
- [x] **Frontend Typecheck:** `pnpm exec tsc --noEmit`
- [x] **Frontend Unit Tests:** `pnpm test`
- [x] **Go API Unit Tests:** `cd services/go-api && go test -v ./...`
- [x] **Python AI Unit Tests:** `cd services/py-ai && .venv/bin/pytest -v`

---

## 5. Open Questions & Decisions Required

- [x] **Approval to update `Dockerfile`, `services/go-api/Dockerfile`, and `compose.yaml` (`AGENTS.md` §5):** Approved by User.
- [x] **Script Location & Naming:** Approved by User (`scripts/start-docker.sh` and `scripts/start-local-dev.sh`).
