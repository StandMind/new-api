#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$ROOT_DIR/web/default"

BACKEND_PORT="${BACKEND_PORT:-3000}"
FRONTEND_PORT="${FRONTEND_PORT:-3001}"
BACKEND_URL="${BACKEND_URL:-http://localhost:${BACKEND_PORT}}"

export PATH="$HOME/.local/bin:$PATH"

backend_pid=""
frontend_pid=""

log() {
  printf '[dev] %s\n' "$*"
}

require_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    printf '[dev] missing command: %s\n' "$name" >&2
    exit 1
  fi
}

ensure_embed_placeholders() {
  local default_dist="$ROOT_DIR/web/default/dist"
  local classic_dist="$ROOT_DIR/web/classic/dist"

  mkdir -p "$default_dist" "$classic_dist"

  if [[ ! -f "$default_dist/index.html" ]]; then
    printf '<!doctype html><html><head><title>dev</title></head><body>use frontend dev server</body></html>\n' >"$default_dist/index.html"
  fi

  if [[ ! -f "$classic_dist/index.html" ]]; then
    printf '<!doctype html><html><head><title>dev</title></head><body>use frontend dev server</body></html>\n' >"$classic_dist/index.html"
  fi
}

ensure_frontend_deps() {
  if [[ "${SKIP_BUN_INSTALL:-}" == "1" ]]; then
    return
  fi

  if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    log "installing frontend dependencies"
    (cd "$FRONTEND_DIR" && bun install)
  fi
}

cleanup() {
  trap - EXIT INT TERM
  if [[ -n "$frontend_pid" ]] && kill -0 "$frontend_pid" >/dev/null 2>&1; then
    kill "$frontend_pid" >/dev/null 2>&1 || true
  fi
  if [[ -n "$backend_pid" ]] && kill -0 "$backend_pid" >/dev/null 2>&1; then
    kill "$backend_pid" >/dev/null 2>&1 || true
  fi
  wait "$frontend_pid" "$backend_pid" >/dev/null 2>&1 || true
}

main() {
  require_command go
  require_command bun

  ensure_embed_placeholders
  ensure_frontend_deps

  log "backend:  ${BACKEND_URL}"
  log "frontend: http://localhost:${FRONTEND_PORT}"

  (
    cd "$ROOT_DIR"
    PORT="$BACKEND_PORT" GIN_MODE="${GIN_MODE:-debug}" go run main.go
  ) &
  backend_pid="$!"

  (
    cd "$FRONTEND_DIR"
    VITE_REACT_APP_SERVER_URL="$BACKEND_URL" bun run dev -- --host 0.0.0.0 --port "$FRONTEND_PORT"
  ) &
  frontend_pid="$!"

  trap cleanup EXIT INT TERM

  wait -n "$backend_pid" "$frontend_pid"
}

main "$@"
