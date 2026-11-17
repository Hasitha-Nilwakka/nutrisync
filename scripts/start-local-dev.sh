#!/usr/bin/env bash
# ==============================================================================
# NutriSync (numbers-dont-lie) — Local Development Launch Script
# ==============================================================================
# Starts PostgreSQL 17 in Docker (:5432) and runs the 3 application services
# natively on the host with live hot-reload:
#   1. Python 3-Layer AI & Analytics Service (uvicorn --reload on :8000)
#   2. Go Core & Auth API                    (go run . on :8080)
#   3. Next.js 16 Web Frontend               (pnpm dev on :3000)
#
# Usage:
#   ./scripts/start-local-dev.sh                  # Auto-detect LLM (MLX / Ollama / Fallback)
#   ./scripts/start-local-dev.sh --llm=mlx        # Also start MLX LoRA server on :11435
#   ./scripts/start-local-dev.sh --llm=ollama     # Use Ollama on :11434
#   ./scripts/start-local-dev.sh --llm=fallback   # Force deterministic clinical fallback
#   ./scripts/start-local-dev.sh --help           # Print usage instructions
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

LLM_MODE="auto"
STOP_DB_ON_EXIT="false"
CHILD_PIDS=()

print_usage_instructions() {
  cat <<EOF
Usage: ./scripts/start-local-dev.sh [OPTIONS]

Starts PostgreSQL 17 in Docker (:5432) and runs Python AI (:8000), Go API (:8080),
and Next.js (:3000) natively with hot-reload in a single terminal session.

Options:
      --llm=MODE   Select LLM backend mode: auto (default), mlx, ollama, or fallback
      --stop-db    Also stop the PostgreSQL Docker container when exiting (Ctrl+C)
  -h, --help       Show this help message and exit
EOF
}

parse_command_line_arguments() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --llm=*)
        LLM_MODE="${1#*=}"
        shift
        ;;
      --stop-db)
        STOP_DB_ON_EXIT="true"
        shift
        ;;
      -h|--help)
        print_usage_instructions
        exit 0
        ;;
      *)
        echo "[ERROR] Unknown option: $1" >&2
        print_usage_instructions
        exit 1
        ;;
    esac
  done

  case "${LLM_MODE}" in
    auto|mlx|ollama|fallback) ;;
    *)
      echo "[ERROR] Invalid --llm mode '${LLM_MODE}'. Expected: auto, mlx, ollama, or fallback." >&2
      exit 1
      ;;
  esac
}

cleanup_background_processes() {
  echo ""
  echo "[INFO] Shutting down local development services..."
  for pid in "${CHILD_PIDS[@]:-}"; do
    if [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1; then
      kill "${pid}" >/dev/null 2>&1 || true
    fi
  done

  if [[ "${STOP_DB_ON_EXIT}" == "true" ]]; then
    echo "[INFO] Stopping PostgreSQL Docker container..."
    docker compose stop db >/dev/null 2>&1 || true
  fi
  echo "[INFO] Local development services stopped."
}

verify_local_dev_prerequisites() {
  local missing_tools=()
  for required_cmd in docker go python3 pnpm curl; do
    if ! command -v "${required_cmd}" >/dev/null 2>&1; then
      missing_tools+=("${required_cmd}")
    fi
  done

  if [[ ${#missing_tools[@]} -gt 0 ]]; then
    echo "[ERROR] Missing required local development tools: ${missing_tools[*]}" >&2
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    echo "[ERROR] Docker daemon is not running. Please start Docker Desktop / daemon so PostgreSQL can run." >&2
    exit 1
  fi
}

ensure_environment_file_exists() {
  if [[ ! -f "${REPO_ROOT}/.env" ]] && [[ -f "${REPO_ROOT}/.env.example" ]]; then
    cp "${REPO_ROOT}/.env.example" "${REPO_ROOT}/.env"
    echo "[INFO] Created .env from .env.example."
  fi
}

start_postgresql_database_container() {
  echo "[INFO] Starting PostgreSQL 17 container (db) on localhost:5432..."
  docker compose up -d db

  local attempt=1
  local max_attempts=30
  while [[ ${attempt} -le ${max_attempts} ]]; do
    if docker compose exec -T db pg_isready -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-nutrisync}" >/dev/null 2>&1; then
      echo "[OK]   PostgreSQL 17 is healthy and accepting connections on localhost:5432."
      return 0
    fi
    sleep 1
    attempt=$((attempt + 1))
  done

  echo "[ERROR] PostgreSQL container did not become healthy in time." >&2
  exit 1
}

ensure_service_dependencies_installed() {
  if [[ ! -d "${REPO_ROOT}/node_modules" ]] || [[ ! -x "${REPO_ROOT}/node_modules/.bin/next" ]]; then
    echo "[INFO] Installing frontend dependencies via pnpm..."
    pnpm install --frozen-lockfile
  fi

  if [[ ! -x "${REPO_ROOT}/services/py-ai/.venv/bin/uvicorn" ]]; then
    echo "[INFO] Creating Python virtual environment in services/py-ai/.venv..."
    python3 -m venv "${REPO_ROOT}/services/py-ai/.venv"
    "${REPO_ROOT}/services/py-ai/.venv/bin/pip" install --upgrade pip
    "${REPO_ROOT}/services/py-ai/.venv/bin/pip" install -r "${REPO_ROOT}/services/py-ai/requirements.txt"
  fi
}

configure_and_start_llm_backend() {
  local selected_mode="${LLM_MODE}"

  if [[ "${selected_mode}" == "auto" ]]; then
    if curl -fsS --max-time 1 "http://localhost:11435/v1/models" >/dev/null 2>&1; then
      selected_mode="mlx"
    elif curl -fsS --max-time 1 "http://localhost:11434/api/tags" >/dev/null 2>&1; then
      selected_mode="ollama"
    elif [[ -x "${REPO_ROOT}/services/py-ai/.venv-train/bin/python" ]] && \
         [[ -d "${REPO_ROOT}/services/py-ai/training/outputs/nutrisync-llama-1b-qlora" ]]; then
      selected_mode="mlx"
    else
      selected_mode="fallback"
    fi
  fi

  case "${selected_mode}" in
    mlx)
      if ! curl -fsS --max-time 1 "http://localhost:11435/v1/models" >/dev/null 2>&1; then
        if [[ -x "${REPO_ROOT}/services/py-ai/.venv-train/bin/python" ]] && \
           [[ -d "${REPO_ROOT}/services/py-ai/training/outputs/nutrisync-llama-1b-qlora" ]]; then
          echo "[INFO] Starting Apple Silicon MLX LoRA server on http://localhost:11435..."
          (
            cd "${REPO_ROOT}/services/py-ai"
            ./.venv-train/bin/python -m mlx_lm.server \
              --model mlx-community/Llama-3.2-1B-Instruct-4bit \
              --adapter-path training/outputs/nutrisync-llama-1b-qlora \
              --port 11435
          ) &
          CHILD_PIDS+=("$!")
        else
          echo "[WARN] MLX training venv or LoRA adapter not found; falling back to deterministic clinical engine."
        fi
      else
        echo "[INFO] Using running MLX LoRA server at http://localhost:11435."
      fi
      export LLM_API_URL="http://localhost:11435/v1/chat/completions"
      export LLM_MODEL="mlx-community/Llama-3.2-1B-Instruct-4bit"
      export LLM_API_FORMAT="openai"
      export LLM_TIMEOUT_SECONDS="20.0"
      ;;
    ollama)
      echo "[INFO] Configuring Python AI service to use Ollama at http://localhost:11434/api/chat."
      export LLM_API_URL="http://localhost:11434/api/chat"
      export LLM_MODEL="${LLM_MODEL:-nutrisync-llama}"
      export LLM_API_FORMAT="ollama_chat"
      export LLM_TIMEOUT_SECONDS="15.0"
      ;;
    fallback)
      echo "[INFO] Using Layer 3 deterministic clinical insight engine (no local LLM server required)."
      export LLM_API_URL="http://localhost:11434/api/chat"
      export LLM_MODEL="nutrisync-llama"
      export LLM_API_FORMAT="ollama_chat"
      export LLM_TIMEOUT_SECONDS="2.0"
      ;;
  esac
}

wait_for_http_health_endpoint() {
  local service_name="$1"
  local url="$2"
  local max_attempts="${3:-30}"
  local attempt=1

  while [[ ${attempt} -le ${max_attempts} ]]; do
    if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
      echo "[OK]   ${service_name} is ready (${url})"
      return 0
    fi
    sleep 1
    attempt=$((attempt + 1))
  done

  echo "[ERROR] ${service_name} failed to become ready at ${url}." >&2
  return 1
}

main() {
  parse_command_line_arguments "$@"
  verify_local_dev_prerequisites
  ensure_environment_file_exists
  ensure_service_dependencies_installed

  trap cleanup_background_processes EXIT INT TERM

  start_postgresql_database_container
  configure_and_start_llm_backend

  echo "[INFO] Starting Python AI & Analytics Service (services/py-ai) on http://localhost:8000..."
  (
    cd "${REPO_ROOT}/services/py-ai"
    ./.venv/bin/uvicorn main:app --host 0.0.0.0 --port 8000 --reload
  ) &
  CHILD_PIDS+=("$!")
  wait_for_http_health_endpoint "Python AI Service" "http://localhost:8000/health" 30

  echo "[INFO] Starting Go Core & Auth API (services/go-api) on http://localhost:8080..."
  (
    cd "${REPO_ROOT}/services/go-api"
    DATABASE_URL="${DATABASE_URL:-postgresql://app:app_password@localhost:5432/nutrisync?sslmode=disable}" \
    PY_AI_URL="http://localhost:8000" \
    APP_URL="http://localhost:3000" \
    go run .
  ) &
  CHILD_PIDS+=("$!")
  wait_for_http_health_endpoint "Go Core API" "http://localhost:8080/health" 45

  echo "[INFO] Starting Next.js Web Frontend on http://localhost:3000..."
  (
    cd "${REPO_ROOT}"
    GO_API_URL="http://localhost:8080" pnpm dev
  ) &
  CHILD_PIDS+=("$!")
  wait_for_http_health_endpoint "Next.js Web Frontend" "http://localhost:3000" 60

  echo ""
  echo "=============================================================================="
  echo " NutriSync Local Development Environment is Running!"
  echo "  - Web App (Next.js):       http://localhost:3000"
  echo "  - Go Core & Auth API:      http://localhost:8080/health"
  echo "  - Python AI & Analytics:   http://localhost:8000/health"
  echo "  - PostgreSQL 17:           localhost:5432"
  echo " Press Ctrl+C to stop all services."
  echo "=============================================================================="

  wait
}

main "$@"
