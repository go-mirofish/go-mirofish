#!/usr/bin/env bash
# Bootstrap: install Node deps for Vue build tooling only.
# Go binaries are built via `go build` or docker compose.
# Python is not a product dependency.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/dev/common.sh
source "$SCRIPT_DIR/common.sh"

install_frontend_deps() {
  run_in_root npm install --prefix frontend
}

case "${1:-all}" in
  all)
    install_frontend_deps
    ;;
  --js-only|js|frontend)
    install_frontend_deps
    ;;
  *)
    echo "usage: $0 [all|--js-only|frontend]" >&2
    exit 1
    ;;
esac
