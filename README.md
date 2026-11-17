# NutriSync (`numbers-dont-lie`)

**NutriSync** is a privacy-first AI Health & Wellness tracking and analytics platform built on a polyglot 3-service architecture:

1. **Web Frontend (`/` root — `:3000`):** Next.js `16.4` (App Router, Turbopack), React `19.3`, TypeScript `5`, Tailwind CSS v4, Shadcn/ui, React Hook Form, Zod v4.
2. **Core Backend & Auth Service (`services/go-api` — `:8080`):** Go `1.26` (`net/http`), GORM, PostgreSQL `17`, JWT + Refresh Token rotation, Google/GitHub OAuth2, TOTP 2FA, GDPR data export, and outbound PII anonymization.
3. **Stateless Analytics & 3-Layer AI Engine (`services/py-ai` — `:8000`):** Python FastAPI, Pydantic v2, Pandas/NumPy deterministic Wellness Score engine (`0.3 BMI + 0.3 Activity + 0.2 Progress + 0.2 Habits`), 3-Layer SciSpacy/LLaMA Guardrail Pipeline, and QLoRA/LoRA fine-tuning (`services/py-ai/training/`).

---

## Prerequisites & Environment Setup

1. Copy the example environment file to `.env` in the repository root:
   ```bash
   cp .env.example .env
   ```
2. Fill in your secrets in `.env` (at minimum `JWT_SECRET`, and optionally SMTP and Google/GitHub OAuth credentials):
   ```env
   APP_URL=http://localhost:3000
   GO_API_URL=http://localhost:8080
   DATABASE_URL=postgresql://app:app_password@localhost:5432/nutrisync?sslmode=disable
   JWT_SECRET=your-secure-random-jwt-secret-here

   # Python AI / Local LLM Configuration
   LLM_API_URL=http://localhost:11434/api/chat
   LLM_MODEL=nutrisync-llama
   LLM_API_FORMAT=ollama_chat
   LLM_TIMEOUT_SECONDS=15.0
   ```

---

## Method 1: Start Using Docker Compose (Recommended for Users / Full-Stack Evaluators)

Docker Compose starts **PostgreSQL (`:5432`)**, **Go Core API (`:8080`)**, **Python AI Service (`:8000`)**, and the **Next.js Web Server (`:3000`)** in a single command.

### 1. (Optional) Start the Local LLM Server on Host
If you want live LLM inference (instead of the built-in deterministic clinical fallback), start **Ollama** on your host machine first so the `py-ai` container can reach it via `host.docker.internal:11434`:
```bash
ollama serve &
ollama pull llama3.2:1b
ollama create nutrisync-llama -f services/py-ai/training/Modelfile
```
*(Note: If Ollama is not running, NutriSync still works 100% end-to-end—Layer 3 automatically falls back to its deterministic clinical insight engine.)*

### 2. Build and Start All Containers
From the repository root:
```bash
docker compose up --build
```
Or run in detached background mode:
```bash
docker compose up -d --build
```

### 3. Access the Application
- **Web App (Next.js):** [http://localhost:3000](http://localhost:3000)
- **Go Core API Health Check:** [http://localhost:8080/health](http://localhost:8080/health)
- **Python AI Service Health Check:** [http://localhost:8000/health](http://localhost:8000/health)

To stop the containers (preserving your PostgreSQL data volume):
```bash
docker compose down
```

---

## Method 2: Start in Local Development Environment (Hot-Reload Across All 3 Services)

In local development, run PostgreSQL in Docker and run **Go API**, **Python AI**, and **Next.js** natively in separate terminals for instant hot-reload and Apple Silicon GPU access.

### Terminal 1 — Start PostgreSQL Database (`:5432`)
Start only the `db` container from `compose.yaml` (and make sure port `5432` is exposed or running locally):
```bash
docker compose up -d db
```

### Terminal 2 — Start the Local Fine-Tuned LLM Server (`:11435` or `:11434`)
Choose **either** Option A (your Apple Silicon M4 MLX LoRA adapter server on `:11435`) **or** Option B (Ollama on `:11434`):

- **Option A — Serve Trained M4 LoRA Adapter directly via `mlx_lm.server` (`:11435`):**
  ```bash
  cd services/py-ai
  source .venv-train/bin/activate
  python -m mlx_lm.server \
    --model mlx-community/Llama-3.2-1B-Instruct-4bit \
    --adapter-path training/outputs/nutrisync-llama-1b-qlora \
    --port 11435
  ```
- **Option B — Serve via Ollama (`:11434`):**
  ```bash
  ollama serve &
  ollama create nutrisync-llama -f services/py-ai/training/Modelfile
  ```

### Terminal 3 — Start the Python AI & Analytics Service (`services/py-ai` on `:8000`)
- **If using Option A (`mlx_lm.server` on `:11435`):**
  ```bash
  cd services/py-ai
  LLM_API_URL=http://localhost:11435/v1/chat/completions \
  LLM_MODEL=mlx-community/Llama-3.2-1B-Instruct-4bit \
  LLM_API_FORMAT=openai \
  LLM_TIMEOUT_SECONDS=20.0 \
  .venv/bin/uvicorn main:app --host 0.0.0.0 --port 8000 --reload
  ```
- **If using Option B (Ollama on `:11434`):**
  ```bash
  cd services/py-ai
  LLM_API_URL=http://localhost:11434/api/chat \
  LLM_MODEL=nutrisync-llama \
  LLM_API_FORMAT=ollama_chat \
  LLM_TIMEOUT_SECONDS=15.0 \
  .venv/bin/uvicorn main:app --host 0.0.0.0 --port 8000 --reload
  ```

### Terminal 4 — Start the Go Core & Auth API (`services/go-api` on `:8080`)
```bash
cd services/go-api
PY_AI_URL=http://localhost:8000 go run .
```
*(Automatically runs GORM `AutoMigrate` on startup and connects to `services/py-ai` at `http://localhost:8000`.)*

### Terminal 5 — Start the Next.js Frontend (`/` on `:3000`)
```bash
pnpm install --frozen-lockfile
pnpm dev
```
Open [http://localhost:3000](http://localhost:3000) in your browser. Next.js automatically rewrites `/api/auth/*`, `/api/go/*`, and `/api/health/*` to `http://localhost:8080`.

---

## Running Verification & Test Suites

| Service | Command |
| :--- | :--- |
| **Frontend (TypeScript + ESLint + Vitest)** | `./.agents/skills/create-nextjs-feature/scripts/verify-frontend.sh` |
| **Go Core API (`gofmt` + `go vet` + `go test`)** | `./.agents/skills/create-go-endpoint/scripts/verify-go-endpoint.sh` |
| **Python AI (`pytest` + Stateless DB Audit)** | `./.agents/skills/build-py-ai-pipeline/scripts/verify-py-ai.sh` |
| **QLoRA Fine-Tuning & Benchmark Guide** | See [`services/py-ai/training/README.md`](./services/py-ai/training/README.md) |
| **Architecture Decision Records (ADRs)** | See [`docs/adr/README.md`](./docs/adr/README.md) |
| **Agentic Workflow Setup Guide** | See [`docs/agentic_setup/AGENTIC_SETUP.md`](./docs/agentic_setup/AGENTIC_SETUP.md) |
