#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
BACKEND_VENV="$BACKEND_DIR/.venv"
FRONTEND_DIST_DIR="${FRONTEND_DIST_DIR:-$ROOT_DIR/frontend/dist}"
BIND_HOST="${BIND_HOST:-127.0.0.1}"
BACKEND_PORT="${BACKEND_PORT:-5001}"
GATEWAY_PORT="${GATEWAY_PORT:-3000}"

backend_pid=""

is_windows_shell() {
  case "${OSTYPE:-}" in
    msys*|cygwin*|win32*) return 0 ;;
    *) return 1 ;;
  esac
}

default_gateway_bin() {
  if is_windows_shell; then
    printf '%s\n' "$ROOT_DIR/gateway/bin/go-mirofish-gateway.exe"
  else
    printf '%s\n' "$ROOT_DIR/gateway/bin/go-mirofish-gateway"
  fi
}

GATEWAY_BIN="${GATEWAY_BIN:-$(default_gateway_bin)}"

cleanup() {
  if [[ -n "$backend_pid" ]] && kill -0 "$backend_pid" 2>/dev/null; then
    kill "$backend_pid" >/dev/null 2>&1 || true
    wait "$backend_pid" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

require_file() {
  local target="$1"
  local message="$2"
  if [[ ! -e "$target" ]]; then
    echo "error: $message" >&2
    exit 1
  fi
}

wait_for_backend() {
  local url="http://127.0.0.1:${BACKEND_PORT}/health"
  local attempts=0

  until curl --silent --fail "$url" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if (( attempts > 60 )); then
      echo "error: backend did not become healthy at $url" >&2
      exit 1
    fi
    sleep 1
  done
}

ensure_backend_python() {
  local python_path=""
  local pip_path=""

  if [[ -x "$BACKEND_VENV/bin/python" ]]; then
    python_path="$BACKEND_VENV/bin/python"
    pip_path="$BACKEND_VENV/bin/pip"
  elif [[ -x "$BACKEND_VENV/Scripts/python.exe" ]]; then
    python_path="$BACKEND_VENV/Scripts/python.exe"
    pip_path="$BACKEND_VENV/Scripts/pip.exe"
  elif [[ -x "$BACKEND_VENV/Scripts/python" ]]; then
    python_path="$BACKEND_VENV/Scripts/python"
    pip_path="$BACKEND_VENV/Scripts/pip"
  fi

  if [[ -n "$python_path" ]]; then
    printf '%s\n' "$python_path"
    return
  fi

  local host_python=""
  if command -v python3.11 >/dev/null 2>&1; then
    host_python="$(command -v python3.11)"
  elif command -v python3 >/dev/null 2>&1; then
    host_python="$(command -v python3)"
  elif command -v python >/dev/null 2>&1; then
    host_python="$(command -v python)"
  fi

  if [[ -z "$host_python" ]]; then
    echo "error: python 3.11.x is required to create backend/.venv (3.12+ is not supported yet; see docs/getting-started/installation.md#python-uv-and-venv)" >&2
    exit 1
  fi

  "$host_python" -m venv "$BACKEND_VENV"

  if [[ -x "$BACKEND_VENV/bin/python" ]]; then
    python_path="$BACKEND_VENV/bin/python"
    pip_path="$BACKEND_VENV/bin/pip"
  elif [[ -x "$BACKEND_VENV/Scripts/python.exe" ]]; then
    python_path="$BACKEND_VENV/Scripts/python.exe"
    pip_path="$BACKEND_VENV/Scripts/pip.exe"
  elif [[ -x "$BACKEND_VENV/Scripts/python" ]]; then
    python_path="$BACKEND_VENV/Scripts/python"
    pip_path="$BACKEND_VENV/Scripts/pip"
  fi

  if [[ -z "$python_path" || -z "$pip_path" ]]; then
    echo "error: failed to locate backend virtualenv executables in $BACKEND_VENV" >&2
    exit 1
  fi

  "$pip_path" install --upgrade pip >/dev/null
  "$pip_path" install -r "$BACKEND_DIR/requirements.txt"

  printf '%s\n' "$python_path"
}

require_file "$GATEWAY_BIN" "prebuilt gateway binary not found at $GATEWAY_BIN"
require_file "$FRONTEND_DIST_DIR/index.html" "frontend build output not found at $FRONTEND_DIST_DIR/index.html"

BACKEND_PYTHON="$(ensure_backend_python)"

(
  cd "$BACKEND_DIR"
  exec "$BACKEND_PYTHON" run.py
) &
backend_pid="$!"

wait_for_backend

export BACKEND_URL="http://127.0.0.1:${BACKEND_PORT}"
export FRONTEND_DIST_DIR
export GATEWAY_BIND_HOST="$BIND_HOST"
export GATEWAY_PORT

exec "$GATEWAY_BIN"
