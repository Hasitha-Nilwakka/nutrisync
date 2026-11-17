#!/usr/bin/env bash
# ==============================================================================
# NutriSync (numbers-dont-lie) — Reviewer Docker Launch Script
# ==============================================================================
# Starts the complete 4-container NutriSync stack (PostgreSQL 17, Go Core API,
# Python 3-Layer AI Engine, and Next.js Web Server) using Docker Compose.
#
# Usage:
#   ./scripts/start-docker.sh             # Build & start in foreground
#   ./scripts/start-docker.sh --detach    # Build & start in background (-d)
#   ./scripts/start-docker.sh --no-build  # Start without rebuilding images
#   ./scripts/start-docker.sh --down      # Stop and remove running containers
#   ./scripts/start-docker.sh --help      # Print usage instructions
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

DETACH_MODE="false"
BUILD_IMAGES="true"
STOP_STACK="false"

print_usage_instructions() {
  cat <<EOF
Usage: ./scripts/start-docker.sh [OPTIONS]

Reviewer one-command launch script for the containerized NutriSync stack:
  - PostgreSQL 17 Database   (:5432)
  - Python 3-Layer AI Engine (:8000)
  - Go Core & Auth API       (:8080)
  - Next.js Web Application  (:3000)

Options:
  -d, --detach     Start containers in detached background mode and verify health checks
      --no-build   Skip rebuilding Docker images before starting
      --down       Stop and remove running NutriSync Docker Compose containers
  -h, --help       Show this help message and exit
EOF
}

parse_command_line_arguments() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -d|--detach)
        DETACH_MODE="true"
        shift
        ;;
      --no-build)
        BUILD_IMAGES="false"
        shift
        ;;
      --down)
        STOP_STACK="true"
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
}

verify_docker_prerequisites() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "[ERROR] 'docker' is not installed or not on PATH. Please install Docker Desktop or Docker Engine." >&2
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    echo "[ERROR] 'docker compose' plugin is not available. Please upgrade Docker." >&2
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    echo "[ERROR] Docker daemon is not running. Please start Docker Desktop / daemon and retry." >&2
    exit 1
  fi
}

ensure_environment_file_exists() {
  if [[ ! -f "${REPO_ROOT}/.env" ]]; then
    if [[ -f "${REPO_ROOT}/.env.example" ]]; then
      cp "${REPO_ROOT}/.env.example" "${REPO_ROOT}/.env"
      echo "[INFO] Created .env from .env.example (update .env if you want custom SMTP or OAuth credentials)."
    else
      echo "[WARN] Neither .env nor .env.example found; using compose.yaml defaults."
    fi
  else
    echo "[INFO] Found existing .env configuration."
  fi
}

configure_optional_host_ollama_model() {
  if command -v ollama >/dev/null 2>&1 && curl -fsS --max-time 2 "http://localhost:11434/api/tags" >/dev/null 2>&1; then
    echo "[INFO] Detected running Ollama server on host (http://localhost:11434)."
    if ! ollama list 2>/dev/null | awk '{print $1}' | grep -Eq '^nutrisync-llama(:|$)'; then
      if [[ -f "${REPO_ROOT}/services/py-ai/training/Modelfile" ]]; then
        echo "[INFO] Registering 'nutrisync-llama' model in host Ollama from services/py-ai/training/Modelfile..."
        ollama create nutrisync-llama -f "${REPO_ROOT}/services/py-ai/training/Modelfile" || \
          echo "[WARN] Could not build nutrisync-llama in Ollama; Layer 3 deterministic clinical fallback remains active."
      fi
    else
      echo "[INFO] Ollama model 'nutrisync-llama' is already registered."
    fi
  else
    echo "[INFO] Host Ollama server not detected on :11434 — Python AI Layer 3 will automatically use its deterministic clinical fallback."
  fi
}

wait_for_http_health_endpoint() {
  local service_name="$1"
  local url="$2"
  local max_attempts="${3:-45}"
  local attempt=1

  while [[ ${attempt} -le ${max_attempts} ]]; do
    if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
      echo "[OK]   ${service_name} is ready (${url})"
      return 0
    fi
    sleep 2
    attempt=$((attempt + 1))
  done

  echo "[WARN] ${service_name} did not respond at ${url} within $((max_attempts * 2))s. Check 'docker compose logs'." >&2
  return 1
}

main() {
  parse_command_line_arguments "$@"
  verify_docker_prerequisites

  if [[ "${STOP_STACK}" == "true" ]]; then
    echo "[INFO] Stopping NutriSync Docker Compose stack..."
    docker compose down
    exit 0
  fi

  ensure_environment_file_exists
  configure_optional_host_ollama_model

  # Export internal Docker network URLs so host-only localhost entries in .env never break container DNS
  export DOCKER_DATABASE_URL="postgresql://${POSTGRES_USER:-app}:${POSTGRES_PASSWORD:-app_password}@db:5432/${POSTGRES_DB:-nutrisync}?sslmode=disable"
  export DOCKER_GO_API_URL="http://go-api:8080"
  export DOCKER_PY_AI_URL="http://py-ai:8000"
  export DOCKER_LLM_API_URL="${DOCKER_LLM_API_URL:-http://host.docker.internal:11434/api/chat}"

  local compose_args=("up")
  if [[ "${BUILD_IMAGES}" == "true" ]]; then
    compose_args+=("--build")
  fi
  if [[ "${DETACH_MODE}" == "true" ]]; then
    compose_args+=("-d")
  fi

  echo "=============================================================================="
  echo " Starting NutriSync Full-Stack Docker Environment"
  echo "  - Web Frontend (Next.js):  http://localhost:3000"
  echo "  - Go Core & Auth API:      http://localhost:8080/health"
  echo "  - Python AI & Analytics:   http://localhost:8000/health"
  echo "=============================================================================="

  docker compose "${compose_args[@]}"

  if [[ "${DETACH_MODE}" == "true" ]]; then
    echo "[INFO] Waiting for containerized services to pass health checks..."
    wait_for_http_health_endpoint "Python AI Service" "http://localhost:8000/health" 45
    wait_for_http_health_endpoint "Go Core API" "http://localhost:8080/health" 45
    wait_for_http_health_endpoint "Next.js Web Frontend" "http://localhost:3000" 60
    echo ""
    echo "[READY] NutriSync is running at http://localhost:3000"
    echo "        Run './scripts/start-docker.sh --down' or 'docker compose down' to stop."
  fi
}

main "$@"
