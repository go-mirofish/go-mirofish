#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

run_gateway_tests() {
  local root
  root="$(repo_root)"
  cd "$root/gateway"
  GOCACHE="${GOCACHE:-/tmp/go-build-cache}" \
  GOMODCACHE="${GOMODCACHE:-/tmp/go-mod-cache}" \
  go test ./...
}

run_backend_tests() {
  local root py status
  root="$(repo_root)"

  if have_cmd uv; then
    cd "$root/backend"
    set +e
    uv run pytest
    status=$?
    set -e
  elif py="$(backend_python)"; then
    cd "$root/backend"
    set +e
    "$py" -m pytest
    status=$?
    set -e
  else
    log "test" "backend runtime not bootstrapped; skipping backend pytest (run make bootstrap)"
    return 0
  fi

  if [[ $status -eq 0 || $status -eq 5 ]]; then
    if [[ $status -eq 5 ]]; then
      log "test" "backend pytest reported no tests collected; skipping"
    fi
    return 0
  fi

  return "$status"
}

run_frontend_smoke() {
  "$SCRIPT_DIR/frontend.sh" build
}

case "${1:-all}" in
  all)
    run_gateway_tests
    run_backend_tests
    run_frontend_smoke
    ;;
  gateway)
    run_gateway_tests
    ;;
  backend)
    run_backend_tests
    ;;
  frontend)
    run_frontend_smoke
    ;;
  *)
    echo "usage: $0 [all|gateway|backend|frontend]" >&2
    exit 1
    ;;
esac
